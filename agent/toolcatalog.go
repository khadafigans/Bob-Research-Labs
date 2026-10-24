package agent

import (
	"context"
	"encoding/json"

	actool "github.com/Autumn-27/norma/tool"
)

// This file converts "built-in tools" from pure code into an enumerable, DB-overridable catalog:
//   - BuiltinToolSeeds(): expands each executor agent's built-in toolset into seed records
//     (key + description + parameter schema + default-bound agents), seeded idempotently
//     into the tools table on server startup.
//   - ToolResolve hook: at runtime, post-processes assembled tools against the DB tools table
//     (filter by agent binding, override description/schema, inject parameter defaults).
//     key/handler stay in code; DB only changes "prose and defaults."
// handler (Call behavior) always comes from code — DB cannot change it, only the model-visible description and default params.

// ToolSeed is a seedable snapshot of a built-in tool: Key = CoreTool.Name() (bound to handler, read-only in UI),
// Desc/Schema come from the code tool definition, Agents are the default-bound agent keys.
type ToolSeed struct {
	Key    string         // = CoreTool.Name()，主键，不可改
	Desc   string         // 顶层描述（可在 UI 覆盖）
	Schema map[string]any // 参数 JSON-Schema（结构只读，description/default 可在 UI 改）
	Agents []string       // 默认绑定的 agent key（worker/planner/mainagent）
}

// builtinToolsByAgent constructs each executor agent's domain toolset using a "read-only shell"
// ToolSet (nil stores). Tool constructors only stuff closures into Spec without dereferencing stores,
// so nil is safe — these tools are only used here to read Name()/Description()/InputSchema(), never Called.
//
// Intentionally EXCLUDES SDK generic tools actool.DefaultTools() (Read/Write/Edit/MultiEdit/LS/Glob/
// Grep/Bash): every agent has them unconditionally, there's no per-agent binding trade-off, and
// their descriptions are mostly in Prompt() (covering only Description() would cause partial-override confusion).
// Not seeded → no DB row → ToolResolve passes them through unchanged — same behavior as before.
// Only the platform's own domain tools are managed in this table.
func builtinToolsByAgent() map[string][]actool.CoreTool {
	ts := NewToolSet(nil, "")
	return map[string][]actool.CoreTool{
		"mainagent": ts.MainAgentTools(),
		"planner":   ts.PlannerTools(),
		"worker":    ts.WorkerTools(),
		// goals (goal decomposer) defaults to set_goals + set_constraints: used to write
		// decomposed goals and extracted operational constraints to the DB.
		"goals": {ts.setGoals(), ts.setConstraints()},
		// auto defaults to finding reporting + asset management tools; other tools can be enabled in UI.
		"auto": {ts.addFinding(), ts.insertAssets(), ts.addCompanyScope(), ts.listAssets(), ts.listCompanies()},
		// pentest (standalone pentest agent) defaults to: list assets / insert assets / report finding / list findings / list companies.
		"pentest": {ts.listAssets(), ts.insertAssets(), ts.addFinding(), ts.listFindings(), ts.listCompanies()},
	}
}

// defaultUnbound: these system tools are still seeded into the catalog (visible in UI, manually bindable)
// but default to NO agent binding — ToolResolve drops tools with empty bindings for all agents; explicit opt-in required.
// They remain in some agent's base toolset (e.g. goal_met in PlannerTools) so seed can construct them for desc/schema,
// and so ToolResolve can retain them when manually re-bound.
//
// goal_met: bypasses per-goal prove_goal to declare the ENTIRE task complete globally; high-impact, risk of false positives,
// and redundant with "prove_goal marks last goal → auto-close", so default-unbound; bind manually when needed.
var defaultUnbound = map[string]bool{"goal_met": true}

