package agent

import (
	"context"

	"github.com/Autumn-27/artex/db"
)

// FindingRecorder is injected by the host; agents never synthesize or copy
// evidence bodies themselves. Its implementation owns the atomic write.
type FindingRecorder interface {
	Record(context.Context, db.RecordFindingInput, []db.TrafficRef) (*db.RecordedFinding, error)
}

// Tool-use guidance is appended without replacing the user's editable prompt.
// It does not require capture or claim that unavailable traffic tools exist.
const findingTrafficGuidance = "\n\n**（）**： report_finding ， HTTP /， traffic_refs  ID；，。TCP  HTTP 、 []， evidence 、，。 ID，。"

func (t *ToolSet) SetFindingRecorder(r FindingRecorder)   { t.findingRecorder = r }
func (w *Worker) SetFindingRecorder(r FindingRecorder)    { w.findingRecorder = r }
func (p *Planner) SetFindingRecorder(r FindingRecorder)   { p.findingRecorder = r }
func (m *MainAgent) SetFindingRecorder(r FindingRecorder) { m.findingRecorder = r }
