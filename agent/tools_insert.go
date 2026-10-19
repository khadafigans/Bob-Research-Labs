package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Autumn-27/artex/db"
	actool "github.com/Autumn-27/norma/tool"
)

// assetInterceptCandidates  /IP/URL ，。
// URL  host ，「 URL」/ /IP 。
func assetInterceptCandidates(item assetInputItem) (domains, ips, urls []string) {
	add := func(dst *[]string, s string) {
		if s = strings.TrimSpace(s); s != "" {
			*dst = append(*dst, s)
		}
	}
	add(&domains, item.Domain)
	for _, d := range item.BoundDomains {
		add(&domains, d)
	}
	add(&ips, item.IP)
	add(&ips, item.ServiceIP)
	add(&urls, item.URL)
	if item.URL != "" {
		if u, err := url.Parse(item.URL); err == nil {
			if h := u.Hostname(); h != "" {
				if net.ParseIP(h) != nil {
					add(&ips, h)
				} else {
					add(&domains, h)
				}
			}
		}
	}
	return domains, ips, urls
}

// assetInputLabel ，。
func assetInputLabel(item assetInputItem) string {
	typ := strings.TrimSpace(item.Type)
	var target string
	switch {
	case strings.TrimSpace(item.Domain) != "":
		target = strings.TrimSpace(item.Domain)
	case strings.TrimSpace(item.URL) != "":
		target = strings.TrimSpace(item.URL)
	case strings.TrimSpace(item.IP) != "":
		target = strings.TrimSpace(item.IP)
	case strings.TrimSpace(item.ServiceIP) != "":
		target = strings.TrimSpace(item.ServiceIP)
	default:
		target = "()"
	}
	if typ != "" {
		return fmt.Sprintf("[%s] %s", typ, target)
	}
	return target
}

// =====================================================================
// Unified asset insertion tools
// =====================================================================

// SetAssetStore wires the asset store and company store onto this ToolSet
// so the insert_assets, add_company_scope, and list_assets tools are active.
func (t *ToolSet) SetAssetStore(as *db.AssetStore, cs *db.CompanyStore) {
	t.as = as
	t.cs = cs
}

// assetInputItem is one element of the insert_assets "assets" array.
type assetInputItem struct {
	Type string `json:"type"` // root_domain|ip|subdomain|app|service|endpoint

	// ---- root_domain / subdomain ----
	Domain      string   `json:"domain"`
	ICP         string   `json:"icp"`
	RecordType  string   `json:"record_type"`
	RecordValue []string `json:"record_value"`

	// ---- ip ----
	IP           string           `json:"ip"`
	BoundDomains []string         `json:"bound_domains"`
	OpenPorts    []db.PortService `json:"open_ports"`

	// ---- app ----
	AppName     string `json:"app_name"`
	BundleID    string `json:"bundle_id"`
	Category    string `json:"category"`
	Description string `json:"description"`
	AppICP      string `json:"app_icp"`
	CompanyID   *int64 `json:"company_id"` // explicit company link (app only; others auto-attribute via scope)

	// ---- service (http) ----
	URL           string           `json:"url"`
	Technologies  []string         `json:"technologies"`
	StatusCode    *int             `json:"status_code"`
	ContentLength *int64           `json:"content_length"`
	PageTitle     string           `json:"page_title"`
	FaviconMMH3   string           `json:"favicon_mmh3"`
	Auth          []map[string]any `json:"auth"`
	ServiceName   string           `json:"service_name"`
	ServiceIP     string           `json:"service_ip"` // optional enrichment IP

	// ---- service (other) ----
	Port  int    `json:"port"`
	Proto string `json:"proto"`

	// ---- endpoint ----
	Method string           `json:"method"`
	Params []map[string]any `json:"params"`
}

