package selfupdate

import (
	"archive/zip"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testPaths 。 ResolvePaths()——
func testPaths(t *testing.T) Paths {
	t.Helper()
	dir := t.TempDir()
	return Paths{
		Dir:     dir,
		Current: filepath.Join(dir, "artex"),
		New:     filepath.Join(dir, "artex.new"),
		Sum:     filepath.Join(dir, "artex.new.sha256"),
		Old:     filepath.Join(dir, "artex.old"),
		Marker:  filepath.Join(dir, "artex.upgrade.json"),
	}
}

// fakeBin  artex。smokeTest  -h ，
func fakeBin(t *testing.T, path, marker string, exitCode int) {
	t.Helper()
	script := "#!/bin/sh\necho " + marker + "\nexit " + itoa(exitCode) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf(" %s: %v", path, err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return string(rune('0' + n))
}

// stage  bin ""： artex.new 。
func stage(t *testing.T, p Paths, marker string, exitCode int) {
	t.Helper()
	fakeBin(t, p.New, marker, exitCode)
	sum, err := fileSHA256(p.New)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if err := os.WriteFile(p.Sum, []byte(sum), 0o644); err != nil {
		t.Fatalf(": %v", err)
	}
}

func readAll(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(" %s: %v", path, err)
	}
	return string(b)
}

func requireUnix(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip(" sh ，Windows ")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b       string
		want       int
		comparable bool
	}{
		{"0.3.7", "0.3.8", -1, true},
		{"0.3.8", "0.3.7", 1, true},
		{"0.3.7", "0.3.7", 0, true},
		{"v0.3.7", "0.3.8", -1, true}, // build.sh  v，tag  v，
		{"0.3.7", "v0.3.7", 0, true},
		{"0.9.0", "0.10.0", -1, true}, // 
		{"1.0.0", "0.99.99", 1, true},
		{"dev", "0.3.8", 0, false},
		{"0.3.7-2-gabc1234", "0.3.8", 0, false},
		{"0.3.7-dirty", "0.3.8", 0, false},
		{"0.3", "0.3.8", 0, false},
		{"", "0.3.8", 0, false},
	}
	for _, c := range cases {
		got, ok := CompareVersions(c.a, c.b)
		if ok != c.comparable {
			t.Errorf("CompareVersions(%q,%q) comparable=%v,  %v", c.a, c.b, ok, c.comparable)
			continue
		}
		if ok && got != c.want {
			t.Errorf("CompareVersions(%q,%q)=%d,  %d", c.a, c.b, got, c.want)
		}
	}
}

func TestResolvePathsNaming(t *testing.T) {
	p, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	for name, path := range map[string]string{"New": p.New, "Sum": p.Sum, "Old": p.Old, "Marker": p.Marker} {
		if filepath.Dir(path) != p.Dir {
			t.Errorf("%s : %s ( %s)", name, path, p.Dir)
		}
	}
	// Windows  .new/.old  .exe，。
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(p.New, ".exe") || !strings.HasSuffix(p.Old, ".exe") {
			t.Errorf("Windows  .new/.old  .exe : new=%s old=%s", p.New, p.Old)
		}
	}
}

func TestVerifyStagedRejectsTamperedBinary(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	stage(t, p, "new", 0)

	fakeBin(t, p.New, "tampered", 0)
	if err := verifyStaged(p); err == nil {
		t.Fatal(" SHA256 ，")
	}
}

func TestVerifyStagedRejectsUnrunnableBinary(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	stage(t, p, "broken", 1) //  0

	if err := verifyStaged(p); err == nil {
		t.Fatal("，")
	}
}

func TestApplyStagedHappyPath(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "old", 0)
	stage(t, p, "new", 0)
	if err := writeMarker(p.Marker, marker{From: "0.3.7", To: "0.3.8"}); err != nil {
		t.Fatalf(": %v", err)
	}

	action, st := applyStaged(p)
	if action != Restart {
		t.Fatalf(" Restart， %v", action)
	}
	if !st.Pending {
		t.Error(" Pending")
	}
	if !strings.Contains(readAll(t, p.Current), "new") {
		t.Error("artex ")
	}
	if !strings.Contains(readAll(t, p.Old), "old") {
		t.Error(" artex.old")
	}
	if _, err := os.Stat(p.New); !os.IsNotExist(err) {
		t.Error(" artex.new ")
	}
	if _, err := os.Stat(p.Sum); !os.IsNotExist(err) {
		t.Error("")
	}
	if _, ok := readMarker(p.Marker); !ok {
		t.Error("")
	}
}

func TestApplyStagedKeepsCurrentWhenVerifyFails(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "old", 0)
	stage(t, p, "new", 0)
	fakeBin(t, p.New, "tampered", 0) // 

	action, st := applyStaged(p)
	if action != Continue {
		t.Fatalf(" Continue， %v", action)
	}
	if !st.FailedStage {
		t.Error(" FailedStage")
	}
	if !strings.Contains(readAll(t, p.Current), "old") {
		t.Fatal("")
	}
	if _, err := os.Stat(p.New); !os.IsNotExist(err) {
		t.Error("，")
	}
}

func TestSwapOverwritesPreviousBackup(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "v2", 0)
	fakeBin(t, p.Old, "v1", 0) // 
	stage(t, p, "v3", 0)

	if err := swap(p); err != nil {
		t.Fatalf("swap: %v", err)
	}
	if !strings.Contains(readAll(t, p.Current), "v3") {
		t.Error(" v3")
	}
	if !strings.Contains(readAll(t, p.Old), "v2") {
		t.Error(" v2")
	}
}

