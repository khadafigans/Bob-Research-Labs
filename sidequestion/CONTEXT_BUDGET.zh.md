
2026-09-11 。 JSON  token， 32K ， HTML、JS 。


- [Grok CLI ](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/agent.ts#L739)：、， 2000， 400 。。
- [Grok CLI ](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/utils/side-question.ts)：； 2048 tokens；。
- [Grok CLI ](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/compaction.ts)： token、、，。
- [OpenCode ](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/compaction.ts)：、/、、； 8000、 4096 tokens。
- [OpenCode ](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/runner/llm.ts)：，。

ARTEX 、、。 norma v0.3.6 ， Grok ； OpenCode  ARTEX  transcript。


-  norma  UTF-8  4/3 ；、 schema 。 token 。
-  8192 tokens，。 `ARTEX_BTW_MAX_OUTPUT_TOKENS`  256–32768 ；。
- ； 200K。 5%， 128、 8192 tokens。
-  20 。 20 ， token  1/4， 16K。
- 。；，。
- ，， 8K tokens；。。
-  UTF-8 ， 2048 tokens；、、。 12 ， 120 ；，。
- ，；，。。
- ，、，。Provider ，。


`side_question_sessions.memory` 、 ordinal，。`side_question_requests.context_info` 、、。

；，。。 v3 ； v3 ，v1/v2 。

POST ，，。SSE/、、、；。，；。


- 19/20/21/50 、 20 、：。
- 、、、、：。
- ////、、、：。
-  PostgreSQL 、、v1/v2/v3 、、 v3 、20 ：。
- Go 、 TypeScript 、 Biome ， Next.js ：。
-  UI ， 1280×720  390×844 、、、、。。
-  Worker ，； Worker #3  293085 。。
-  Worker  Grok ，，。
- 2026-09-11 00:37 ，、。 SHA-256 ； `/api/health` 。

（）：

```sh
go test -race ./sidequestion ./db ./server -run 'TestSide|TestCheckpoint|TestSnapshot|TestBuildRequest|TestService|TestMainSide|TestTaskArchive' -count=1
go build ./cmd/artex
npx tsc --noEmit
npm run build -- --webpack
```

， `.next`。 `/private/tmp/artex-btw-budget-candidate`， `/private/tmp/artex-btw-preview/artex` ； `artex.before-context-budget`。
