package agent

import (
	"strings"
	"testing"
)

func TestRenderSystemOverrideAndFallback(t *testing.T) {
	t.Cleanup(func() { PromptOverride = nil })

	// no override → built-in default
	PromptOverride = nil
	if got := renderSystem("planner", "DEFAULT", PlannerVars{Goal: "g"}); got != "DEFAULT" {
		t.Fatalf("no override should give default, got %q", got)
	}

	// override → rendered with vars
	PromptOverride = func(k string) (string, bool) {
		if k == "planner" {
			return ":{{.Goal}} :{{.Scope}}", true
		}
		return "", false
	}
	if got := renderSystem("planner", "DEFAULT", PlannerVars{Goal: "X", Scope: "*.x.com"}); got != ":X :*.x.com" {
		t.Fatalf("override render: %q", got)
	}

	// override referencing a non-catalog var → execution error → fallback to default
	PromptOverride = func(k string) (string, bool) { return "{{.NotInCatalog}}", true }
	if got := renderSystem("planner", "DEFAULT", PlannerVars{Goal: "x"}); got != "DEFAULT" {
		t.Fatalf("bad var should fall back to default, got %q", got)
	}

	// full plannerSystem path: DB body [A] is honored, then the code-owned tail
	// [C] () is ALWAYS appended — editing the body can't drop it.
	PromptOverride = func(k string) (string, bool) { return "PLANNER {{.Goal}}", true }
	got := plannerSystem("X", "/data", "/data")
	if !strings.HasPrefix(got, "PLANNER X") {
		t.Fatalf("plannerSystem body not honored: %q", got)
	}
	if !strings.Contains(got, "Intermediate artifact rule") || !strings.Contains(got, "/data") {
		t.Fatalf("plannerSystem missing code-owned artifact tail: %q", got)
	}

	// worker dual-text via {{if .ProxyAddr}} in a user template, plus the code tail:
	// [B] trafficTool present only when RECORDING (caCert set — the MITM is on, so
	// the traffic_* tools exist), [C] artifact spec always present. The trafficTool
	// block is gated on the CA (arg 2), NOT on ProxyAddr — a global egress proxy
	// with capture off routes traffic but records nothing.
	PromptOverride = func(k string) (string, bool) {
		return "{{if .ProxyAddr}} {{.ProxyAddr}}{{else}}{{end}}", true
	}
	recording := workerSystem("127.0.0.1:8080", "/ca.pem", "/data", "/data")
	if !strings.HasPrefix(recording, " 127.0.0.1:8080") {
		t.Fatalf("worker proxy branch body: %q", recording)
	}
	if !strings.Contains(recording, "traffic_search") {
		t.Fatalf("worker while recording should inject trafficTool: %q", recording)
	}
	if strings.Contains(recording, "traffic_refs") {
		t.Fatalf("worker bypassed shared optional evidence policy: %q", recording)
	}
	if !strings.Contains(recording, "Intermediate artifact rule") {
		t.Fatalf("worker missing artifact tail: %q", recording)
	}
	// Egress proxy set but capture OFF (no CA): the ProxyAddr template branch still
	// renders, but the trafficTool block must NOT — those tools are not registered.
	egressOnly := workerSystem("127.0.0.1:8080", "", "/data", "/data")
	if !strings.HasPrefix(egressOnly, " 127.0.0.1:8080") {
		t.Fatalf("worker egress-only branch body: %q", egressOnly)
	}
	if strings.Contains(egressOnly, "traffic_search") {
		t.Fatalf("worker without recording must NOT inject trafficTool: %q", egressOnly)
	}
	noProxy := workerSystem("", "", "/data", "/data")
	if !strings.HasPrefix(noProxy, "") {
		t.Fatalf("worker no-proxy branch body: %q", noProxy)
	}
	if strings.Contains(noProxy, "traffic_search") {
		t.Fatalf("worker without proxy must NOT inject trafficTool: %q", noProxy)
	}
}

// TestLangDirectiveAppendedToUserFacingRoles pins the English output-language
// tail: every user-facing role's system prompt must end with the code-owned
// English directive, and a DB-edited body must NOT be able to drop it.
func TestLangDirectiveAppendedToUserFacingRoles(t *testing.T) {
	t.Cleanup(func() { PromptOverride = nil })

	dir := langDirective()
	if !strings.Contains(dir, "English") {
		t.Fatalf("langDirective must force English output, got %q", dir)
	}
	if !strings.Contains(dir, "payload") || !strings.Contains(dir, "verbatim") {
		t.Fatalf("langDirective must keep commands/payloads verbatim, got %q", dir)
	}
	if !strings.Contains(dir, "never output Chinese or Korean") {
		t.Fatalf("langDirective must forbid leaking Chinese/Korean to the user, got %q", dir)
	}
	if !strings.Contains(dir, "another language") {
		t.Fatalf("langDirective must forbid mirroring the target/material language, got %q", dir)
	}
	if !strings.Contains(dir, "situation summar") {
		t.Fatalf("langDirective must name the planner situation summary as user-facing, got %q", dir)
	}

	PromptOverride = func(string) (string, bool) { return "BODY-ONLY", true }
	cases := map[string]string{
		"worker":    workerSystem("", "", "/data", "/data"),
		"planner":   plannerSystem("g", "/data", "/data"),
		"mainagent": mainAgentSystem("g", "/data", "/data"),
		"chat":      chatSystem("chat", "/data", "/data"),
		"goals":     goalsSystem("/data", true),
	}
	for role, sys := range cases {
		if !strings.HasPrefix(sys, "BODY-ONLY") {
			t.Fatalf("%s: DB body not honored: %q", role, sys)
		}
		if !strings.Contains(sys, "English") {
			t.Fatalf("%s: missing English output-language tail: %q", role, sys)
		}
		if strings.Index(sys, "English") <= strings.Index(sys, "BODY-ONLY") {
			t.Fatalf("%s: langDirective must be appended after the body: %q", role, sys)
		}
	}
}
