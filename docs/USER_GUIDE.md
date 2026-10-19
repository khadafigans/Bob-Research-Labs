# Bob Research Labs — User Guide

How to use the English UI after install. Default URL: **http://localhost:8787**

---

## 1. Getting started

### Install and open the UI

1. Run `./install.sh` (prefer **Local run** / option 2, or Docker option 1 which builds from **this** tree).
2. Open **http://localhost:8787** (or your host/port).
3. First visit → **/setup** → create the admin password (8+ characters).
4. Log in at **/login**.

> Do **not** pull `autumn27/artex` from Docker Hub. That image is upstream Korean ARTEX, not this English fork.

### Sidebar map

| Area | Purpose |
|------|---------|
| **Dashboard** | High-level status |
| **Chat** | Talk to conversation agents (Auto, custom agents, Report Writer, …) |
| **Function → Tasks / Findings / Assets / Traffic / …** | Pentest workspace |
| **System → LLM** | Model configs (API keys, formats) |
| **System → Agents** | Agent keys, prompts, tools, triggers |
| **System → Skills / Tools / MCP** | Extensibility |
| **System → Notify** | Alert channels |
| **System → Settings** | Proxy, web search, traffic capture, updates |

### Recommended first-time order

1. Add at least one **LLM config** and mark it active (section 2).
2. Optionally attach that config to built-in agents (`planner`, `worker`, `auto`, `reporter`).
3. Create a **Task** with a clear description + goal (authorized target only).
4. Watch the task graph / chat; findings appear under **Findings**.
5. (Optional) Wire **Notify** so new findings ping Telegram/email.

### Built-in agents

Seeded on first boot. Edit prompts/tools/triggers carefully; avoid deleting unless you know the side effects.

| Key | Role |
|-----|------|
| `auto` | Platform operator assistant (chat / ops tools) |
| `planner` | Plans exploration intents for a task |
| `worker` | Executes one intent |
| `mainagent` | Human-facing task interface |
| `reporter` | Writes Markdown reports when `report_finding` fires |
| `retester` | Retests a finding in an isolated session |

UI and user-facing agent output are English (`NEXT_PUBLIC_LOCALE=en`).

---

## 2. Adding LLM models

Path: **System → LLM**

An **LLM config** (profile) is a named connection: format + model + API key (+ optional base URL / proxy / rate limits). Agents and tasks pick a profile or follow the **global active** one.

### Create a profile

1. Open **System → LLM**.
2. Click **Create** / **New**.
3. Fill:

| Field | Notes |
|-------|--------|
| **Name** | Display name (e.g. `Claude Sonnet`, `DeepSeek Flash`) |
| **Format** | Protocol adapter — see table below |
| **Model** | Model id for that provider |
| **API key** | Provider key (Bedrock: Access Key + Secret + Region) |
| **Base URL** | Optional override; empty = format default |
| **Proxy** | Optional HTTP(S) proxy for this profile only |
| **RPS / RPM / context window / max tokens** | Optional limits |

4. Save. Use **Test** / connection check if offered.

### Supported formats

| Format | Typical Base URL | Notes |
|--------|------------------|--------|
| `anthropic` | `https://api.anthropic.com` | Official Anthropic Messages API |
| `openai` | `https://api.openai.com/v1` | Chat Completions |
| `openai-responses` | `https://api.openai.com/v1` | OpenAI Responses API |
| `bedrock` | _(none — uses AWS)_ | Access Key \| Secret \| Region; Claude on Bedrock |
| `deepseek` | `https://api.deepseek.com/v1` | |
| `gemini` | `https://generativelanguage.googleapis.com/v1beta/openai` | OpenAI-compat Gemini |
| `zai` | `https://api.z.ai/api/paas/v4` | Z.ai / GLM |
| `openrouter` | `https://openrouter.ai/api/v1` | Prefer `anthropic/...` models |
| `moonshot` | `https://api.moonshot.ai/v1` | Kimi (use `.ai`, not `.cn`) |

### Activate / priority

- Mark one profile as **global active**.
- Call priority: **agent-attached → task/session chain → global active**.

### LLM pool (failover)

On the LLM page, open **Pool / Failover** if present: enable, order a chain, fail over on quota/overload. See on-page hints for cooldown / cursor behavior.

### Overrides

- **Task**: task detail → LLM control → config chain.
- **Agent**: **System → Agents → (agent) → Config & Prompt → Default model** → **Save config**.

### Env fallbacks (optional)

