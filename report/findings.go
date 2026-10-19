package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Autumn-27/artex/db"
)

//   「」 :   findings    Markdown,
//  Markdown,  CSV  . JSON  server  DTO   

// sortFindingsForExport   ,    .    .
func sortFindingsForExport(fs []*db.DBFinding) {
	sort.SliceStable(fs, func(i, j int) bool {
		ri, rj := sevRank[fs[i].Severity], sevRank[fs[j].Severity]
		if ri != rj {
			return ri < rj // sevRank    
		}
		return fs[i].CreatedAt.After(fs[j].CreatedAt)
	})
}

// findingTitle        :  →  → 「」.
func findingTitle(f *db.DBFinding) string {
	return nz(f.Name, nz(f.VulnClass, ""))
}

// FindingsMarkdown    findings     ( + 
func FindingsMarkdown(fs []*db.DBFinding, generatedAt time.Time) string {
	items := append([]*db.DBFinding(nil), fs...)
	sortFindingsForExport(items)

	var b strings.Builder
	b.WriteString("#   \n\n")
	fmt.Fprintf(&b, "- ** **: %s\n", generatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- ** **: %d\n\n", len(items))

	counts := map[string]int{}
	for _, f := range items {
		counts[f.Severity]++
	}
	b.WriteString("## \n\n")
	b.WriteString("|  |  |\n| --- | --- |\n")
	for _, s := range []struct{ key, label string }{
		{"critical", ""}, {"high", ""}, {"medium", ""}, {"low", ""},
	} {
		fmt.Fprintf(&b, "| %s | %d |\n", s.label, counts[s.key])
	}
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString("_  ._\n")
		return b.String()
	}

	b.WriteString("##  \n\n")
	for i, f := range items {
		fmt.Fprintf(&b, "### %d. [%s] %s\n\n", i+1, strings.ToUpper(nz(f.Severity, "info")), findingTitle(f))
		if f.VulnClass != "" {
			fmt.Fprintf(&b, "- ****: %s\n", f.VulnClass)
		}
		fmt.Fprintf(&b, "- ****: %s\n", nz(f.Status, "pending"))
		if desc := strings.TrimSpace(f.TaskDescription); desc != "" {
			fmt.Fprintf(&b, "- ** **: %s\n", desc)
		}
		fmt.Fprintf(&b, "- ** **: %s\n\n", f.CreatedAt.Format("2006-01-02 15:04:05"))
		if s := strings.TrimSpace(f.Summary); s != "" {
			fmt.Fprintf(&b, "%s\n\n", s)
		}
		if e := strings.TrimSpace(f.Evidence); e != "" {
			fmt.Fprintf(&b, "**:**\n\n```\n%s\n```\n\n", e)
		}
		if rep := strings.TrimSpace(f.Report); rep != "" {
			b.WriteString("** :**\n\n")
			b.WriteString(rep)
			b.WriteString("\n\n")
		}
		b.WriteString(findingTrafficMarkdown(f, false))
		b.WriteString("---\n\n")
	}
	return b.String()
}

// SingleFindingMarkdown     Markdown   (「   」  ).
func SingleFindingMarkdown(f *db.DBFinding, generatedAt time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# [%s] %s\n\n", strings.ToUpper(nz(f.Severity, "info")), findingTitle(f))
	if f.VulnClass != "" {
		fmt.Fprintf(&b, "- ****: %s\n", f.VulnClass)
	}
	fmt.Fprintf(&b, "- ****: %s\n", nz(f.Severity, "info"))
	fmt.Fprintf(&b, "- ****: %s\n", nz(f.Status, "pending"))
	if desc := strings.TrimSpace(f.TaskDescription); desc != "" {
		fmt.Fprintf(&b, "- ** **: %s\n", desc)
	}
	fmt.Fprintf(&b, "- ** **: %s\n", f.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- ** **: %s\n\n", generatedAt.Format("2006-01-02 15:04:05"))
	if s := strings.TrimSpace(f.Summary); s != "" {
		fmt.Fprintf(&b, "## \n\n%s\n\n", s)
	}
	if e := strings.TrimSpace(f.Evidence); e != "" {
		fmt.Fprintf(&b, "## \n\n```\n%s\n```\n\n", e)
	}
	if rep := strings.TrimSpace(f.Report); rep != "" {
		b.WriteString("##  \n\n")
		b.WriteString(rep)
		b.WriteString("\n")
	}
	b.WriteString(findingTrafficMarkdown(f, true))
	return b.String()
}

var unsafeFilenameChars = regexp.MustCompile(`[^\p{Han}\p{L}\p{N}._-]+`)

// FindingFilename  「   」   .md   .
// : `critical_SQL_#123.md`.      zip    .
func FindingFilename(f *db.DBFinding) string {
	sev := nz(f.Severity, "info")
	title := findingTitle(f)
	name := fmt.Sprintf("%s_%s_#%d", sev, title, f.ID)
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if name == "" {
		name = fmt.Sprintf("finding_%d", f.ID)
	}
	name = path.Base(name)
	if len(name) > 120 {
		name = name[:120]
	}
	return name + ".md"
}

// FindingsCSV    findings  CSV  (UTF-8 BOM , Excel  
//  ).  report/evidence       .
//   Markdown/JSON  .
func FindingsCSV(fs []*db.DBFinding) []byte {
	items := append([]*db.DBFinding(nil), fs...)
	sortFindingsForExport(items)

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ID", "", "", "", "", " ", " ", "", "  ", "  ID"})
	for _, f := range items {
		_ = w.Write([]string{
			fmt.Sprintf("%d", f.ID),
			findingTitle(f),
			f.VulnClass,
			nz(f.Severity, "info"),
			nz(f.Status, "pending"),
			f.TaskDescription,
			f.CreatedAt.Format("2006-01-02 15:04:05"),
			strings.TrimSpace(f.Summary),
			fmt.Sprint(len(f.TrafficBindings)), findingTrafficIDs(f),
		})
	}
	w.Flush()
	return buf.Bytes()
}

func findingTrafficIDs(f *db.DBFinding) string {
	ids := make([]string, 0, len(f.TrafficBindings))
	for _, b := range f.TrafficBindings {
		ids = append(ids, fmt.Sprint(b.ID))
	}
	return strings.Join(ids, ",")
}

func findingTrafficMarkdown(f *db.DBFinding, attachments bool) string {
	stale := f.Report != "" && f.EvidenceVersion != f.ReportEvidenceVersion
	if len(f.TrafficBindings) == 0 && !stale {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n##   \n\n")
	fmt.Fprintf(&out, " : %d,  : %d.\n\n", f.EvidenceVersion, len(f.TrafficBindings))
	if stale {
		out.WriteString("     .\n\n")
	}
	for i, b := range f.TrafficBindings {
		fmt.Fprintf(&out, "%d. ** #%d · %s**: `%s %s`,   %d\n", i+1, b.ID, b.Role, b.Snapshot.Method, strings.ReplaceAll(b.Snapshot.URL, "`", "%60"), b.Snapshot.Status)
		if b.Note != "" {
			fmt.Fprintf(&out, "   %s\n", strings.ReplaceAll(b.Note, "\n", "\n   "))
		}
		if attachments {
			fmt.Fprintf(&out, "   [ ](evidence/%d/%d/request.http) · [ ](evidence/%d/%d/response.http)\n", f.ID, b.ID, f.ID, b.ID)
		}
	}
	out.WriteString("\n")
	return out.String()
}
