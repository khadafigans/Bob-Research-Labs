package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// findings_groups.go         .
//   F3a   assertKoreanError( ·  0) .

// TestDeepenFindingBodyTooLargeLocalized   (413)    HTTP 
// .   MaxBytesReader    s.m.pg(DB)    
func TestDeepenFindingBodyTooLargeLocalized(t *testing.T) {
	s := &Server{}
	body := `{"description":"` + strings.Repeat("a", 33<<10) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/exploration/findings/1/deepen", strings.NewReader(body))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	s.deepenFinding(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("  = %d,  = %d ( %q)", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf(" JSON  : %v ( %q)", err, rec.Body.String())
	}
	const want = "   "
	if resp.Error != want {
		t.Fatalf("  = %q,  = %q", resp.Error, want)
	}
	assertKoreanError(t, "body_too_large", resp.Error)
}

// TestFindingFollowUpAuditSummaryLocalized        .
//    AddFindingFollowUpIntent  DB   DB   
func TestFindingFollowUpAuditSummaryLocalized(t *testing.T) {
	assertKoreanError(t, "follow_up_summary", auditFindingFollowUpSummary)
}
