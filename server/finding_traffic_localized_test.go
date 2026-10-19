package server

import "testing"

// finding_traffic.go    HTTP       
// .   F3a   assertKoreanError( ·  0)
// .    (findingTrafficAccess·bindFindingTraffic·
// editFindingTraffic)  s.m.pg.GetFinding(DB)   DB   
//        (goals_api·notify_api  DB 
//
// readEvidencePreview  errors.New(offset/length )·  
// roTool("get_finding_traffic")       
func TestFindingTrafficErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"inherited_readonly", errFindingTrafficInheritedReadonly},
		{"select_required", errFindingTrafficSelectRequired},
		{"version_required", errFindingTrafficVersionRequired},
		{"binding_ids_required", errFindingTrafficBindingIDsRequired},
	} {
		assertKoreanError(t, c.label, c.msg)
	}
}
