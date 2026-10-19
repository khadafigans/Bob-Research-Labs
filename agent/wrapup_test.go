package agent

import (
	"strings"
	"testing"
	"unicode"
)

// Wrap-up / settlement prompts are injected when a run or task hits its
// step/time budget. They directly instruct the final user-visible summary, so
// they must be English, free of Hangul/Han remnants, and keep tool names plus
// the one-sentence plain-text instruction.
//
// DB seeds leave wrapup_prompt / task_timeout_wrapup_prompt empty (builtin agent
// INSERT does not fill those columns); empty falls through to these constants.
func TestWrapupPromptsLocalizedToEnglish(t *testing.T) {
	all := map[string]string{
		"settleWrapUpPrompt":        settleWrapUpPrompt,
		"plannerWrapUpDefault":      plannerWrapUpDefault,
		"mainAgentWrapUpDefault":    mainAgentWrapUpDefault,
		"genericWrapUpDefault":      genericWrapUpDefault,
		"workerTaskTimeoutDefault":  workerTaskTimeoutDefault,
		"plannerTaskTimeoutDefault": plannerTaskTimeoutDefault,
	}

	hasScript := func(s string, table *unicode.RangeTable) bool {
		for _, r := range s {
			if unicode.Is(table, r) {
				return true
			}
		}
		return false
	}

	for name, p := range all {
		if hasScript(p, unicode.Hangul) {
			t.Errorf("%s: still has Hangul: %q", name, p)
		}
		if hasScript(p, unicode.Han) {
			t.Errorf("%s: still has Han characters: %q", name, p)
		}
		if !strings.Contains(strings.ToLower(p), "english") && name != "plannerWrapUpDefault" && name != "plannerTaskTimeoutDefault" {
			// planner wrap-ups intentionally omit a summary sentence
			if name == "mainAgentWrapUpDefault" || name == "genericWrapUpDefault" || name == "settleWrapUpPrompt" || name == "workerTaskTimeoutDefault" {
				t.Errorf("%s: must instruct English summary output", name)
			}
		}
	}

	mustContain := func(name, p string, subs ...string) {
		for _, s := range subs {
			if !strings.Contains(p, s) {
				t.Errorf("%s: missing required instruction %q", name, s)
			}
		}
	}
	mustContain("settleWrapUpPrompt", settleWrapUpPrompt,
		"insert_assets", "record_fact", "report_finding", "one plain-text", "English")
	mustContain("workerTaskTimeoutDefault", workerTaskTimeoutDefault,
		"insert_assets", "record_fact", "report_finding", "one plain-text", "English")
	mustContain("genericWrapUpDefault", genericWrapUpDefault, "one plain-text", "English")
	mustContain("mainAgentWrapUpDefault", mainAgentWrapUpDefault, "one plain-text", "English")
	mustContain("plannerWrapUpDefault", plannerWrapUpDefault, "add_intent", "prove_goal", "TodoWrite")
	mustContain("plannerTaskTimeoutDefault", plannerTaskTimeoutDefault, "prove_goal")
}

func TestWrapupDefaultsRouting(t *testing.T) {
	if WrapupDefault("worker") != settleWrapUpPrompt {
		t.Error("worker per-run default is not settleWrapUpPrompt")
	}
	if WrapupDefault("planner") != plannerWrapUpDefault {
		t.Error("planner per-run default is not plannerWrapUpDefault")
	}
	if WrapupDefault("mainagent") != mainAgentWrapUpDefault {
		t.Error("mainagent per-run default is not mainAgentWrapUpDefault")
	}
	if WrapupDefault("unknown-agent") != genericWrapUpDefault {
		t.Error("unknown key must fall back to genericWrapUpDefault")
	}
	if TaskTimeoutWrapupDefault("worker") != workerTaskTimeoutDefault {
		t.Error("worker task-timeout default is not workerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("planner") != plannerTaskTimeoutDefault {
		t.Error("planner task-timeout default is not plannerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("mainagent") != "" {
		t.Error("mainagent must have empty task-timeout wrap-up")
	}
	if workerTaskTimeoutDefault == settleWrapUpPrompt {
		t.Error("worker per-run and task-timeout prompts must differ")
	}
	if plannerTaskTimeoutDefault == plannerWrapUpDefault {
		t.Error("planner per-run and task-timeout prompts must differ")
	}
}
