package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Autumn-27/norma/harness"
)

// runTrace retains the latest tool call so an interrupted run can identify the
// operation that was still in flight.
type runTrace struct {
	startedAt time.Time
	id        string
	name      string
	input     string
	at        time.Time
	pending   bool
}

func (t *runTrace) start(id, name, input string) {
	t.id, t.name, t.input, t.at, t.pending = id, name, input, time.Now(), true
}

func (t *runTrace) done(id string) {
	if id == t.id {
		t.pending = false
	}
}

var reasonHint = map[harness.TerminalReason]string{
	harness.ReasonCompleted:         "，；",
	harness.ReasonMaxTurns:          "(MaxTurns)：SDK ， exhausted，，",
	harness.ReasonTimeout:           "(MaxDuration)：，， exhausted",
	harness.ReasonModelError:        " API （、、、 5xx ）， blocked——；（get_worker_trace）",
	harness.ReasonBlockingLimit:     "，；",
	harness.ReasonPromptTooLong:     "，",
	harness.ReasonImageError:        "；",
	harness.ReasonStopHookPrevented: "Stop ，； Guard ",
	harness.ReasonHookStopped:       "，； tool_result ",
	harness.ReasonAbortedStreaming:  "",
	harness.ReasonAbortedTools:      "",
}

// terminalText renders a terminal event with no final text into a compact summary
// and a Markdown detail block.
func terminalText(ctx context.Context, term *harness.Terminal, tr *runTrace) (string, string) {
	reason := term.Reason
	aborted := reason == harness.ReasonAbortedStreaming || reason == harness.ReasonAbortedTools
	// Prompt may return ctx.Err directly without a terminal event. Preserve the
	// cancellation cause instead of falling back to an empty/unknown terminal reason.
	if reason == "" && ctx.Err() != nil {
		aborted = true
	}

	var sum string
	if aborted {
		_, short, _, ok := AbortReason(ctx)
		if !ok {
			short = ""
		}
		stage := ""
		switch reason {
		case harness.ReasonAbortedStreaming:
			stage = ""
		case harness.ReasonAbortedTools:
			stage = ""
		}
		sum = "（：" + short + "；" + stage + progressSuffix(term, tr) + "，）"
	} else if reason == harness.ReasonMaxTurns || reason == harness.ReasonTimeout {
		sum = "（(" + string(reason) + ")，" + progressSuffix(term, tr) + "；）"
	} else {
		hint := terminalReasonHint(reason)
		sum = "（， " + terminalReasonLabel(reason) + "：" + firstLine(hint, 80) + "）"
	}

	var b strings.Builder
	b.WriteString(sum)
	b.WriteString("\n\n")
	displayReason := terminalReasonLabel(reason)
	fmt.Fprintf(&b, "- ****: `%s` - %s\n", displayReason, terminalReasonHint(reason))
	if aborted {
		code, _, why, ok := AbortReason(ctx)
		if ok {
			fmt.Fprintf(&b, "- **** (`%s`): %s\n", code, why)
		} else {
			b.WriteString("- ****: ； context.WithCancelCause \n")
		}
	}
	if term.Err != nil {
		fmt.Fprintf(&b, "- ****: `%v`\n", term.Err)
	}
	if aborted && strings.TrimSpace(term.Text) != "" {
		b.WriteString("- ****:\n\n")
		b.WriteString(term.Text)
		b.WriteString("\n\n")
	}
	if term.Turns > 0 {
		fmt.Fprintf(&b, "- ****: %d \n", term.Turns)
	}
	if !tr.startedAt.IsZero() {
		fmt.Fprintf(&b, "- ****: %s\n", roundDur(time.Since(tr.startedAt)))
	}
	if u := term.Usage; u.InputTokens+u.OutputTokens+u.CacheReadTokens+u.CacheWriteTokens > 0 {
		fmt.Fprintf(&b, "- ** token**:  %d /  %d /  %d /  %d\n",
			u.InputTokens, u.OutputTokens, u.CacheReadTokens, u.CacheWriteTokens)
	}
	if tr.name == "" {
		b.WriteString("- ****: \n")
	} else if tr.pending {
		fmt.Fprintf(&b, "- ****: `%s`（ %s，****）\n\n  ```json\n  %s\n  ```\n",
			tr.name, roundDur(time.Since(tr.at)), firstLine(tr.input, 300))
	} else {
		fmt.Fprintf(&b, "- ****: `%s`（）\n", tr.name)
	}
	return sum, b.String()
}

func terminalReasonLabel(reason harness.TerminalReason) string {
	if reason == "" {
		return "context_canceled"
	}
	return string(reason)
}

func terminalReasonHint(reason harness.TerminalReason) string {
	if hint := reasonHint[reason]; hint != "" {
		return hint
	}
	if reason == "" {
		return " context ， Terminal "
	}
	return "；harness  TerminalReason， reasonHint"
}

func progressSuffix(term *harness.Terminal, tr *runTrace) string {
	var parts []string
	if term.Turns > 0 {
		parts = append(parts, fmt.Sprintf("%d ", term.Turns))
	}
	if !tr.startedAt.IsZero() {
		parts = append(parts, roundDur(time.Since(tr.startedAt)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "， " + strings.Join(parts, " / ")
}

func roundDur(d time.Duration) string {
	switch {
	case d < time.Minute:
		return d.Round(100 * time.Millisecond).String()
	case d < time.Hour:
		return d.Round(time.Second).String()
	default:
		return d.Round(time.Minute).String()
	}
}
