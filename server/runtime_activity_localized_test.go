package server

import (
	"fmt"
	"strings"
	"testing"
)

// engine.go·engine_timeout.go·goals.go  " "     
//    .    (activity transcript) 
// (planner.workerOutput·get_worker_output ) intent (node_id) 'result'/
// )      .   assertKoreanError(
// ·  0, intercept_archive_localized_test.go) . [[G132]]
func TestRuntimeActivitySummariesLocalized(t *testing.T) {
	plain := map[string]string{
		"goalless_task_done": goallessTaskDoneSummary,
		"goal_breakdown_r0":  goalBreakdownRound0Summary,
		"queued_no_llm":      queuedNoLLMSummary,
		"queued_fifo":        queuedFIFOSummary,
	}
	for label, msg := range plain {
		assertKoreanError(t, label, msg)
	}

	fmtCases := map[string]string{
		"planner_round":       plannerRoundSummaryFmt,
		"timeout_final_round": timeoutFinalRoundSummaryFmt,
		"queued_conc_limit":   queuedConcurrencyLimitSummaryFmt,
	}
	for label, f := range fmtCases {
		if !strings.Contains(f, "%d") {
			t.Fatalf("%s:   %%d  : %q", label, f)
		}
		got := fmt.Sprintf(f, 7)
		assertKoreanError(t, label, got)
		if !strings.Contains(got, "7") {
			t.Fatalf("%s:     : %q", label, got)
		}
	}
}
