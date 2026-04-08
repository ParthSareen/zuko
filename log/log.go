package log

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Timestamp time.Time `json:"ts"`
	Tool      string    `json:"tool"`
	Args      []string  `json:"args,omitempty"`
	Action    string    `json:"action"` // "blocked", "granted", "allowed", "auth_failed"
	Scope     string    `json:"scope,omitempty"`
	Error     string    `json:"error,omitempty"`
	PID       int       `json:"pid"`
	PPID      int       `json:"ppid"`
	Process   string    `json:"proc,omitempty"`
	Parent    string    `json:"parent,omitempty"`
}

func LogsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zuko", "logs.jsonl")
}

// Write appends a log entry to the logs file.
func Write(entry Entry) {
	entry.Timestamp = time.Now()
	entry.PID = os.Getpid()
	entry.PPID = os.Getppid()
	entry.Process = getProcessName(entry.PID)
	entry.Parent = getProcessName(entry.PPID)

	f, err := os.OpenFile(LogsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.Encode(entry)

	// Rotate logs asynchronously if file is getting large (>10000 lines)
	if info, err := f.Stat(); err == nil && info.Size() > 2*1024*1024 {
		go Rotate(1000)
	}
}

func getProcessName(pid int) string {
	var name string
	if runtime.GOOS == "darwin" {
		// macOS: use ps
		out, _ := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
		name = strings.TrimSpace(string(out))
	} else {
		// Linux: read from /proc
		data, _ := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
		name = strings.TrimSpace(string(data))
	}
	// Get just the basename
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		return name[idx+1:]
	}
	return name
}

// Read returns all log entries (newest first), limited to maxEntries.
func Read(maxEntries int) ([]Entry, error) {
	data, err := os.ReadFile(LogsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	// Reverse to get newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if maxEntries > 0 && len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	return entries, nil
}

// Clear removes all logs.
func Clear() error {
	return os.Remove(LogsPath())
}

// Rotate truncates the log file to keep only the last maxEntries entries.
func Rotate(maxEntries int) error {
	if maxEntries <= 0 {
		return nil
	}

	data, err := os.ReadFile(LogsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := splitLines(data)
	if len(lines) <= maxEntries {
		return nil
	}

	// Keep only the last maxEntries lines
	start := len(lines) - maxEntries
	kept := lines[start:]

	// Rewrite file with kept entries
	f, err := os.Create(LogsPath())
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range kept {
		if len(line) == 0 {
			continue
		}
		f.Write(line)
		f.Write([]byte{'\n'})
	}
	return nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// FormatEntry returns a human-readable string for an entry.
func FormatEntry(e Entry) string {
	cmd := e.Tool
	if len(e.Args) > 0 {
		cmd = fmt.Sprintf("%s %s", e.Tool, joinArgs(e.Args))
	}

	who := e.Process
	if who == "" {
		who = fmt.Sprintf("pid:%d", e.PID)
	}
	if e.Parent != "" {
		who = fmt.Sprintf("%s (%s)", who, e.Parent)
	}

	switch e.Action {
	case "blocked":
		return fmt.Sprintf("[%s] BLOCKED: %s (scope: %s) [%s]", formatTime(e.Timestamp), cmd, e.Scope, who)
	case "blocked_dangerous":
		return fmt.Sprintf("[%s] BLOCKED_DANGEROUS: %s (scope: %s) [%s] - clipboard only", formatTime(e.Timestamp), cmd, e.Scope, who)
	case "granted":
		return fmt.Sprintf("[%s] GRANTED: %s (scope: %s) [%s]", formatTime(e.Timestamp), cmd, e.Scope, who)
	case "allowed":
		return fmt.Sprintf("[%s] ALLOWED: %s [%s]", formatTime(e.Timestamp), cmd, who)
	case "auth_failed":
		return fmt.Sprintf("[%s] AUTH_FAILED: %s (%s) [%s]", formatTime(e.Timestamp), cmd, e.Error, who)
	default:
		return fmt.Sprintf("[%s] %s: %s [%s]", formatTime(e.Timestamp), e.Action, cmd, who)
	}
}

func formatTime(t time.Time) string {
	return t.Format("15:04:05")
}

func joinArgs(args []string) string {
	var b strings.Builder
	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		if needsQuote(a) {
			fmt.Fprintf(&b, "%q", a)
		} else {
			b.WriteString(a)
		}
	}
	return b.String()
}

func needsQuote(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '\\' {
			return true
		}
	}
	return false
}
