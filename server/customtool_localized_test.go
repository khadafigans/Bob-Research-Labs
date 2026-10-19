package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// customtool.go     CRUD·   writeErr  
// assertKoreanError( ·  0),    task_categories 
// decodeErrorField  ( package server).   actool.Errorf 

// TestCustomToolErrorConstantsLocalized  writeErr    8  
// .   (errCustomToolKeyExists·errCustomToolEditCustomOnly)  
// pg.GetTool(DB)     HTTP   .  
// (goals_api·task_control  DB    ).
func TestCustomToolErrorConstantsLocalized(t *testing.T) {
	for _, c := range []struct{ name, msg string }{
		{"keyFormat", errCustomToolKeyFormat},
		{"kindInvalid", errCustomToolKindInvalid},
		{"httpSchemaRequired", errCustomToolHTTPSchemaRequired},
		{"keyExists", errCustomToolKeyExists},
		{"editCustomOnly", errCustomToolEditCustomOnly},
		{"badBody", errCustomToolBadBody},
		{"shellNoExec", errCustomToolShellNoExec},
		{"unknownKindPrefix", errCustomToolUnknownKindPrefix},
	} {
		assertKoreanError(t, "customtool."+c.name, c.msg)
	}
}

// newCustomToolServer  pg (s.pg)   DB   Server  .
// &db.DB{}   *sql.DB  nil    non-nil  pg()  503  
func newCustomToolServer() *Server {
	return &Server{m: &Manager{pg: &db.DB{}}}
}

// TestCustomToolCreateValidationLocalized  pgCreateCustomTool    3
//  HTTP      .    pg.GetTool(68) 
func TestCustomToolCreateValidationLocalized(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"bad-key", `{"key":"BadKey","kind":"command"}`, errCustomToolKeyFormat},
		{"bad-kind", `{"key":"goodkey","kind":"bogus"}`, errCustomToolKindInvalid},
		{"http-no-schema", `{"key":"goodkey","kind":"http"}`, errCustomToolHTTPSchemaRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCustomToolServer()
			req := httptest.NewRequest(http.MethodPost, "/api/tools/custom", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			s.pgCreateCustomTool(rec, req)
			if rec.Code != 400 && rec.Code != 409 {
				t.Fatalf("     %d ( %q)", rec.Code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("  : got %q want %q", got, c.want)
			}
			assertKoreanError(t, "customtool.create."+c.name, got)
		})
	}
}

// TestCustomToolTestRunValidationLocalized  pgTestCustomTool    3
func TestCustomToolTestRunValidationLocalized(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"bad-body", `not-json`, errCustomToolBadBody},
		{"shell", `{"kind":"shell"}`, errCustomToolShellNoExec},
		{"unknown-kind", `{"kind":"bogus"}`, errCustomToolUnknownKindPrefix + "bogus"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCustomToolServer()
			req := httptest.NewRequest(http.MethodPost, "/api/tools/custom/test", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			s.pgTestCustomTool(rec, req)
			if rec.Code != 400 {
				t.Fatalf("  400   %d ( %q)", rec.Code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("  : got %q want %q", got, c.want)
			}
			// unknown-kind   req.Kind("bogus")     0  .
			assertKoreanError(t, "customtool.test."+c.name, errCustomToolUnknownKindPrefix)
		})
	}
}
