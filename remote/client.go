package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func RequestApproval(ctx context.Context, req ApprovalRequest) (Decision, error) {
	state, err := LoadServeState()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(state.URL, "/") + "/v1/requests"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Zuko-Local-Token", state.LocalToken)

	client := &http.Client{Timeout: time.Until(req.ExpiresAt) + 5*time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return DecisionTimeout, nil
		}
		return "", ErrNoServer
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrNoServer
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("zuko serve returned HTTP %d", resp.StatusCode)
	}

	var result ApprovalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Decision, nil
}
