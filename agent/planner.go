package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/intercept"
	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
	"github.com/Autumn-27/norma/transcript"
)

// Planner is the event-driven LLM planner (docs §4.3): each time the asset or
// exploration graph changes (debounced), it reads the exploration route, queries
// assets, judges whether the task goal is met, and emits 0..N exploration intents
// into the frontier. It is the sole intent generator.
type Planner struct {
	findingRecorder   FindingRecorder
	prov              llm.Provider
	model             string
	tx                *transcript.Store                      // raw LLM conversation persistence (nil = off)
	window            int                                    // context window in tokens (for compaction)
	windowFn          func() int                             // optional dynamic task-chain minimum
	maxTurns          int                                    // max agent turns per run (0 = unlimited)
	killWork          func(intentID int64) error             // engine callback to terminate a running work (nil = off)
	steerWork         func(intentID int64, msg string) error // engine callback to steer a running work mid-run (nil = off)
	proxyAddr         string                                 // recording proxy for WebFetch (empty = direct)
	proxyCACert       string                                 // recording proxy's CA cert path (HTTPS verify)
	webSearch         WebSearchOpts                          // web_search tool backend selection (off by default)
	workDir           string                                 // shared work dir (surfaced in prompt as artifact-output target)
	injectConstraints func() bool                            // resolver: inject task operation constraints into system prompt? (nil = yes)
	nonStreamingFn    func() bool                            // resolver: use non-streaming (Complete) path? (nil = streaming)
	noaEnabledFn      func() bool                            // resolver: use experimental noa compaction? (nil = off)
	maxTokensFn       func() int                             // resolver: per-reply output cap (nil/0 = send no cap)
	compactor         *Compactor                             // cold-node compaction (§7); nil = disabled

	// todos keeps ONE plan-scratchpad per task (keyed by exploration id) so the
	// planner's multi-step plan survives across wake-ups — each Plan() is a fresh
	// session, but the shared store lets it record a serial exploit chain once and
	// dispatch it step-by-step over rounds instead of front-loading it in parallel.
	todoMu sync.Mutex
	todos  map[int64]*actool.TodoStore
}

func NewPlanner(prov llm.Provider, model, workDir string, tx *transcript.Store, window, maxTurns int) *Planner {
	return &Planner{prov: prov, model: model, workDir: workDir, tx: tx, window: window, maxTurns: maxTurns, todos: map[int64]*actool.TodoStore{}}
}

func (p *Planner) SetCompactionWindowResolver(fn func() int) { p.windowFn = fn }

// SetCompactor wires the cold-node compactor (cold-digest §7). Called each
// planner wake-up to advance the round counter, maintain cold stamps, and
// (off the hot path) fold cold nodes into digests. nil = feature disabled.
func (p *Planner) SetCompactor(c *Compactor) { p.compactor = c }

// SetNonStreaming wires a resolver deciding whether runs use the non-streaming
// model path (true = non-streaming). nil/unset = streaming (default).
func (p *Planner) SetNonStreaming(fn func() bool) { p.nonStreamingFn = fn }

func (p *Planner) nonStreaming() bool { return p.nonStreamingFn != nil && p.nonStreamingFn() }

// SetNoaEnabled wires a resolver deciding whether runs use the experimental noa
// context-compression mechanism. nil/unset = off (built-in compaction). Read per
// run so the settings toggle takes effect without rebuilding the agent.
func (p *Planner) SetNoaEnabled(fn func() bool) { p.noaEnabledFn = fn }

// SetMaxTokens wires a resolver for the per-reply output cap. nil/unset or 0 =
// send no cap and let the endpoint decide. Read per run, like nonStreaming.
func (p *Planner) SetMaxTokens(fn func() int) { p.maxTokensFn = fn }

func (p *Planner) maxTokens() int {
	if p.maxTokensFn == nil {
		return 0
	}
	return p.maxTokensFn()
}

func (p *Planner) compactionWindow() int {
	if p.windowFn != nil {
		return p.windowFn()
	}
	return p.window
}

// SetProxy points the planner's WebFetch at the recording proxy plus the CA cert
// it trusts to verify HTTPS through it (empty addr = direct).
func (p *Planner) SetProxy(addr, caCert string) { p.proxyAddr, p.proxyCACert = addr, caCert }

// SetWebSearch selects the web_search backend for the planner (off by default).
func (p *Planner) SetWebSearch(o WebSearchOpts) { p.webSearch = o }

