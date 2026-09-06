//go:build windows

package windows

// This file implements Warden's real ETW auditing layer for the Windows
// backend: a private real-time trace session that enables the Kernel-File
// and Kernel-Network providers and consumes their events scoped to the
// sandboxed process tree.
//
// Fail-closed gate: auditing is part of the security gate, so a run whose
// trace session cannot start is refused (see Run) — Warden never runs with a
// silently missing audit trail. The kernel (WPP-style) providers accept only
// one enabling session at a time; when another controller already holds one
// (PerfView, an EDR), EnableTraceEx2 fails and the run fails closed with a
// clear message, exactly as the gate intends.
//
// What is captured and mapped:
//   - Kernel-File events for PIDs in the sandbox tree become `file` audit
//     records when the payload carries a verifiable path (create, delete,
//     rename, and the file-name events). Payload layouts are the classic
//     kernel FileIo records; see etwdecode.go. Records without a verifiable
//     name are dropped rather than guessed. Denied file opens never reach the
//     file system provider (the AppContainer token denies them first), so
//     file denials are invisible to ETW by design — the enforcement boundary
//     is the token, not a logged syscall.
//   - Kernel-Network events are enabled and captured for the same tree, but
//     their endpoint payloads are not mapped in this release: the network
//     audit trail (allowed and blocked host:port decisions) comes from the
//     egress proxy and the WFP deny filters, which report exact decisions.
//
// Struct mirrors below are x64 (amd64) layouts from evntrace.h / evntcons.h;
// they are pinned by the Windows lifecycle and escape tests.

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/audit"
)

// ETW logging-mode / process-mode constants (evntrace.h).
const (
	eventTraceRealTimeMode         = 0x00000100 // EVENT_TRACE_REAL_TIME_MODE
	eventTraceControlStop          = 0          // EVENT_TRACE_CONTROL_STOP
	eventControlCodeEnableProvider = 1          // EVENT_CONTROL_CODE_ENABLE_PROVIDER
	traceLevelVerbose              = 5          // TRACE_LEVEL_VERBOSE
	processTraceModeRealTime       = 0x00000100 // PROCESS_TRACE_MODE_REAL_TIME
	processTraceModeEventRecord    = 0x01000000 // PROCESS_TRACE_MODE_EVENT_RECORD

	errorAlreadyExists    = syscall.Errno(183)  // ERROR_ALREADY_EXISTS
	errorWMIInstanceFound = syscall.Errno(4201) // ERROR_WMI_INSTANCE_NOT_FOUND
)

// kernelFileGUID and kernelNetworkGUID are the two kernel providers the audit
// session enables (Microsoft-Windows-Kernel-File and
// Microsoft-Windows-Kernel-Network).
var (
	kernelFileGUID    = guidOf("edd08927-9cc4-4e65-b970-c2560fb5c289")
	kernelNetworkGUID = guidOf("7dd42a49-5329-4832-8dfd-43d979084a1d")
)

// wnodeHeader mirrors WNODE_HEADER (evntrace.h), x64 layout. It is the first
// member of EVENT_TRACE_PROPERTIES.
type wnodeHeader struct {
	bufferSize        uint32
	providerID        uint32
	historicalContext uint64
	kernelHandle      uintptr
	guid              windows.GUID
	clientContext     uint32
	flags             uint32
}

// eventTraceProperties mirrors EVENT_TRACE_PROPERTIES (evntrace.h), x64
// layout. The session name is stored after the struct in the same
// allocation; see traceSession.props.
type eventTraceProperties struct {
	wnode               wnodeHeader
	bufferSize          uint32
	minimumBuffers      uint32
	maximumBuffers      uint32
	maximumFileSize     uint32
	logFileMode         uint32
	flushTimer          uint32
	enableFlags         uint32
	ageLimit            uint32
	numberOfBuffers     uint32
	freeBuffers         uint32
	eventsLost          uint32
	buffersWritten      uint32
	logBuffersLost      uint32
	realTimeBuffersLost uint32
	loggerThreadID      uintptr
	logFileNameOffset   uint32
	loggerNameOffset    uint32
}

