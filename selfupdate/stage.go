package selfupdate

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

// sumsAsset  release.yml ， Release  zip。
const sumsAsset = "SHA256SUMS"

// maxBinarySize ， zip 。
const maxBinarySize = 512 << 20 // 512 MiB

// Phase ， SSE  phase 。
type Phase string

const (
	PhaseIdle     Phase = "idle"
	PhaseDownload Phase = "downloading"
	PhaseVerify   Phase = "verifying"
	PhaseExtract  Phase = "extracting"
	PhaseStaged   Phase = "staged"
	PhaseFailed   Phase = "failed"
)

type Progress func(ph Phase, pct int, msg string)

// Stage  Release ， artex.new。
//
//  zip ，： Release  SHA256SUMS 
// skills/， skill 。 skills  KB。
//
func Stage(ctx context.Context, c *http.Client, rel *Release, currentVersion string, prog Progress) error {
	if prog == nil {
		prog = func(Phase, int, string) {}
	}
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	if err := checkWritable(p.Dir); err != nil {
		return err
	}

	name := AssetName(rel.TagName, runtime.GOOS, runtime.GOARCH)
	asset, ok := rel.FindAsset(name)
	if !ok {
		return fmt.Errorf("  %s/%s     (%s )", runtime.GOOS, runtime.GOARCH, name)
	}

	prog(PhaseDownload, 0, "   …")
	sums, err := fetchSums(ctx, c, rel)
	if err != nil {
		return err
	}
	want, ok := sums[name]
	if !ok {
		return fmt.Errorf("%s  %s       ", sumsAsset, name)
	}

	zipPath := p.New + ".zip.part"
	binPath := p.New + ".part"
	defer func() {
		_ = os.Remove(zipPath)
		_ = os.Remove(binPath)
	}()

	prog(PhaseDownload, 0, fmt.Sprintf("%s  (%s)…", name, humanSize(asset.Size)))
	got, err := download(ctx, c, asset, zipPath, prog)
	if err != nil {
		return err
	}

	prog(PhaseVerify, -1, "SHA256  …")
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("SHA256   :  %s,  %s(  )", short(want), short(got))
	}

	prog(PhaseExtract, -1, "    …")
	if err := extractBinary(zipPath, binPath); err != nil {
		return err
	}
	if err := smokeTest(binPath); err != nil {
		return fmt.Errorf("     : %w", err)
	}

	binSum, err := fileSHA256(binPath)
	if err != nil {
		return fmt.Errorf("    : %w", err)
	}
	if err := os.WriteFile(p.Sum, []byte(binSum), 0o644); err != nil {
		return fmt.Errorf("  : %w", err)
	}
	if err := os.Rename(binPath, p.New); err != nil {
		_ = os.Remove(p.Sum)
		return fmt.Errorf("   : %w", err)
	}

	if err := writeMarker(p.Marker, marker{
		From:     currentVersion,
		To:       strings.TrimPrefix(rel.TagName, "v"),
		StagedAt: time.Now().Unix(),
	}); err != nil {
		prog(PhaseStaged, -1, ":            ")
	}

	prog(PhaseStaged, 100, "  .  …")
	return nil
}

// fetchSums  SHA256SUMS，  → 。
func fetchSums(ctx context.Context, c *http.Client, rel *Release) (map[string]string, error) {
	asset, ok := rel.FindAsset(sumsAsset)
	if !ok {
		return nil, fmt.Errorf("  %s        ", sumsAsset)
	}
	body, err := get(ctx, c, asset.URL)
	if err != nil {
		return nil, fmt.Errorf("%s  : %w", sumsAsset, err)
	}
	defer body.Close()

	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%s  : %w", sumsAsset, err)
	}
	out := parseSums(string(raw))
	if len(out) == 0 {
		return nil, fmt.Errorf("%s       ", sumsAsset)
	}
	return out, nil
}

// parseSums  sha256sum ，  → 。
//
func parseSums(raw string) map[string]string {
	out := map[string]string{}
	for line := range strings.Lines(raw) {
		//  "<sha256>  <filename>"（sha256sum ；shasum 
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || !isHexSHA256(fields[0]) {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == "" {
			continue
		}
		out[name] = strings.ToLower(fields[0])
	}
	return out
}

func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// download  dst， SHA256  Content-Length 。
func download(ctx context.Context, c *http.Client, a Asset, dst string, prog Progress) (string, error) {
	body, err := get(ctx, c, a.URL)
	if err != nil {
		return "", fmt.Errorf("%s  : %w", a.Name, err)
	}
	defer body.Close()

	f, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("   : %w", err)
	}
	defer f.Close()

	h := sha256.New()
	pw := &progressWriter{total: a.Size, prog: prog, name: a.Name, last: time.Now()}
	if _, err := io.Copy(io.MultiWriter(f, h, pw), body); err != nil {
		return "", fmt.Errorf(" : %w", err)
	}
	if err := f.Sync(); err != nil {
		return "", fmt.Errorf("  : %w", err)
	}
	if a.Size > 0 && pw.written != a.Size {
		return "", fmt.Errorf("  :  %d,  %d", a.Size, pw.written)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func get(ctx context.Context, c *http.Client, rawURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if err := checkURL(req.URL); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "artex-selfupdate")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// extractBinary  artex 。
//
//  artex-<>-<os>-<arch>/artex，****
func extractBinary(zipPath, dst string) error {
	want := "artex"
	if runtime.GOOS == "windows" {
		want = "artex.exe"
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("   : %w", err)
	}
	defer zr.Close()

	for _, entry := range zr.File {
		if entry.FileInfo().IsDir() || !strings.EqualFold(path.Base(entry.Name), want) {
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return fmt.Errorf("%s  : %w", entry.Name, err)
		}
		defer rc.Close()

		f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return fmt.Errorf("   : %w", err)
		}
		defer f.Close()

		n, err := io.Copy(f, io.LimitReader(rc, maxBinarySize+1))
		if err != nil {
			return fmt.Errorf("%s   : %w", entry.Name, err)
		}
		if n > maxBinarySize {
			return fmt.Errorf("     %s     ", humanSize(maxBinarySize))
		}
		if n == 0 {
			return fmt.Errorf("   %s   ", want)
		}
		return f.Sync()
	}
	return fmt.Errorf("  %s   ", want)
}

// checkWritable 。， root 、
func checkWritable(dir string) error {
	probe, err := os.CreateTemp(dir, ".artex-update-probe-*")
	if err != nil {
		return fmt.Errorf("  %s         (    ): %w", dir, err)
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return nil
}

// progressWriter ， 32KiB  SSE。
type progressWriter struct {
	total   int64
	written int64
	name    string
	prog    Progress
	last    time.Time
}

func (w *progressWriter) Write(b []byte) (int, error) {
	w.written += int64(len(b))
	if time.Since(w.last) < 300*time.Millisecond {
		return len(b), nil
	}
	w.last = time.Now()
	pct := -1
	if w.total > 0 {
		pct = int(w.written * 100 / w.total)
	}
	w.prog(PhaseDownload, pct, fmt.Sprintf("  %s / %s", humanSize(w.written), humanSize(w.total)))
	return len(b), nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

func short(sum string) string {
	if len(sum) > 12 {
		return sum[:12] + "…"
	}
	return sum
}