// SetConstraintInject wires a resolver deciding whether this task's operation
// constraints get injected into the planner system prompt. Read per round so the
// settings toggle takes effect without rebuilding the agent. nil = inject (default).
func (p *Planner) SetConstraintInject(fn func() bool) { p.injectConstraints = fn }

// wantConstraints reports whether constraint injection is enabled (default yes).
func (p *Planner) wantConstraints() bool { return p.injectConstraints == nil || p.injectConstraints() }

// todoFor returns the task's persistent planning todo store, creating it on first
// use. Shared across all of this task's planner wake-ups.
func (p *Planner) todoFor(expID int64) *actool.TodoStore {
	p.todoMu.Lock()
	defer p.todoMu.Unlock()
	s := p.todos[expID]
	if s == nil {
		s = actool.NewTodoStore()
		p.todos[expID] = s
	}
	return s
}

// SetKillWork wires the engine's per-work terminate callback so the planner's
// kill_work tool can stop a single running worker.
func (p *Planner) SetKillWork(fn func(intentID int64) error) { p.killWork = fn }

// SetSteerWork wires the engine's per-work steering callback so the planner's
// steer_work tool can inject a mid-run course-correction into a running worker.
func (p *Planner) SetSteerWork(fn func(intentID int64, msg string) error) { p.steerWork = fn }

// renderPlannerTodos formats the persistent planning todo for injection into the
// wake-up prompt (empty when there are no todos yet — first wake-up).
func renderPlannerTodos(items []actool.Todo) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n【（，）】：\n")
	for _, it := range items {
		mark := map[actool.TodoStatus]string{actool.TodoPending: "☐", actool.TodoInProgress: "▶", actool.TodoCompleted: "✔"}[it.Status]
		if mark == "" {
			mark = "☐"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", mark, it.Content))
	}
	b.WriteString("：【 /  fact 】； TodoWrite （ fact  completed）。 pending/in_progress 。")
	return b.String()
}

// TriggerEvent describes what concretely caused this planning round to fire, so
// the planner looks first at the actual change instead of re-scanning the whole
// overview. Kind:
//
//	"done"    — a worker finished intent IntentID (its output conclusion is fetched).
//	"finding" — a worker reported a finding on intent IntentID (Detail = ).
//	"goal"    — the human (via  agent  set_goals) added one OR MORE goals in a
//	            single call (Goals = ，1+ ；set_goals ).
//	"goal_deleted" — the human deleted a goal from  (Detail = ).
//	"goal_edited"  — the human edited a goal from  (OldGoal→NewGoal ).
//	"cancelled" — the human deleted intent IntentID (Detail = ). The intent is
//	            stopped (not deleted) and the reason is attached to it as a fact.
type TriggerEvent struct {
	Kind     string
	IntentID int64
	Detail   string
	Summary  string   // Kind=="cancelled" ：（，）
	Goals    []string // Kind=="goal" ： set_goals （1 ）
	OldGoal  string   // Kind=="goal_edited" ：
	NewGoal  string   // Kind=="goal_edited" ：
	Hints    []string // Kind=="hint" ： add_hint （1 ）
}

// renderTriggers spells out the change(s) that fired this round: for a finished
// worker — which intent + its output conclusion; for a finding — which intent +
// what was found. Empty for time/heartbeat wakes. Reads the store (best-effort;
// a blank field never blocks the round).
func renderTriggers(ts *db.ExplorationStore, evs []TriggerEvent) string {
	if len(evs) == 0 || ts == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n【（，）】：")
	for _, ev := range evs {
		switch ev.Kind {
		case "goal":
			if len(ev.Goals) == 1 {
				b.WriteString(fmt.Sprintf("\n- （ agent）：%s —— ，（）。", ev.Goals[0]))
			} else {
				b.WriteString(fmt.Sprintf("\n- （ agent） %d ：%s —— ，。", len(ev.Goals), strings.Join(ev.Goals, "；")))
			}
		case "hint":
			if len(ev.Hints) == 1 {
				b.WriteString(fmt.Sprintf("\n- （ agent）：%s —— ，/（）。", ev.Hints[0]))
			} else {
				b.WriteString(fmt.Sprintf("\n- （ agent） %d ：%s —— ，/。", len(ev.Hints), strings.Join(ev.Hints, "；")))
			}
		case "goal_deleted":
			b.WriteString(fmt.Sprintf("\n- ：%s —— ，/（）。", ev.Detail))
		case "goal_edited":
			b.WriteString(fmt.Sprintf("\n- ，「%s」「%s」—— （）。", ev.OldGoal, ev.NewGoal))
		case "finding":
			b.WriteString(fmt.Sprintf("\n-  #%d（%s） worker  finding：%s", ev.IntentID, intentSummary(ts, ev.IntentID), ev.Detail))
		case "cancelled":
			//  Summary（，intentSummary ）。
			sm := ev.Summary
			if sm == "" {
				sm = intentSummary(ts, ev.IntentID)
			}
			b.WriteString(fmt.Sprintf("\n-  #%d ，：%s、：%s。（）；。", ev.IntentID, sm, ev.Detail))
		default: // "done"
			b.WriteString(fmt.Sprintf("\n-  #%d（%s） worker ，：%s", ev.IntentID, intentSummary(ts, ev.IntentID), workerOutput(ts, ev.IntentID)))
			if fids := factIDsYielded(ts, ev.IntentID); fids != "" {
				b.WriteString(fmt.Sprintf("； id：%s ", fids))
			}
		}
	}
	b.WriteString("\n（ node_detail / get_worker_output / list_findings 。）")
	return b.String()
}