func TestConfirmCountsAttemptsThenRollsBack(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "broken-new", 0)
	fakeBin(t, p.Old, "good-old", 0)
	m := marker{From: "0.3.7", To: "0.3.8"}

	for i := 1; i <= maxAttempts; i++ {
		action, st := confirmOrRollback(p, m)
		if action != Continue {
			t.Fatalf(" %d  Continue， %v", i, action)
		}
		if !st.Pending {
			t.Errorf(" %d  Pending", i)
		}
		got, ok := readMarker(p.Marker)
		if !ok || got.Attempts != i {
			t.Fatalf(" %d  attempts=%d（ok=%v）， %d", i, got.Attempts, ok, i)
		}
		m = got
	}

	action, st := confirmOrRollback(p, m)
	if action != Restart {
		t.Fatalf(" Restart， %v", action)
	}
	if !st.RolledBack {
		t.Error(" RolledBack")
	}
	if !strings.Contains(readAll(t, p.Current), "good-old") {
		t.Fatal("")
	}
	if _, err := os.Stat(p.Marker); !os.IsNotExist(err) {
		t.Error("，")
	}
	if _, err := os.Stat(p.Current + ".failed"); err != nil {
		t.Error(" .failed ")
	}
}

func TestManualRollbackIsReversible(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "v2", 0)
	fakeBin(t, p.Old, "v1", 0)

	// Rollback()  ResolvePaths()，。
	tmp := p.Current + ".swap"
	if err := os.Rename(p.Current, tmp); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(p.Old, p.Current); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, p.Old); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readAll(t, p.Current), "v1") {
		t.Error(" v1")
	}
	if !strings.Contains(readAll(t, p.Old), "v2") {
		t.Error(" v2，")
	}
}

func TestParseSums(t *testing.T) {
	const (
		linuxSum = "1111111111111111111111111111111111111111111111111111111111111111"
		winSum   = "ABCDEF0000000000000000000000000000000000000000000000000000000000"
	)
	// sha256sum ；shasum -a 256  *。
	raw := linuxSum + "  artex-0.3.8-linux-amd64.zip\n" +
		winSum + " *artex-0.3.8-windows-amd64.zip\n" +
		"\n" +
		"garbage line\n" + // ，
		"deadbeef  artex-0.3.8-darwin-arm64.zip\n" // 

	out := parseSums(raw)
	if out["artex-0.3.8-linux-amd64.zip"] != linuxSum {
		t.Errorf("linux : %v", out)
	}
	if got := out["artex-0.3.8-windows-amd64.zip"]; got != strings.ToLower(winSum) {
		t.Errorf("windows （* 、）: %q", got)
	}
	if len(out) != 2 {
		t.Errorf("、， %v", out)
	}
}

func TestExtractBinaryFindsNestedEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(" Windows  artex.exe， Unix ")
	}
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "release.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	// ：artex-<>-<os>-<arch>/artex，。
	for name, body := range map[string]string{
		"artex-0.3.8-linux-amd64/README.md":           "readme",
		"artex-0.3.8-linux-amd64/skills/a.md":         "skill",
		"artex-0.3.8-linux-amd64/artex":               "#!/bin/sh\nexit 0\n",
		"artex-0.3.8-linux-amd64/config.example.json": "{}",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	dst := filepath.Join(dir, "out")
	if err := extractBinary(zipPath, dst); err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	if got := readAll(t, dst); !strings.Contains(got, "exit 0") {
		t.Errorf(" artex : %q", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("")
	}
}

func TestExtractBinaryMissingEntry(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "release.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("artex-0.3.8-linux-amd64/README.md")
	_, _ = w.Write([]byte("readme"))
	_ = zw.Close()
	f.Close()

	if err := extractBinary(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("")
	}
}

func TestCheckURLRejectsNonGitHub(t *testing.T) {
	bad := []string{
		"http://github.com/x",           //  HTTPS
		"https://evil.com/artex.zip",    // 
		"https://github.com.evil.com/x", // 
		"https://raw.githubusercontent.com.evil.com/x",
	}
	for _, raw := range bad {
		u := mustParse(t, raw)
		if err := checkURL(u); err == nil {
			t.Errorf("checkURL(%q) ", raw)
		}
	}
	good := []string{
		"https://api.github.com/repos/x/releases/latest",
		"https://objects.githubusercontent.com/blah",
		"https://GitHub.com/x", // 
	}
	for _, raw := range good {
		u := mustParse(t, raw)
		if err := checkURL(u); err != nil {
			t.Errorf("checkURL(%q) ，: %v", raw, err)
		}
	}
}

func TestAssetNameMatchesBuildScript(t *testing.T) {
	// build.sh  package_binary  artex-<>-<os>-<arch>.zip，
	if got := AssetName("v0.3.8", "linux", "amd64"); got != "artex-0.3.8-linux-amd64.zip" {
		t.Errorf("AssetName = %q", got)
	}
	if got := AssetName("0.3.8", "windows", "amd64"); got != "artex-0.3.8-windows-amd64.zip" {
		t.Errorf("AssetName = %q", got)
	}
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf(" %q: %v", raw, err)
	}
	return u
}

func TestSettleClearsMarkerAndStopsRollback(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "new", 0)
	fakeBin(t, p.Old, "old", 0)
	if err := writeMarker(p.Marker, marker{From: "0.3.7", To: "0.3.8", Attempts: 2}); err != nil {
		t.Fatal(err)
	}

	settle(p)

	if _, err := os.Stat(p.Marker); !os.IsNotExist(err) {
		t.Fatal("")
	}
	if _, ok := readMarker(p.Marker); ok {
		t.Error("")
	}
	if _, err := os.Stat(p.Old); err != nil {
		t.Error("")
	}
}

func TestSettleIsNoopWithoutMarker(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "cur", 0)
	settle(p) // ， panic 
	if _, err := os.Stat(p.Current); err != nil {
		t.Error(" settle ")
	}
}
