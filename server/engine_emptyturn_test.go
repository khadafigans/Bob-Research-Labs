package server

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/Autumn-27/norma/llm"
)

// (、)， steerHooks.Stop。

func assistantThinking(text string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{
		{Type: llm.BlockThinking, Thinking: text, Signature: "sig"},
	}}
}

func TestIsThinkingOnlyTurn(t *testing.T) {
	toolUse := llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{
		{Type: llm.BlockThinking, Thinking: ""},
		{Type: llm.BlockToolUse, ID: "t1", Name: "run_nuclei"},
	}}
	cases := []struct {
		name string
		msgs []llm.Message
		want bool
	}{
		{"", []llm.Message{llm.UserText(""), assistantThinking("")}, true},
		{"+", []llm.Message{llm.UserText(""), toolUse}, false},
		{"+", []llm.Message{assistantThinking(""), {
			Role:    llm.RoleAssistant,
			Content: []llm.ContentBlock{{Type: llm.BlockThinking, Thinking: "x"}, llm.TextBlock("")},
		}}, false},
		{"", []llm.Message{{
			Role:    llm.RoleAssistant,
			Content: []llm.ContentBlock{{Type: llm.BlockThinking, Thinking: "x"}, llm.TextBlock("  \n ")},
		}}, true},
		{" assistant ", []llm.Message{{Role: llm.RoleAssistant}}, true},
		//  user ， assistant，。
		{"", []llm.Message{toolUse, {
			Role:    llm.RoleUser,
			Content: []llm.ContentBlock{{Type: llm.BlockToolResult, ToolUseID: "t1"}},
		}}, false},
		{" assistant ", []llm.Message{llm.UserText("")}, false},
		{"", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isThinkingOnlyTurn(c.msgs); got != c.want {
				t.Fatalf("isThinkingOnlyTurn = %v, want %v", got, c.want)
			}
		})
	}
}

// fakeHooks  inner HookRunner， steerHooks  inner 。
type fakeHooks struct {
	prevent  bool
	blocking []string
	msg      string
}

func (f fakeHooks) PreToolUse(context.Context, string, []byte) (bool, string, []byte) {
	return false, "", nil
}
func (f fakeHooks) PostToolUse(context.Context, string, []byte, []byte, bool) {}
func (f fakeHooks) Stop(context.Context, []llm.Message) (bool, []string, string) {
	return f.prevent, f.blocking, f.msg
}

func TestSteerHooksStopNudgesEmptyTurn(t *testing.T) {
	empty := []llm.Message{assistantThinking("")}

	t.Run("", func(t *testing.T) {
		h := steerHooks{nudges: &atomic.Int64{}, limit: defaultEmptyTurnNudges, label: "worker-1 · #1"}
		prevent, blocking, _ := h.Stop(context.Background(), empty)
		if prevent {
			t.Fatal("")
		}
		if len(blocking) != 1 || blocking[0] != emptyTurnNudge {
			t.Fatalf("blocking = %v, want [emptyTurnNudge]", blocking)
		}
	})

	t.Run("", func(t *testing.T) {
		h := steerHooks{nudges: &atomic.Int64{}, limit: defaultEmptyTurnNudges}
		normal := []llm.Message{{
			Role:    llm.RoleAssistant,
			Content: []llm.ContentBlock{llm.TextBlock("，")},
		}}
		if _, blocking, _ := h.Stop(context.Background(), normal); blocking != nil {
			t.Fatalf(": %v", blocking)
		}
		if n := h.nudges.Load(); n != 0 {
			t.Fatalf(", got %d", n)
		}
	})

	t.Run("", func(t *testing.T) {
		const limit = 5 // 「」 5
		h := steerHooks{nudges: &atomic.Int64{}, limit: limit}
		for i := 1; i <= limit; i++ {
			if _, blocking, _ := h.Stop(context.Background(), empty); len(blocking) != 1 {
				t.Fatalf(" %d , blocking = %v", i, blocking)
			}
		}
		if _, blocking, _ := h.Stop(context.Background(), empty); blocking != nil {
			t.Fatalf(": %v", blocking)
		}
	})

	// 「」 -1 = ，emptyTurnNudgeLimit  0。
	t.Run("", func(t *testing.T) {
		h := steerHooks{nudges: &atomic.Int64{}, limit: 0}
		if _, blocking, _ := h.Stop(context.Background(), empty); blocking != nil {
			t.Fatalf(": %v", blocking)
		}
	})

	t.Run("inner ", func(t *testing.T) {
		h := steerHooks{inner: fakeHooks{prevent: true, msg: "guard "}, nudges: &atomic.Int64{}, limit: defaultEmptyTurnNudges}
		prevent, blocking, msg := h.Stop(context.Background(), empty)
		if !prevent || msg != "guard " || blocking != nil {
			t.Fatalf("inner : prevent=%v blocking=%v msg=%q", prevent, blocking, msg)
		}
		if n := h.nudges.Load(); n != 0 {
			t.Fatalf(" inner , got %d", n)
		}
	})

	t.Run("inner ", func(t *testing.T) {
		h := steerHooks{inner: fakeHooks{blocking: []string{"guard "}}, nudges: &atomic.Int64{}, limit: defaultEmptyTurnNudges}
		_, blocking, _ := h.Stop(context.Background(), empty)
		if len(blocking) != 1 || blocking[0] != "guard " {
			t.Fatalf("inner : %v", blocking)
		}
	})

	t.Run("", func(t *testing.T) {
		h := steerHooks{limit: defaultEmptyTurnNudges} //  nudges
		if _, blocking, _ := h.Stop(context.Background(), empty); blocking != nil {
			t.Fatalf(": %v", blocking)
		}
	})
}