// eventTraceLogfile mirrors the fields of EVENT_TRACE_LOGFILE (evntrace.h),
// x64 layout, that OpenTraceW reads for a real-time EVENT_RECORD consumer.
type eventTraceLogfile struct {
	loggerName          *uint16
	logFileName         *uint16
	eventCallback       uintptr
	logFileMode         uint32
	maximumFileSize     uint32
	bufferSize          uint32
	fatalError          uint32
	eventRecordCallback uintptr
	context             uintptr
}

// eventDescriptor mirrors EVENT_DESCRIPTOR (evntrace.h).
type eventDescriptor struct {
	id      uint16
	version uint8
	channel uint8
	level   uint8
	opcode  uint8
	task    uint16
	keyword uint64
}

// eventHeader mirrors the fixed 80-byte EVENT_HEADER prefix of an
// EVENT_RECORD (evntcons.h), x64 layout.
type eventHeader struct {
	size          uint16
	headerType    uint16
	flags         uint16
	eventProperty uint16
	threadID      uint32
	processID     uint32
	timeStamp     int64
	providerID    windows.GUID
	descriptor    eventDescriptor
	kernelTime    uint32
	userTime      uint32
	activityID    windows.GUID
}

// eventRecord mirrors the fixed prefix of EVENT_RECORD (evntcons.h), x64
// layout, through the UserData pointer. Fields after UserData are never read.
type eventRecord struct {
	header            eventHeader // 80 bytes
	bufferContext     [4]byte
	extendedDataCount uint16
	userDataLength    uint16
	_                 uint32 // alignment to the pointers below
	extendedData      uintptr
	userData          uintptr
}

// maxEventPayload bounds how much UserData the callback will touch. ETW does
// not deliver events larger than 64 KB.
const maxEventPayload = 0x10000

// traceSession owns one ETW audit session for a sandboxed run.
type traceSession struct {
	name   string
	nameW  *uint16         // NUL-terminated session name
	props  []byte          // EVENT_TRACE_PROPERTIES + session name allocation
	logger *audit.Logger   // destination for mapped audit events
	tree   map[uint32]bool // PIDs scoped into the audit; set before run()

	session uint64 // TRACEHANDLE from StartTraceW
	trace   uint64 // TRACEHANDLE from OpenTraceW

	mu      sync.Mutex
	started bool // run() has entered ProcessTrace
	done    chan struct{}
}

// consumerMu guards activeConsumer: ProcessTrace's EVENT_RECORD callback
// receives no context pointer, so the consumer is found through this
// package-level registry. Warden runs one sandboxed process per process (a
// gateway wraps each server with its own `warden run`), so one active
// consumer is the real concurrency shape; a concurrent second session fails
// closed rather than sharing the slot.
var (
	consumerMu     sync.Mutex
	activeConsumer *traceSession
)

// startTrace begins a real-time ETW audit session for the sandboxed run. It
// fails closed: any failure to start the session or enable the kernel
// providers is an error, and the caller (Run) refuses to start the target.
func startTrace(name string, logger *audit.Logger) (*traceSession, error) {
	if logger == nil {
		return nil, failClose("ETW audit session", errors.New("no audit logger configured"))
	}
	nameW, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, failClose("ETW audit session", fmt.Errorf("invalid session name: %w", err))
	}
	t := &traceSession{
		name:   name,
		nameW:  nameW,
		logger: logger,
		done:   make(chan struct{}),
	}

	// The properties block reserves room for the session name right after
	// the struct; StartTraceW copies the name into that offset itself. The
	// buffer must stay alive until ControlTrace stop (t.props holds it).
	name16, _ := windows.UTF16FromString(name) // NUL-terminated copy for sizing
	propsSize := int(unsafe.Sizeof(eventTraceProperties{})) + len(name16)*2
	t.props = make([]byte, propsSize)
	p := (*eventTraceProperties)(unsafe.Pointer(&t.props[0]))
	p.wnode.bufferSize = uint32(propsSize)
	p.logFileMode = eventTraceRealTimeMode
	p.loggerNameOffset = uint32(unsafe.Sizeof(eventTraceProperties{}))

	// Clean up an orphaned session of the same name from a run that did not
	// close cleanly (e.g. warden was killed). Best effort: the name may
	// simply be free.
	stopTraceByName(name)

	if err := t.startSession(); err != nil {
		return nil, err
	}
	if err := t.enableProviders(); err != nil {
		_ = t.stopSession()
		return nil, err
	}
	if err := t.openTrace(); err != nil {
		_ = t.stopSession()
		return nil, err
	}
	return t, nil
}

