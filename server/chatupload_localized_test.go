package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// chatupload.go     API       .
//   F3a  assertKoreanError( ·  0),   
// task_categories  decodeErrorField  ( package server).

// TestChatUploadErrorConstantsLocalized    7   .
func TestChatUploadErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"scope_invalid": errChatUploadScopeInvalid,
		"bad_id":        errChatUploadBadID,
		"task_deleting": errChatUploadTaskDeleting,
		"mkdir":         errChatUploadMkdir,
		"parse":         errChatUploadParse,
		"no_file":       errChatUploadNoFile,
		"save_failed":   errChatUploadSaveFailed,
	}
	for label, msg := range cases {
		assertKoreanError(t, label, msg)
	}
}

// TestChatUploadHandlerResponsesLocalized  DB     chatUpload 
//  HTTP   . scope/id  Manager    ,
// no-file  scope=session( ) ·DB    .
// task-deleting  goals    (engine.deleting)  409  .
func TestChatUploadHandlerResponsesLocalized(t *testing.T) {
	//   "file"   multipart  — ParseMultipartForm    0.
	noFileBody := func() (*bytes.Buffer, string) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("other", "x")
		_ = mw.Close()
		return &buf, mw.FormDataContentType()
	}

	t.Run("scope-invalid", func(t *testing.T) {
		s := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=bogus&id=x", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("  = %d,  = 400 ( %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadScopeInvalid {
			t.Fatalf("  = %q,  = %q", got, errChatUploadScopeInvalid)
		}
		assertKoreanError(t, "scope-invalid", errChatUploadScopeInvalid)
	})

	t.Run("bad-id", func(t *testing.T) {
		s := &Server{}
		// scope=session( ) Manager    id  .
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=session&id=../evil", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("  = %d,  = 400 ( %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadBadID {
			t.Fatalf("  = %q,  = %q", got, errChatUploadBadID)
		}
		assertKoreanError(t, "bad-id", errChatUploadBadID)
	})

	t.Run("no-file", func(t *testing.T) {
		s := &Server{m: &Manager{dir: t.TempDir()}}
		body, ctype := noFileBody()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=session&id=sess1", body)
		req.Header.Set("Content-Type", ctype)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("  = %d,  = 400 ( %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadNoFile {
			t.Fatalf("  = %q,  = %q", got, errChatUploadNoFile)
		}
		assertKoreanError(t, "no-file", errChatUploadNoFile)
	})

	t.Run("task-deleting", func(t *testing.T) {
		s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
		s.engine.deleting.Store("t1", true)
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=task&id=t1", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("  = %d,  = 409 ( %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadTaskDeleting {
			t.Fatalf("  = %q,  = %q", got, errChatUploadTaskDeleting)
		}
		assertKoreanError(t, "task-deleting", errChatUploadTaskDeleting)
	})
}