// insertAssets is the unified insert_assets agent tool.
func (t *ToolSet) insertAssets() actool.CoreTool {
	return writeTool(
		"insert_assets",
		"，（type ）。\n"+
			"：root_domain→domain；ip→ip（ IPv4/IPv6，）；subdomain→domain；app→app_name；service(HTTP)→url；service(HTTP)→service_name+port（ip/domain ）；endpoint→url+method。。\n"+
			"auth/technologies/params (append)，。\n"+
			"：{results:[{index,id,type}], errors:[{index,error}]}",
		obj(map[string]any{
			// task_id ：worker  task  SetTaskID ( handler)。
			"assets": map[string]any{
				"type":        "array",
				"description": "，",
				"items": obj(map[string]any{
					"type": map[string]any{
						"type":        "string",
						"enum":        []string{"root_domain", "ip", "subdomain", "app", "service", "endpoint"},
						"description": "",
					},
					// root_domain / subdomain
					"domain":      str("（root_domain/subdomain ）"),
					"icp":         str("ICP （）"),
					"record_type": str("DNS ：A/AAAA/CNAME/MX （subdomain ）"),
					"record_value": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "DNS （subdomain ， [\"1.2.3.4\",\"2.3.4.5\"]）",
					},
					// ip
					"ip": str("IP ， IPv4/IPv6 ，（ type=subdomain  domain ）；ip ；service/endpoint ， IP"),
					"bound_domains": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": " IP （ip ）",
					},
					"open_ports": map[string]any{
						"type":        "array",
						"description": "（ip ）",
						"items": obj(map[string]any{
							"port":    intp(""),
							"service": str("， http/ssh/mysql （）"),
						}, "port"),
					},
					// app
					"app_name":    str("（app ）"),
					"bundle_id":   str("Bundle ID（app ）"),
					"category":    str("（）"),
					"description": str("（）"),
					"app_icp":     str(" ICP （）"),
					"company_id":  intp(" id（app ；app  scope ，。id  add_company_scope ）"),
					// service (http)
					"url":         str(" URL，（HTTP ；service_type  http）"),
					"status_code": intp("HTTP ， 200/301/403/404（）"),
					"content_length": map[string]any{
						"type":        "integer",
						"description": "HTTP （）",
					},
					"page_title":   str(" <title> （）"),
					"favicon_mmh3": str("favicon MMH3 （）"),
					"technologies": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "/， [\"Nginx\",\"Vue\",\"Bootstrap\"]（）",
					},
					"auth": map[string]any{
						"type":        "array",
						"description": "， type/username/password （，）",
						"items":       map[string]any{"type": "object"},
					},
					// service (other， HTTP)
					"service_name": str("， ssh/mysql/redis（service  HTTP ）"),
					"port":         intp("（service  HTTP ）"),
					// endpoint
					"method": str("HTTP ：GET/POST/PUT/PATCH/DELETE （endpoint ）"),
					"params": map[string]any{
						"type":        "array",
						"description": "， location(query/body/header/path)/name/value/type（，）",
						"items":       map[string]any{"type": "object"},
					},
				}, "type"),
			},
		}, "assets"),
		func(_ context.Context, in json.RawMessage) (actool.Result, error) {
			if t.as == nil {
				return actool.Errorf("insert_assets : AssetStore "), nil
			}
			var a struct {
				Assets []assetInputItem `json:"assets"`
			}
			if err := json.Unmarshal(in, &a); err != nil {
				return actool.Errorf("invalid input: " + err.Error()), nil
			}
			// task_id (worker: SetTaskID)，——/
			// 。(auto/pentest/chat) t.taskID=0。
			taskID := t.taskID

			type result struct {
				Index int    `json:"index"`
				ID    int64  `json:"id"`
				Type  string `json:"type"`
			}
			type errEntry struct {
				Index int    `json:"index"`
				Error string `json:"error"`
			}

			var results []result
			var errs []errEntry

			blockRules, _ := t.as.ListAssetInterceptRules()
			var allowRules []db.AssetInterceptRule
			if t.taskID > 0 {
				if tb, ta, err := t.as.TaskInterceptRulesSplit(t.taskID); err == nil {
					blockRules = append(blockRules, tb...)
					allowRules = ta
				}
			}

			for i, item := range a.Assets {
				domains, ips, urls := assetInterceptCandidates(item)
				if d := db.EvaluateAssetGate(blockRules, allowRules, domains, ips, urls); !d.Allowed {
					errs = append(errs, errEntry{
						Index: i,
						Error: fmt.Sprintf(" %s %s，", assetInputLabel(item), d.Reason),
					})
					continue
				}

				typ := strings.TrimSpace(item.Type)
				var id int64
				var err error

				switch typ {
				case "root_domain":
					id, err = t.as.UpsertRootDomain(db.UpsertRootDomainReq{
						Domain: item.Domain,
						ICP:    item.ICP,
						TaskID: taskID,
					})

				case "ip":
					id, err = t.as.UpsertIP(db.UpsertIPReq{
						IP:           item.IP,
						BoundDomains: item.BoundDomains,
						OpenPorts:    item.OpenPorts,
						TaskID:       taskID,
					})

				case "subdomain":
					id, err = t.as.UpsertSubdomain(db.UpsertSubdomainReq{
						Domain:      item.Domain,
						RecordType:  item.RecordType,
						RecordValue: item.RecordValue,
						ICP:         item.ICP,
						TaskID:      taskID,
					})

				case "app":
					id, err = t.as.UpsertApp(db.UpsertAppReq{
						Name:        item.AppName,
						BundleID:    item.BundleID,
						Category:    item.Category,
						Description: item.Description,
						ICP:         item.AppICP,
						CompanyID:   item.CompanyID,
						TaskID:      taskID,
					})

				case "service":
					// distinguish HTTP vs other by presence of url
					if item.URL != "" {
						// agent may send "ip" or "service_ip" for the enrichment IP; accept both
						svcIP := item.ServiceIP
						if svcIP == "" {
							svcIP = item.IP
						}
						id, err = t.as.UpsertHTTPService(db.UpsertHTTPServiceReq{
							URL:           item.URL,
							Technologies:  item.Technologies,
							StatusCode:    item.StatusCode,
							ContentLength: item.ContentLength,
							PageTitle:     item.PageTitle,
							FaviconMMH3:   item.FaviconMMH3,
							Auth:          item.Auth,
							IP:            svcIP,
							TaskID:        taskID,
						})
					} else {
						id, err = t.as.UpsertOtherService(db.UpsertOtherServiceReq{
							Domain:      item.Domain,
							IP:          item.IP,
							Port:        item.Port,
							ServiceName: item.ServiceName,
							Auth:        item.Auth,
							TaskID:      taskID,
						})
					}

				case "endpoint":
					id, err = t.as.UpsertEndpoint(db.UpsertEndpointReq{
						URL:    item.URL,
						Method: item.Method,
						Params: item.Params,
						IP:     item.ServiceIP,
						TaskID: taskID,
					})

				default:
					errs = append(errs, errEntry{Index: i, Error: "unknown type: " + typ})
					continue
				}

				if err != nil {
					errs = append(errs, errEntry{Index: i, Error: err.Error()})
					continue
				}
				results = append(results, result{Index: i, ID: id, Type: typ})
				t.writes.Assets++
				t.anchorOwner(id)
				if taskID > 0 {
					var sourceNodeID *int64
					if t.ownerNode > 0 {
						nodeID := t.ownerNode
						sourceNodeID = &nodeID
					}
					summary := "Agent  insert_assets "
					if t.ownerNode > 0 {
						summary = fmt.Sprintf("Worker  #%d  insert_assets ", t.ownerNode)
					}
					_ = t.as.SetTaskAssetSource(taskID, id, "agent", summary, sourceNodeID)
				}
				// (source='auto')： worker ，
				// ；side-effect ，。taskID=0 。
				// ：task_scope (list/)，
				{
					svcIP := item.ServiceIP
					if svcIP == "" {
						svcIP = item.IP
					}
					_ = t.as.AddAutoScope(taskID, typ, item.Domain, item.URL, svcIP)
				}
			}

			return jsonResult(map[string]any{
				"results": results,
				"errors":  errs,
			})
		},
	)
}

