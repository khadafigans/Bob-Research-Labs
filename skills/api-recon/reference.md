
Grep 、`config.json` 。 grep  `js/` 。bundle  `js-beautify`  `sed 's/}/}\n/g'`， raw grep 。


`scripts/` ****，。：

|  |  |
|---|---|
| `harvest_static.py` | endpoint 、webpack/Vite manifest 、 publicPath、/ |
| `runtime_harvest.js` | neutralize 、stub  body 、routes 、WS 、`waitUntil`/`routeTimeout`/`proxy` |
| `preload.js` | `loginPathRe`、L1 stubs、`neutralize.fields`、`apiPattern`、 L3、`recordDetail`、`observe.*`、`neutralizeVueRouter` |
| `spider_mpa.py` | `--exclude` 、cookie、depth/max、 |
| `extract_route_map.py` | `routeMap` / `routeLink` 、KEY  |
| `build_perm_tree.py` | `userRouteAuth` 、`ROOTS`/`PREFIX_PARENT` 、stub  |
| `config.json` |  |

（ `recon/`），。

---



```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```

 `isLogin = f(getUser())` → `getUser = decode(storage.read(KEY))`， ****、****（Cookie vs localStorage）、****：

|  | config  |
|---|---|
|  / `"1"` / token | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT | /`alg:none` JWT， bundle  |
| （SM2/AES/RSA） | ； blob  forge； |

→  `cookies` / `localStorage`。


```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(|||unauthorized|||token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```

：****、****（ `0`  `200`）、****。 junk session ：

```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```

→  `neutralize.fields` + `neutralize.success`。


```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```


| API |  payload |  |
|---|---|---|
| `.../role_permissions` | `{ permissions: string[], role_type }` | 、 ACL |
| `.../permissions/all` | `tree[{ code, position, children }]` |  |
| bundle  `userRouteAuth` | `{ CODE: { url, name? } }` | code →  path |
| bundle  `routeMap` | `{ KEY: { name, link } }` | （webpack `o.DASHBOARD`） |

：`getResultTree(tree, permissions)` 、`v-if` / `hasAuth(code)` 。

** forge**（）： permissive payload → `stubs`。


---


```json
{
  "baseUrl": "https://target/",
  "runtimeMode": "both",
  "chromium": "/usr/bin/chromium",

  "cookies": [
    { "name": "auth", "value": "b64json:{\"id\":1,\"username\":\"admin\",\"role\":\"admin\",\"func\":{},\"permissions\":[\"*\"]}" }
  ],
  "localStorage": { "token": "faketoken", "isLogin": "1" },

  "neutralize": {
    "fields": ["response_code", "code", "errno", "ret", "status"],
    "success": 0,
    "flags": { "success": true, "message": "ok" }
  },
  "forward": true,
  "loginUrlPattern": "/login",
  "apiPattern": "/api/|/rest/|/graphql",

  "mockTier": "L1+L2",
  "recordDetail": true,
  "observe": {
    "storageReads": false,
    "cookieReads": false,
    "xhrHeaders": true
  },
  "neutralizeVueRouter": true,
  "stubs": [
    {
      "match": "permissions/all|/menu|role_permissions",
      "body": {
        "response_code": 0, "code": 0,
        "data": {
          "permissions": ["*"],
          "menus": [
            { "name": "dashboard", "path": "/dashboard", "show": true, "children": [] },
            { "name": "alert", "path": "/alert", "show": true, "children": [] }
          ]
        }
      }
    }
  ],

  "explore": {
    "clickTabs": true,
    "clickTables": true,
    "pushStateFallback": true,
    "maxMenuItems": 50
  },

  "routes": ["/dashboard", "/alert", "/asset", "/device", "/report", "/config", "/system"],
  "waitMs": 1500, "perRouteMs": 900, "headless": true,
  "waitUntil": "domcontentloaded",
  "routeTimeout": 12000,
  "proxy": "",

  "captureResponses": true, "recordWs": true, "respMax": 600
}
```

：
- `runtimeMode`：`depth`（Puppeteer）、`coverage`（browser MCP）、`both`
- `cookies[].value` ：`b64json:` → base64(JSON)；`json:` →  JSON； → 
- `forward: true` ；`false`  stub
- `mockTier`：coverage  preload ， `L1+L2`、`L1+L2+L3`
- `routes`  `routes.txt`；forge  harness  `<a href>`
- `captureResponses` / `recordWs`  depth 
- `waitUntil`： SPA  `domcontentloaded`， `networkidle2` 
- `routeTimeout`： `page.goto` （）
- `proxy`：Puppeteer `--proxy-server`； `HTTP_PROXY` / `HTTPS_PROXY`