`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `ARTEX_LLM_PROVIDER`, `ARTEX_LLM_MODEL`, `ARTEX_LLM_BASE_URL`, `ARTEX_LLM_PROXY` — UI profiles are preferred for multi-model use.

### Troubleshooting

| Symptom | Check |
|---------|--------|
| “No LLM config” | Create + activate at least one profile |
| 401 / invalid key | Key, region (Bedrock), or Base URL |
| Empty model list | Enter model id manually |
| OpenRouter Anthropic errors | Use `anthropic/...` model ids |
| Moonshot region errors | Base URL `api.moonshot.ai` (not `.cn`) |

---

## 3. Adding agents

Path: **System → Agents**

An **agent** is prompt + tool bindings + optional LLM profile + triggers. Built-ins already exist; custom agents are for chat / specialized workflows.

### Create

1. **System → Agents** → **New**.
2. **Key**: `^[a-z][a-z0-9_]*$` (e.g. `triage_bot`).
3. **Name** / **Description** for the UI.
4. Create → editor opens.

### Config & Prompt

| Setting | Meaning |
|---------|---------|
| **Default model** | Pin a profile, or follow task / global active |
| **Max iterations** | Cap turns (`0` = unlimited) |
| **Run time (seconds)** | Wall-clock budget (`0` = unlimited) |
| **Web search** | Allow `web_search` (global web search must be on in Settings) |
| **Interactive Shell** | PTY tools for msfconsole / ssh / REPLs |
| **Prompt** | Empty = built-in default for known roles; click UI variables to insert |
| **Save config** | Persists model / limits / toggles |
| **Save as new version** / **Restore default** | Prompt versioning |

Always **Save config** after changing the LLM binding or toggles.

### Tools / skills / MCP

- Enable tools in the agent editor (or **System → Tools → Bind to agent**).
- Bind **Skills** and select **MCP** servers the agent may use.
- Prefer a minimal tool set for custom chat agents (fewer tokens, less noise).

### Wrapup

On max turns/time, settlement injects a wrapup (“persist findings, then one English summary”). Empty = built-in English default. **Task timeout wrapup** (worker/planner) is separate from per-run wrapup.

### Triggers

| Condition | Fires when |
|-----------|------------|
| **On tool call** | Named tool(s) return (e.g. `report_finding` → Report Writer) |
| **On finding** | Finding recorded |
| **On goal met** | Goal marked met |
| **On task create** | New task |
| **On task timeout** | Task deadline |
| **On interval** | Every N seconds |

Per trigger: **Message**, run mode (serial/parallel), merge mode, queue vs drop policy.

### Using agents

- **Chat**: **+ New Conversation** → pick agent (`Name (count)`).
- **Tasks**: planner/worker/mainagent run automatically; usually only tune prompts/tools.
- **Triggered**: fires from tool/finding/schedule (no manual chat).

Keep keys stable. After prompt edits, start a **new** conversation or wait for the next trigger.

---

## 4. Skills, tools & MCP

| Page | Role |
|------|------|
| **System → Skills** | Playbooks under `skills/` (`SKILL.md` + scripts); bind on the agent |
| **System → Tools** | Host tools + agent bindings (`auto` / `planner` / `worker` / `reporter` pre-seeded) |
| **System → MCP** | External MCP servers; enable, then select per agent |

**Settings** related: traffic capture (`traffic_*`), agent auto traffic binding, egress proxy, web search master switch.

---

## 5. Tasks & chat

### Chat

1. **Chat** → **+ New Conversation**.
2. Pick an agent (`All Agents`, `Auto`, `Report Writer`, …).
3. Send messages; optional per-conversation LLM switch if shown.

Use **Auto** for platform ops (“create a task…”). Use specialized agents for narrow jobs.

### Tasks

Path: **Function → Tasks**

1. **New task** — set description, **goal**, optional company/assets, LLM chain, timeout, planner heartbeat, seed-first-intent, coverage.
2. Start/resume → planner emits intents → workers execute → findings under **Findings**.
3. While running: graph, intents, worker traces, LLM records; pause/resume; add hints; soft/hard-delete bad intents.

**Report Writer** usually auto-runs on `report_finding`. Traffic evidence: manual link or Settings auto-binding (see `docs/finding-traffic-evidence-en.md`).

Only test systems you own or are authorized to test.

---

## 6. Notifications

Path: **System → Notify**

1. Create a channel (Telegram, email, webhook, Feishu, DingTalk, WeCom, …).
2. Fill credentials/endpoints → save → **Test** ping if available.
3. Events: new findings (single/digest), status changes, optional system notices.

Do not commit channel secrets to git or public zips.