// addCompanyScope writes to company_scope table and triggers asset attribution.
func (t *ToolSet) addCompanyScope() actool.CoreTool {
	return writeTool(
		"add_company_scope",
		"/IP/CIDR/ICP/【】——、ICP，Agent。\n"+
			"：company ，()。\n"+
			"scope ，： / URL /  IP / CIDR  / ICP / 。\n"+
			" reason (whois//ASN )。\n"+
			"： TLD (IPv4/16-/32、IPv6/32-/128)， errors 。",
		obj(map[string]any{
			"company": str("(、；)"),
			"scope":   str("，： / URL / IP / CIDR / ICP / "),
			"reason":  str("(/)，"),
			"logo":    str(" URL(；)"),
		}, "company", "scope"),
		func(_ context.Context, in json.RawMessage) (actool.Result, error) {
			if t.cs == nil {
				return actool.Errorf("add_company_scope : CompanyStore "), nil
			}
			var a struct {
				Company string `json:"company"`
				Scope   string `json:"scope"`
				Reason  string `json:"reason"`
				Logo    string `json:"logo"`
			}
			if err := json.Unmarshal(in, &a); err != nil {
				return actool.Errorf(err.Error()), nil
			}
			if strings.TrimSpace(a.Company) == "" {
				return actool.Errorf("company "), nil
			}
			companyID, _, err := t.cs.UpsertCompany(a.Company, a.Logo)
			if err != nil {
				return actool.Errorf("/: " + err.Error()), nil
			}
			lines := splitLines(a.Scope)
			added, skipped, invalid, errMsgs := t.cs.AddScope(companyID, lines, a.Reason)
			out := map[string]any{
				"company_id": companyID,
				"added":      added,
				"skipped":    skipped,
				"invalid":    invalid,
			}
			if len(errMsgs) > 0 {
				out["errors"] = errMsgs
			}
			return jsonResult(out)
		},
	)
}

