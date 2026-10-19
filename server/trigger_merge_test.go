package server

import (
	"strings"
	"testing"
)

// A long task goal repeated per event was the dominant bloat. These tests pin the
// fix: the task-context header (description + goal) is rendered ONCE per task, no
// matter how many same-task fires are merged.

const longGoal = " f2-05  flag  submit_flag ；，flag ……"

func sameTaskFires(n int) []triggeredRun {
	items := make([]triggeredRun, n)
	for i := range items {
		items[i] = triggeredRun{
			agentKey: "tec_benchmark", taskID: 72, taskDesc: "f2-05 reverse", taskGoal: longGoal,
			message: "[Triggered by tool call]\nTool: submit_flag\nInput: {...}\nOutput: {correct:false}", mergeable: true,
		}
	}
	return items
}

func TestMergeAllRunsWritesTaskGoalOnce(t *testing.T) {
	out := mergeAllRuns(sameTaskFires(39))
	if got := strings.Count(out.message, longGoal); got != 1 {
		t.Fatalf("same-task goal should appear exactly once in a merged-all run, got %d", got)
	}
	if strings.Count(out.message, "-- trigger ") < 1 || !strings.Contains(out.message, "trigger 39") {
		t.Fatalf("all 39 event bodies should be present: %q", out.message)
	}
	if out.taskDesc != "" || out.taskGoal != "" {
		t.Fatalf("merged run must clear taskDesc/taskGoal to avoid a duplicate header")
	}
	if finalTriggerMessage(out) != out.message {
		t.Fatalf("finalTriggerMessage must not prepend another header for a merged run")
	}
}

func TestMergeAllRunsGroupsInterleavedTasks(t *testing.T) {
	mk := func(id int64, goal string) triggeredRun {
		return triggeredRun{agentKey: "a", taskID: id, taskDesc: "d", taskGoal: goal, message: "body", mergeable: true}
	}
	out := mergeAllRuns([]triggeredRun{mk(1, "GOAL_A"), mk(2, "GOAL_B"), mk(1, "GOAL_A"), mk(2, "GOAL_B")})
	if got := strings.Count(out.message, "GOAL_A"); got != 1 {
		t.Fatalf("task #1 goal should appear once despite interleaving, got %d", got)
	}
	if got := strings.Count(out.message, "GOAL_B"); got != 1 {
		t.Fatalf("task #2 goal should appear once despite interleaving, got %d", got)
	}
	if !strings.Contains(out.message, "2 tasks total") {
		t.Fatalf("header should report 2 tasks: %q", out.message)
	}
	if got := strings.Count(out.message, "-- trigger "); got != 4 {
		t.Fatalf("all 4 event bodies should be present, got %d", got)
	}
}

func TestMergeTriggeredRunsWritesTaskGoalOnce(t *testing.T) {
	out := mergeTriggeredRuns(sameTaskFires(5))
	if got := strings.Count(out.message, longGoal); got != 1 {
		t.Fatalf("same-task goal should appear exactly once in a by-task merge, got %d", got)
	}
}

func TestFinalTriggerMessageSingleFirePrependsHeaderOnce(t *testing.T) {
	item := sameTaskFires(1)[0]
	msg := finalTriggerMessage(item)
	if got := strings.Count(msg, longGoal); got != 1 {
		t.Fatalf("single fire should carry the task goal exactly once, got %d", got)
	}
	if !strings.HasPrefix(msg, "[Task #72") {
		t.Fatalf("single fire should be prefixed with the task-context header: %q", msg)
	}
}

func TestTaskContextHeaderEmptyForIntervalFire(t *testing.T) {
	if h := taskContextHeader(0, "", ""); h != "" {
		t.Fatalf("interval/none trigger (no task) must produce no header, got %q", h)
	}
	item := triggeredRun{message: "scheduled body"}
	if finalTriggerMessage(item) != "scheduled body" {
		t.Fatalf("interval fire message must pass through unchanged")
	}
}

func TestTaskContextHeaderTruncatesLongGoal(t *testing.T) {
	huge := strings.Repeat("x", 5000)
	h := taskContextHeader(72, "d", huge)
	if len([]rune(h)) > 800 {
		t.Fatalf("header should be bounded even for a huge goal, got %d runes", len([]rune(h)))
	}
}

func TestTriggerSynthesisEnglishFraming(t *testing.T) {
	if h := taskContextHeader(7, "d", "g"); !strings.Contains(h, "(Goal:") {
		t.Fatalf("task context header goal framing changed: %q", h)
	}
	if out := mergeTriggeredRuns(sameTaskFires(3)); !strings.Contains(out.message, "This session merged") {
		t.Fatalf("by-task merge intro framing changed: %q", out.message)
	}
}
