---
name: api-recon
description: APIskill。
---


****，：** API**（、、、）、****、**UI **（Tab、、）。

---


 skill ** API / **，。


|  |  |  |
|---|---|---|
| **** |  path、method、、、UI  | SQLi/XSS///fuzz 、、 |
| **** | Hook + stub/mock **** | ； |
| **** |  hook ， mock  SPA  |  |


1.  `preload.js` / `runtime_harvest.js` ** stub** 、、 bootstrap ；
2.  **、、**  mock body；
3.  401 ， XHR/fetch/WebSocket；
4. **、、 UI **——。

****： mock ，** outbound **；，****。


|  |  |
|---|---|
| Phase 1  grep/curl/Read  entry `index-*.js`  API path |  `OUTDIR/harvest_static.py` |
|  `extract_apis.py`  harvest  |  `OUTDIR/harvest_static.py`  |
|  grep/ ≥2  | ： tool_logs、 harvest、 reference |
|  A/B， `scripts/`  |  OUTDIR  |
| /、OTP、OAuth  | stub/mock（） |
| 「」 stub，/ |  outbound， recon  |
| 、、 | coverage  |
|  runtime + ， | 「」 |
|  + diff， | Phase 3b  + Phase 5 diff |
|  runtime / |  diff / |

---


|  |  |  |
|---|---|---|
| ****（JS bundle） |  endpoint 、、 |  HTTP ； Phase 1b； URL |
| ****（） |  + body +  +  URL + WS/SSE； diff  | ；/ |

|  |  |  |
|---|---|---|
| **depth** | `runtime_harvest.js`（Puppeteer） | API 、METHOD/params/、WS/SSE、 |
| **coverage** | browser + `preload.js` |  Tab//， |
| **both** |  depth  coverage | ， |

****（）：path  harvest/； ** + UI  +  diff + **（grep  [reference.md](reference.md) J ）。

---


 recon ：

- [ ] ****：Phase 1 harvest  `api_static.txt`、`routes.txt`、`js/`
- [ ] ****： depth  coverage ；coverage/both  **Hook  + **
- [ ] ****： path  `/login`（ hash ）
- [ ] ****：coverage/both  + `param_samples.json`；Phase 5  `params_merged.json`
- [ ] ****（）：Phase 4 ， **module  API**（ locale/bootstrap）
- [ ] ****：Phase 5 （ Phase 5 ）；`insert_assets` 

---


`scripts/` ，****。

****： →  →  `OUTDIR`（ `recon/`）→  `CHANGES.md`；，。

|  |  |  → OUTDIR  |  |
|---|---|---|---|
| **A（）** | Phase 0 、**** harvest/spider  | `harvest_static.py` / `spider_mpa.py` | ** regex **； manifest/ endpoint 、webpack/Vite `publicPath`、MPA exclude/cookie |
| **B（）** | Phase 2 、 depth/coverage  | `runtime_harvest.js` / `preload.js` + `config.json` | Cookie/localStorage 、neutralize 、stubs、login 、api 、hash/history |


|  |  |  |
|---|---|---|
| Phase 0  |  Bash = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | curl/grep/Read  entry `index-*.js`（ >500KB） |
|  A |  →  → **** |  API  harvest |
| Phase 1  | `wc -l` ；404  harvest  |  extract ； URL  grep |
| Phase 1b  | grep  `OUTDIR/js/*.js` |  bundle  harvest |

- ✅  `harvest_static.py` → （） regex → ****
- ❌ curl  bundle → grep  →  extract →  harvest
- **MPA**：Phase 0  Bash = `python3 OUTDIR/spider_mpa.py ...`

---


|  |  |
|---|---|
|  | >100KB  `index-*.js` **** Read/grep ； OUTDIR  |
| grep  |  `\| head -20`  `-m 5`； path ， bundle  |
|  |  `wc -l`、`ls \| wc -l`； Read  |
| regex  | 、≤1 、 ≤50KB  chunk  HTML； harvest  |
| reference | // [reference.md](reference.md)， inline  |

---


