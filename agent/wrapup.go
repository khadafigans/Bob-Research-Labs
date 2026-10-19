package agent

import (
	"strings"

	"github.com/Autumn-27/norma/harness"
)

// (wrap-up / settlement prompt): agent 【(MaxTurns)】
// 【(run_seconds/MaxDuration)】,SDK  settlement ,
//
//  agent ( agents.wrapup_prompt),

// WrapupOverride, if set, returns the stored wrap-up prompt for an agent key and
// whether a non-empty one exists. Wired by the server to the agents table (like
// PromptOverride for system prompts). nil / empty → the built-in default is used.
var WrapupOverride func(agentKey string) (string, bool)

// WrapupMaxTurnsOverride, if set, returns the admin-configured turn budget for the
// wrap-up phase of an agent and whether a positive one exists. Wired to the agents
// table. nil / ≤0 → the built-in per-agent default (wrapupTurnDefaults) is used.
var WrapupMaxTurnsOverride func(agentKey string) (int, bool)

// , agent key 。worker  settleWrapUpPrompt
// ( worker.go),planner/mainagent ;( agent)。
var wrapupDefaults = map[string]string{
	"worker":    settleWrapUpPrompt,
	"planner":   plannerWrapUpDefault,
	"mainagent": mainAgentWrapUpDefault,
}

// wrapupTurnDefaults:  agent 【】( >0 )。
//  10 ,。 genericWrapupTurns。
var wrapupTurnDefaults = map[string]int{
	"worker":    10,
	"planner":   10,
	"mainagent": 10,
}

const genericWrapupTurns = 10

const plannerWrapUpDefault = "This planning round's step budget is almost exhausted. Only 【this round】 is ending — the system will wake you again later as the situation changes. The overall task is not over, so you do not need to wrap up the whole plan here. Persist conclusions you have already clearly decided so this round is not wasted, but 【do not invent intents just to look finished】 (zero intents this round is a fully valid outcome). (1) If you already decided exploration directions that 【must be dispatched now】, submit them in one add_intent call (do not sit on decided work). (2) Mark goals already proven by findings/facts as met via prove_goal (do not skip any). (3) If you identified a serial exploit chain that needs staged work, record it with TodoWrite so the next wake can continue. When done, end this round immediately and do not output a summary sentence."

const mainAgentWrapUpDefault = "The step budget is almost exhausted and this interaction is about to end. Do not start any new exploration or operations. Summarize for the user in **exactly one plain-text English sentence**: current progress, key conclusion, and recommended next step."

const genericWrapUpDefault = "About to end because the budget is exhausted. First persist any completed results you have not saved yet, then summarize in **exactly one plain-text English sentence** what you did and the key conclusion (that single sentence is shown as this run's result)."

// WrapupDefault returns the built-in default wrap-up prompt for an agent key —
// used by the admin UI as the "restore default" value and empty-field placeholder.
func WrapupDefault(agentKey string) string {
	if d, ok := wrapupDefaults[agentKey]; ok {
		return d
	}
	return genericWrapUpDefault
}

// WrapupTurnsDefault returns the built-in wrap-up turn budget for an agent key —
// used by the admin UI as the "0 = default N" hint.
func WrapupTurnsDefault(agentKey string) int {
	if n, ok := wrapupTurnDefaults[agentKey]; ok {
		return n
	}
	return genericWrapupTurns
}

// resolveWrapup returns the effective wrap-up prompt: the DB override (if set and
// non-empty) over the built-in default.
func resolveWrapup(agentKey string) string {
	if WrapupOverride != nil {
		if t, ok := WrapupOverride(agentKey); ok && strings.TrimSpace(t) != "" {
			return t
		}
	}
	return WrapupDefault(agentKey)
}

// resolveWrapupTurns returns the effective wrap-up turn budget: a positive DB
// override over the built-in per-agent default.
func resolveWrapupTurns(agentKey string) int {
	if WrapupMaxTurnsOverride != nil {
		if v, ok := WrapupMaxTurnsOverride(agentKey); ok && v > 0 {
			return v
		}
	}
	return WrapupTurnsDefault(agentKey)
}

