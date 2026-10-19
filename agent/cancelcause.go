package agent

import (
	"context"
	"errors"
	"fmt"
)

// AbortCause names why an agent run's context was cancelled. Every cancellation
// site should attach one so the activity trace can report the real initiator.
type AbortCause struct {
	Code  string
	Short string
	Text  string
}

func (c *AbortCause) Error() string { return c.Text }

func cause(code, short, text string) *AbortCause {
	return &AbortCause{Code: code, Short: short, Text: text}
}

// Causef builds a cause that includes runtime-specific detail.
func Causef(code, short, format string, args ...any) *AbortCause {
	return &AbortCause{Code: code, Short: short, Text: fmt.Sprintf(format, args...)}
}

var (
	// Task-level execution context.
	AbortPausedByUser = cause("paused_by_user", "",
		"（POST /api/tasks/{id}/control，action=pause）。 Planner/Worker ； frontier(open)，")
	AbortPausedByOrchestrator = cause("paused_by_orchestrator", " Agent ",
		" Agent  pause_task 。 Planner/Worker ； frontier(open)，")
	AbortTaskDeleted = cause("task_deleted", "",
		"（DELETE /api/tasks/{id}）， Planner、Worker  Agent；")
	AbortPausedOnReload = cause("paused_on_reload", "",
		"。； Agent")
	AbortGoalMet = cause("goal_met", "",
		" done， Worker； stopped，")
	AbortSettleDrainTimeout = cause("settle_drain_timeout", "",
		" timeout  Worker ， 90  drain ，； exhausted，")

	// Per-work context.
	AbortKilledByPlanner = cause("killed_by_planner", "",
		" kill_work ，； stopped，")
	AbortWorkPausedByUser = cause("work_paused_by_user", " Worker ",
		" Worker。， paused；、、，")
	AbortWorkCancelledByUser = cause("work_cancelled_by_user", " Worker ",
		" Worker。；Worker ，——，")
	AbortWorkFinished = cause("work_finished", "Worker  context",
		"Worker ， detachWork  context 。；，")
	AbortPausedRaceGuard = cause("paused_race_guard", "",
		"， context， claim  Worker ； frontier")

	// Main Agent and standalone conversation contexts.
	AbortChatStoppedByUser = cause("chat_stopped_by_user", "",
		"， Agent  Agent 。，")
	AbortChatPausedWithTask = cause("chat_paused_with_task", " Agent ",
		"， Agent 。；")
	AbortChatTurnFinished = cause("chat_turn_finished", " context",
		"， context 。；，")

	// Process-level and per-run hard backstop.
	AbortShutdown = cause("shutdown", "",
		" SIGINT  SIGTERM，、。 Agent ； running  open ")
	AbortRunHardTimeout = cause("run_hard_timeout", "",
		"，，。")
)

// AbortReason resolves the named cause attached to a cancelled run context.
func AbortReason(ctx context.Context) (code, short, text string, ok bool) {
	c := context.Cause(ctx)
	if c == nil {
		return "", "", "", false
	}
	var ac *AbortCause
	if errors.As(c, &ac) {
		return ac.Code, ac.Short, ac.Text, true
	}
	switch {
	case errors.Is(c, context.DeadlineExceeded):
		return "deadline_exceeded", " context  deadline",
			" context  deadline， WithTimeoutCause : " + c.Error(), true
	case errors.Is(c, context.Canceled):
		return "canceled_no_cause", "",
			" context ， context.WithCancelCause ； agent/cancelcause.go ", true
	default:
		return "other", firstLine(c.Error(), 80), c.Error(), true
	}
}