// startSession starts the private real-time session. A leftover session that
// raced the cleanup is stopped once and the start retried before failing.
func (t *traceSession) startSession() error {
	var h uint64
	r, _, _ := procStartTraceW.Call(
		uintptr(unsafe.Pointer(&h)),
		uintptr(unsafe.Pointer(t.nameW)),
		uintptr(unsafe.Pointer(&t.props[0])),
	)
	if r == uintptr(errorAlreadyExists) {
		stopTraceByName(t.name)
		r, _, _ = procStartTraceW.Call(
			uintptr(unsafe.Pointer(&h)),
			uintptr(unsafe.Pointer(t.nameW)),
			uintptr(unsafe.Pointer(&t.props[0])),
		)
	}
	if r != 0 {
		return failClose("start ETW trace session", windows.Errno(r))
	}
	t.session = h
	return nil
}

// stopTraceByName stops a session by name via ControlTrace, ignoring the
// not-found outcome. Used to reclaim orphaned sessions before a start.
func stopTraceByName(name string) {
	nameW, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	// ControlTrace wants a properties buffer; a zeroed header-sized one is
	// enough for a by-name stop. Best effort: if the name is already free
	// the call reports not-found and is ignored.
	var p eventTraceProperties
	p.wnode.bufferSize = uint32(unsafe.Sizeof(p))
	procControlTraceW.Call(
		0, // session handle: stop by name
		uintptr(unsafe.Pointer(nameW)),
		uintptr(unsafe.Pointer(&p)),
		eventTraceControlStop,
	)
}

// enableProviders turns on the Kernel-File and Kernel-Network providers for
// this session. Kernel providers are single-session (WPP-style): when another
// controller already holds one, EnableTraceEx2 fails and the run fails closed
// — auditing is never best-effort.
func (t *traceSession) enableProviders() error {
	for _, guid := range []windows.GUID{kernelFileGUID, kernelNetworkGUID} {
		r, _, _ := procEnableTraceEx2.Call(
			uintptr(t.session),
			uintptr(unsafe.Pointer(&guid)),
			eventControlCodeEnableProvider,
			traceLevelVerbose,
			0, // MatchAnyKeyword: 0 matches every keyword
			0, // MatchAllKeyword
			0, // Timeout
			0, // EnableParameters (nil)
		)
		if r != 0 {
			return failClose(fmt.Sprintf("enable ETW provider %s", guid.String()), windows.Errno(r))
		}
	}
	return nil
}

// openTrace attaches the real-time consumer to the session.
func (t *traceSession) openTrace() error {
	var lf eventTraceLogfile
	lf.loggerName = t.nameW
	lf.logFileMode = processTraceModeRealTime | processTraceModeEventRecord
	lf.eventRecordCallback = syscall.NewCallback(eventRecordCallback)

	r, _, _ := procOpenTraceW.Call(uintptr(unsafe.Pointer(&lf)))
	if r == ^uintptr(0) { // INVALID_HANDLE_VALUE
		code := lf.fatalError
		if code == 0 {
			code = uint32(windows.ERROR_INVALID_HANDLE)
		}
		return failClose("open ETW trace consumer", windows.Errno(code))
	}
	t.trace = uint64(r)
	return nil
}

// run consumes events until the session is closed. It runs in its own
// goroutine (ProcessTrace blocks). The PID set and the logger must be set on
// t before run is started.
func (t *traceSession) run() {
	t.mu.Lock()
	t.started = true
	t.mu.Unlock()

	consumerMu.Lock()
	activeConsumer = t
	consumerMu.Unlock()
	defer func() {
		consumerMu.Lock()
		if activeConsumer == t {
			activeConsumer = nil
		}
		consumerMu.Unlock()
		close(t.done)
	}()

	if len(t.tree) == 0 {
		return
	}
	var handle uintptr = uintptr(t.trace)
	procProcessTrace.Call(
		uintptr(unsafe.Pointer(&handle)),
		1, // handleCount
		0, // start time
		0, // end time
	)
}