### B1.  stub （role_permissions + permissions/all）

```json
"stubs": [
  {
    "match": "role_permissions",
    "body": {
      "response_code": 0,
      "data": {
        "permissions": ["MONITOR", "MONITOR_ALERT", "THREAT", "ASSETS_RISK"],
        "role_type": "SUPER_ADMIN"
      }
    }
  },
  {
    "match": "permissions/all",
    "body": {
      "response_code": 0,
      "data": [
        {
          "code": "MONITOR",
          "position": 1,
          "children": [
            { "code": "MONITOR_ALERT", "position": 1, "children": [] }
          ]
        }
      ]
    }
  }
]
```

（`response_code` / `code` / `data`） A2 ；`permissions`  tree  leaf code。

---

## C. coverage ：preload 

 `scripts/preload.js`  `CONFIG` ， CDP ：

```javascript
const CONFIG = {
  loginPathRe: /\/(login|signin)(\/|$|\?)/i,
  mockTier: 'L1+L2',
  forward: true,
  recordDetail: true,
  extractUrlsFromResponse: true,
  neutralizeVueRouter: true,
  observe: { storageReads: false, cookieReads: false, xhrHeaders: true },
  neutralize: { fields: ['response_code', 'code'], success: 0 },
  stubs: [ /*  config.json stubs */ ],
  apiPattern: /\/(api|apis|v\d+|dev|internal|graphql)\//i,
};
```

：`window.__API_RECON_PRELOAD__ === true`  pathname 。

：

```javascript
JSON.stringify({
  apis: [...window.__API_RECON_LOG__],
  detail: window.__API_RECON_DETAIL__,
  routes: [...(window.__API_RECON_ROUTES__ || [])],
  observe: window.__API_RECON_OBSERVE__,
}, null, 2)
```

---

## D. preload / runtime Hook 

preload（coverage） runtime_harvest（depth） Hook ：

| Hook  |  API  |  |
|---|---|---|
| Hook fetch / XHR.open |  URL/ | ✅ `recordDetail` + `__API_RECON_LOG__` |
| Hook XHR.setRequestHeader |  Authorization  | ✅ `observe.xhrHeaders` |
| Hook localStorage/cookie  |  | ⚠️  `observe.storageReads/cookieReads` |
| Vue  |  frontendRoutes | ✅ `__API_RECON_ROUTES__`（） |
| Vue  /  |  API | ✅ `neutralizeVueRouter` +  |
| React  |  | ⚠️  + ； Hook |
| （ path） |  | ⚠️  path， |
| Hook （CryptoJS/SM ） |  →  API body | ❌  Hook ； config |
|  bypass |  runtime  API | ❌ ； |

---


 `harvest_static.py`  `extract_endpoints` ，：

```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```

---


|  |  →  |
|---|---|
|  API  | endpoint  → （D ） |
| chunk  ≪ manifest | CSS-only  chunk；404  |
| runtime  |  →  A1：、、、domain |
|  |  → forge （A3）；`routes` path  |
|  bootstrap/locale |  → I ； `role_permissions` + `permissions/all`  stub |
|  | tree  intermediate  code  `userRouteAuth`  |
|  API  |  →  `neutralize`； walk  |
| WS  0 |  subscribe； `perRouteMs` |
|  |  `forward: true`  |
| Chromium  |  chromium  `config.chromium` / `CHROMIUM` |
| Mock  | Hook  `location.href` setter → document-start + preload |
|  | L3 ； Tab// |
|  Redux action  |  get/set/change/clear/toggle/upload  path |
| Vue  | preload  document-start → ； `neutralizeVueRouter: false`  |
|  URL  log |  `extractUrlsFromResponse`； `__API_RECON_DETAIL__`  |
|  Authorization  |  `observe.xhrHeaders`  DevTools  |
| runtime  /  |  `waitUntil: domcontentloaded`； `routeTimeout`； `networkidle2` |
|  |  `proxy` / ；Puppeteer  curl  |

---


（ forge  cookie、 stub ），runtime  shell 。：

- ** endpoint ** —  path 
- ，**** harness：`forward: true`、 neutralize， methods/params/responses

---


