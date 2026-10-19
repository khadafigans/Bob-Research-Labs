package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// intent_intervention.go  Worker  API      
// .   F3a   assertKoreanError( ·  0) .

// TestIntentInterventionErrorConstantsLocalized    11   .
// ·   s.engine·Store   DB     
// ,     (conversations.go ).    4
func TestIntentInterventionErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"request_too_large", errIntentRequestTooLarge},
		{"message_empty", errIntentMessageEmpty},
		{"message_too_long", errIntentMessageTooLong},
		{"bad_request_id", errIntentBadRequestID},
		{"task_deleting", errIntentTaskDeleting},
		{"task_paused", errIntentTaskPaused},
		{"task_queued", errIntentTaskQueued},
		{"task_terminal", errIntentTaskTerminal},
		{"task_settling", errIntentTaskSettling},
		{"inherited_readonly", errIntentInheritedReadonly},
		{"not_paused", errIntentNotPaused},
	}
	for _, c := range cases {
		assertKoreanError(t, c.name, c.msg)
	}
}

// TestSendWorkerMessageInputValidationLocalized     4  HTTP 
//  . sendWorkerMessage   4 s.m.Task( )   
// s.engine·Store·DB   , tasks     DB   .
func TestSendWorkerMessageInputValidationLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}}

	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/intents/1/message", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		req.SetPathValue("iid", "1")
		rec := httptest.NewRecorder()
		s.sendWorkerMessage(rec, req)
		return rec
	}
	errBody := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var resp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf(" JSON  : %v ( %q)", err, rec.Body.String())
		}
		return resp.Error
	}

	cases := []struct {
		name string
		body string
		code int
		want string
	}{
		// 64KB    JSON . MaxBytesReader      .
		{"request_too_large", `{"message":"` + strings.Repeat("a", maxWorkerMessageBytes+1024) + `"}`, http.StatusRequestEntityTooLarge, errIntentRequestTooLarge},
		{"message_empty", `{"message":"  "}`, http.StatusBadRequest, errIntentMessageEmpty},
		{"message_too_long", `{"message":"` + strings.Repeat("", 4001) + `"}`, http.StatusBadRequest, errIntentMessageTooLong},
		{"bad_request_id", `{"message":"","request_id":" "}`, http.StatusBadRequest, errIntentBadRequestID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := call(c.body)
			if rec.Code != c.code {
				t.Fatalf("  = %d,  = %d ( %q)", rec.Code, c.code, rec.Body.String())
			}
			if got := errBody(t, rec); got != c.want {
				t.Fatalf("  = %q,  = %q", got, c.want)
			}
			assertKoreanError(t, c.name, errBody(t, rec))
		})
	}
}
