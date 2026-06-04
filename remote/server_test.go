package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerApprovalFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, clientToken, err := PairClient("phone")
	if err != nil {
		t.Fatalf("PairClient failed: %v", err)
	}

	server := httptest.NewServer(NewServer("local-token"))
	defer server.Close()

	request := NewApprovalRequest("git", []string{"commit", "-m", "test"}, "git:commit", time.Minute)
	resultCh := make(chan ApprovalResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		var result ApprovalResponse
		body, err := json.Marshal(request)
		if err != nil {
			errCh <- err
			return
		}
		req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/requests", bytes.NewReader(body))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Zuko-Local-Token", "local-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("create request returned HTTP %d", resp.StatusCode)
			return
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	approval := waitForApproval(t, server.URL, clientToken)
	if approval.ID != request.ID {
		t.Fatalf("approval ID = %q, want %q", approval.ID, request.ID)
	}

	decisionBody := []byte(`{"decision":"approve"}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/approvals/"+request.ID+"/decision", bytes.NewReader(decisionBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+clientToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("decision returned HTTP %d", resp.StatusCode)
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case result := <-resultCh:
		if result.Decision != DecisionApprove {
			t.Fatalf("decision = %q, want %q", result.Decision, DecisionApprove)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for approval response")
	}
}

func TestServerRejectsInvalidRemoteToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(NewServer("local-token"))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/approvals", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer invalid")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func waitForApproval(t *testing.T, serverURL, token string) ApprovalRequest {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, serverURL+"/v1/approvals", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var payload struct {
			Approvals []ApprovalRequest `json:"approvals"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list returned HTTP %d", resp.StatusCode)
		}
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if len(payload.Approvals) > 0 {
			return payload.Approvals[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for pending approval")
	return ApprovalRequest{}
}
