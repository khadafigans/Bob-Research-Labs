package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/llm"
	acperm "github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
	"github.com/Autumn-27/norma/transcript"
)

// goalsDefaultTmpl is the built-in EDITABLE body ( [A]) of the goals-decomposer
// prompt, seeded into agent_prompts. No template vars are used today.
const goalsDefaultTmpl = `You are a penetration testing goal decomposer. Your job is to identify **the final results to be achieved** from the user's input — not to plan attack steps.

**Step 1 (before splitting goals): Extract operational constraints**
From the "task goal / task description," identify any explicit rules about **what operations are and are not permitted**, and register them one by one with set_constraints (skip if no operational constraints exist):
- type=deny: prohibited operations (e.g. "no port scanning," "no write/delete on production," "no brute force," "don't touch this subdomain").
- type=allow: explicitly permitted/limited scope (e.g. "passive recon only," "only target this domain").
- Constraints ≠ goals, and ≠ attack steps: they define the boundary of permitted behavior.
- **Constraints must be self-contained with concrete specifics**: replace pronouns like "current target / current port / current IP / current domain / this site" with the **actual values** from the task goal/description.
  Example: if the target is https://abc.example.net, write "only test abc.example.net" not "only test the current target."
- **Only register constraints explicitly stated in the goal/description — never fabricate**; when unsure of type, use deny.
- If there are no operational constraints, **do not** call set_constraints.
After registering constraints (if any), proceed to goal decomposition.

**Goal = final deliverable / verifiable result**

**What is NOT a goal (do not list as sub-goals)**:
- Information gathering, recon, endpoint scanning
- Vulnerability analysis and validation process
- Attack steps, exploitation techniques
- Result verification steps

**Decomposition rules**:
- If the user describes a single final goal → output one
- If there are multiple **mutually independent** final deliverables → list them separately
- Label with vulnclass when it corresponds to a clear vulnerability class; leave empty for recon/business logic
- Never fabricate goals the user did not mention

Call set_goals to submit the results.`

// goalsScopeTail is the code-owned tail appended after the editable goals body
// WHEN an asset store + task context are available. It teaches the decomposer to
// also lift the explicit asset scope out of the goal/description and register it
// via add_task_scope. Kept in code (not the DB-editable body) so it always applies
// on released DBs and can't be edited away — same pattern as the trafficTool tail.
const goalsScopeTail = `

**Additional responsibility: register test asset scope**
In addition to decomposing goals, identify the **explicitly stated test asset scope** from the "task goal / task description" and register it with add_task_scope. **Minimum scope principle: only register what the user explicitly mentioned — never expand.**
- If the target is a URL or address with hostname (e.g. https://xxx.example.com/path, app.example.com) → use the **full hostname**, kind=subdomain, value=full hostname.
  Example: target https://a1b2c3.lab.example.net/path → kind=subdomain, value=a1b2c3.lab.example.net (**not** example.net).
  **Never** collapse a hostname with subdomain into the root domain — registering example.com when you see xxx.example.com expands scope beyond the user's target.
- Only when the user gives a **bare root domain with no subdomain** (e.g. just example.com), or explicitly says "entire site / all subdomains / full domain" → use kind=root_domain, value=example.com.
- Pure IP or subnet → kind=ip / cidr, value=IP or CIDR.
- **Do not** register company-level scope — company-level scope is handled in the planning phase.
Rules:
- Only register scope **explicitly written in the goal/description**; never fabricate or infer unmentioned domains/IPs.
- reason should briefly note which statement it came from, for auditability.
- If there is no explicit asset scope, **do not** call add_task_scope.
Call add_task_scope to register scope first (if any), then call set_goals to submit goals.`

// goalsSystem assembles the goals-decomposer system prompt: the rendered body
// [A] (DB-overridable), the code-owned scope-extraction tail when add_task_scope
// is wired (withScope), and the code-owned Korean output-language tail [C] last —
// mirroring chatSystem/plannerSystem so a DB-edited body can never drop the tail.
// DecomposeGoalsWithProvider and the localization test share this one assembly, so
// the langDirective tail can't drift between runtime and test. EngagementDescription
// is intentionally left empty: the task description rides in the user message, not
// the {{.EngagementDescription}} var (see DecomposeGoalsWithProvider).
func goalsSystem(dataDir string, withScope bool) string {
	sys := renderSystem("goals", goalsDefaultTmpl, GoalsVars{DataDir: dataDir, Now: nowStr()})
	if withScope {
		sys += goalsScopeTail
	}
	return sys + langDirective()
}

// GoalSpec is one decomposed objective.
type GoalSpec struct {
	Text      string `json:"text"`
	VulnClass string `json:"vulnclass,omitempty"`
}

