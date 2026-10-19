package server

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/db"
)

func assertEnglishError(t *testing.T, label, msg string) {
	t.Helper()
	for _, r := range msg {
		if unicode.In(r, unicode.Hangul) {
			t.Errorf("%s: unexpected Hangul in %q", label, msg)
			return
		}
		if r >= 0x4E00 && r <= 0x9FFF {
			t.Errorf("%s: unexpected CJK han in %q", label, msg)
			return
		}
	}
}

func TestNotifyErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"notifyErrBadJSON":         notifyErrBadJSON,
		"notifyErrKindInvalidFmt":  fmt.Sprintf(notifyErrKindInvalidFmt, "email / webhook"),
		"notifyErrNameMissing":     notifyErrNameMissing,
		"notifyErrNameEmpty":       notifyErrNameEmpty,
		"notifyErrModeInvalid":     notifyErrModeInvalid,
		"notifyErrRateNegative":    notifyErrRateNegative,
		"notifyErrChannelID":       notifyErrChannelID,
		"notifyErrKindUnregFmt":    fmt.Sprintf(notifyErrKindUnregFmt, "bogus"),
		"notifyErrDeliveryID":      notifyErrDeliveryID,
		"notifyErrChannelNotFound": notifyErrChannelNotFound,
		"notifyTestName":           notifyTestName,
		"notifyTestClass":          notifyTestClass,
		"notifyTestSummary":        notifyTestSummary,
	}
	for label, msg := range cases {
		assertEnglishError(t, label, msg)
	}
}

func TestNotifyChannelLookupErrLocalized(t *testing.T) {
	rec := httptest.NewRecorder()
	notifyChannelLookupErr(rec, db.ErrNotificationChannelNotFound)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeErrorField(t, rec.Body.Bytes())
	if got != notifyErrChannelNotFound {
		t.Fatalf("error = %q, want %q", got, notifyErrChannelNotFound)
	}
	assertEnglishError(t, "channel_not_found", got)
}

func TestNotifyTestMessageLocalized(t *testing.T) {
	const base = "https://artex.example.test"
	msg := notifyTestMessage(base)
	if len(msg.Items) != 1 {
		t.Fatalf("test message items = %d, want 1", len(msg.Items))
	}
	item := msg.Items[0]
	assertEnglishError(t, "test.name", item.Name)
	assertEnglishError(t, "test.class", item.VulnClass)
	assertEnglishError(t, "test.summary", item.Summary)
	if item.Name != notifyTestName || item.VulnClass != notifyTestClass || item.Summary != notifyTestSummary {
		t.Fatalf("unexpected test payload: %+v", item)
	}
	if msg.HomeURL != base || item.DetailURL != base {
		t.Fatalf("link preserve failed: HomeURL=%q DetailURL=%q (want %q)", msg.HomeURL, item.DetailURL, base)
	}
}