// BuiltinToolSeeds deduplicates and merges all agents' built-in toolsets into a seed list:
// same-named tools (e.g. list_assets shared by multiple agents) are merged into one entry with
// Agents as the union; tools in defaultUnbound have their binding forced to empty.
func BuiltinToolSeeds() []ToolSeed {
	byAgent := builtinToolsByAgent()
	order := []string{"mainagent", "goals", "planner", "worker", "auto", "pentest"}

	type acc struct {
		tool   actool.CoreTool
		agents []string
	}
	m := map[string]*acc{}
	var keys []string
	for _, ak := range order {
		for _, t := range byAgent[ak] {
			a, ok := m[t.Name()]
			if !ok {
				a = &acc{tool: t}
				m[t.Name()] = a
				keys = append(keys, t.Name())
			}
			a.agents = append(a.agents, ak)
		}
	}

	out := make([]ToolSeed, 0, len(keys))
	for _, k := range keys {
		a := m[k]
		agents := a.agents
		if defaultUnbound[k] {
			agents = []string{} // seeded in catalog, manually bindable, but default-unbound (stores [] not null, consistent with other tools)
		}
		out = append(out, ToolSeed{
			Key:    k,
			Desc:   a.tool.Description(),
			Schema: a.tool.InputSchema(),
			Agents: agents,
		})
	}
	return out
}

// ToolResolve, if set, post-processes an agent's fully-assembled tool list against
// the DB tools table: it drops tools not bound to this agent (or globally disabled)
// and wraps the rest so the model sees the DB-overridden description/schema and
// default params get injected. Tools with no matching DB row (MCP/skill/host tools like
// traffic) pass through untouched. nil = tools unchanged. Wired in server/assembly.go.
var ToolResolve func(ctx context.Context, agentKey string, tools []actool.CoreTool) []actool.CoreTool

// DecorateTool wraps t so Description()/InputSchema() report the DB overrides and
// Call() injects scalar parameter defaults (from schema's "default" props) whenever
// the model omitted them. Name/Prompt/permission/scheduler flags delegate to t, so
// the tool's identity and handler are unchanged. Empty desc/schema fall back to t's.
func DecorateTool(t actool.CoreTool, desc string, schema map[string]any) actool.CoreTool {
	if desc == "" {
		desc = t.Description()
	}
	if len(schema) == 0 {
		schema = t.InputSchema()
	}
	return &overriddenTool{CoreTool: t, desc: desc, schema: schema}
}

// overriddenTool is a CoreTool decorator: it embeds the original (so all behavioral
// methods — Prompt/IsReadOnly/IsConcurrencySafe/CheckPermissions/Name — delegate)
// and overrides only the model-facing description/schema plus default injection.
type overriddenTool struct {
	actool.CoreTool
	desc   string
	schema map[string]any
}

func (o *overriddenTool) Description() string         { return o.desc }
func (o *overriddenTool) InputSchema() map[string]any { return o.schema }

func (o *overriddenTool) Call(ctx context.Context, in json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
	return o.CoreTool.Call(ctx, injectDefaults(in, o.schema), tc)
}

// injectDefaults fills scalar parameter defaults declared in the (possibly edited)
// schema into the input JSON whenever the model omitted the field or left it empty/
// null. Structure (names/types/required) is untouched — only default values are merged in.
func injectDefaults(in json.RawMessage, schema map[string]any) json.RawMessage {
	defs := scalarDefaults(schema)
	if len(defs) == 0 {
		return in
	}
	m := map[string]json.RawMessage{}
	if len(in) > 0 {
		if err := json.Unmarshal(in, &m); err != nil {
			return in // non-object input: don't touch it
		}
	}
	changed := false
	for k, dv := range defs {
		if cur, ok := m[k]; !ok || isEmptyJSON(cur) {
			m[k] = dv
			changed = true
		}
	}
	if !changed {
		return in
	}
	b, err := json.Marshal(m)
	if err != nil {
		return in
	}
	return b
}

// scalarDefaults extracts properties[k]["default"] for scalar params (string/
// integer/number/boolean). Array/object defaults are skipped: merging them is
// ambiguous and not worth the surprise.
func scalarDefaults(schema map[string]any) map[string]json.RawMessage {
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	out := map[string]json.RawMessage{}
	for name, raw := range props {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		dv, ok := p["default"]
		if !ok || dv == nil {
			continue
		}
		switch p["type"] {
		case "string", "integer", "number", "boolean":
			if b, err := json.Marshal(dv); err == nil {
				out[name] = b
			}
		}
	}
	return out
}

func isEmptyJSON(raw json.RawMessage) bool {
	s := string(raw)
	return s == "null" || s == `""`
}