// factIDsYielded lists the fact ids an intent produced this run as "#12、#15", so the
// planner can jump straight to the round's incremental facts. Empty (best-effort) when
// the intent yielded no facts or the lookup fails.
func factIDsYielded(ts *db.ExplorationStore, id int64) string {
	ids, err := ts.FactsYielded(id)
	if err != nil || len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, fid := range ids {
		parts[i] = fmt.Sprintf("#%d", fid)
	}
	return strings.Join(parts, "、")
}

// intentSummary reads an intent node's one-line summary (best-effort, "?" on miss).
func intentSummary(ts *db.ExplorationStore, id int64) string {
	n, err := ts.GetNode(id)
	if err != nil || n == nil {
		return "?"
	}
	var p map[string]any
	if json.Unmarshal(n.Payload, &p) == nil {
		if s, ok := p["summary"].(string); ok && s != "" {
			return s
		}
	}
	return "?"
}

// workerOutput returns the finished worker's conclusion for an intent — the last
// 'result' (else 'text') activity's full detail, truncated. Same source get_worker_output uses.
func workerOutput(ts *db.ExplorationStore, id int64) string {
	acts, _, err := ts.ActivityList(&id, 0, 1000)
	if err != nil {
		return "()"
	}
	var pick *db.Activity
	for i := range acts {
		if acts[i].Kind == "result" {
			pick = &acts[i]
		} else if acts[i].Kind == "text" && pick == nil {
			pick = &acts[i]
		}
	}
	if pick == nil {
		return "( work )"
	}
	out, _ := ts.ActivityDetail(pick.ID)
	if out == "" {
		out = pick.Summary
	}
	return truncOutput(out, 800)
}

// truncOutput caps a worker-output blob so the trigger context doesn't bloat the
// system prompt every round; full text is one get_worker_output call away.
func truncOutput(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + " …（， get_worker_output）"
}

// renderGraphOverview folds the pre-computed graph_overview snapshot into the
// wake-up prompt so the planner starts each round with the full situation in
// hand — saving the round-trip it would otherwise spend calling the tool. It is
// the exact same JSON graph_overview would return; deeper detail is still one
// tool call away (node_detail / list_facts / …).
func renderGraphOverview(data map[string]any) string {
	b, err := json.Marshal(data)
	if err != nil {
		return "" // fall back to the model calling graph_overview itself
	}
	return "\n\n【（graph_overview ，； node_detail/list_facts ）】：\n" + string(b)
}