// wrapupSettlement builds the settlement config for an agent's run. Prompt and the
// turn budget are admin-editable per agent; disabled tools are code-owned policy so
// a user can't edit away the "stop probing" guardrail. Resolved fresh each run
// (reads DB live), so edits apply on the next run without a restart.
func wrapupSettlement(agentKey string, disabledTools []string) *harness.Settlement {
	return &harness.Settlement{
		Prompt:        resolveWrapup(agentKey),
		DisabledTools: disabledTools,
		MaxTurns:      resolveWrapupTurns(agentKey),
	}
}

//
//  per-run 【】：per-run " run "；
// "、"。（ planner：per-run ""，
// "、"）。 worker/planner 。

// WrapupTaskTimeoutOverride / …TurnsOverride： DB 
// （wire  agents.task_timeout_wrapup_prompt / _max_turns， worker/planner）。
var (
	WrapupTaskTimeoutOverride      func(agentKey string) (string, bool)
	WrapupTaskTimeoutTurnsOverride func(agentKey string) (int, bool)
)

var taskTimeoutWrapupDefaults = map[string]string{
	"worker":  workerTaskTimeoutDefault,
	"planner": plannerTaskTimeoutDefault,
}

const workerTaskTimeoutDefault = "**The whole task has hit its time limit and is about to end** (not this run's budget — the entire exploration is finishing). This is the last chance. (1) Persist 【everything】 already identified but not yet recorded — new assets via insert_assets, conclusions/facts via record_fact, confirmed vulnerabilities via report_finding. (2) Do not start any new commands or probes. (3) **End with exactly one plain-text English sentence** summarizing the key conclusion from this intent."

const plannerTaskTimeoutDefault = "**The whole task has hit its time limit and is about to end** (not this round — the entire task is finishing). Make a final goal judgment from 【all】 facts and findings so far. Mark goals proven by evidence as met via prove_goal (do not skip any). **Do not create any new intents** (dispatching now will not run). When judgment is done, end immediately and do not output a summary sentence."

// TaskTimeoutWrapupDefault  agent （/）。
func TaskTimeoutWrapupDefault(agentKey string) string {
	return taskTimeoutWrapupDefaults[agentKey] // (mainagent/chat)
}

// resolveTaskTimeoutWrapup：DB () > 。 agent 
// （ worker/planner）， per-run 。
func resolveTaskTimeoutWrapup(agentKey string) string {
	if WrapupTaskTimeoutOverride != nil {
		if t, ok := WrapupTaskTimeoutOverride(agentKey); ok && strings.TrimSpace(t) != "" {
			return t
		}
	}
	return TaskTimeoutWrapupDefault(agentKey)
}

func resolveTaskTimeoutTurns(agentKey string) int {
	if WrapupTaskTimeoutTurnsOverride != nil {
		if v, ok := WrapupTaskTimeoutTurnsOverride(agentKey); ok && v > 0 {
			return v
		}
	}
	return resolveWrapupTurns(agentKey) //  per-run 
}

// wrapupSettlementForTask builds settlement for a worker/planner run that is aware
// of the task deadline. See §5 of the design doc:
//   - clamped=true  →  run  deadline ： Timeout =→；
//      MaxTurns =、→ per-run 。
//   - clamped=false → ： reason  per-run （ wrapupSettlement）。
//
//  harness  PromptByReason 【】reason ， build 。
func wrapupSettlementForTask(agentKey string, disabledTools []string, clamped bool) *harness.Settlement {
	perRun := resolveWrapup(agentKey)
	st := &harness.Settlement{
		Prompt:        perRun, // ( clamped  reason )
		DisabledTools: disabledTools,
		MaxTurns:      resolveWrapupTurns(agentKey),
	}
	if clamped {
		if tt := resolveTaskTimeoutWrapup(agentKey); tt != "" {
			st.PromptByReason = map[harness.TerminalReason]string{
				harness.ReasonTimeout:  tt,     // 
				harness.ReasonMaxTurns: perRun, // 、
			}
			st.MaxTurns = resolveTaskTimeoutTurns(agentKey)
		}
	}
	return st
}
