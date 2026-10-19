package intercept

import (
	"encoding/json"
	"io"
	"strings"
)

// The application owns the envelope contract, including for saved custom prompts.
const JudgeContextBoundary = `# 
 JSON。 tool_name  arguments（）；working_directory  Agent ， Shell 。
background ，source=user_message。Worker ， Worker ， Agent 。，，。
、、、 Worker 。， Agent 、。、、；。
、、。，，。
；、。，。，；，。
， /srv、/var、/data ， /tmp、test、fixture 。，；，“”“”。
background.truncated  true ；。，、、。
。，，。`

func EffectiveJudgePrompt(prompt string) string {
	if !strings.Contains(prompt, JudgeContextBoundary) {
		prompt += "\n\n" + JudgeContextBoundary
	}
	if !strings.Contains(prompt, JudgeOutputContract) {
		prompt += "\n\n" + JudgeOutputContract
	}
	return prompt
}

// Output is an application contract, also applied to saved custom policies.
// It changes the explanation format, not the user's policy or rule precedence.
const JudgeOutputContract = `# （，）
 JSON ： {、 }。、、（）；JSON 。
 decision  comment ；。 YAML  decision: ... / comment: ...。
decision  allow、ask、deny，、、。
comment “：…；：…；：…”，；、， comment  120 （，）。
： tool_name  arguments ；background 、Write/Edit （ command  cat “”）。
：，。
：（： A1–A6、 D1–D6、 ASK、 DEFAULT），。
`

// DefaultJudgePrompt is the built-in system prompt for the LLM fallback judge.
// It runs ONLY on tool calls that no intercept rule matched, and classifies the
// single call into ALLOW / ASK / DENY under authorized-pentest semantics: block
// only direct, hard-to-undo damage to the target's real assets; allow normal
// offensive/recon/read techniques; ask a human only when genuinely undecidable.
//
// Users may customize the policy via llm_judge_prompt; an empty setting uses
// this constant. EffectiveJudgePrompt always appends the input trust boundary.
const DefaultJudgePrompt = `# 
「」。,「」。。、,:

    「」「」?

 → DENY;(//)→ ALLOW;、 → ASK。

「、」,:

1) :,?
   - 、、、/ → ,。
   - 、、、、、 → 。

2) :「」?
   -  / :、、、。 test/bak、 /tmp 、Worker ;。
   - :、/、、、 → 。

:   = DENY。 ALLOW。、 = ASK。

- ,、。、、、。。
-  ALLOW。 /etc/shadow、.env、、——「」。
- ,,。 ASK。
- 「」「」「」 DENY 。DENY 。
- 。「」「 ALLOW」「…」,,。

D1 :/、、,。
D2 :/、、,。,,。
D3 :、Web/DB/、、、。
D4 :///——(DELETE/PUT/PATCH 、、、,), DROP/TRUNCATE/ UPDATE/DELETE 、rm 、、。
D5 ://(systemctl stop、kill 、shutdown、reboot)。
D6 : flood /  / ( hping3 --flood、-c 100000、)。
:D1–D6 ; D4。

A1  payload:SQLi(UNION///)、XSS、、SSTI、、SSRF、XXE、、。
A2 、,;、。
A3 :、、(hydra/medusa/ncrack ),。
A4 ://、、、、。
A5 :,、file://、、、、、、(--dump)。
A6 :whoami/id/uname/hostname/ls/cat/ps/netstat/ifconfig 。

- //,「」「/」(:drop 、)。
- ,「」「//」。
- :ASK 「」,「」。 ALLOW, DENY; ASK,。

1. , D1–D6 ?→  DENY。
2.  A1–A6 ?→  ALLOW。
3. //,?→ ASK。
4.  →  ALLOW(,)。

ALLOW  execute_command cat /etc/shadow                         ,。
ALLOW  http GET /view?file=../../../etc/passwd                 ,。
ALLOW  sqlmap -u '...' --dump                                  /,。
ALLOW   shell.php  /uploads/                              ,。
ALLOW  hydra -l admin -P rockyou.txt -t 4 tgt http-post-form   , DoS。
ALLOW  POST /login  {user:admin,pass:123456}                   。
DENY   rm -rf /var/www/html                                    ,+(D4)。
DENY   http DELETE /api/orders/8821                            (D4)。
DENY   mysql -e "UPDATE accounts SET balance=0"                (D4)。
DENY   passwd admin / UPDATE users SET pass=... WHERE       (D1)。
DENY   systemctl stop nginx                                    (D5)。
DENY   hping3 --flood -p 80 tgt                                 flood(D6)。
ASK    mysql -e "DROP TABLE users_bak_0921"                    ,。
ASK     /data/uploads                     。
ASK    ，                         ，。

，：
：{"decision":"allow","comment":"：；：，；：A2"}
（ cat report.md）：{"decision":"allow","comment":"： report.md ；：，；：A5"}
：{"decision":"ask","comment":"：；：，；：ASK（）"}
：{"decision":"deny","comment":"：；：；：D4"}
` + JudgeOutputContract

