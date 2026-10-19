package selfupdate

//  (F20). update.go  updateHub  Stage     SSE
// progress.message , Stage/FetchLatest/Rollback    progress.error·
// writeErr ·boot_notice   (update-card) ,  

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

func hasHanzi(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// assertKoreanError   ,  ,    .
func assertKoreanError(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s:   nil ", label)
	}
	msg := err.Error()
	if !hasHangul(msg) {
		t.Errorf("%s:  : %q", label, msg)
	}
	if hasHanzi(msg) {
		t.Errorf("%s:    : %q", label, msg)
	}
}

// rtFunc  http.Client    . GitHub    .
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubClient(status int, body string) *http.Client {
	return &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
}

// TestCheckURLErrorsLocalized:  TestCheckURLRejectsNonGitHub    
func TestCheckURLErrorsLocalized(t *testing.T) {
	nonHTTPS := checkURL(mustParse(t, "http://api.github.com/x"))
	assertKoreanError(t, "non-https", nonHTTPS)
	if !strings.Contains(nonHTTPS.Error(), "HTTPS") {
		t.Errorf("non-https  HTTPS  : %q", nonHTTPS.Error())
	}

	nonGitHub := checkURL(mustParse(t, "https://evil.example.com/x"))
	assertKoreanError(t, "non-github-host", nonGitHub)
	if !strings.Contains(nonGitHub.Error(), "GitHub") {
		t.Errorf("non-github  GitHub  : %q", nonGitHub.Error())
	}
}

// TestFetchLatestErrorsLocalized:      ( ··
// · ·tag )  .  transport    .
func TestFetchLatestErrorsLocalized(t *testing.T) {
	ctx := context.Background()

	_, errRate := FetchLatest(ctx, stubClient(http.StatusForbidden, ""))
	assertKoreanError(t, "rate-limited", errRate)
	if !strings.Contains(errRate.Error(), "60") {
		t.Errorf("   60  : %q", errRate.Error())
	}

	_, errNF := FetchLatest(ctx, stubClient(http.StatusNotFound, ""))
	assertKoreanError(t, "not-found", errNF)

	_, errStatus := FetchLatest(ctx, stubClient(http.StatusInternalServerError, ""))
	assertKoreanError(t, "bad-status", errStatus)
	if !strings.Contains(errStatus.Error(), "GitHub") {
		t.Errorf("  GitHub  : %q", errStatus.Error())
	}

	_, errJSON := FetchLatest(ctx, stubClient(http.StatusOK, "{not json"))
	assertKoreanError(t, "bad-json", errJSON)

	_, errTag := FetchLatest(ctx, stubClient(http.StatusOK, "{}"))
	assertKoreanError(t, "missing-tag", errTag)
	if !strings.Contains(errTag.Error(), "tag") {
		t.Errorf("tag   tag  : %q", errTag.Error())
	}
}

// TestStageMissingAssetLocalized:      Stage 
//  checkWritable          .
func TestStageMissingAssetLocalized(t *testing.T) {
	rel := &Release{TagName: "v9.9.9"} // Assets  
	err := Stage(context.Background(), &http.Client{}, rel, "0.0.1", nil)
	assertKoreanError(t, "stage-missing-asset", err)
}
