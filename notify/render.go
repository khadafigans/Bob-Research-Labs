package notify

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const ellipsis = "…"

// TruncateBytes  s  max ， UTF-8 。
//
//  rune （utf8.RuneStart  0b10xxxxxx）。
//
func TruncateBytes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	budget := max - len(ellipsis)
	suffix := ellipsis
	if budget < 0 {
		budget = max
		suffix = ""
	}
	cut := budget
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + suffix
}

func OneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	return TruncateRunes(s, max)
}

// TruncateRunes  s  max （），。
//
//  TruncateBytes ：，Telegram 。
func TruncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + ellipsis
}

// TruncateHTML  HTML ，。
//
//  HTML  `<a href="htt` ，
//
// （ </b> ）：Telegram  HTML ，
func TruncateHTML(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	cut := TruncateRunes(s, max)
	if lt := strings.LastIndex(cut, "<"); lt >= 0 && !strings.Contains(cut[lt:], ">") {
		cut = cut[:lt]
	}
	if amp := strings.LastIndex(cut, "&"); amp >= 0 && !strings.Contains(cut[amp:], ";") {
		cut = cut[:amp]
	}
	return cut
}

// packItemCount ****，。
//
//
// ：maxSize<=0 ；reserve /；
// size （：/，Telegram ——
//
func packItemCount(items []Item, maxSize, reserve int, footer string, size func(string) int, render func(Item, int) string) int {
	if maxSize <= 0 {
		return len(items)
	}
	budget := maxSize - reserve - size(footer)
	if budget < 0 {
		budget = 0
	}
	used := 0
	for i, it := range items {
		used += size(render(it, i))
		if used > budget && i > 0 {
			return i
		}
	}
	return len(items)
}

// byteSize / runeSize  packItemCount ，
//  func(s string) int ，。
func byteSize(s string) int { return len(s) }
func runeSize(s string) int { return utf8.RuneCountInString(s) }

// assetLine ， limit 。
func assetLine(assets []string, limit int) string {
	if len(assets) == 0 {
		return ""
	}
	if limit <= 0 || len(assets) <= limit {
		return strings.Join(assets, ", ")
	}
	return strings.Join(assets[:limit], ", ") + fmt.Sprintf(" … (%d total)", len(assets))
}

// itoa  strconv.Itoa ，， import strconv。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
