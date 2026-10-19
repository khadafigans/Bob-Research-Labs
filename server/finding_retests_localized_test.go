package server

import "testing"

// finding_retests.go    HTTP       
// .   F3a   assertKoreanError( ·  0)
// .    startFindingRetest    s.pg(w)(DB)
// (finding_traffic·goals_api·notify_api  DB    ).  
//
//  (get_finding_retest_context·record_finding_retest_result)· ·
// actool.Errorf(  ) seedFindingRetester  DB   ·
// ,     (finding_retests.go    ).
func TestFindingRetestErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"notes_too_long", errFindingRetestNotesTooLong},
		{"agent_missing", errFindingRetestAgentMissing},
		{"tool_required", errFindingRetestToolRequired},
		{"service_stopping", errFindingRetestServiceStopping},
	} {
		assertKoreanError(t, c.label, c.msg)
	}
}
