---

## name: scopesentry-mcp
description:  ScopeSentry MCP （、、、、）。 ScopeSentry、MCP、API Key、、。

# ScopeSentry MCP 

** ScopeSentry **。 Cursor（ MCP ），。



-  Web ：`http://<>`
- MCP ：`http://<>/mcp`（， `/mcp` ）


1.  ScopeSentry Web 
2.  **API Key** （）
3.  `ssk_...` （****）


Cursor → Settings → MCP → ：

```json
{
  "mcpServers": {
    "scopesentry": {
      "url": "http://<>:8082/mcp",
      "headers": {
        "X-API-Key": "ssk_"
      }
    }
  }
}
```

：`Authorization: Bearer ssk_`

 MCP  Cursor， `list_projects`、`list_assets` 。

---



|                      |                 |
| ---------------------- | ----------------- |
| `list_projects`        | （ ID） |
| `list_projects_data`   | ，     |
| `get_project`          |               |
| `create_project`       |               |
| `list_tasks`           |             |
| `get_task`             |               |
| `list_scan_templates`  |             |
| `get_scan_template`    |               |
| `list_plugin_modules`  |           |
| `list_plugins`         | （ hash、） |
| `create_scan_template` |             |
| `create_scan_task`     |             |
| `list_assets`          | （）       |
| `count_assets`         | （`/api/assets/common/total`） |
| `get_asset_detail`     |            |
| `add_asset_tag`        |            |
| `list_nodes`           |             |


 MCP （schema）；`list_assets` / `count_assets`  search、filter ， `list_assets` description。

「」 `count_assets`（ Web ）， `list_assets`。

---



****， `filter.project` ，。，。

1. `list_projects`  `list_projects_data`  **ObjectID**（`id` / `children[].value`）
2. `list_assets`  `filter.project`（** ID，**）

```json
{
  "asset_type": "asset",
  "pageIndex": 1,
  "pageSize": 20,
  "search": "domain=^example.com",
  "filter": {
    "project": ["<ObjectID>"]
  }
}
```


1. `list_nodes` 
2. `list_scan_templates`  `create_scan_template`  **ObjectID**
3. `create_scan_task`：`name`、`node` ，`template`  ID（）

** `targetSource`（ Web ）：**

| targetSource |  |  |
| --- | --- | --- |
| `general` |  | `target` |
| `project` |  | `project`（ ObjectID ） |
| `asset` |  Web  | `search`； `project`、`filter`、`targetNumber` |
| `RootDomain` |  | `search`； `project`、`filter`、`targetNumber` |
| `subdomain` |  | `search`； `project`、`filter`、`targetNumber` |
| `UrlScan` |  URL  | `search`； `project`、`filter`、`targetNumber` |
| `*Source`（ `subdomainSource`） | 「/」 | `targetTp=search`  `search`；`targetTp=select`  `targetIds` |


```json
{
  "name": "example-",
  "node": ["node-1"],
  "template": "<ObjectID>",
  "targetSource": "general",
  "target": "example.com\nfoo.com",
  "project": ["<ObjectID>"]
}
```


```json
{
  "name": "example-",
  "node": ["node-1"],
  "template": "<ObjectID>",
  "targetSource": "subdomain",
  "search": "task==\"example-\"",
  "project": ["<ObjectID>"]
}
```


********，，。



1. ** — **
   - `targetSource`: `general`
   - `target`: （）
   - ： `SubdomainScan`、`SubdomainSecurity`（ + ）
   -  `get_task` 

2. ** — **
   - `targetSource`: `subdomain`
   - `search`: `task=="<>"`（）
   -  `project` 
   - ：、、（ SubdomainScan）
   - ，

 Web 「」，「」，。

```mermaid
flowchart LR
  A[] --> B[1: general + SubdomainScan]
  B --> C[]
  C --> D[2: subdomain + task==1]
  D --> E[//]
```


1. `list_plugin_modules` → 
2. `list_plugins`（ `module` ）→  `hash`  `parameter`
3. `create_scan_template`： `modules` 「 →  hash 」

---

## 4. （`list_assets` / `count_assets`）

`count_assets`  `list_assets`  `asset_type`、`search`、`filter`， `{ "total": N }`， Web  `/api/assets/common/total`。

```json
{
  "asset_type": "subdomain",
  "search": "task==\"\"",
  "filter": {"project": ["<ObjectID>"]}
}
```