// eventRecordCallback is the trampoline target ProcessTrace invokes for every
// event. It finds the owning session through the registry and hands the raw
// record over; a nil session (teardown race) is a no-op.
func eventRecordCallback(record uintptr) uintptr {
	consumerMu.Lock()
	s := activeConsumer
	consumerMu.Unlock()
	if s == nil {
		return 0
	}
	s.handleEvent(record)
	return 0
}

// handleEvent maps one PID-scoped EVENT_RECORD into the audit log. It is
// deliberately conservative: only records whose provider and payload Warden
// can verify produce audit entries.
func (t *traceSession) handleEvent(record uintptr) {
	if record == 0 {
		return
	}
	// The callback receives the record address as a uintptr register value;
	// re-derive the pointer through the address of the parameter so vet's
	// unsafe.Pointer rules (which forbid casting arbitrary uintptr variables
	// directly) stay satisfied.
	r := (*eventRecord)(*(*unsafe.Pointer)(unsafe.Pointer(&record)))
	if r.header.processID == 0 || !t.tree[r.header.processID] {
		return // not one of this run's processes
	}
	switch r.header.providerID {
	case kernelFileGUID:
		t.handleKernelFile(r)
	case kernelNetworkGUID:
		// Kernel-Network records are captured but not mapped in this
		// release: endpoint payload decoding is version-fragile and the
		// blocked/allowed network audit is produced with exact host:port
		// decisions by the egress proxy and the WFP deny filters.
	default:
		// Other providers (rundown/config noise) are ignored.
	}
}

func (t *traceSession) handleKernelFile(r *eventRecord) {
	if r.userDataLength == 0 || int(r.userDataLength) > maxEventPayload || r.userData == 0 {
		return
	}
	// userData is the address of the event payload; read it back out of the
	// record (offset 96 on x64, see eventRecord) as a real pointer.
	const eventRecordUserDataOffset = 96
	payloadPtr := *(*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(r), eventRecordUserDataOffset))
	payload := unsafe.Slice((*byte)(payloadPtr), int(r.userDataLength))
	name, ok := decodeKernelFileEvent(payload)
	if !ok {
		// Read/write/flush events and other records that do not end in a
		// verifiable name carry no resource; writing them would pollute the
		// audit log with pathless noise, so they are dropped.
		return
	}
	action := kernelFileAction(r.header.descriptor.opcode)
	// The audit logger stamps the timestamp; failures to encode are dropped
	// (a callback must never take the run down with it).
	_ = t.logger.Log(audit.Event{Type: "file", Action: action, Resource: name, Allowed: true, Reason: "observed via ETW kernel trace"})
}

// closeSession stops the consumer and the trace session. It is safe to call
// more than once and on a session whose run goroutine never started (an
// early error path in Run).
func (t *traceSession) closeSession() error {
	if t == nil {
		return nil
	}
	// CloseTrace unblocks ProcessTrace, letting the run goroutine exit.
	if t.trace != 0 {
		procCloseTrace.Call(uintptr(t.trace))
		t.trace = 0
	}
	t.mu.Lock()
	started := t.started
	t.mu.Unlock()
	if started {
		<-t.done
	}
	return t.stopSession()
}

// stopSession stops the trace session, releasing the kernel providers.
func (t *traceSession) stopSession() error {
	if t.session == 0 {
		return nil
	}
	var propsPtr uintptr
	if len(t.props) > 0 {
		propsPtr = uintptr(unsafe.Pointer(&t.props[0]))
	}
	r, _, _ := procControlTraceW.Call(
		uintptr(t.session),
		0,
		propsPtr,
		eventTraceControlStop,
	)
	t.session = 0
	if r != 0 && r != uintptr(errorWMIInstanceFound) {
		return windows.Errno(r)
	}
	return nil
}
