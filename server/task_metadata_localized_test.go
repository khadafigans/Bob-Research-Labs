package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// task_metadata.go     API      
// .   F3a   assertKoreanError( ·  0) .

// TestTaskMetadataErrorConstantsLocalized    4   .
func TestTaskMetadataErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"errTaskMetaRequestTooLarge": errTaskMetaRequestTooLarge,
		"errTaskMetaNoFields":        errTaskMetaNoFields,
		"errTaskMetaNameEmpty":       errTaskMetaNameEmpty,
		"errTaskMetaNameTooLongFmt":  fmt.Sprintf(errTaskMetaNameTooLongFmt, maxTaskNameRunes),
	}
	for label, msg := range cases {
		assertKoreanError(t, label, msg)
	}
}

// TestUpdateTaskMetadataResponsesLocalized     4  HTTP  
// . updateTaskMetadata   4( · · ·  )
// s.m.Task( )    s.m.UpdateTaskMetadata(DB)   ,
func TestUpdateTaskMetadataResponsesLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}}

	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/tasks/t1/metadata", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		rec := httptest.NewRecorder()
		s.updateTaskMetadata(rec, req)
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
		// 16KB    JSON . MaxBytesReader      .
		{"request_too_large", `{"name":"` + strings.Repeat("a", maxTaskMetadataRequestSize+1024) + `"}`, http.StatusRequestEntityTooLarge, errTaskMetaRequestTooLarge},
		{"no_fields", `{}`, http.StatusBadRequest, errTaskMetaNoFields},
		{"name_empty", `{"name":"  "}`, http.StatusBadRequest, errTaskMetaNameEmpty},
		{"name_too_long", `{"name":"` + strings.Repeat("", maxTaskNameRunes+1) + `"}`, http.StatusBadRequest, fmt.Sprintf(errTaskMetaNameTooLongFmt, maxTaskNameRunes)},
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
