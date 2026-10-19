package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// (system/settings  update-card) progress.message·reason·writeErr  
// assertKoreanError·decodeErrorField   F3b  ( package server) .

func TestUpdateMessageConstantsLocalized(t *testing.T) {
	assertKoreanError(t, "updateMsgPreparing", updateMsgPreparing)
	assertKoreanError(t, "updateMsgFailed", updateMsgFailed)
	assertKoreanError(t, "updateMsgStaged", updateMsgStaged)
	assertKoreanError(t, "updateErrInProgress", updateErrInProgress)
	assertKoreanError(t, "updateErrRollbackInProgress", updateErrRollbackInProgress)

	notRelease := fmt.Sprintf(updateErrNotReleaseFmt, "v0.0.0-dev")
	assertKoreanError(t, "updateErrNotReleaseFmt", notRelease)
	if !strings.Contains(notRelease, "v0.0.0-dev") {
		t.Fatalf("updateErrNotReleaseFmt:    : %q", notRelease)
	}
	latest := fmt.Sprintf(updateErrAlreadyLatestFmt, "v1.2.3")
	assertKoreanError(t, "updateErrAlreadyLatestFmt", latest)
	if !strings.Contains(latest, "v1.2.3") {
		t.Fatalf("updateErrAlreadyLatestFmt:    : %q", latest)
	}
}

// begin/finish  SSE           .
func TestUpdateProgressMessagesLocalized(t *testing.T) {
	h := &updateHub{subs: map[chan updateProgress]struct{}{}}

	if !h.begin("v1.2.3") {
		t.Fatal("begin    true  ")
	}
	cur, running := h.snapshot()
	if !running {
		t.Fatal("begin  running  ")
	}
	if cur.Message != updateMsgPreparing {
		t.Fatalf("   : %q", cur.Message)
	}
	if h.begin("v1.2.4") {
		t.Fatal("  begin  false  ")
	}

	h.finish(errors.New(" "))
	cur, running = h.snapshot()
	if running {
		t.Fatal("finish  running   ")
	}
	if cur.Message != updateMsgFailed {
		t.Fatalf("  : %q", cur.Message)
	}

	h.finish(nil)
	cur, _ = h.snapshot()
	if cur.Message != updateMsgStaged {
		t.Fatalf("   : %q", cur.Message)
	}
}

// updateRollback  "   " 409   DB   HTTP  .
func TestUpdateRollbackInProgressLocalized(t *testing.T) {
	updHub.mu.Lock()
	prev := updHub.running
	updHub.running = true
	updHub.mu.Unlock()
	defer func() {
		updHub.mu.Lock()
		updHub.running = prev
		updHub.mu.Unlock()
	}()

	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/update/rollback", nil)
	rec := httptest.NewRecorder()
	s.updateRollback(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("  409 ,  %d ( %s)", rec.Code, rec.Body.Bytes())
	}
	msg := decodeErrorField(t, rec.Body.Bytes())
	assertKoreanError(t, "updateRollback   409", msg)
	if msg != updateErrRollbackInProgress {
		t.Fatalf("   : %q", msg)
	}
}
