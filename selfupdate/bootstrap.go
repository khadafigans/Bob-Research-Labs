package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// smokeEnv  Bootstrap。
//
// ： os.Executable()  artex.new，
const smokeEnv = "ARTEX_SELFUPDATE_SMOKE"

// Action  Bootstrap  main 。
type Action int

const (
	// Continue： server。
	Continue Action = iota
	// Restart： ExitRestart ，。
	Restart
)

// State ， /api/update/check 
type State struct {
	Pending     bool   // 
	RolledBack  bool   // 
	FailedStage bool   // /，
	Detail      string // 
}

// Bootstrap  main ，、。
//
//
func Bootstrap() (Action, State) {
	if os.Getenv(smokeEnv) != "" {
		return Continue, State{}
	}
	p, err := ResolvePaths()
	if err != nil {
		log.Printf("[update] ：%v", err)
		return Continue, State{}
	}

	if _, err := os.Stat(p.New); err == nil {
		return applyStaged(p)
	}

	m, ok := readMarker(p.Marker)
	if !ok {
		return Continue, State{}
	}
	return confirmOrRollback(p, m)
}

//
func applyStaged(p Paths) (Action, State) {
	m, _ := readMarker(p.Marker)

	if err := verifyStaged(p); err != nil {
		log.Printf("[update] ，，：%v", err)
		cleanStaged(p)
		_ = os.Remove(p.Marker)
		return Continue, State{FailedStage: true, Detail: "    : " + err.Error()}
	}

	if err := swap(p); err != nil {
		log.Printf("[update] ，：%v", err)
		cleanStaged(p)
		_ = os.Remove(p.Marker)
		return Continue, State{FailedStage: true, Detail: "  : " + err.Error()}
	}

	m.Attempts = 0
	if m.StagedAt == 0 {
		m.StagedAt = time.Now().Unix()
	}
	if err := writeMarker(p.Marker, m); err != nil {
		log.Printf("[update] （）：%v", err)
	}
	log.Printf("[update]  %s，（exit %d）", orUnknown(m.To), ExitRestart)
	return Restart, State{Pending: true}
}

// confirmOrRollback ""：，。
//
func confirmOrRollback(p Paths, m marker) (Action, State) {
	m.Attempts++
	if m.Attempts > maxAttempts {
		if err := rollback(p); err != nil {
			log.Printf("[update]  %d ，：%v", maxAttempts, err)
			_ = os.Remove(p.Marker)
			return Continue, State{Detail: "     : " + err.Error()}
		}
		log.Printf("[update]  %d ， %s，（exit %d）",
			maxAttempts, orUnknown(m.From), ExitRestart)
		_ = os.Remove(p.Marker)
		return Restart, State{RolledBack: true, Detail: fmt.Sprintf("      (%s)", orUnknown(m.From))}
	}
	if err := writeMarker(p.Marker, m); err != nil {
		log.Printf("[update] ：%v", err)
	}
	log.Printf("[update] （ %d/%d ），",
		m.Attempts, maxAttempts)
	return Continue, State{Pending: true}
}

//
func Settle() {
	p, err := ResolvePaths()
	if err != nil {
		return
	}
	settle(p)
}

func settle(p Paths) {
	if _, ok := readMarker(p.Marker); !ok {
		return // ，
	}
	if err := os.Remove(p.Marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("[update] ：%v", err)
		return
	}
	log.Printf("[update] ，（ %s）", p.Old)
}

const SettleDelay = 30 * time.Second

// verifyStaged ： SHA256，。
func verifyStaged(p Paths) error {
	want, err := os.ReadFile(p.Sum)
	if err != nil {
		return fmt.Errorf("  : %w", err)
	}
	got, err := fileSHA256(p.New)
	if err != nil {
		return fmt.Errorf("  : %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(string(want)), got) {
		return errors.New("SHA256   (  )")
	}
	return smokeTest(p.New)
}

// 、（exec format error）、。
func smokeTest(bin string) error {
	if err := os.Chmod(bin, 0o755); err != nil {
		return fmt.Errorf("   : %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "-h")
	cmd.Env = append(os.Environ(), smokeEnv+"=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return errors.New("   (   )")
	}
	if err != nil {
		snippet := strings.TrimSpace(string(out))
		if len(snippet) > 300 {
			snippet = snippet[:300] + "…"
		}
		return fmt.Errorf("  : %v: %s", err, snippet)
	}
	return nil
}

//
// Unix  Windows  rename （Windows 
func swap(p Paths) error {
	// Windows  rename ， .old 。
	if err := os.Remove(p.Old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("  %s  : %w", p.Old, err)
	}
	if err := os.Rename(p.Current, p.Old); err != nil {
		return fmt.Errorf("   : %w", err)
	}
	if err := os.Rename(p.New, p.Current); err != nil {
		if rerr := os.Rename(p.Old, p.Current); rerr != nil {
			return fmt.Errorf("   (%v),    : %w", err, rerr)
		}
		return fmt.Errorf("   : %w", err)
	}
	_ = os.Remove(p.Sum)
	return nil
}

// rollback  swap 。
func rollback(p Paths) error {
	if _, err := os.Stat(p.Old); err != nil {
		return fmt.Errorf("  (%s): %w", p.Old, err)
	}
	failed := p.Current + ".failed"
	_ = os.Remove(failed)
	if err := os.Rename(p.Current, failed); err != nil {
		return fmt.Errorf("   : %w", err)
	}
	if err := os.Rename(p.Old, p.Current); err != nil {
		return fmt.Errorf("   : %w", err)
	}
	return nil
}

// Rollback  /api/update/rollback ：。
func Rollback() error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	if _, err := os.Stat(p.Old); err != nil {
		return errors.New("   (" + p.Old + " )")
	}
	cleanStaged(p)
	if err := smokeTest(p.Old); err != nil {
		return fmt.Errorf("      : %w", err)
	}
	tmp := p.Current + ".swap"
	_ = os.Remove(tmp)
	if err := os.Rename(p.Current, tmp); err != nil {
		return fmt.Errorf("   : %w", err)
	}
	if err := os.Rename(p.Old, p.Current); err != nil {
		_ = os.Rename(tmp, p.Current)
		return fmt.Errorf("   : %w", err)
	}
	if err := os.Rename(tmp, p.Old); err != nil {
		log.Printf("[update] （）：%v", err)
	}
	_ = os.Remove(p.Marker)
	return nil
}

func HasBackup() bool {
	p, err := ResolvePaths()
	if err != nil {
		return false
	}
	_, err = os.Stat(p.Old)
	return err == nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "   "
	}
	return s
}