// addTaskScope lets the plan agent add test scope to THE CURRENT TASK — the coverage
// denominator and the task's authorization edge. Worker discoveries are auto-scoped
// (precise host) by insertAssets; this tool is for DELIBERATELY WIDENING: pull a whole
// root domain or whole company into scope, or add a specific subdomain / ip.
func (t *ToolSet) addTaskScope() actool.CoreTool {
	return writeTool(
		"add_task_scope",
		"【】——，。\n"+
			"kind ：company() / root_domain(，) / subdomain() / ip / cidr / icp / keyword。\n"+
			"：worker 【】()；【】——/，/IP。\n"+
			"value：company  id()；root_domain/subdomain ；ip/cidr  IP ；icp/keyword 。\n"+
			" reason ()。 entries 。",
		obj(map[string]any{
			"entries": map[string]any{"type": "array", "description": "：[{kind, value}]。kind∈company/root_domain/subdomain/ip/cidr/icp/keyword。", "items": map[string]any{"type": "object"}},
			"kind":    str("[] company / root_domain / subdomain / ip / cidr / icp / keyword"),
			"value":   str("[] id /  / IP / CIDR / ICP / "),
			"reason":  str("()，"),
		}),
		func(_ context.Context, in json.RawMessage) (actool.Result, error) {
			if t.as == nil {
				return actool.Errorf("add_task_scope : AssetStore "), nil
			}
			if t.taskID <= 0 {
				return actool.Errorf("add_task_scope ( task)"), nil
			}
			type scopeEntry struct {
				Kind  string `json:"kind"`
				Value string `json:"value"`
			}
			var a struct {
				Entries    []scopeEntry `json:"entries"`
				scopeEntry              // 
				Reason     string       `json:"reason"`
			}
			_ = json.Unmarshal(in, &a)
			items := a.Entries
			if len(items) == 0 {
				items = []scopeEntry{a.scopeEntry}
			}
			var added []map[string]any
			errs := map[string]string{}
			for i, e := range items {
				ts, err := t.as.AddAgentScope(t.taskID, strings.TrimSpace(e.Kind), e.Value, a.Reason, "agent")
				if err != nil {
					errs[strconv.Itoa(i)] = err.Error()
					continue
				}
				added = append(added, map[string]any{"kind": ts.Kind, "domain": ts.Domain, "net": ts.Net, "value": ts.Value, "company_id": ts.CompanyID})
			}
			out := map[string]any{"added": added}
			if len(errs) > 0 {
				out["errors"] = errs
			}
			return jsonResult(out)
		},
	)
}