// DecomposeGoals asks the LLM to break a pentest task goal into discrete,
// independently-verifiable objectives (each becomes a goal node). Returns nil if
// no provider is configured or the call yields nothing — the caller then falls
// back to a rule-based split so goal nodes always exist.
//
// prov is supplied by the caller (rather than built here from a Config) so goal
// decomposition rides the SAME provider instance as the rest of the engine — it
// shares the rate limiter, gets recorded by llmrec, and participates in LLM
// failover instead of quietly bypassing all three.
//
// desc is the task's free-text description (：/flag /).
// It is fed alongside the goal so the decomposer no longer splits blind — the
// prompt still forbids inventing anything the two texts don't state.
//
// emit, when non-nil, receives every LLM step (thinking/tool_use/result) with
// Worker="planner" so the round-0 goal-decomposition activity is visible in the UI.
//
// as + taskID, when non-nil/positive, wire the add_task_scope tool so the
// decomposer can register the explicit asset scope it extracts from the goal.
//
// ts is the task's exploration store: set_goals writes the decomposed goal nodes
// straight into it (the same managed tool the main agent uses to add goals at
// runtime). The returned specs are read back from the store so callers can emit
// per-goal activity and detect the "LLM produced nothing" case for their fallback.
func DecomposeGoals(ctx context.Context, prov llm.Provider, dataDir, goalText, desc string, as *db.AssetStore, ts *db.ExplorationStore, taskID int64, emit func(db.Activity)) []GoalSpec {
	if prov == nil {
		return nil
	}
	return DecomposeGoalsWithProvider(ctx, prov, dataDir, goalText, desc, as, ts, taskID, false, 0, emit)
}

// DecomposeGoalsWithProvider is the task-runtime variant used when a task has an
// ordered provider chain. It preserves the same tools and write behavior while
// letting the caller own provider selection/failover. maxTokens is the profile's
// per-reply output cap (0 = send none).
func DecomposeGoalsWithProvider(ctx context.Context, prov llm.Provider, dataDir, goalText, desc string, as *db.AssetStore, ts *db.ExplorationStore, taskID int64, nonStreaming bool, maxTokens int, emit func(db.Activity)) []GoalSpec {
	if prov == nil {
		return nil
	}
	// ： transcript store， agentcore  ctx 
	// session id（ writer ， agentcore.Prompt）。 session-id 
	// /（opencode zen  x-opencode-session  400
	// MissingSessionID） ctx ——「、 400」。
	// planner/worker ， llmrec.parseSession 。
	if ts != nil {
		ctx = transcript.WithSessionID(ctx, fmt.Sprintf("exp%d-goals", ts.ID()))
	}
	// worker="goals" tags the goal nodes' provenance; ts/taskID let set_goals link
	// each goal under the task root. This is the catalog's real set_goals tool, so a
	// web-edited description/schema on it applies here too.
	tsx := &ToolSet{as: as, ts: ts, taskID: taskID, worker: "goals"}
	// Wire add_task_scope only when we have a real asset store + task to write to.
	// goalsSystem appends the scope-extraction tail in lockstep (withScope) so the
	// prompt never asks for a tool that isn't present, and it owns the output-language
	// tail last so a DB-edited body can't drop it. Description rides in the user
	// message, NOT the {{.EngagementDescription}} var, so a prompt can't inject it twice.
	withScope := as != nil && taskID > 0
	sys := goalsSystem(dataDir, withScope)
	// set_constraints ( asset store):「」
	tools := []actool.CoreTool{tsx.setGoals(), tsx.setConstraints()}
	if withScope {
		tools = append(tools, tsx.addTaskScope())
	}
	userMsg := "：\n" + goalText
	if d := strings.TrimSpace(desc); d != "" {
		userMsg += "\n\n（，/flag /；，）：\n" + d
	}
	// Use captureRun so every LLM step is emitted as an activity record (visible in
	// the plan tab under the round-0 marker). Falls back gracefully when emit is nil.
	captureEmit := func(r db.Activity) {
		if emit != nil {
			r.Worker = "planner"
			emit(r)
		}
	}
	captureRun(ctx, agentcore.Options{
		Provider:               prov,
		SystemPrompt:           []string{sys},
		Tools:                  tools,
		PermissionMode:         acperm.ModeBypass,
		DisableBackgroundTasks: true,
		MaxTurns:     8,
		NonStreaming: nonStreaming, //  profile  Provider.Complete
		MaxTokens:    maxTokens,    // 0 = ,
	}, userMsg, captureEmit)
	// set_goals persisted the goals directly; read them back so the caller sees what
	// was written (empty slice ⇒ the LLM produced nothing ⇒ caller falls back).
	if ts == nil {
		return nil
	}
	nodes, _ := ts.ListByKind(db.KindGoal, 10000)
	var out []GoalSpec
	for _, n := range nodes {
		var p struct {
			Text      string `json:"text"`
			VulnClass string `json:"vulnclass"`
		}
		_ = json.Unmarshal(n.Payload, &p)
		if strings.TrimSpace(p.Text) != "" {
			out = append(out, GoalSpec{Text: p.Text, VulnClass: p.VulnClass})
		}
	}
	return out
}
