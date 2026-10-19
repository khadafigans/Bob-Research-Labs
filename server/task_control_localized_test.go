package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// task_control.go  ·  API       .
//   F3a  assertKoreanError( ·  0),   
// task_categories  decodeErrorField  ( package server).

// TestTaskControlErrorConstantsLocalized     10  
// .   (applyIntentControl)   t.Store.GetNode(DB) 
// (intent_intervention·goals_api ).  ·     
func TestTaskControlErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"task_deleting":          errTaskCtrlDeleting,
		"task_terminal_pause":    errTaskCtrlTerminalPause,
		"task_already_paused":    errTaskCtrlAlreadyPaused,
		"batch_size":             errTaskCtrlBatchSizeFmt,
		"intent_inherited":       errIntentCtrlInheritedReadonly,
		"intent_only_running":    errIntentCtrlOnlyRunningPause,
		"intent_only_paused":     errIntentCtrlOnlyPausedResume,
		"intent_state_conflict":  errIntentCtrlStateConflictFmt,
		"intent_only_deletable":  errIntentCtrlOnlyDeletable,
		"intent_reason_required": errIntentCtrlReasonRequired,
	}
	for label, msg := range cases {
		assertKoreanError(t, label, msg)
	}
}

// TestTaskControlApplyResponsesLocalized  DB·     pause 
//  applyTaskControl   ,      
// .    s.m.Task( ) →   → t.lifecycleSnapshot()(
//  Status/Paused  ) , tasks     Engine  
// Store·DB  .  ( ) ApplyTaskPause(DB) 
func TestTaskControlApplyResponsesLocalized(t *testing.T) {
	//   : isTerminalStatus(done) .
	terminalServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "done"}
		return &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}, tk
	}
	//    : lifecycle.Paused .
	pausedServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "running", Paused: true}
		return &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}, tk
	}
	deletingServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "running"}
		s := &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}
		s.engine.deleting.Store("1", true)
		return s, tk
	}

	cases := []struct {
		name  string
		setup func() (*Server, *Task)
		want  string
	}{
		{"terminal", terminalServer, errTaskCtrlTerminalPause},
		{"already-paused", pausedServer, errTaskCtrlAlreadyPaused},
		{"deleting", deletingServer, errTaskCtrlDeleting},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, tk := c.setup()
			_, err := s.applyTaskControl(tk, "pause")
			if err == nil {
				t.Fatalf("  nil  ")
			}
			if err.Error() != c.want {
				t.Fatalf("  = %q,  = %q", err.Error(), c.want)
			}
			assertKoreanError(t, c.name, err.Error())
		})
	}
}

// TestControlTasksBatchResponsesLocalized       
// HTTP  . ①  task_ids  s.m   writeErr  400 +   
// &Server{}  . ②      pause  200 items[].error 
//     , applyTaskControl   JSON   
func TestControlTasksBatchResponsesLocalized(t *testing.T) {
	t.Run("size-error", func(t *testing.T) {
		s := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/control",
			strings.NewReader(`{"task_ids":[],"action":"pause"}`))
		rec := httptest.NewRecorder()
		s.controlTasksBatch(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("  = %d,  = 400 ( %q)", rec.Code, rec.Body.String())
		}
		got := decodeErrorField(t, rec.Body.Bytes())
		want := fmt.Sprintf(errTaskCtrlBatchSizeFmt, maxBatchControlIDs)
		if got != want {
			t.Fatalf("  = %q,  = %q", got, want)
		}
		if !strings.Contains(got, "100") {
			t.Fatalf("  100    : %q", got)
		}
		assertKoreanError(t, "size-error", got)
	})

	t.Run("item-error-surfaces", func(t *testing.T) {
		tk := &Task{ID: "1", Status: "done"}
		s := &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/control",
			strings.NewReader(`{"task_ids":["1"],"action":"pause"}`))
		rec := httptest.NewRecorder()
		s.controlTasksBatch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("  = %d,  = 200 ( %q)", rec.Code, rec.Body.String())
		}
		var out struct {
			Items []struct {
				ID    string `json:"id"`
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf(" JSON  : %v ( %q)", err, rec.Body.String())
		}
		if len(out.Items) != 1 {
			t.Fatalf("items  = %d,  = 1", len(out.Items))
		}
		if out.Items[0].OK || out.Items[0].Error != errTaskCtrlTerminalPause {
			t.Fatalf("items[0] = %+v, error  = %q", out.Items[0], errTaskCtrlTerminalPause)
		}
		assertKoreanError(t, "item-error", out.Items[0].Error)
	})
}

// TestIntentStateConflictWrapsSentinel       %w 
// db.ErrIntentStateConflict   , errors.Is    .
func TestIntentStateConflictWrapsSentinel(t *testing.T) {
	err := fmt.Errorf(errIntentCtrlStateConflictFmt, db.ErrIntentStateConflict)
	if !errors.Is(err, db.ErrIntentStateConflict) {
		t.Fatalf("errors.Is(.., ErrIntentStateConflict) = false, %%w  : %q", err.Error())
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Fatalf("enum paused    : %q", err.Error())
	}
	assertKoreanError(t, "state-conflict", err.Error())
}
