package notify

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Filter  notification_channels.filter  JSONB ：。
type Filter struct {
	// MinSeverity （low/medium/high/critical），=。
	MinSeverity string `json:"min_severity"`
	// TaskIDs / AssetIDs ；。
	TaskIDs  []int64 `json:"task_ids"`
	AssetIDs []int64 `json:"asset_ids"`
	// VulnClassInclude ； vulnclass 。
	// VulnClassExclude （）。
	VulnClassInclude []string `json:"vulnclass_include"`
	VulnClassExclude []string `json:"vulnclass_exclude"`
	// OnStatusChange （ realtime ）。
	OnStatusChange bool `json:"on_status_change"`
}

//
func ParseFilter(raw []byte) Filter {
	var f Filter
	if len(raw) == 0 {
		return f
	}
	_ = json.Unmarshal(raw, &f)
	return f
}

// ValidMinSeverity  s （）。
func ValidMinSeverity(s string) bool {
	if s == "" {
		return true
	}
	_, ok := severityRank[s]
	return ok
}

//
//  min_severity （"hgih"），****
//
//  Validate ****。 ParseFilter ，
func (f Filter) Validate() error {
	if !ValidMinSeverity(f.MinSeverity) {
		return fmt.Errorf("Invalid min severity: %q. Choose low / medium / high / critical, or leave empty for no limit", f.MinSeverity)
	}
	return nil
}

//
// ** error**， ParseFilter：「」。
func Match(f Filter, s Snapshot) bool {
	if s.Kind == EventFindingStatusChanged && !f.OnStatusChange {
		return false
	}
	if !AtLeast(s.Severity, f.MinSeverity) {
		return false
	}
	if len(f.TaskIDs) > 0 && !slices.Contains(f.TaskIDs, s.TaskID) {
		return false
	}
	if len(f.AssetIDs) > 0 && !intersectsInt(f.AssetIDs, s.AssetIDs) {
		return false
	}
	if len(f.VulnClassExclude) > 0 && containsAnyFold(s.VulnClass, f.VulnClassExclude) {
		return false
	}
	if len(f.VulnClassInclude) > 0 && !containsAnyFold(s.VulnClass, f.VulnClassInclude) {
		return false
	}
	return true
}

func intersectsInt(a, b []int64) bool {
	for _, v := range b {
		if slices.Contains(a, v) {
			return true
		}
	}
	return false
}

// containsAnyFold  s  keywords （）。
func containsAnyFold(s string, keywords []string) bool {
	lower := strings.ToLower(s)
	for _, kw := range keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw != "" && strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
