package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestSyncScopeSentryConstantsLocalized guards F3b (sync_scopesentry.go): every
// user-facing message constant — surfaced either as a writeErr HTTP body or inside
// the sync result's warnings/errors JSON arrays — must be Korean (Hangul present,
// no Chinese Han). Reverting any literal to Chinese fails this test.
func TestSyncScopeSentryConstantsLocalized(t *testing.T) {
	cases := []struct {
		label string
		msg   string
	}{
		{"errSSDataSourceMissingFmt", errSSDataSourceMissingFmt},
		{"errSSDataSourceNoURLFmt", errSSDataSourceNoURLFmt},
		{"errSSListProjectsPrefix", errSSListProjectsPrefix},
		{"errSSParseProjectsPrefix", errSSParseProjectsPrefix},
		{"errSSListTasksPrefix", errSSListTasksPrefix},
		{"errSSParseTasksPrefix", errSSParseTasksPrefix},
		{"errSSDimension", errSSDimension},
		{"errSSTargetsEmpty", errSSTargetsEmpty},
		{"warnSSProjectMetaFmt", warnSSProjectMetaFmt},
		{"warnSSCompanyCreateFmt", warnSSCompanyCreateFmt},
		{"warnSSUnknownAssetPrefix", warnSSUnknownAssetPrefix},
		{"errSSFetchFmt", errSSFetchFmt},
		{"warnSSTruncatedFmt", warnSSTruncatedFmt},
		{"errSSParseSubdomainPrefix", errSSParseSubdomainPrefix},
		{"errSSParseAppPrefix", errSSParseAppPrefix},
		{"errSSParseServicePrefix", errSSParseServicePrefix},
	}
	for _, c := range cases {
		assertKoreanError(t, c.label, c.msg)
	}

	// dimension/targets validation keep their JSON field names and enum values.
	if !strings.Contains(errSSDimension, "project") || !strings.Contains(errSSDimension, "task") {
		t.Fatalf("errSSDimension  enum  : %q", errSSDimension)
	}
}

// TestSyncScopeSentryFormattedLocalized exercises the exact Sprintf/Errorf format
// strings used by the sync handler so a verb mismatch or a reverted literal is
// caught by running the real format operation (not just inspecting the constant).
func TestSyncScopeSentryFormattedLocalized(t *testing.T) {
	ds := fmt.Errorf(errSSDataSourceMissingFmt, scopeSentryMCPName).Error()
	assertKoreanError(t, "scopeSentryClient/missing", ds)
	if !strings.Contains(ds, "ScopeSentry") {
		t.Fatalf("   : %q", ds)
	}

	noURL := fmt.Errorf(errSSDataSourceNoURLFmt, scopeSentryMCPName).Error()
	assertKoreanError(t, "scopeSentryClient/noURL", noURL)
	if !strings.Contains(noURL, "URL") {
		t.Fatalf("URL  : %q", noURL)
	}

	meta := fmt.Sprintf(warnSSProjectMetaFmt, "proj-1", errors.New("boom"))
	assertKoreanError(t, "warnSSProjectMetaFmt", meta)
	if !strings.Contains(meta, "proj-1") || !strings.Contains(meta, "boom") {
		t.Fatalf("/   : %q", meta)
	}

	company := fmt.Sprintf(warnSSCompanyCreateFmt, "ACME", errors.New("boom"))
	assertKoreanError(t, "warnSSCompanyCreateFmt", company)
	if !strings.Contains(company, "ACME") {
		t.Fatalf("   : %q", company)
	}

	fetch := fmt.Sprintf(errSSFetchFmt, "subdomain", "t1", errors.New("boom"))
	assertKoreanError(t, "errSSFetchFmt", fetch)
	if !strings.Contains(fetch, "subdomain") || !strings.Contains(fetch, "t1") {
		t.Fatalf(" /  : %q", fetch)
	}

	truncated := fmt.Sprintf(warnSSTruncatedFmt, "app", "t1", syncMaxPerType)
	assertKoreanError(t, "warnSSTruncatedFmt", truncated)
	if !strings.Contains(truncated, "5000") {
		t.Fatalf("   : %q", truncated)
	}
}

// TestSyncScopeSentryIngestParseErrorsLocalized runs ssIngest on malformed JSON for
// each asset kind. The unmarshal failure returns before any store access, so this is
// a real code path (nil AssetStore never dereferenced) that pins the parse-error
// strings to Korean.
func TestSyncScopeSentryIngestParseErrorsLocalized(t *testing.T) {
	s := &Server{}
	bad := json.RawMessage("not json")
	for _, kind := range []string{"subdomain", "app", "service"} {
		got := s.ssIngest(nil, kind, bad, map[string]int{})
		if got == "" {
			t.Fatalf("%s:  JSON    ", kind)
		}
		assertKoreanError(t, "ssIngest/"+kind, got)
		if !strings.Contains(got, kind) {
			t.Fatalf("%s:    : %q", kind, got)
		}
	}
}
