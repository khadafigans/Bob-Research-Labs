package server

import "testing"

// goals.go   (admitPausedTask)       
//   .   F3a   assertKoreanError( ·
//  0) .    admitTaskWhen(requirePaused=true) 
// s.concMu   s.m.Task()·s.engine.IsDeleting()·beginTaskOperation()  
// (finding_retests·finding_traffic      ).   
//
//   (goals.go    ):   (server.go:1194 →
// writeErr 409)·(task_control.go:298 → items[].error)    
//  .  (orchestration.go:343) action="pause"  
func TestGoalResumeErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"resume_terminal", errGoalResumeTerminal},
		{"resume_not_paused", errGoalResumeNotPaused},
	} {
		assertKoreanError(t, c.label, c.msg)
	}
}
