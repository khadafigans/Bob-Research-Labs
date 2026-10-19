package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// conversations.go  () API       .
//   F3a   assertKoreanError( ·  0) .

// TestConversationErrorConstantsLocalized    12   .
func TestConversationErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"request_too_large", convErrRequestTooLarge},
		{"agent_key_empty", convErrAgentKeyEmpty},
		{"agent_key_too_long", fmt.Sprintf(convErrAgentKeyTooLong, maxConversationAgentKeyRunes)},
		{"agent_not_found", convErrAgentNotFound},
		{"llm_profile", convErrLLMProfile},
		{"title_too_long", fmt.Sprintf(convErrTitleTooLong, maxConversationTitleRunes)},
		{"title_or_pinned", convErrTitleOrPinned},
		{"title_empty", convErrTitleEmpty},
		{"bad_conv_id", convErrBadConvID},
		{"ids_count", fmt.Sprintf(convErrIDsCount, maxConversationDeleteBatch)},
		{"message_empty", convErrMessageEmpty},
		{"busy", convErrBusy},
	}
	for _, c := range cases {
		assertKoreanError(t, c.name, c.msg)
	}
}

// TestDecodeConversationRequestTooLargeLocalized       HTTP 
//  . decodeConversationRequest  Server(DB)    
func TestDecodeConversationRequestTooLargeLocalized(t *testing.T) {
	// 64KB    JSON . MaxBytesReader      .
	body := `{"title":"` + strings.Repeat("a", maxConversationRequestBytes+1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(body))
	rec := httptest.NewRecorder()
	var dst struct {
		Title string `json:"title"`
	}
	if decodeConversationRequest(rec, req, &dst) {
		t.Fatal("   decodeConversationRequest  true  ")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("  = %d,  = %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf(" JSON  : %v (=%q)", err, rec.Body.String())
	}
	if resp.Error != convErrRequestTooLarge {
		t.Fatalf(" error = %q,  = %q", resp.Error, convErrRequestTooLarge)
	}
	assertKoreanError(t, "decode.too_large.response", resp.Error)
}

// TestConversationDefaultTitlesLocalized      (F8)  
//  . convDefaultTitle       ,
func TestConversationDefaultTitlesLocalized(t *testing.T) {
	assertKoreanError(t, "default_title", convDefaultTitle)
	assertKoreanError(t, "attachment_title", convAttachmentTitle)
	if convDefaultTitle == convAttachmentTitle {
		t.Fatal("       ")
	}
}

// TestIsDefaultConversationTitle       .  
// (convDefaultTitle)     ,    . 
func TestIsDefaultConversationTitle(t *testing.T) {
	if !isDefaultConversationTitle("") {
		t.Fatal("     ")
	}
	if !isDefaultConversationTitle(convDefaultTitle) {
		t.Fatalf("  %q     ", convDefaultTitle)
	}
	if isDefaultConversationTitle("  ") {
		t.Fatal("       ")
	}
}

// TestConversationRetestReasonsLocalized      (F9) 
// .   finding_retests.error     item.error 
func TestConversationRetestReasonsLocalized(t *testing.T) {
	assertKoreanError(t, "retest_failed_to_start", convRetestFailedToStart)
	assertKoreanError(t, "retest_status_read_failed", convRetestStatusReadFailed)
	assertKoreanError(t, "retest_stopped_or_closed", convRetestStoppedOrClosed)
}

// TestTranscriptErrorSummaryLocalized     (F9)  .  
func TestTranscriptErrorSummaryLocalized(t *testing.T) {
	cases := []struct {
		name  string
		label string
		errm  string
		want  string
	}{
		{"chat_turn", "", "connection reset", "(: connection reset)"},
		{"main_agent", " ", "connection reset", "(  : connection reset)"},
	}
	for _, c := range cases {
		got := transcriptErrorSummary(c.label, c.errm)
		if got != c.want {
			t.Fatalf("%s: transcriptErrorSummary = %q,  = %q", c.name, got, c.want)
		}
		if !strings.Contains(got, c.errm) {
			t.Fatalf("%s: err   : %q", c.name, got)
		}
		//  · ASCII   (F1·fluent-korean ).
		if strings.ContainsAny(got, "（）：") {
			t.Fatalf("%s:    : %q", c.name, got)
		}
		wrapper := strings.ReplaceAll(got, c.errm, "")
		assertKoreanError(t, c.name+".wrapper", wrapper)
	}
}
