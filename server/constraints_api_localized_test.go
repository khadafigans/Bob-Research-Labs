package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// constraints_api.go  ()  API      
// .   F3a  assertKoreanError( ·  0),  
//  task_categories  decodeErrorField  ( package server).

// TestConstraintErrorConstantsLocalized    5   .
func TestConstraintErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"task_deleting_add":    errConstraintTaskDeletingAdd,
		"task_deleting_edit":   errConstraintTaskDeletingEdit,
		"task_deleting_delete": errConstraintTaskDeletingDelete,
		"text_empty":           errConstraintTextEmpty,
		"kind_invalid":         errConstraintKindInvalid,
	}
	for label, msg := range cases {
		assertKoreanError(t, label, msg)
	}
}

// TestConstraintHandlersResponsesLocalized  DB      
//  s.m.Task( ) → s.engine.beginTaskOperation(sync.Map   ) → 
// ·  , tasks     Engine   Store·DB  .
func TestConstraintHandlersResponsesLocalized(t *testing.T) {
	// deleting   : beginTaskOperation  false   409  .
	deletingServer := func() *Server {
		s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
		s.engine.deleting.Store("t1", true)
		return s
	}
	//   : beginTaskOperation  true     .
	liveServer := func() *Server {
		return &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
	}

	newReq := func(body, cid string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/constraints", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		if cid != "" {
			req.SetPathValue("cid", cid)
		}
		return req
	}

	cases := []struct {
		name    string
		handler func(*Server) http.HandlerFunc
		server  func() *Server
		body    string
		cid     string
		code    int
		want    string
	}{
		{
			name:    "add/task-deleting",
			handler: func(s *Server) http.HandlerFunc { return s.addConstraint },
			server:  deletingServer,
			body:    `{"text":"  ","kind":"allow"}`,
			code:    http.StatusConflict,
			want:    errConstraintTaskDeletingAdd,
		},
		{
			name:    "add/text-empty",
			handler: func(s *Server) http.HandlerFunc { return s.addConstraint },
			server:  liveServer,
			body:    `{"text":"   ","kind":"allow"}`,
			code:    http.StatusBadRequest,
			want:    errConstraintTextEmpty,
		},
		{
			name:    "add/kind-invalid",
			handler: func(s *Server) http.HandlerFunc { return s.addConstraint },
			server:  liveServer,
			body:    `{"text":"   ","kind":"maybe"}`,
			code:    http.StatusBadRequest,
			want:    errConstraintKindInvalid,
		},
		{
			name:    "edit/task-deleting",
			handler: func(s *Server) http.HandlerFunc { return s.editConstraint },
			server:  deletingServer,
			body:    `{"text":" ","kind":"deny"}`,
			cid:     "5",
			code:    http.StatusConflict,
			want:    errConstraintTaskDeletingEdit,
		},
		{
			name:    "edit/text-empty",
			handler: func(s *Server) http.HandlerFunc { return s.editConstraint },
			server:  liveServer,
			body:    `{"text":"  ","kind":"deny"}`,
			cid:     "5",
			code:    http.StatusBadRequest,
			want:    errConstraintTextEmpty,
		},
		{
			name:    "edit/kind-invalid",
			handler: func(s *Server) http.HandlerFunc { return s.editConstraint },
			server:  liveServer,
			body:    `{"text":" ","kind":"nope"}`,
			cid:     "5",
			code:    http.StatusBadRequest,
			want:    errConstraintKindInvalid,
		},
		{
			name:    "delete/task-deleting",
			handler: func(s *Server) http.HandlerFunc { return s.deleteConstraint },
			server:  deletingServer,
			body:    ``,
			cid:     "5",
			code:    http.StatusConflict,
			want:    errConstraintTaskDeletingDelete,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := c.server()
			rec := httptest.NewRecorder()
			c.handler(s)(rec, newReq(c.body, c.cid))
			if rec.Code != c.code {
				t.Fatalf("  = %d,  = %d ( %q)", rec.Code, c.code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("  = %q,  = %q", got, c.want)
			}
			assertKoreanError(t, c.name, got)
		})
	}
}
