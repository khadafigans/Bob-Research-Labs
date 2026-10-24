package agent

// anthropic_oauth.go — Claude Code / claude.ai OAuth token (sk-ant-oat01-*) support.
//
// Anthropic Console API keys (sk-ant-api*) use x-api-key. Claude Code OAuth tokens
// require Authorization: Bearer plus anthropic-beta oauth capability, and for every
// non-Haiku model the Messages API also requires the system prompt to begin with the
// Claude Code identity string (as its own first system block).
//
// We reuse FormatAnthropic (norma always sets x-api-key) and wrap the HTTP client
// with AnthropicOAuthTransport, which:
//  1. Replaces x-api-key with Authorization: Bearer <token>
//  2. Sets anthropic-beta: claude-code-20250219,oauth-2025-04-20
//  3. Prepends the Claude Code identity as the first system content block

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	claudeCodeOAuthSystemIdentity = "You are Claude Code, Anthropic's official CLI for Claude."
	anthropicOAuthBeta            = "claude-code-20250219,oauth-2025-04-20"
)

// AnthropicOAuthTransport rewrites Anthropic Messages requests for Claude Code OAuth.
type AnthropicOAuthTransport struct {
	Token string
	Inner http.RoundTripper
}

// RoundTrip rewrites auth headers and injects the Claude Code system identity.
func (t *AnthropicOAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())

	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("anthropic-oauth: read body: %w", err)
		}
		_ = req.Body.Close()
	}

	token := strings.TrimSpace(t.Token)
	if token == "" {
		token = strings.TrimSpace(clone.Header.Get("x-api-key"))
	}
	clone.Header.Del("x-api-key")
	if token != "" {
		clone.Header.Set("Authorization", "Bearer "+token)
	}
	clone.Header.Set("anthropic-beta", anthropicOAuthBeta)
	if clone.Header.Get("User-Agent") == "" {
		clone.Header.Set("User-Agent", "claude-cli/2.1.85 (external, cli)")
	}
	if clone.Header.Get("x-app") == "" {
		clone.Header.Set("x-app", "cli")
	}

	if len(bodyBytes) > 0 {
		if rewritten, err := injectClaudeCodeSystem(bodyBytes); err == nil {
			bodyBytes = rewritten
		}
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		clone.ContentLength = int64(len(bodyBytes))
		clone.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
	} else {
		clone.Body = http.NoBody
		clone.ContentLength = 0
	}

	inner := t.Inner
	if inner == nil {
		inner = http.DefaultTransport
	}
	return inner.RoundTrip(clone)
}

// injectClaudeCodeSystem ensures the Claude Code identity is the first system block.
// Norma emits system as [{"type":"text","text":"..."}, ...]; a plain string is also handled.
func injectClaudeCodeSystem(body []byte) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return body, err
	}

	identityBlock, err := json.Marshal(map[string]string{
		"type": "text",
		"text": claudeCodeOAuthSystemIdentity,
	})
	if err != nil {
		return body, err
	}

	sysRaw, hasSys := root["system"]
	if !hasSys || len(sysRaw) == 0 || string(sysRaw) == "null" {
		arr, err := json.Marshal([]json.RawMessage{identityBlock})
		if err != nil {
			return body, err
		}
		root["system"] = arr
		return json.Marshal(root)
	}

	// Array of content blocks (norma path).
	var blocks []json.RawMessage
	if err := json.Unmarshal(sysRaw, &blocks); err == nil {
		if len(blocks) > 0 && systemBlockHasIdentity(blocks[0]) {
			return body, nil
		}
		out := make([]json.RawMessage, 0, len(blocks)+1)
		out = append(out, identityBlock)
		out = append(out, blocks...)
		arr, err := json.Marshal(out)
		if err != nil {
			return body, err
		}
		root["system"] = arr
		return json.Marshal(root)
	}

	// Plain string system.
	var s string
	if err := json.Unmarshal(sysRaw, &s); err != nil {
		return body, nil // unknown shape — leave alone
	}
	if strings.HasPrefix(s, claudeCodeOAuthSystemIdentity) {
		return body, nil
	}
	textBlock, err := json.Marshal(map[string]string{"type": "text", "text": s})
	if err != nil {
		return body, err
	}
	arr, err := json.Marshal([]json.RawMessage{identityBlock, textBlock})
	if err != nil {
		return body, err
	}
	root["system"] = arr
	return json.Marshal(root)
}

func systemBlockHasIdentity(block json.RawMessage) bool {
	var b struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(block, &b); err != nil {
		return false
	}
	return strings.HasPrefix(b.Text, claudeCodeOAuthSystemIdentity)
}
