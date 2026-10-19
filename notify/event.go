package notify

// Snapshot  notification_events.snapshot  JSONB 。 db 
//
// 。 fan-out  JOIN findings/tasks/assets 。
type Snapshot struct {
	// ：finding_created / finding_status_changed
	Kind      string  `json:"kind"`
	FindingID int64   `json:"finding_id"`
	TaskID    int64   `json:"task_id"`
	VulnClass string  `json:"vulnclass"`
	Name      string  `json:"name"`
	Severity  string  `json:"severity"`
	Summary   string  `json:"summary"`
	AssetIDs  []int64 `json:"asset_ids"`
	//  kind=finding_status_changed 。
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status,omitempty"`
}

type Item struct {
	FindingID int64
	Name      string
	VulnClass string
	Severity  string
	Summary   string
	// Assets （/IP）。 server ——
	Assets []string
	// DetailURL ； public_base_url，。
	DetailURL string
	FromStatus string
	ToStatus   string
}

// IsStatusChange 。
func (i Item) IsStatusChange() bool { return i.FromStatus != "" || i.ToStatus != "" }

// Title ： name， vulnclass，
func (i Item) Title() string {
	if i.Name != "" {
		return i.Name
	}
	if i.VulnClass != "" {
		return i.VulnClass
	}
	return "(unnamed finding)"
}

type Message struct {
	Items []Item
	Batch bool
	// WindowMinutes （）， Batch=true 「 N 」。
	WindowMinutes int
	// HomeURL （ public_base_url）；。
	HomeURL string
}