1. 
2. **** `scripts/harvest_static.py` →  →  →  `api_static.txt`、`routes.txt`
3. **Phase 1b**：path  +  → `param_candidates.json`（J ）
4.  A1/A2/A3 →  `config.json`
5. **** `runtime_harvest.js` / `preload.js` 
6. `runtimeMode=depth`：`npm install` →  harvest 
7. `runtimeMode=coverage/both`：document-start  preload → browser MCP  + ****
8.  → **I ** → patch stubs → 
9.  diff +  → `params_merged.json`
10.  → `site_map.json` + `api_merged.txt`，、

---


 forge  `menus: [{ path, show: true }]` 、 mount 。


```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```

：** API path**、****、** chunk **。


```bash
python3 scripts/extract_route_map.py recon/js recon/
#  recon/route_map.json
```

 `[!] no routeMap pattern found`： `extract_route_map.py` ， grep：

```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```


```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```

：
1.  `userRouteAuth={MONITOR:{url:...},...}`（ webpack  `He=o.DASHBOARD`）
2.  `route_map.json`  alias →  path
3.  code  parent（`MONITOR_ALERT` → `MONITOR`）
4.  `permissions_tree.json`、`permissions_all_stub.json`、`role_permissions_stub.json`
5. `--config`  `config.json`  `stubs`  `routes`

- `DEFAULT_ROOTS`： code 
- `DEFAULT_PREFIX_PARENT`：`PREFIX_` → parent 
- `DEFAULT_EXTRA_PARENT`： orphan 


```bash
# permissions  ≈ userRouteAuth 
wc -l recon/perm_codes_all.txt
# routes  route_map  link
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```


```bash
node recon/runtime_harvest.js recon/config.json
#  forge  runtime_api.json ； /attack、/asset  API
```

| forge  | forge （） |
|---|---|
|  3–5  bootstrap |  module API |
|  `/api/locale/language` |  `/api/web/...`  endpoint |
| `routes.txt`  | `routes` 80–110+  route_map |


- **coverage **： + Tab， gating 
- **stub **： API（curl +  session） stub  nesting
- ****：grep `hasPermission|checkRole|func.` ， `role_permissions.permissions`
- ****： API path  `api_static.txt`，runtime  METHOD/body； `param_candidates.json` + 

---




```bash
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```


```bash
# axios /  request
grep -rhoaE '(axios|request)\.(get|post|put|delete|patch)\(' js | head
grep -rhoaE 'interceptors\.(request|response)' js | head

# GraphQL
grep -rhoaE '(query|mutation)\s+\w+|gql`|graphql\(' js | head
grep -rhoaE '\$[a-zA-Z_]+\s*:\s*(Int|String|Boolean|\[)' js | head

# FormData / multipart
grep -rhoaE 'FormData|\.append\(' js | head

grep -rhoaE 'path:\s*"/[^"]*:[^"]+"' js | head
grep -rhoaE 'useParams|route\.params|\$route\.params' js | head
```


```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```


```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```

runtime ：DevTools → Network →  → ****（call stack） `fetch`/`send` 。


```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```

**** — Hook ****， plaintext payload； `config.json` / `param_candidates.json`。


，diff  body/query：

|  |  |
|---|---|
|  |  |
|  | keyword、filters |
|  | optional  |
| / |  entity |
| / | `ids[]`、`exportType` |
| / | `sortField`、`order` |

 `param_samples.json`：`[{ "path", "method", "action": "search", "body", "query", "headers" }]`


|  |  |
|---|---|
| **** |  callsite + runtime ≥2  |
| **** | ， 1  runtime |
| **** | /， |
| **** | ，UI/ |


|  |  |
|---|---|
| REST  | J1  → J6  diff → J3 rules |
| / | J3 Form name → J4 submit  → runtime  +  400 |
| GraphQL | J2 variables  → runtime  operation  variables |
|  body | J5 Hook  →  params |


| api-recon |  recon |
|---|---|
| Phase 1  | J1  |
| Phase 2 A2  | （tenantId、sign） |
| Phase 3 runtime | J6  + `param_samples.json` |
| Phase 4  |  →  |
| Phase 5  | `params_merged.json` + ； |


|  |  |
|---|---|
|  runtime  | 「」； /  /  select  option |
|  path  body  |  —  `action` ， schema |
| stub  params | ** outbound ** body/headers， stub  |
| 400  nested field |  `data`/`bizData`/`variables` |
| GraphQL  operation  |  `variables` JSON； `$var: Type` |

---
