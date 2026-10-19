package notify

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateBytesKeepsValidUTF8(t *testing.T) {
	inputs := []string{
		"",
		" mixed  content",
		"abcde",
		"🔴🟠🟡🔵", // 4  emoji，
		strings.Repeat("", 100),
	}
	for _, in := range inputs {
		for max := 1; max <= len(in)+2; max++ {
			got := TruncateBytes(in, max)
			if !utf8.ValidString(got) {
				t.Fatalf(" %q max=%d:  UTF-8 %q", in, max, got)
			}
			if len(got) > max {
				t.Fatalf(" %q max=%d:  %d ", in, max, len(got))
			}
			if len(in) <= max && got != in {
				t.Fatalf(" %q max=%d:  -> %q", in, max, got)
			}
		}
	}
}

func TestTruncateBytesZeroMeansUnlimited(t *testing.T) {
	long := strings.Repeat("x", 10000)
	if got := TruncateBytes(long, 0); got != long {
		t.Fatal("max=0 ")
	}
	if got := TruncateBytes(long, -5); got != long {
		t.Fatal("max<0 ")
	}
}

func TestTruncateBytesEllipsisBudget(t *testing.T) {
	got := TruncateBytes("abcdefgh", 1)
	if len(got) > 1 {
		t.Fatalf("max=1  %q  %d ", got, len(got))
	}
	if got := TruncateBytes("abcdefgh", 5); !strings.HasSuffix(got, ellipsis) {
		t.Fatalf("， %q", got)
	}
}

func TestTruncateRunesCountsCharactersNotBytes(t *testing.T) {
	//  TruncateBytes ：Telegram ，
	s := ""
	got := TruncateRunes(s, 5)
	if n := utf8.RuneCountInString(got); n != 5 {
		t.Fatalf(" 5 ， %d  (%q)", n, got)
	}
	if utf8.RuneCountInString(TruncateBytes(s, 5)) >= 5 {
		t.Fatal("")
	}
}

func TestOneLineCollapsesWhitespace(t *testing.T) {
	got := OneLine("\n\n\t   ", 0)
	if strings.ContainsAny(got, "\n\t") {
		t.Fatalf("， %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Fatalf("， %q", got)
	}
	got = OneLine("", 4)
	if n := utf8.RuneCountInString(got); n != 4 {
		t.Fatalf(" 4 ， %d (%q)", n, got)
	}
}

func TestTruncateHTMLNeverCutsTagInHalf(t *testing.T) {
	//  HTML  `<a href="htt` ，。
	s := `<b></b><a href="https://example.com/very/long/path"></a>`
	for max := 1; max <= utf8.RuneCountInString(s)+2; max++ {
		got := TruncateHTML(s, max)
		if n := utf8.RuneCountInString(got); max > 0 && n > max {
			t.Fatalf("max=%d:  %d ", max, n)
		}
		if lt := strings.LastIndex(got, "<"); lt >= 0 && !strings.Contains(got[lt:], ">") {
			t.Fatalf("max=%d:  -> %q", max, got)
		}
	}
}

func TestAssetLineOmitsExcess(t *testing.T) {
	if got := assetLine(nil, 3); got != "" {
		t.Fatalf("， %q", got)
	}
	if got := assetLine([]string{"a", "b"}, 3); got != "a, b" {
		t.Fatalf("， %q", got)
	}
	got := assetLine([]string{"a", "b", "c", "d", "e"}, 2)
	if !strings.Contains(got, "(5 total)") {
		t.Fatalf(" 5， %q", got)
	}
}

func TestSeverityAndStatusLabels(t *testing.T) {
	if AtLeast("", "low") {
		t.Fatal(" 0，")
	}
	if !AtLeast("critical", "") {
		t.Fatal("")
	}
	if got := StatusLabel("fixed"); got != "Fixed" {
		t.Fatalf("， %q", got)
	}
	if got := StatusLabel("weird_status"); got != "weird_status" {
		t.Fatalf("， %q", got)
	}
}

// TestTruncateHTMLNeverCutsEntity ：
//
func TestTruncateHTMLNeverCutsEntity(t *testing.T) {
	s := "aaaa&amp;bbbb&lt;cccc&quot;dddd"
	for max := 1; max <= utf8.RuneCountInString(s)+2; max++ {
		got := TruncateHTML(s, max)
		if amp := strings.LastIndex(got, "&"); amp >= 0 && !strings.Contains(got[amp:], ";") {
			t.Fatalf("max=%d:  %q", max, got[amp:])
		}
		if strings.Contains(got, "&amp\x00") {
			t.Fatalf("max=%d: ", max)
		}
	}
}
