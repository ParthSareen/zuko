package remote

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Server struct {
	localToken string

	mu      sync.Mutex
	pending map[string]*pendingApproval
}

type pendingApproval struct {
	request  ApprovalRequest
	decision chan Decision
}

func NewServer(localToken string) *Server {
	return &Server{
		localToken: localToken,
		pending:    make(map[string]*pendingApproval),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	case r.Method == http.MethodPost && r.URL.Path == "/v1/requests":
		s.handleCreateRequest(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/v1/approvals":
		s.handleListApprovals(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/approvals/") && strings.HasSuffix(r.URL.Path, "/decision"):
		s.handleDecision(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (s *Server) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Zuko-Local-Token") != s.localToken {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}
	if req.ID == "" || req.Tool == "" || req.Scope == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid approval request"})
		return
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	if req.ExpiresAt.IsZero() || time.Until(req.ExpiresAt) <= 0 {
		req.ExpiresAt = time.Now().Add(2 * time.Minute)
	}

	pending := &pendingApproval{
		request:  req,
		decision: make(chan Decision, 1),
	}
	s.mu.Lock()
	s.pending[req.ID] = pending
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.pending, req.ID)
		s.mu.Unlock()
	}()

	timer := time.NewTimer(time.Until(req.ExpiresAt))
	defer timer.Stop()

	select {
	case decision := <-pending.decision:
		writeJSON(w, http.StatusOK, ApprovalResponse{Decision: decision})
	case <-timer.C:
		writeJSON(w, http.StatusOK, ApprovalResponse{Decision: DecisionTimeout})
	case <-r.Context().Done():
		return
	}
}

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedRemoteClient(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	now := time.Now()
	approvals := make([]ApprovalRequest, 0)
	s.mu.Lock()
	for id, pending := range s.pending {
		if now.After(pending.request.ExpiresAt) {
			delete(s.pending, id)
			continue
		}
		approvals = append(approvals, pending.request)
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"approvals": approvals})
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedRemoteClient(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/approvals/"), "/decision")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing approval id"})
		return
	}

	var payload struct {
		Decision Decision `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}
	if payload.Decision != DecisionApprove && payload.Decision != DecisionDeny {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "decision must be approve or deny"})
		return
	}

	s.mu.Lock()
	pending := s.pending[id]
	s.mu.Unlock()
	if pending == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "approval not found"})
		return
	}

	select {
	case pending.decision <- payload.Decision:
	default:
	}
	writeJSON(w, http.StatusOK, ApprovalResponse{Decision: payload.Decision})
}

func (s *Server) authorizedRemoteClient(r *http.Request) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	token, ok := strings.CutPrefix(auth, "Bearer ")
	return ok && ValidClientToken(strings.TrimSpace(token))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