// plannerDefaultTmpl is the built-in EDITABLE body ( [A]) of the planner prompt,
// seeded into agent_prompts. Goal is a {{.Goal}} template var; the 
// tail is code-owned (artifactSpec) and appended by plannerSystem after rendering.
const plannerDefaultTmpl = `You are the "planner" of a network security platform's authorized penetration testing system, invoked frequently (woken up on every graph change). Responsibility: read the situation → assess the goal → **only add new exploration intents when there are genuinely uncovered new directions**. You are a planner, not an executor: all output this round must be [generate/clarify intents] or [declare goal achieved] — never do the work itself inside a plan.

━━ When to add intents ━━
**Condition for adding**: a mechanistically NEW direction has not yet been attempted by any intent in the graph (including running, queued, or failed). Similarity of wording doesn't matter; what matters is whether the mechanism is different.
**Don't add** when: the direction already has a running or queued intent (even if phrased differently); a previous attempt already has a failed result from this direction; you're just rephrasing an existing intent.
**Add** when: a new entry point / parameter / code path / mechanism was just discovered that has not been tried; a previous attempt failed but new material information (new tech stack, new path, new credential) suggests a genuinely different mechanism worth trying.

━━ Situational awareness reading ━━
- get_task_graph to see the full picture: goal / subgoals / intents / findings / constraints.
- Focus on: which directions have been tried (and results), what new assets/entries were just discovered, which constraints still apply, what hasn't been tried yet.
- Findings and assets are the most important new information — they often open new directions.

━━ Intent writing rules ━━
- **One sentence**: what to investigate + concrete target (include specific URL/endpoint/parameter if known).
- **Differentiation**: when adding a new intent similar to an existing one, explicitly state in the intent "mechanism differs: [specific difference]" — forcing you to confirm there's a real new mechanism.
- **Never vague**: "try more" / "keep trying" / "explore" without a concrete direction are not intents.
- spawn_intent(goal_id=..., desc=...) or add_task_hint for the actively running intent.

━━ Goal assessment ━━
At the end of each call, assess each goal:
- **Achieved**: finding already confirmed by a worker in get_task_graph, with PoC/evidence visible — call complete_goal(goal_id, summary).
- **Not yet**: there are reasonable directions not yet tried, or running intents haven't returned — continue planning.
- **Truly exhausted**: all reasonable mechanistically-distinct directions have been tried and failed — call complete_goal with summary explaining why goal is unreachable.

━━ Discipline ━━
- Only read state; never directly exploit / run commands / call OS tools / call worker tools.
- Respond concisely: list what you found, what intents you added (if any), and your goal assessment.
- When goal is achieved or all directions exhausted, complete_goal immediately — do not keep spawning intents.`

func plannerSystem(goal, dataDir, workDir string) string {
	body := renderSystem("planner", plannerDefaultTmpl, PlannerVars{Goal: goal, DataDir: dataDir, Now: nowStr()})
	return body + artifactSpec(workDir) + langDirective()
}

