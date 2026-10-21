//go:build windows

package clientconfig

// processAlive cannot be probed cheaply without OpenProcess, and a wrong
// answer here would let one process delete another's live lock. Report alive
// and let the age-based fallback in lockIsStale reclaim abandoned locks.
func processAlive(pid int) bool { return true }