// listUntestedAssets lets the plan agent pull the current + directly inherited
// scope's not-yet-tested assets on demand (filter by type, paginated).
func (t *ToolSet) listUntestedAssets() actool.CoreTool {
	return readTool(
		"list_untested_assets",
		"【】、（，，）。\n"+
			"：root_domain/subdomain/service/app/endpoint/ip。\n"+
			"：page  1 、page_size  10。 {assets:[{id,type,label}], total, page, page_size}。。",
		obj(map[string]any{
			"type":      str("（）：root_domain/subdomain/service/app/endpoint/ip"),
			"page":      intp("， 1 （ 1）"),
			"page_size": intp("（ 10）"),
		}),
		func(_ context.Context, in json.RawMessage) (actool.Result, error) {
			if t.as == nil {
				return actool.Errorf("list_untested_assets : AssetStore "), nil
			}
			if t.taskID <= 0 || t.ts == nil {
				return actool.Errorf("list_untested_assets "), nil
			}
			var a struct {
				Type     string `json:"type"`
				Page     int    `json:"page"`
				PageSize int    `json:"page_size"`
			}
			_ = json.Unmarshal(in, &a)
			if a.Page <= 0 {
				a.Page = 1
			}
			if a.PageSize <= 0 {
				a.PageSize = 10
			}
			offset := (a.Page - 1) * a.PageSize
			assets, total, err := t.as.ListUntestedAssetsWithSources(t.taskID, strings.TrimSpace(a.Type), a.PageSize, offset)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			return jsonResult(map[string]any{
				"assets": assets, "total": total, "page": a.Page, "page_size": a.PageSize,
			})
		},
	)
}

// listAssets lets an agent query the asset table.
func (t *ToolSet) listAssets() actool.CoreTool {
	return readTool(
		"list_assets",
		"：DSL ， id/ids ；。【】。\n"+
			"DSL：field=value (ILIKE) | field==value  | field!=value  |  > >= < <= | =；AND/OR (AND )，。 type ， DSL。\n"+
			" id/ids  dsl （）。\n"+
			"：domain(//)、root_domain、ip、url、page_title、icp、service_name、app_name、method( GET/POST)、service_type(http|other)、record_type( A/CNAME)、technology(，= ==)、port/status_code/company_id()。\n"+
			"：status_code>=400 AND technology=shiro ；(port==80 OR port==443) AND technology=nginx",
		obj(map[string]any{
			"dsl":    str(`DSL （/）。 id/ids 。`),
			"type":   str("：root_domain|ip|subdomain|app|service|endpoint（， dsl ； type ， dsl）"),
			"id":     intp(" id （， dsl/type ）"),
			"ids":    map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": " id （， dsl/type ）"},
			"limit":  intp("， 10（）"),
			"offset": intp("， 0（）"),
		}),
		func(_ context.Context, in json.RawMessage) (actool.Result, error) {
			if t.as == nil {
				return actool.Errorf("list_assets : AssetStore "), nil
			}
			var a struct {
				DSL    string  `json:"dsl"`
				Type   string  `json:"type"`
				ID     int64   `json:"id"`
				IDs    []int64 `json:"ids"`
				Limit  int     `json:"limit"`
				Offset int     `json:"offset"`
			}
			_ = json.Unmarshal(in, &a)
			if a.Limit <= 0 {
				a.Limit = 10
			}

			var assets []*db.Asset
			var err error
			switch {
			case a.ID > 0:
				assets, err = t.as.GetByIDsInScope(t.taskID, []int64{a.ID})
			case len(a.IDs) > 0:
				assets, err = t.as.GetByIDsInScope(t.taskID, a.IDs)
			case a.DSL != "":
				assets, err = t.as.QueryDSLInScope(a.DSL, a.Type, t.taskID, a.Limit, a.Offset)
			default:
				return actool.Errorf(" id/ids  dsl ：，"), nil
			}
			if err != nil {
				return actool.Errorf("DSL : " + err.Error()), nil
			}
			return jsonResult(map[string]any{
				"count":  len(assets),
				"assets": assets,
			})
		},
	)
}