// Plan runs one planning round. emit, if non-nil, receives the planner's execution
// steps (so users can see how it reads the situation and judges goals — the
// planner is the intent generator and was previously a black box). Returns whether
// the planner judged the goal met.
// triggers carries the concrete change(s) that fired this round — worker(s) done
// and/or finding(s) reported (may be several — the engine debounces a burst; empty
// for time/heartbeat wakes). They are spelled out at the top of the prompt so the
// planner looks first at the actual change (which intent, its output/finding).
func (p *Planner) Plan(ctx context.Context, taskID int64, as *db.AssetStore, ts *db.ExplorationStore, goal string, triggers []TriggerEvent, emit func(db.Activity)) (met bool, reason string, err error) {
	// cold-digest §2.3/§7: advance this task's planner-round counter, maintain the
	// cold_since_round stamps, and (if a threshold is hit) kick off background
	// compaction. Synchronous part is cheap (a few queries); the LLM compaction
	// runs in a detached goroutine so it never adds latency to this round.
	p.compactor.OnPlannerRound(ctx, ts)
	tsx := NewToolSet(ts, "planner")
	tsx.SetFindingRecorder(p.findingRecorder)
	if as != nil {
		tsx.SetAssetStore(as, as.Companies())
	}
	tsx.SetTaskID(taskID)
	tsx.SetCoverageEnabled(as == nil || as.CoverageEnabled(taskID))
	tsx.killWork = p.killWork   // enable kill_work tool (nil = unavailable)
	tsx.steerWork = p.steerWork // enable steer_work tool (nil = unavailable)
	if origin, _ := ts.OriginFactID(); origin > 0 {
		tsx.SetOwnerNode(origin) // planner-side anchors default to the task root (origin fact)
	}
	//  + （Read/Write/Edit/MultiEdit/LS/Glob/Grep/Bash）
	//  add_task_scope/list_untested_assets（ prompt）。
	base := append(tsx.DropCoverageTools(tsx.PlannerTools()), actool.DefaultTools()...)
	ctx = WithRunInfo(ctx, RunInfo{TaskID: taskID, ExplorationID: explorationID(ts)})
	tools, def, cleanup := AugmentTools(ctx, "planner", base)
	defer cleanup()
	// （ + ）【 user 】( input)，system
	// 。move-out  system 、；，
	//  compaction （planner ，）。situational  input。
	situational := renderTriggers(ts, triggers) + renderGraphOverview(tsx.graphOverviewData())
	//  deadline / ( ctx , taskclock.go)。
	// planner 【】 user ( situational),
	tc := taskClockFrom(ctx)
	if tc.Final {
		situational += "\n\n【（，）】：" + resolveTaskTimeoutWrapup("planner")
	}
	//  <workDir>/tasks/<taskID>，。
	taskDir := ensureRunDir(p.workDir, taskID, 0)
	ctx = intercept.WithReviewContext(ctx, taskDir, intercept.ReviewBackground{})
	sysBody := plannerSystem(goal, p.workDir, taskDir)
	if p.wantConstraints() {
		sysBody += constraintBlock(ts) // (),
	}
	system, boundary := deferredSystem(sysBody, def)
	// planner ; deadline  MaxDuration ,
	maxDur, clamped := clampMaxDuration(tc.DeadlineUnix, 0)
	settle := wrapupSettlement("planner", nil)
	if tc.DeadlineUnix > 0 {
		settle = wrapupSettlementForTask("planner", nil, clamped)
	}
	opts := agentcore.Options{
		Provider:        p.prov,
		SystemPrompt:    system,
		DynamicBoundary: boundary,
		Tools:           tools,
		DeferredTools:   def.Deferred,
		UnlockSet:       def.Unlock,
		PermissionMode:  permission.ModeBypass,
		EnableWebFetch:  true, // ； CA  MITM  HTTPS 
		WebFetchProxy:   p.proxyAddr,
		WebFetchCACert:  p.proxyCACert,
		// ()。ddgs  key；brave-free  BraveKey；tavily  TavilyKey。
		// WebSearchProxy (http/https/socks5)， MITM ；。
		EnableWebSearch:       p.webSearch.Enabled,
		WebSearchBackend:      p.webSearch.Backend,
		BraveSearchAPIKey:     p.webSearch.BraveKey,
		TavilySearchAPIKey:    p.webSearch.TavilyKey,
		DeepSeekSearchBaseURL: p.webSearch.DeepSeekBaseURL,
		DeepSeekSearchAPIKey:  p.webSearch.DeepSeekAPIKey,
		DeepSeekSearchModel:   p.webSearch.DeepSeekModel,
		WebSearchProxy:        p.webSearch.Proxy,
		BashEnv:               proxyEnv(p.proxyAddr, p.proxyCACert), // Bash + CA
		WorkingDir:            taskDir,                              //  <workDir>/tasks/<taskID>
		ToolOutputDir:         cmdOutDir(taskDir),
		MaxTurns:              p.maxTurns, // 0 = unlimited (configurable in agent management)
		MaxDuration:           maxDur,     // 0=; deadline = deadline 
		Compaction:            compactionConfig(p.compactionWindow()),
		// ：（session ，store ）。
		Todos: p.todoFor(ts.ID()),
		// 【】→ SDK :( add_intent、
		//  prove_goal、 TodoWrite),——planner 。
		// clamped( deadline ) PromptByReason( wrapupSettlementForTask)。
		Settlement:   settle,
		NonStreaming: p.nonStreaming(), //  profile  Provider.Complete
		MaxTokens:    p.maxTokens(),    // 0 = ,
	}
	if p.tx != nil { // persist raw LLM conversation; one accumulating file per task's planner
		opts.Transcript = p.tx
		opts.SessionID = fmt.Sprintf("exp%d-planner", ts.ID())
	}
	// : noa ( <workDir>/noa/<SessionID> ,)。
	noaSession := fmt.Sprintf("exp%d-planner", ts.ID())
	enableNoa(&opts, p.noaEnabledFn, p.workDir, noaSession, noaWarn(noaSession))
	// （ + ） user （ input）。user 
	lead := "（【】），："
	if len(triggers) == 0 {
		lead = "**（）/**——。： steer_work 、 kill_work ；、："
		// /, open  running  → ( worker 、
		if active, err := ts.HasActiveIntent(); err == nil && !active {
			lead = "**（）**,** open  running **—— worker 、,。****、****( 0 );,："
		}
	}
	input := lead + situational + "\n\n，。【】（/） prove_goal 。**：、 open  running （frontier_open=0  running_intents ），—— work 、， 0 =。 open/running 、，。**" +
		renderPlannerTodos(opts.Todos.List())
	// MaxDuration ( ctx ),
	// , ctx 。ctx  pause / kill / shutdown。
	_, _, err = captureRun(ctx, opts, input,
		func(r db.Activity) {
			if emit != nil {
				r.Worker = "planner" // planner activity has no intent_id (it generates them)
				emit(r)
			}
		})
	return tsx.GoalMet, tsx.Reason, err
}