// Verdict is the parsed outcome of the judge's JSON reply.
type Verdict struct {
	Action string // "allow" | "ask" | "deny" | "" (unparseable)
	Reason string
}

// stripCodeFence unwraps a fenced reply (```json … ```) before strict parsing.
// This is a deterministic unwrap, not a repair: the payload still goes through
// ParseVerdict unchanged, so truncated, ambiguous or prose replies stay
// unparseable. A reply cut off at MaxTokens has no closing fence and is left
// alone on purpose — completing it would invent a verdict the model never gave.
//
// It exists because the fail action defaults to allow: without it a model that
// merely wraps its JSON in markdown turns a DENY into a silent allow.
func stripCodeFence(text string) string {
	t := strings.TrimSpace(text)
	if len(t) <= 6 || !strings.HasPrefix(t, "```") || !strings.HasSuffix(t, "```") {
		return t
	}
	t = strings.TrimSpace(t[3 : len(t)-3])
	if !strings.HasPrefix(t, "{") {
		// Drop the opening fence's language tag line (```json).
		if _, rest, ok := strings.Cut(t, "\n"); ok {
			t = strings.TrimSpace(rest)
		}
	}
	return t
}

// ParseVerdict requires a complete verdict and explanation for every action.
// Never extract a decision keyword from prose, arguments, or a broken JSON
// reply. Invalid/incomplete responses follow the configured model-failure path.
func ParseVerdict(text string) Verdict {
	d := json.NewDecoder(strings.NewReader(stripCodeFence(text)))
	if tok, err := d.Token(); err != nil || tok != json.Delim('{') {
		return Verdict{}
	}
	fields := map[string]string{}
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return Verdict{}
		}
		key, ok := tok.(string)
		if _, duplicate := fields[key]; !ok || duplicate || (key != "decision" && key != "comment") {
			return Verdict{}
		}
		var value *string
		if d.Decode(&value) != nil || value == nil {
			return Verdict{}
		}
		fields[key] = *value
	}
	if tok, err := d.Token(); err != nil || tok != json.Delim('}') {
		return Verdict{}
	}
	if _, err := d.Token(); err != io.EOF || len(fields) != 2 {
		return Verdict{}
	}
	action, reason := fields["decision"], strings.TrimSpace(fields["comment"])
	if action != "allow" && action != "ask" && action != "deny" {
		return Verdict{}
	}
	if len(reason) > 2400 || !strings.HasPrefix(reason, "：") {
		return Verdict{}
	}
	operation, rest, ok := strings.Cut(strings.TrimPrefix(reason, "："), "；：")
	if !ok || strings.TrimSpace(operation) == "" {
		return Verdict{}
	}
	consequence, rule, ok := strings.Cut(rest, "；：")
	if !ok || strings.TrimSpace(consequence) == "" || strings.TrimSpace(rule) == "" {
		return Verdict{}
	}
	return Verdict{Action: action, Reason: reason}
}