// listCompanies lets an agent enumerate companies () with their scope + asset count.
func (t *ToolSet) listCompanies() actool.CoreTool {
	return readTool(
		"list_companies",
		"【/】(scope)。、"+
			" company_id（insert_assets  app、list_assets  company_id ）。"+
			" search ()，。",
		obj(map[string]any{
			"search": str("(，)；"),
		}),
		func(_ context.Context, in json.RawMessage) (actool.Result, error) {
			if t.cs == nil {
				return actool.Errorf("list_companies : CompanyStore "), nil
			}
			var a struct {
				Search string `json:"search"`
			}
			_ = json.Unmarshal(in, &a)
			cos, err := t.cs.ListCompanies()
			if err != nil {
				return actool.Errorf(": " + err.Error()), nil
			}
			q := strings.ToLower(strings.TrimSpace(a.Search))
			type companyOut struct {
				ID         int64    `json:"id"`
				Name       string   `json:"name"`
				AssetCount int      `json:"asset_count"`
				Scope      []string `json:"scope"`
			}
			out := make([]companyOut, 0, len(cos))
			for _, c := range cos {
				if q != "" && !strings.Contains(strings.ToLower(c.Name), q) {
					continue
				}
				scope := make([]string, 0, len(c.Scope))
				for _, r := range c.Scope {
					scope = append(scope, r.Raw)
				}
				out = append(out, companyOut{ID: c.ID, Name: c.Name, AssetCount: c.AssetCount, Scope: scope})
			}
			return jsonResult(map[string]any{"count": len(out), "companies": out})
		},
	)
}

// splitLines splits a multi-line string into non-empty trimmed lines.
func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// WorkerTools returns the tool set for a work agent.
func (t *ToolSet) WorkerTools() []actool.CoreTool {
	return []actool.CoreTool{
		// list_findings ：，。
		t.listFindings(),
		t.addFinding(), t.recordFact(),
		// asset management (handlers guard nil store internally)。
		// add_company_scope  worker：//Auto ，worker 。
		t.insertAssets(), t.listAssets(),
		//  work ：worker  work ，。
		// search_all_worker_traces： intent_id，；
		// get_worker_trace： work //。
		t.searchAllWorkerTraces(), t.getWorkerTrace(),
		// node_detail：worker  intent_id/ id （）。
		t.nodeDetail(),
		// 【】worker， planner/main（、 work ，
		// worker ）：list_facts / list_companies / list_worker_traces。
	}
}

// MainAgentTools returns the human-interface tool set.
func (t *ToolSet) MainAgentTools() []actool.CoreTool {
	return []actool.CoreTool{
		t.graphOverview(), t.listFindings(), t.listFacts(), t.nodeDetail(),
		t.expandDigest(), // cold-digest §6.1
		t.getWorkerOutput(), t.getWorkerTrace(), t.searchAllWorkerTraces(), t.addHint(), t.addIntent(),
		// steer_work：(work)（、）。
		t.steerWorkTool(),
		t.setGoals(),
		// set_constraints：/（allow/deny）， planner/worker 。
		t.setConstraints(),
		// asset management (handlers guard nil store internally)
		t.insertAssets(), t.addCompanyScope(), t.listAssets(),
		t.addFinding(), t.recordFact(),
		t.addTaskScope(),
		// list_untested_assets：(+)，。
		t.listUntestedAssets(),
	}
}

// AllDomainTools returns the union of all domain tools across all agent types,
// deduped by name (mainagent order wins). Used by the server to build a registry
// for injecting domain tools into agents (Auto, custom) that don't own a per-task
// ToolSet. The caller provides real stores; tools are callable at taskID=0 scope.
func (t *ToolSet) AllDomainTools() []actool.CoreTool {
	seen := map[string]bool{}
	var out []actool.CoreTool
	all := append(append(t.MainAgentTools(), t.PlannerTools()...), t.WorkerTools()...)
	for _, tool := range all {
		if !seen[tool.Name()] {
			seen[tool.Name()] = true
			out = append(out, tool)
		}
	}
	return out
}
