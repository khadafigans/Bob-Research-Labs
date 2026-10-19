package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//    : ① requireAuth  pg     HTTP
//   F3a   assertKoreanError ( ·  0) .

// TestRequireAuthMessagesLocalized      requireAuth  
func TestRequireAuthMessagesLocalized(t *testing.T) {
	s := &Server{jwtKey: []byte(strings.Repeat("k", 32))}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := s.requireAuth(next)

	cases := []struct {
		name   string
		bearer string
		want   string
	}{
		{" ", "", authErrUnauthorized},
		{" ", "Bearer not-a-valid-token", authErrTokenInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
			if c.bearer != "" {
				r.Header.Set("Authorization", c.bearer)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("  = %d,  401", rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, c.want) {
				t.Errorf("  %q  : %s", c.want, body)
			}
			assertKoreanError(t, c.name, body)
		})
	}
}

// TestAuthErrorConstantsLocalized         
func TestAuthErrorConstantsLocalized(t *testing.T) {
	consts := map[string]string{
		"authErrUnauthorized":         authErrUnauthorized,
		"authErrTokenInvalid":         authErrTokenInvalid,
		"authErrPasswordAlreadySet":   authErrPasswordAlreadySet,
		"authErrPasswordEmpty":        authErrPasswordEmpty,
		"authErrNewPasswordEmpty":     authErrNewPasswordEmpty,
		"authErrPasswordHash":         authErrPasswordHash,
		"authErrSaveFailedPrefix":     authErrSaveFailedPrefix,
		"authErrTokenGen":             authErrTokenGen,
		"authErrBadRequest":           authErrBadRequest,
		"authErrPasswordNotInit":      authErrPasswordNotInit,
		"authErrCurrentPasswordWrong": authErrCurrentPasswordWrong,
		"authErrBadCredential":        authErrBadCredential,
	}
	for name, msg := range consts {
		assertKoreanError(t, name, msg)
	}
}
