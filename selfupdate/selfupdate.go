// Package selfupdate implements ARTEX ： GitHub Release 
//
// （ start.sh / start.bat）：
//
//
// （sha256sum / shasum / certutil），——
//
//
//	①  server  /api/update/apply →  →  artex.new → exit 75
//	②  → Bootstrap  artex.new → + →  → exit 75
//
package selfupdate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ExitRestart ""（EX_TEMPFAIL）。
const ExitRestart = 75

// settleDelay ； maxAttempts ，。
const maxAttempts = 3

type Paths struct {
	Dir     string // 
	Current string //         artex      / artex.exe
	New     string //             artex.new  / artex.new.exe
	Sum     string //  sha256（hex）  artex.new.sha256 / artex.new.exe.sha256
	Old     string //       artex.old  / artex.old.exe
	Marker  string //             artex.upgrade.json
}

// ResolvePaths 。
//
// Windows  .new/.old  .exe ，，
func ResolvePaths() (Paths, error) {
	exe, err := os.Executable()
	if err != nil {
		return Paths{}, fmt.Errorf("    : %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	name := filepath.Base(exe)
	ext := filepath.Ext(name) // Windows  ".exe"，Unix 
	stem := strings.TrimSuffix(name, ext)

	join := func(suffix string) string { return filepath.Join(dir, stem+suffix+ext) }
	return Paths{
		Dir:     dir,
		Current: exe,
		New:     join(".new"),
		Sum:     join(".new") + ".sha256",
		Old:     join(".old"),
		Marker:  filepath.Join(dir, stem+".upgrade.json"),
	}, nil
}

type marker struct {
	From     string `json:"from"`     // 
	To       string `json:"to"`       // 
	Attempts int    `json:"attempts"` // 
	StagedAt int64  `json:"staged_at"`
}

func readMarker(path string) (marker, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return marker{}, false
	}
	var m marker
	if json.Unmarshal(b, &m) != nil {
		return marker{}, false
	}
	return m, true
}

func writeMarker(path string, m marker) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func cleanStaged(p Paths) {
	_ = os.Remove(p.New)
	_ = os.Remove(p.Sum)
}

// CompareVersions ， -1/0/1（a<b / a==b / a>b）。
// git describe  "0.3.7-2-gabc1234-dirty"），，
func CompareVersions(a, b string) (int, bool) {
	av, aok := parseVersion(a)
	bv, bok := parseVersion(b)
	if !aok || !bok {
		return 0, false
	}
	for i := range 3 {
		if av[i] != bv[i] {
			if av[i] < bv[i] {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

// parseVersion  "v0.3.7" / "0.3.7"  [3]int。
//
// ：build.sh  tag  git describe 
func parseVersion(s string) ([3]int, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return [3]int{}, false
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// InDocker 。Docker ，
// `docker compose up -d` ——
func InDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	b, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	s := string(b)
	return strings.Contains(s, "docker") || strings.Contains(s, "containerd")
}
