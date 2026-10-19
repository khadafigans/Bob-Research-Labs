# ARTEX `/btw`

、 MainAgent  Worker 。 `/btw ` ， `/btw` “”。， Drawer。

 Agent ，、、。、 SSE 。；，。


 Go、norma v0.3.7、Next.js、 Markdown / ResizablePanel / Drawer / AlertDialog ； norma 。Planner、 Worker、。

```mermaid
flowchart LR
    A[ Agent QueryDeps] --> B[ Provider ]
    B --> C[]
    B --> D[ Agent ]
    C --> E[(PostgreSQL )]
    E --> F[ +  + ]
    F --> G[SideQuestionService  Provider ]
    G --> H[()]
    H --> I[ SSE / ]
```

- `capture.go`  `Options.Deps.CallModel / CallModelSync` 。Provider 、，；。
- 、、。； norma  `MessagesForAPI` ，。。
-  JSON 、、。。
- `SideQuestionService`  Provider；，/。 agent session，、 transcript、，。；。。
- ，， 120 。。
- ；。，、、， Agent 。。


`db/schema.sql`  `side_question_sessions`  `side_question_requests`。、、、；、、、、、、。

 conversation ID， task ID + exploration ID + intent ID。Worker 。

， 250 ms ， `(run_id, version)` 。。；。 250 ms ，。

 `running`  `interrupted`，，。。 Agent， UI 。

；。，Worker ，。、、； v3， v1/v2。

， 20 。 20 ， token ；。，。、、 120 。 [](CONTEXT_BUDGET.md)。


 `{parent}`，：

- `/api/conversations/{id}`
- `/api/tasks/{id}/chat`
- `/api/tasks/{id}/intents/{iid}`

|  |  |
| --- | --- |
| `GET {parent}/side-questions?before={ordinal}` | `items` 、 `current` 、`snapshot` 、`next_cursor`； 0  /  |
| `POST {parent}/side-questions` | JSON `{ "question": "…", "client_request_id": "UUID" }`； 202 ； ID  200 |
| `DELETE {parent}/side-questions` |  |
| `GET /api/side-questions/{requestID}/events` | `snapshot` SSE ，`id` ，`data` ； `cleared` |
| `POST /api/side-questions/{requestID}/cancel` | ； SSE  |

 4000 。、、 ID  409； 429。 SSE ，。 ID + ，、。


、 [VALIDATION.md](VALIDATION.md)。

 [Grok CLI side-question.ts（）](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/utils/side-question.ts)， [OpenCode（）](https://github.com/anomalyco/opencode/tree/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b)。ARTEX  norma ，。
