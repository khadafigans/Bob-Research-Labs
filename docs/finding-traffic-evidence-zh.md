
「」、、。，。，。

「Agent 」， `/api/settings`  `agent_traffic_binding`。/、 Token ； Agent 。，，。、、。

，「 →  Agent →  → 」。 `evidence` 、、 ID ； Agent ， `traffic_search` / `traffic_get`  `bind_finding_traffic`， `version` 。， Agent /，；。

：`report_finding`  `traffic_refs` / `evidence_hint_id` 。：TCP  HTTP 、，；、，，。 ID 。；。，。


`report_finding` ，：

```json
{
  "traffic_refs": [
    {"traffic_id": "ID", "role": "baseline", "note": ""},
    {"traffic_id": "ID", "role": "proof", "note": ""}
  ]
}
```

 `baseline`（）、`proof`（）、`verification`（）、`supporting`（，）。 `traffic_search` / `traffic_get` ；，。

 [CyberStrikeAI ](https://github.com/RuoJi6/CyberStrikeAI/blob/54d56774b8bd285817d16d48b70a4a5e6e0963f7/internal/app/vulnerability_tools.go)，，。 ID，。

 `finding recorded: < ID>`； JSON  `finding_id`、 `finding_node_id` 。

- `get_finding_traffic(finding_id)` ** ID**， `version`； `binding_id`、`side=request|response`、`offset`、`length` ， 8192 。
- `update_finding_report`  `finding_id` ** ID**。 `evidence_version` ；，。
- ，。、、，。

，`add_hint` / `add_task_hint`  `hints`  `traffic_refs`。Planner  `evidence_hint_id`，，。、。。

 `bind_finding_traffic(finding_id, traffic_refs)` ，。`list_findings` / `list_task_findings` / `node_detail` / `get_task_node_detail`  `finding_id`  `finding_node_id`； `id` 。

 schema ， Agent； Agent。、、。：， Agent 、。 Agent，。；。 `report_finding`  Agent。

## API

：`/api/exploration/findings/{finding_id}/traffic`， ID。；`context_task` 。

|  /  |  /  |
| --- | --- |
| `GET` | 、、 |
| `POST` | `{"traffic_refs":[...]}`  |
| `PATCH /{binding_id}` | `{"version":1,"role":"proof","note":""}` |
| `DELETE /{binding_id}` | `{"version":1}` |
| `PUT /order` | `{"version":1,"binding_ids":["2","1"]}`， |
| `GET /{binding_id}` |  |
| `GET /{binding_id}/body` | `side`、`offset`、`length`；`download=1`  |

/、 `409`； `403`； `404`；/。


 PostgreSQL： `traffic_evidence_snapshots`、`finding_traffic_bindings`， `findings.evidence_version` / `report_evidence_version`（ 0）。。

 ID、、URL、、、/、 SHA-256。 `<data>/evidence/blobs/<>/<hash>.bin`， `data/traffic` ，/。，。

， blob 。， PostgreSQL 、、、；。，。

PostgreSQL advisory lock `7337741004`  SQL ；。。。

，， 24 。。 PostgreSQL  `data/evidence`。


Markdown ；JSON ；CSV  ID。`md-zip`  Markdown，：

```text
evidence/<finding_id>/<binding_id>/
  manifest.json
  request.http
  response.http
  request.bin
  response.bin
```

Markdown 。、、、 ZIP  CRC ；/。。

 v3 ，。；。，，。v1/v2 ， 0。


、 PostgreSQL ， `ARTEX_PG_DSN` ，/。，：

```sh
go test ./<package> -count=1
go test -race -p 1 ./evidence ./db ./agent ./server -run 'TestEvidence|TestFindingTraffic|TestFindingEvidence|TestReportFindingAtomicContract|TestTaskArchive'
```

 `npx tsc --noEmit`、 Biome 、Webpack  `NEXT_EXPORT=1` 。，。

、 HTTP /  HTTPS ，、、/、、、，、、、。

；/，。。， HTTPS  IP 。
