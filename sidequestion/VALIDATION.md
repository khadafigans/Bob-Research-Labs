
 · [](VALIDATION.zh.md)

: 2026-09-10. : `codex/btw-side-question`.  (baseline): `8dae851b9b622f2ff2631f332fde9719d0b16fba`.

>     (`VALIDATION.zh.md`)   . (upstream)       .

 PostgreSQL  DB    .             ,     . Go 1.26.3, norma v0.3.6, Next.js 16.2.9.

  ,  ,  , Qwen    [validation-2026-09-10.json](validation-2026-09-10.json)  ,   API   .


   ,    .

-      : ( `TestCheckpointDeepCopyAndBoundaries`).
- ·   ,     ,     : ( `TestCheckpointDeepCopyAndBoundaries`, `TestSnapshotExcludesPartialStreamAndSelectsPoolMember`).
-     : ( `TestSnapshotExcludesPartialStreamAndSelectsPoolMember`).
-   , 20  ,    : ( `TestBuildRequestCompactionToolPairingAndBudget`).
-    ,   : (:   Provider, `TestMainSideConcurrencyAndIndependentCancellation`).
-   , ·,    : ( `TestServiceNoToolsAndUsageOnFailure`).
-  norma ChatAgent   Read ,  transcript· : ( `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`  ·  ).
- ,  , ,     : ( `TestSideHistoryIdempotencyPagingAndRecovery`).
-    ,   ,  : ( `TestSideClearLateWritersAndDeletedParent`).
- MainAgent·Worker  (v1·v2·v3): ( `TestSideTaskArchiveVersions`).
-    , ,  , Worker  : ( `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`, `TestSideCheckpointPersistsBeforeAdmissionAndRestart`).
-     ,  SSE ·, , : ( `TestSideHTTPBusyIsolationClearAndReconnect`).
-   1 · 4   : (:   `TestSideHTTP…` ).
-    ,    ,     : ( `TestSideCheckpointPersistsBeforeAdmissionAndRestart`).
-         : ( `TestSideRejectsDeletedOrChangedCachedProfile`).
-        : ( `TestSideTaskDrainPersistsBeforeArchive`).
-          : ( `TestSideUsageRecordedOnceOnConsumerCancellation`).
-    Worker·deadline      : ( `TestSideRestoredWorkerRuntimePublishesNewCheckpoint`).
-   race : (:  ).
- TypeScript   : ( `npx tsc --noEmit`, `npm run build`).
-     Biome : ( `biome check`,   3 ).

    `ARTEX_PG_DSN`       ( DB   ):

```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```

 Go    (green) . `server`          ,   `TempDir RemoveAll … directory not empty`  :

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

          `server`         .    `TestCoreTaskLifecyclePG`        ,   `server`     .   ,     race  .        ,       .

Next.js   lockfile·workspace      ,      .


Codex     Go  Next.js   .  390 × 844    ·      :

-     `/btw`       ,   .
-    .      ,   .
-    ,     .       `/btw`    .
-   Drawer  , , ,      .
-    ,      transcript    .
-  MainAgent   Worker     ,      .
-      Worker    , Worker    `/btw`  .    Worker         ,    .
-  ·   .

         ,      .     Worker        (     ),     .       .


 `grok-4.6`  . OpenAI   `http://127.0.0.1:12580/tingly/openai` .   HTTP 200     `grok-4.6`  `READY`  , 2.82  . 1   Tingly  `glm`  Zhipu  `glm-5.3`    ,       .

-     ·· : `redhaze.top`,     , `BTW-REAL-0910`    (16.97 ).
-          : WebFetch 200, curl  301 → 302 → 200 ,    (7.24 ).
-  Bash     :      (7.74 ).
-       :  transcript  SHA-256      ,     0.
- Go   ·   :    3  ,    ··      .
-   Grok   :  `ATOMIC-0910`   ,  ·(input 11734, output 138, cache_read 11520).

    WebFetch  Bash/curl     ,   `https://id.redhaze.top/home`,    " RedHaze Group · " (    ). Bash            .   "   "   .

 transcript : `e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`.

** :** Tingly  Grok   usage   .  `stream_options.include_usage=true`      HTTP 200,   12 , usage  0 .    0      ,      .    ·    .


  `qwen-flash`, OpenAI   `https://dashscope.aliyuncs.com/compatible-mode/v1`, HTTP 200 .      ,    ,   , `verdict: accept`  `concerns: []`  .  ··         .  : prompt 6625, completion 312, total 6937.

 Qwen          . Qwen  " "    .   curl       ,    . ,   0 , transcript    ,       .