**（`list_assets` / `count_assets` ）：**  `filter.project` ；`search`  `==`  `^` （ [4.3](#43-search-)）， `=` 。。

 `filter.project`  [4.4](#44-filter-) 。


`asset`、`RootDomain`、`subdomain`、`app`、`mp`、`UrlScan`、`SensitiveResult`、`DirScanResult`、`crawler`、`vulnerability`、`PageMonitoring`、`IPAsset`、`SubdomainTakerResult`

：`web`→asset、`vuln`→vulnerability、`ip`→IPAsset、`url`→UrlScan



|                        |                                       |
| ------------------------ | --------------------------------------- |
| `pageIndex` / `pageSize` | ， 1 / 20                            |
| `search`                 | （）                              |
| `filter`                 |  JSON（）                          |
| `sort`                   |  UrlScan、DirScanResult  `length`  |
| `sid`                    |  SensitiveResult：                |


`search`  `filter` ****。


 DSL（** SQL**）：


|   |    |  |                           |
| ---- | ---- | ---- | --------------------------- |
| `=`  | （regex） |  | `domain=example`            |
| `==` | （） | **** | `port==443`                 |
| `!=` |    | — | `port!="80"`                |
| `&&` |     | — | `domain==example.com && port==443` |
| `||` |     | — | `title=admin || body=login` |


**：** `domain`、`ip`、`port`、`title` ， **`==` **  ** `^` **（ `domain=^example.com`）；**`=`  regex ，**，。

** search ：** `tag`、`task`（）、`rootDomain`

**project  search **（ `&&` ）。 `filter.project`。



| asset_type           |                                                                                   |
| -------------------- | ----------------------------------------------------------------------------------- |
| asset                | domain, ip, port, service, app, title, statuscode, icon, banner, type, body, header |
| RootDomain           | domain, icp, company                                                                |
| subdomain            | domain, ip, type, value                                                             |
| app                  | name, icp, company, category, description, url, apk                                 |
| mp                   | name, icp, company, category, description, url                                      |
| UrlScan              | url, input, source, resultId, type                                                  |
| SensitiveResult      | url, sname, body, info, md5                                                         |
| DirScanResult        | url, statuscode, redirect, length                                                   |
| vulnerability        | url, vulname, matched, request, response, level                                     |
| crawler              | url, method, body, resultId                                                         |
| PageMonitoring       | url, hash, diff, response                                                           |
| IPAsset              | ip, domain, port, service, webServer, app                                           |
| SubdomainTakerResult | domain, value, type, response                                                       |



- `domain==www.example.com && port==443`（，）
- `domain=^example.com`（，）
- `ip==192.168.1.1`
- `task==""`
- `level==high`（vulnerability）
- `statuscode==200`（DirScanResult）

 `=`， `title=admin`（，）。


JSON ： key  **OR**， key  **AND**。

** `project`：** ， asset_type  `project`，；。


| filter key   |        |                                                      |
| ------------ | -------- | -------------------------------------------------------- |
| `project`    |      | **ObjectID**， `list_projects` / `list_projects_data`  |
| `task`       |      | ****， `list_tasks`  `name`                         |
| `port`       |        |  `"443"`                                                |
| `service`    | /    |  `"https"`                                              |
| `app`        |      |  `"Nginx"`                                              |
| `icon`       |  hash  |                                                          |
| `statuscode` | HTTP  |  asset                                               |
| `status`     |        | UrlScan/DirScan HTTP ；/                       |
| `level`      |      | critical / high / medium / low / info                    |
| `type`       |        |  A、CNAME                                         |
| `color`      |    | SensitiveResult                                          |
| `sname`      |     | SensitiveResult                                          |
| `tags`       |        |                                                          |




| asset_type                            | filter key                                                      |
| ------------------------------------- | --------------------------------------------------------------- |
| asset                                 | project, port, service, app, icon, statuscode, type, task, tags |
| RootDomain                            | project, tags                                                   |
| subdomain                             | project, type, task, tags                                       |
| app / mp                              | project, tags                                                   |
| UrlScan                               | status, tags                                                    |
| DirScanResult                         | status, tags                                                    |
| SensitiveResult                       | status, color, sname, tags                                      |
| crawler                               | project, task, tags                                             |
| vulnerability                         | project, level, status, task, tags                              |
| PageMonitoring / SubdomainTakerResult | tags                                                            |
| IPAsset                               | project, port, service, app                                     |



```json
{"project": ["<ObjectID>"], "port": ["443"]}
```


```json
{
  "asset_type": "asset",
  "search": "domain=^baidu && port==443",
  "filter": {"project": ["<ObjectID>"]},
  "pageIndex": 1,
  "pageSize": 10
}
```


-  `filter.project`（）；
- `filter.project` 
-  `==`， `^`； `=` 
- UrlScan  HTTP  `filter.status`；DirScanResult  search  `statuscode==200`
- SensitiveResult ：`search`  `sname=`， `filter.sname`


 **UrlScan**、**DirScanResult** ：

```json
{"length": "ascending"}
```

 `sort`，。

---


`TargetHandler`、`SubdomainScan`、`SubdomainSecurity`、`PortScanPreparation`、`PortScan`、`PortFingerprint`、`AssetMapping`、`AssetHandle`、`URLScan`、`WebCrawler`、`URLSecurity`、`DirScan`、`VulnerabilityScan`、`PassiveScan`

---



|         |                                                  |
| --------- | -------------------------------------------------- |
| MCP    |  URL、API Key、ScopeSentry                     |
| 401 / 403 |  API Key                                    |
|      |  `filter.project`  ObjectID； search  project |
| / | `template`  ObjectID；`node`             |
| /   |  `filter.project`；search  `==`  `^` ， `=`； `pageSize` |


---

