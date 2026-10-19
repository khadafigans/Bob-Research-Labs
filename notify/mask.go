package notify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MaskedPrefix 。API ，
//
const MaskedPrefix = "__masked__"

//
//
func MaskedValue(secret string) string {
	if len(secret) <= 6 {
		return MaskedPrefix
	}
	return MaskedPrefix + ":…" + secret[len(secret)-6:]
}

func IsMasked(v string) bool { return strings.HasPrefix(v, MaskedPrefix) }

//
func MaskConfig(kind string, cfg map[string]any) map[string]any {
	channel, ok := Get(kind)
	if !ok {
		return map[string]any{}
	}
	secrets := map[string]bool{}
	for _, k := range channel.SecretKeys() {
		secrets[k] = true
	}
	out := make(map[string]any, len(cfg))
	for k, v := range cfg {
		if !secrets[k] {
			out[k] = v
			continue
		}
		if s, ok := v.(string); ok {
			out[k] = MaskedValue(s)
			continue
		}
		out[k] = MaskedPrefix
	}
	return out
}

// ErrDestinationChangedWithoutCredentials 「，
// 」。， PrepareConfigUpdate。
type ErrDestinationChangedWithoutCredentials struct {
	Changed []string // 
	Missing []string // 
}

func (e *ErrDestinationChangedWithoutCredentials) Error() string {
	return "Target address (" + strings.Join(e.Changed, ", ") + ") changed. Re-enter credential field(s) (" +
		strings.Join(e.Missing, ", ") + "): provide a new value, or leave empty to indicate credentials are no longer needed. " +
		"Existing credentials are only valid for the previous address; reusing them would send them to the new address."
}

// PrepareConfigUpdate ，「」。
//
//
//	webhook  {config:{url:"https://attacker.tld"}}  →  Authorization 
//	telegram {config:{base_url:"https://attacker.tld"}} → /bot<Token>/sendMessage
//	email    {config:{host:"smtp.attacker.tld"}}    → STARTTLS 
//
//
//
// （webhook  headers、email  password）「 200」，
func PrepareConfigUpdate(kind string, stored, incoming map[string]any) (map[string]any, error) {
	channel, ok := Get(kind)
	if !ok {
		return nil, fmt.Errorf("Unregistered channel kind: %q", kind)
	}
	secrets := channel.SecretKeys()
	destinations := channel.DestinationKeys()

	// （ webhook  headers ），
	//
	if err := rejectMaskedInContainers(incoming, secrets); err != nil {
		return nil, err
	}

	var changed []string
	for _, key := range destinations {
		raw, present := incoming[key]
		if !present {
			continue
		}
		s, isStr := raw.(string)
		if isStr && IsMasked(s) {
			continue
		}
		if !sameConfigValue(raw, stored[key]) {
			changed = append(changed, key)
		}
	}
	if len(changed) == 0 {
		return MergeConfig(stored, incoming), nil
	}

	var missing []string
	for _, key := range secrets {
		raw, present := incoming[key]
		if !present {
			missing = append(missing, key)
			continue
		}
		if s, isStr := raw.(string); isStr && IsMasked(s) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, &ErrDestinationChangedWithoutCredentials{Changed: changed, Missing: missing}
	}
	return MergeConfig(stored, incoming), nil
}

// rejectMaskedInContainers 。
//
// 「」。 webhook  headers ，
func rejectMaskedInContainers(incoming map[string]any, secretKeys []string) error {
	for _, key := range secretKeys {
		raw, present := incoming[key]
		if !present {
			continue
		}
		if _, isStr := raw.(string); isStr {
			continue
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		if strings.Contains(string(encoded), MaskedPrefix) {
			return fmt.Errorf("%s field contains a mask marker (%q). Leave the field empty to keep the existing value, or submit a full new value; mask markers cannot be embedded inside structured fields",
				key, MaskedPrefix)
		}
	}
	return nil
}

// sameConfigValue 。 JSON 
//
//  MergeConfig 、 delete 。，
// （Telegram  base_url ：
//
//	 base_url:""  →   MergeConfig 
//	→  incoming  ""、stored ，「」
//
func sameConfigValue(a, b any) bool {
	if isBlankConfigValue(a) && isBlankConfigValue(b) {
		return true
	}
	ra, errA := json.Marshal(a)
	rb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ra) == string(rb)
}

// isBlankConfigValue 「」。
//  MergeConfig （strings.TrimSpace(s) == ""），
// 「MergeConfig 、sameConfigValue 」。
func isBlankConfigValue(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

// MergeConfig  incoming  stored ，。
//
//   - incoming  →  stored （）
//   - stored  incoming  → （）
//
func MergeConfig(stored, incoming map[string]any) map[string]any {
	out := make(map[string]any, len(stored)+len(incoming))
	for k, v := range stored {
		out[k] = v
	}
	for k, v := range incoming {
		if s, ok := v.(string); ok {
			if IsMasked(s) {
				continue //  = ， stored
			}
			if strings.TrimSpace(s) == "" {
				delete(out, k)
				continue
			}
			out[k] = s
			continue
		}
		out[k] = v
	}
	return out
}
