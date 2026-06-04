package remote

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"
)

type Decision string

const (
	DecisionApprove Decision = "approve"
	DecisionDeny    Decision = "deny"
	DecisionTimeout Decision = "timeout"
)

type ApprovalRequest struct {
	ID        string    `json:"id"`
	Tool      string    `json:"tool"`
	Args      []string  `json:"args"`
	Scope     string    `json:"scope"`
	Command   string    `json:"command"`
	Digest    string    `json:"digest"`
	CWD       string    `json:"cwd,omitempty"`
	PID       int       `json:"pid"`
	PPID      int       `json:"ppid"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ApprovalResponse struct {
	Decision Decision `json:"decision"`
}

func NewApprovalRequest(tool string, args []string, scope string, timeout time.Duration) ApprovalRequest {
	id, _ := NewToken()
	cwd, _ := os.Getwd()
	now := time.Now()
	req := ApprovalRequest{
		ID:        id,
		Tool:      tool,
		Args:      append([]string(nil), args...),
		Scope:     scope,
		Command:   joinCommand(tool, args),
		CWD:       cwd,
		PID:       os.Getpid(),
		PPID:      os.Getppid(),
		CreatedAt: now,
		ExpiresAt: now.Add(timeout),
	}
	req.Digest = commandDigest(req)
	return req
}

func commandDigest(req ApprovalRequest) string {
	parts := []string{req.Tool, req.Scope}
	parts = append(parts, req.Args...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func joinCommand(tool string, args []string) string {
	parts := []string{tool}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\r\n'\"\\$`!*?[]{}()<>|&;") {
		return s
	}
	return strconv.Quote(s)
}
