// Package notify  IM / 。
//
//  map[string]any （ notification_channels.config  JSONB ），
//  PostgreSQL ， server 。
//
// ：Channel ****。 Channel 
package notify

// 。 notification_channels.kind ， server 
// （ findings.status ， DB CHECK，）。
const (
	KindDingTalk = "dingtalk" // 
	KindFeishu   = "feishu"   // ( Lark)
	KindWeCom    = "wecom"    // 
	KindWebhook  = "webhook"  //  Webhook：//JSON 
	KindTelegram = "telegram" // Telegram Bot API
	KindEmail    = "email"    // SMTP 
)

// ， notification_events.kind。
const (
	EventFindingCreated       = "finding_created"
	EventFindingStatusChanged = "finding_status_changed"
)

// InitKind  config  kind 。
const InitKind = KindDingTalk

// severityRank 。 0，
var severityRank = map[string]int{
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// SeverityRank ； 0。
func SeverityRank(severity string) int { return severityRank[severity] }

// SeverityLabel  emoji ，。
func SeverityLabel(severity string) string {
	switch severity {
	case "critical":
		return "🔴 Critical"
	case "high":
		return "🟠 High"
	case "medium":
		return "🟡 Medium"
	case "low":
		return "🔵 Low"
	default:
		return severity
	}
}

func StatusLabel(status string) string {
	switch status {
	case "pending":
		return "Pending"
	case "in_progress":
		return "In progress"
	case "confirmed":
		return "Confirmed"
	case "resolved":
		return "Resolved"
	case "fixed":
		return "Fixed"
	case "false_positive":
		return "False positive"
	case "ignored":
		return "Ignored"
	case "duplicate":
		return "Duplicate"
	case "risk_accepted":
		return "Risk accepted"
	default:
		return status
	}
}

// AtLeast  severity  min 。min ，。
//  severity  0， min （ severityRank ）。
func AtLeast(severity, min string) bool {
	if min == "" {
		return true
	}
	return SeverityRank(severity) >= SeverityRank(min)
}
