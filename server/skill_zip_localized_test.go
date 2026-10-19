package server

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// TestSkillZipErrorsLocalized guards the F3b skill_zip.go bundle: every user-facing
// skill-upload error response (fsUploadSkill surfaces these via
// writeErr(400, err.Error()), see server_mgmt.go) must be Korean, and no zip-method
// display name may stay Chinese. Reverting any of these literals to Chinese fails here.
func TestSkillZipErrorsLocalized(t *testing.T) {
	// Message constants rendered with sample arguments.
	assertKoreanError(t, "errSkillZipParse",
		fmt.Errorf(errSkillZipParse, fmt.Errorf("boom")).Error())
	assertKoreanError(t, "errSkillZipEncrypted",
		fmt.Sprintf(errSkillZipEncrypted, "demo/SKILL.md"))
	assertKoreanError(t, "errSkillZipUnsupported",
		fmt.Sprintf(errSkillZipUnsupported, "LZMA", 14, "demo/SKILL.md"))

	// A non-zip upload must reach the user as a Korean hint, not a raw stdlib error.
	if _, err := newSkillZipReader([]byte("this is not a zip archive")); err == nil {
		t.Fatal("-zip    ")
	} else {
		assertKoreanError(t, "newSkillZipReader", err.Error())
		if !strings.Contains(err.Error(), " ") {
			t.Fatalf("parse error = %q, want ' ' ", err.Error())
		}
	}

	// No zip-method display name may carry a Chinese Han ideograph; the two
	// translated ones (AES, unknown fallback) must be Korean.
	for m, name := range zipMethodNames {
		for _, r := range name {
			if unicode.Is(unicode.Han, r) {
				t.Fatalf("zipMethodNames[%d] = %q     ", m, name)
			}
		}
	}
	assertKoreanError(t, "zipMethodName(AES)", zipMethodName(zipMethodAES))
	assertKoreanError(t, "zipMethodName(unknown)", zipMethodName(0xffff))
}
