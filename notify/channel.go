package notify

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
)

type Channel interface {
	Kind() string
	Validate(cfg map[string]any) error
	//
	//
	// ， *PermanentError 。
	Send(ctx context.Context, cfg map[string]any, m Message) (int, error)
	// DefaultRatePerMin ，
	DefaultRatePerMin() int
	// SecretKeys 。API ，
	// （ Webhook ， secret），
	SecretKeys() []string
	// DestinationKeys 「」。
	//
	//  PrepareConfigUpdate。
	DestinationKeys() []string
}

// registry 。 init() ：「」
var registry = map[string]Channel{
	KindDingTalk: dingTalkChannel{},
	KindFeishu:   feishuChannel{},
	KindWeCom:    weComChannel{},
	KindWebhook:  webhookChannel{},
	KindTelegram: telegramChannel{},
	KindEmail:    emailChannel{},
}

func Get(kind string) (Channel, bool) {
	c, ok := registry[kind]
	return c, ok
}

// ValidKind  kind 。
func ValidKind(kind string) bool {
	_, ok := registry[kind]
	return ok
}

func Kinds() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// PermanentError ：、、。
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// Permanent  err 。err  nil  nil，
//  `return Permanent(someCheck())`。
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// IsPermanent  err 。
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

//
//  JSONB ， encoding/json  map[string]any，
//  float64、 []any。 helper ，

func cfgString(cfg map[string]any, key string) string {
	v, ok := cfg[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// cfgInt ， float64（JSON ）。
func cfgInt(cfg map[string]any, key string) int {
	switch v := cfg[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func cfgBool(cfg map[string]any, key string) bool {
	switch v := cfg[key].(type) {
	case bool:
		return v
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

func cfgStrings(cfg map[string]any, key string) []string {
	raw, ok := cfg[key].([]any)
	if !ok {
		if s := cfgString(cfg, key); s != "" {
			return []string{s}
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func cfgMap(cfg map[string]any, key string) map[string]string {
	raw, ok := cfg[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		out[k] = s
	}
	return out
}