```
Phase 0  + OUTDIR
  →  A → Phase 1 harvest（★  ★）
  → Phase 1b 
  → Phase 2  → config.json
  →  B → Phase 3  + 
  → Phase 4 （）→  Phase 3
  → Phase 5  + insert_assets、api，
```

；** Phase**。

1. [ ] **Phase 0**： SPA/MPA； `OUTDIR` → [Phase 0](#phase-0--)
2. [ ] ** A + Phase 1**： → **** harvest → `wc -l`  → [Phase 1](#phase-1--)
3. [ ] **Phase 1b**： +  → `param_candidates.json` → [Phase 1b](#phase-1b--)
4. [ ] **Phase 2**： → `config.json` → [Phase 2](#phase-2--)
5. [ ] ** B**： runtime  → [Phase 3](#phase-3--)
6. [ ] **Phase 3**：depth / coverage / both；； → `param_samples.json`
7. [ ] **Phase 4**（）： → patch stubs →  Phase 3 → [Phase 4](#phase-4--)
8. [ ] **Phase 5**： +  + `insert_assets` → [Phase 5](#phase-5--)

---


 HTML，** `OUTDIR`**（ skill  `scripts/`）：

- **SPA**： + `<div id=app>` + chunk → Phase 1–5
- **MPA**：SSR + `<form>`、 endpoint bundle →  A ：

```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```

 `forms.txt`、`links.txt`、`api_inline.txt`。SPA  forms ≈ 0 →  Phase 1。

---


 [](#) · [](#)。

```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```

harvest： HTML script → webpack/Vite manifest →  lazy chunk →  `js/`、`api_static.txt`、`routes.txt`、`chunkmap.txt`。

```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```

- chunk  vs manifest：404  harvest ， curl  chunk
- `api_static.txt`  →  OUTDIR  endpoint （ reference）


path  Phase 1； recon。grep  [](#)。



|  |  |  |
|---|---|---|
| REST JSON | body + query | path  `(params\|data\|body)\s*:\s*\{` |
| GraphQL | `variables` | gql 、`$page: Int` |
|  form | urlencoded | `<form>`、`FormData` |
|  | multipart | `FormData.append` |
|  | `/user/:id` |  + `useParams` / `$route.params` |
| / |  `sign`/`data` | Hook （reference D ） |

： `transport: query|json|form|graphql|encrypted`。


 path ，：

```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```

|  |  |
|---|---|
| axios  | `data` / `params` |
|  request |  |
| OpenAPI  |  method  |
| React Query / SWR | hook  |
| Vue composable | composable  |

：`yup`/`zod`/rules、`Form.Item name=`、 Swagger。

→ `param_candidates.json`：`{ path, fields[], source: "static-callsite", confidence }`


```
Form field → onFinish/handleSubmit → transform → API payload
```

|  |  |
|---|---|
|  submit |  submit → transform → API |
|  | `getFieldsValue()` → `params` |
|  | `:id` / `?tab=` |
|  |  `tenantId`、、sign |
|  select | `options` → API  |

DevTools call stack  `fetch`/`XHR.send` 。


|  |  |
|---|---|
| **** | payload  build、transform  |
| **** | required、pattern、enum |
| **** | path / query / body / multipart /  |

（Phase 2）（Authorization、`X-Tenant-Id`、sign）。


/；**//** Phase 3  + diff + Phase 5 。

---


 `OUTDIR/js/` grep（ `head`）， `config.json`（ reference）：

|  |  |  |
|---|---|---|
| **** | ？ | `isLogin`、`getToken`、Cookie/localStorage |
| **** |  `/login`？ | `response_code`、`errno`、axios interceptor |
| **** | /？ | `menu`、`permission`、`role`、`acl`、`routes` |

 localStorage —— chunk/。

** =  B**： `config.json`， `OUTDIR/runtime_harvest.js` / `preload.js`。


 OUTDIR  `preload.js` 、Authorization、 API URL：

|  |  |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | headers  |
| `extractUrlsFromResponse: true` |  API |
| `observe.storageReads/cookieReads: true` |  config |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

coverage ：`__API_RECON_LOG__`、`__API_RECON_DETAIL__`、`__API_RECON_ROUTES__`、`__API_RECON_OBSERVE__`。

---


 B； [](#agent---) ·  mock 。

`config.json`  `"runtimeMode": "depth" | "coverage" | "both"`（ reference）。

### Hook  stub（depth + coverage ）

|  |  |  |
|---|---|---|
| L1  | auth//bootstrap stub |  |
| L2  |  JSON  |  →  |
| L3  |  L1  `/api`  | ， UI |

- **depth**：fake auth + `forward`  + `stubs`； `routes`（hash/history）； `runtime_api.json`
- **coverage**：**document-start**  `preload.js`（CDP `addScriptToEvaluateOnNewDocument`  Userscript）

：`window.__API_RECON_PRELOAD__` ； path  `/login`。

```bash
cd recon && npm install
node runtime_harvest.js config.json
```


1. / — ， 1–3s
2. Tab — `role=tab`、`.ant-tabs-tab`
3.  — //
4.  — 、、（****）
5.  —  API/
6. SPA —  `routes.txt`  path  `pushState`（MPA ）


|  |  |
|---|---|
|  |  +  |
|  | keyword、filter |
|  |  optional |
| / |  entity |
| // | `ids[]`、`exportType`、`sortField` |

**stub  outbound body/headers **——。 → `scan_raw.json`、`param_samples.json`、`api_detail.json`。

- **Vue**：`neutralizeVueRouter: true` + document-start preload
- **React**：`routes.txt` +  + `pushState`
- **both**： 3a depth， 3b coverage

---


****： /  bootstrap（ locale）→ 。

|  |  |
|---|---|
|  |  +  |
| / | stub shape  |
|  API  | `v-if permission`  |
| `routes.txt`  bundle |  auth  |

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```

：`role_permissions`（flat codes）+ `permissions/all`（tree）→ `getResultTree` → `userRouteAuth[CODE].url`。

```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```

：`route_map.json`、`userRouteAuth.json`、`permissions_tree.json`、`*_stub.json`、`perm_codes_all.txt`。

stub ： `response_code` ；flat codes  tree ；`routes`  `route_map`  link。

 `config.json` ** Phase 3**。 SPA  `waitUntil`、`routeTimeout`、`perRouteMs`（ reference A3/I ）。

---



|  |  |  |
|---|---|---|
| `js/`、`api_static.txt`、`routes.txt`、`chunkmap.txt` | 1 |  bundle  path |
| `param_candidates.json` | 1b |  |
| `config.json` | 2 |  + runtime  |
| `runtime_api.json` | 3a | depth （ WS/SSE） |
| `param_samples.json`、`scan_raw.json`、`api_detail.json` | 3b | 、、detail |
| `route_map.json`  | 4 | （） |
| `params_merged.json` | 5 |  +  |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | 、API、params、、 |
| **insert_assets** | 5 | 、 |


 `param_samples.json` diff，****。 reference J7（///）。


 400（** recon，**）：`field 'x' is required`、。 `data` 、`variables`、 `bizData`。

：runtimeMode、/ API 、、、 `CHANGES.md` 。

`site_map.json` ：

```json
{
  "site": "https://example.com",
  "runtimeMode": "both",
  "appType": "vue-spa",
  "routeGuardStrategy": ["nav-neutralize", "L1-auth", "L2-patch", "forward"],
  "apisFromStatic": [],
  "apisFromRuntime": [],
  "apis": [],
  "params": [{ "method": "POST", "path": "/api/user/list", "transport": "json", "fields": [] }],
  "frontendRoutes": [],
  "routesVerifiedByClick": [],
  "featuresTriggered": [],
  "limitations": ""
}
```

 grep  [reference.md](reference.md)。

---


- ****：webpack/Vite/Angular lazy load 
- ****：REST/JSON、GraphQL、WebSocket、SSE；gRPC-web 
- **SSR**： fetch ；RSC/Server Actions 
- ****：JSVMP、WASM、HMAC/mTLS  →  + 
- ****：、hidden params、WASM  → 「」/「」
- ****：runtime  endpoint

---


- Grep 、`config.json` 、、Hook、 J 、site_map ：**[reference.md](reference.md)**
-  [](#) 
