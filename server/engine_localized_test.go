package server

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Autumn-27/artex/db"
)

//  .   F3a   assertKoreanError( ·  0)
//
//   - ControlWork   (  work ·  ·  · 
//     ) → applyIntentControl(task_control.go) → controlIntent(server.go:1236) →
//     writeErr 409.    (actool)   .
//   - runDetachedIntent   (Worker · CAS ) →
//     sendWorkerMessage(intent_intervention.go:147) → writeErr.  .
//   -  SteerWork·KillWork(steer_work·kill_work  actool.Errorf)·
//     transitionIntentState(   )   ·  

// TestControlWorkNoRunningWorkErrorLocalized    work    ControlWork 
//     DB·   . run==nil  e.work 
//  NewEngine(nil)   . %w  errWorkControlConflict  
func TestControlWorkNoRunningWorkErrorLocalized(t *testing.T) {
	e := NewEngine(nil)
	err := e.ControlWork(context.Background(), 42, "pause")
	if err == nil {
		t.Fatal("  work   ControlWork  nil  ")
	}
	if !errors.Is(err, errWorkControlConflict) {
		t.Fatalf("errors.Is(err, errWorkControlConflict) = false, err=%v", err)
	}
	assertKoreanError(t, "control_work_no_running", err.Error())
}

// TestEngineUserFacingErrorsLocalized  /DB      
//         (finding_retests·
// finding_traffic      ).       .
func TestEngineUserFacingErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"work_control_busy", fmt.Errorf(errWorkControlBusyFmt, errWorkControlConflict, int64(42), "pause").Error()},
		{"work_control_wait", fmt.Errorf(errWorkControlWaitFmt, int64(42), "pause", context.DeadlineExceeded).Error()},
		{"detached_worker_not_ready", errDetachedWorkerNotReady},
		{"detached_state_conflict", fmt.Errorf(errIntentCtrlStateConflictFmt, db.ErrIntentStateConflict).Error()},
	} {
		assertKoreanError(t, c.label, c.msg)
	}
}
