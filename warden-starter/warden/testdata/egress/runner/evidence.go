package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Classification values. WARN, UNSUPPORTED, BLOCKED and NOT_RUN never count
// as PASS in any summary produced by this runner.
const (
	ClassPass        = "PASS"
	ClassFail        = "FAIL"
	ClassWarn        = "WARN"
	ClassBlocked     = "BLOCKED"
	ClassUnsupported = "UNSUPPORTED"
	ClassNotRun      = "NOT_RUN"
)

// RunMeta is written once per runner invocation.
type RunMeta struct {
	RunID        string         `json:"run_id"`
	Stamp        string         `json:"stamp"`
	StartedAt    time.Time      `json:"started_at"`
	FinishedAt   time.Time      `json:"finished_at,omitempty"`
	OS           string         `json:"os"`
	Arch         string         `json:"arch"`
	Kernel       string         `json:"kernel,omitempty"`
	Platform     string         `json:"platform"`
	Backend      string         `json:"backend"`
	Mode         string         `json:"mode"`
	Privilege    string         `json:"privilege"`
	WardenPath   string         `json:"warden_path"`
	WardenVer    string         `json:"warden_version,omitempty"`
	WardenCommit string         `json:"warden_commit,omitempty"`
	ProbePath    string         `json:"probe_path"`
	MatrixPath   string         `json:"matrix_path"`
	Command      []string       `json:"command"`
	Capabilities map[string]any `json:"capabilities"`
	Zone         string         `json:"zone"`
	Limitations  []string       `json:"limitations,omitempty"`
}

// CheckResult is one evaluated assertion inside a case.
type CheckResult struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Required    bool   `json:"required"`
	Unsupported bool   `json:"unsupported,omitempty"`
	WarnOnly    bool   `json:"warn_only,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

// CaseResult is the complete machine-readable outcome for one matrix row.
type CaseResult struct {
	ID             string            `json:"id"`
	Group          string            `json:"group"`
	Title          string            `json:"title"`
	Kind           string            `json:"kind,omitempty"`
	Backend        string            `json:"backend"`
	Mode           string            `json:"mode"`
	Policy         string            `json:"policy,omitempty"`
	Command        string            `json:"command,omitempty"`
	Expected       string            `json:"expected"`
	StartedAt      time.Time         `json:"started_at"`
	FinishedAt     time.Time         `json:"finished_at"`
	DurationMS     int64             `json:"duration_ms"`
	ExitCode       int               `json:"exit_code"`
	TimedOut       bool              `json:"timed_out,omitempty"`
	ProbeEvents    []map[string]any  `json:"probe_events,omitempty"`
	AuditEvents    []map[string]any  `json:"audit_events,omitempty"`
	CollectorHits  []Hit             `json:"collector_hits,omitempty"`
	DNSHits        []DNSHit          `json:"dns_hits,omitempty"`
	Checks         []CheckResult     `json:"checks,omitempty"`
	Classification string            `json:"classification"`
	Reason         string            `json:"reason,omitempty"`
	Artifacts      map[string]string `json:"artifacts,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	PositiveMarker bool              `json:"positive_marker"`
}

// Summary aggregates a run.
type Summary struct {
	RunID         string         `json:"run_id"`
	Backend       string         `json:"backend"`
	Mode          string         `json:"mode"`
	Counts        map[string]int `json:"counts"`
	Total         int            `json:"total"`
	Verdict       string         `json:"verdict"`
	VerdictReason string         `json:"verdict_reason"`
	Results       []CaseResult   `json:"results"`
}

type evidenceWriter struct {
	dir      string
	casesDir string
	jsonl    *os.File
}

func newEvidenceWriter(dir string) (*evidenceWriter, error) {
	casesDir := filepath.Join(dir, "cases")
	if err := os.MkdirAll(casesDir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "results.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &evidenceWriter{dir: dir, casesDir: casesDir, jsonl: f}, nil
}

func (w *evidenceWriter) close() {
	if w.jsonl != nil {
		_ = w.jsonl.Close()
	}
}

func (w *evidenceWriter) caseDir(id string) string {
	return filepath.Join(w.casesDir, id)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func (w *evidenceWriter) writeCase(r CaseResult) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.jsonl.Write(append(b, '\n')); err != nil {
		return err
	}
	return writeJSON(filepath.Join(w.caseDir(r.ID), "result.json"), r)
}

func (w *evidenceWriter) writeRunMeta(m RunMeta) error {
	return writeJSON(filepath.Join(w.dir, "run.json"), m)
}

func classifyCounts(results []CaseResult) map[string]int {
	counts := map[string]int{
		ClassPass: 0, ClassFail: 0, ClassWarn: 0,
		ClassBlocked: 0, ClassUnsupported: 0, ClassNotRun: 0,
	}
	for _, r := range results {
		counts[r.Classification]++
	}
	return counts
}

// verdict derives the per-backend answer. It never upgrades WARN,
// UNSUPPORTED, BLOCKED or NOT_RUN to PASS.
func verdict(results []CaseResult) (string, string) {
	counts := classifyCounts(results)
	var fails, blocked, unsupported, notRun []string
	for _, r := range results {
		switch r.Classification {
		case ClassFail:
			fails = append(fails, r.ID)
		case ClassBlocked:
			blocked = append(blocked, r.ID)
		case ClassUnsupported:
			unsupported = append(unsupported, r.ID)
		case ClassNotRun:
			notRun = append(notRun, r.ID)
		}
	}
	sort.Strings(fails)
	if len(fails) > 0 {
		return "NO", fmt.Sprintf("%d failing case(s): %s", len(fails), strings.Join(fails, ", "))
	}
	if len(blocked) > 0 || len(notRun) > 0 {
		return "INCONCLUSIVE", fmt.Sprintf("PASS=%d WARN=%d but %d BLOCKED and %d NOT_RUN case(s) remain: %s %s",
			counts[ClassPass], counts[ClassWarn], len(blocked), len(notRun),
			strings.Join(blocked, ","), strings.Join(notRun, ","))
	}
	return "YES", fmt.Sprintf("no failing case; PASS=%d WARN=%d UNSUPPORTED=%d (WARN and UNSUPPORTED cases are excluded from the security claim, never counted as PASS)",
		counts[ClassPass], counts[ClassWarn], len(unsupported))
}
