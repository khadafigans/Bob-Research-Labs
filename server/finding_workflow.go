package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/traffic"
	actool "github.com/Autumn-27/norma/tool"
)

//
//	(1)  · description: report_finding  evidence_hint_id·hints.text,
//	    bind_finding_traffic  wrTool ·finding_id strParam. LLM   
//	(2)   : seedFindingWorkflowTools  legacy ( traffic_search
//	    ) traffic.TrafficSearchDescription(    ) 
//	    UPDATE … WHERE description=$2  .     
//	(3) actool.Errorf  : agentFindingTrafficAccess    
//	    toolBindFindingTraffic( ) get_finding_traffic(finding_traffic.go) 
//	      actool.Errorf(err.Error())    ,
//	    toolBindFindingTraffic   · traffic_refs   actool.Errorf .
//
func (s *Server) seedFindingWorkflowTools() {
	const hostSearchDescriptionFlag = "finding_workflow_tools_v3_host_search_description"
	if value, _, _ := s.m.pg.GetSetting(hostSearchDescriptionFlag); value != "true" {
		// Only replace the original built-in text. A user-edited description is
		// authoritative and must survive upgrades.
		legacy := "（ host， URL ）。body_contains /，（ 3 ），、、、。(id/method/url/status/resp_len)，。 3 、 10 ； page （page=0 ）；/ traffic_get(id)。、， curl  URL。"
		if _, err := s.m.pg.Exec(`UPDATE tools SET description=$1,updated_at=now() WHERE key='traffic_search' AND system AND description=$2`, traffic.TrafficSearchDescription, legacy); err != nil {
			// Log and leave the flag unset so the next startup retries; do not
			// return, or a transient error here would also skip the reporter
			// migration below — the two are independent.
			log.Printf("[evidence] upgrade traffic_search description: %v", err)
		} else {
			_ = s.m.pg.SetSetting(hostSearchDescriptionFlag, "true")
		}
	}
	const flag = "finding_workflow_tools_v2_reporter"
	if value, _, _ := s.m.pg.GetSetting(flag); value == "true" {
		return
	}
	for _, key := range []string{"report_finding", "add_hint", "add_task_hint"} {
		row, err := s.m.pg.GetTool(key)
		if err != nil {
			log.Printf("[evidence] load %s: %v", key, err)
			return
		}
		if row == nil || !row.System {
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(row.Schema, &schema); err != nil {
			log.Printf("[evidence] invalid schema for %s: %v", key, err)
			return
		}
		if schema == nil {
			log.Printf("[evidence] missing object schema for %s", key)
			return
		}
		props := objectProperty(schema, "properties")
		if key == "report_finding" {
			if _, exists := props["evidence_hint_id"]; !exists {
				props["evidence_hint_id"] = map[string]any{"type": "integer", "description": "： hint ID； traffic_refs ，"}
			}
		} else {
			if _, exists := props["traffic_refs"]; !exists {
				props["traffic_refs"] = agent.HintTrafficSchema()
			}
			hints := objectProperty(props, "hints")
			if _, exists := hints["type"]; !exists {
				hints["type"] = "array"
			}
			items := objectProperty(hints, "items")
			if _, ok := items["type"]; !ok {
				items["type"] = "object"
			}
			itemProps := objectProperty(items, "properties")
			for name, value := range map[string]any{"text": strParam(""), "asset_ids": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}, "traffic_refs": agent.HintTrafficSchema()} {
				if _, exists := itemProps[name]; !exists {
					itemProps[name] = value
				}
			}
		}
		raw, _ := json.Marshal(schema)
		result, err := s.m.pg.Exec(`UPDATE tools SET schema=$2::jsonb,updated_at=now() WHERE key=$1 AND system AND schema=$3::jsonb`, key, string(raw), string(row.Schema))
		if err != nil {
			log.Printf("[evidence] upgrade %s: %v", key, err)
			return
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return
		} // preserve concurrent user edits
	}
	// Upgrade only the original default binding. Customized lists and enabled
	// flags survive; the one-time flag also preserves future user unbinding.
	readers := `["worker","reporter"]`
	for _, key := range []string{"traffic_search", "traffic_get", "traffic_blob"} {
		if _, err := s.m.pg.Exec(`UPDATE tools SET agents=$2::jsonb WHERE key=$1 AND system AND (agents='["worker"]'::jsonb OR (agents @> '["worker","planner","mainagent","auto","pentest"]'::jsonb AND jsonb_array_length(agents)=5))`, key, readers); err != nil {
			return
		}
	}
	if _, err := s.m.pg.Exec(`UPDATE tools SET agents=$1::jsonb WHERE key='get_finding_traffic' AND system AND agents @> '["auto","reporter"]'::jsonb AND jsonb_array_length(agents)=2`, `["auto","reporter","worker","planner","mainagent","pentest"]`); err != nil {
		return
	}
	// Replace the previous code default only; preserve customized binding lists.
	if _, err := s.m.pg.Exec(`UPDATE tools SET agents='["reporter"]'::jsonb WHERE key='bind_finding_traffic' AND system AND agents @> '["worker","planner","mainagent","auto","pentest"]'::jsonb AND jsonb_array_length(agents)=5`); err != nil {
		return
	}
	if err := s.m.pg.AddAgentToToolBinding("reporter", []string{"bind_finding_traffic"}); err != nil {
		return
	}
	_ = s.m.pg.SetSetting(flag, "true")
}

func objectProperty(parent map[string]any, key string) map[string]any {
	value, ok := parent[key].(map[string]any)
	if !ok {
		value = map[string]any{}
		parent[key] = value
	}
	return value
}

func (s *Server) agentFindingTrafficAccess(ctx context.Context, id int64, write bool) error {
	if id <= 0 {
		return errors.New("finding_id  ID； ID")
	}
	f, err := s.m.pg.GetFinding(id)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("%w：finding_id=%d。 ID， list_task_findings / get_task_node_detail  finding_id ； id / finding_node_id", db.ErrFindingNotFound, id)
	}
	if ri := agent.RunInfoFrom(ctx); ri.TaskID > 0 {
		task := s.m.ResolveTask(strconv.FormatInt(ri.TaskID, 10))
		if task == nil {
			return errors.New("")
		}
		_, inherited, allowed := findingProvenanceInTask(task, f.TaskID)
		if !allowed {
			return errors.New("")
		}
		if write && inherited {
			return errors.New("，")
		}
	}
	return nil
}

func (s *Server) toolBindFindingTraffic() actool.CoreTool {
	return wrTool("bind_finding_traffic", " HTTP 。finding_id  ID； ID。，。；。",
		objSchema(map[string]any{"finding_id": strParam(" ID， list_task_findings / get_task_node_detail  finding_id "), "traffic_refs": agent.HintTrafficSchema()}, "finding_id", "traffic_refs"),
		func(ctx context.Context, raw json.RawMessage) (actool.Result, error) {
			if !s.m.pg.GetBool(settingAgentTrafficBinding, false) {
				return actool.Errorf("Agent ；，。"), nil
			}
			var args struct {
				FindingID json.RawMessage `json:"finding_id"`
				Refs      []db.TrafficRef `json:"traffic_refs"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return actool.Errorf(err.Error()), nil
			}
			id := parseProfileID(args.FindingID)
			if err := s.agentFindingTrafficAccess(ctx, id, true); err != nil {
				return actool.Errorf(err.Error()), nil
			}
			if len(args.Refs) == 0 {
				return actool.Errorf(" traffic_refs；"), nil
			}
			list, err := s.evidenceStore().Bind(ctx, id, args.Refs)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			return jsonResult(trafficSummary(list))
		})
}
