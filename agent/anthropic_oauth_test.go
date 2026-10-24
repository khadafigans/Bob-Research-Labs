package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInjectClaudeCodeSystem_PrependsIdentity(t *testing.T) {
	in := []byte(`{"model":"claude-sonnet-4-5","system":[{"type":"text","text":"You are a connection test."}],"messages":[]}`)
	out, err := injectClaudeCodeSystem(in)
	if err != nil {
		t.Fatal(err)
	}
	var root struct {
		System []struct {
			Text string `json:"text"`
		} `json:"system"`
	}
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatal(err)
	}
	if len(root.System) != 2 {
		t.Fatalf("got %d blocks, want 2", len(root.System))
	}
	if root.System[0].Text != claudeCodeOAuthSystemIdentity {
		t.Fatalf("first block = %q", root.System[0].Text)
	}
	if root.System[1].Text != "You are a connection test." {
		t.Fatalf("second block = %q", root.System[1].Text)
	}
}

func TestInjectClaudeCodeSystem_Idempotent(t *testing.T) {
	in := []byte(`{"system":[{"type":"text","text":"You are Claude Code, Anthropic's official CLI for Claude."},{"type":"text","text":"extra"}]}`)
	out, err := injectClaudeCodeSystem(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Fatalf("expected unchanged body, got %s", out)
	}
}

func TestAnthropicOAuthTransport_RewritesAuth(t *testing.T) {
	var gotAuth, gotKey, gotBeta string
	var gotBody []byte
	inner := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("x-api-key")
		gotBeta = r.Header.Get("anthropic-beta")
		gotBody, _ = io.ReadAll(r.Body)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})
	tr := &AnthropicOAuthTransport{Token: "sk-ant-oat01-test", Inner: inner}
	req, _ := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", strings.NewReader(
		`{"model":"m","system":[{"type":"text","text":"hi"}],"messages":[]}`,
	))
	req.Header.Set("x-api-key", "sk-ant-oat01-test")
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if gotKey != "" {
		t.Fatalf("x-api-key still set: %q", gotKey)
	}
	if gotAuth != "Bearer sk-ant-oat01-test" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotBeta != anthropicOAuthBeta {
		t.Fatalf("anthropic-beta = %q", gotBeta)
	}
	if !strings.Contains(string(gotBody), claudeCodeOAuthSystemIdentity) {
		t.Fatalf("body missing identity: %s", gotBody)
	}
}
