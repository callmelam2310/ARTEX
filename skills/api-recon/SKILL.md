---
name: api-recon
description: Call this skill when collecting a website's API endpoints.
---

# API Recon (front-end interface reconnaissance)

Under an **authorized** premise, discover as completely as possible: **back-end APIs** (paths, methods, parameters, response bodies), **front-end routes**, and **UI feature trigger points** (tabs, dialogs, table actions, etc.).

---

## Boundaries and prohibitions (Agent must read · violating = out of scope)

This skill **only does API / parameter-surface recon**; it is not the vulnerability-hunting or exploitation stage.

### Task boundaries

| Scope | Allowed | Forbidden |
|---|---|---|
| **Target** | Enumerate paths, methods, parameters, routes, UI trigger points | SQLi/XSS/privilege-escalation/brute-force/fuzz vulnerabilities, packet-tampering attacks, destructive operations |
| **Auth** | Hook + stub/mock to bypass the **client-side** login gate | Asking the user for or guessing credentials; attempting real login-form submission |
| **Runtime** | Hook interfaces without credentials, use mock responses to let the SPA enter the post-login shell | Flows that depend on a real back-end session to continue |

### Credential-free dynamic analysis (Phase 3 default)

1. Use `preload.js` / `runtime_harvest.js` to **intercept and stub** bootstrap interfaces such as login, permissions, and menus;
2. Return **structurally correct, business-code-success, possibly-empty data** mock bodies for business query interfaces;
3. Make the front-end render post-login pages even with no back-end or in a 401 environment, thereby triggering more XHR/fetch/WebSocket;
4. **Empty data, blank tables, and placeholder UI are all expected** — do not switch to real login or vulnerability testing because of them.

**In one sentence**: use mocks to prop open the front-end routes and component mounting, and **only record outbound requests**; what the back-end returns does not matter, what matters is **which interfaces the front-end will still send**.

### Process hard prohibitions

| Forbidden | Alternative |
|---|---|
| grep/curl/Read the main entry `index-*.js` to extract API paths before Phase 1 is done | Run `OUTDIR/harvest_static.py` |
| Hand-writing scripts like `extract_apis.py` to replace harvest | Edit `OUTDIR/harvest_static.py` and re-run |
| Repeating the same grep/command after it failed >=2 times | Change strategy: read tool_logs, edit harvest, check reference |
| Skipping gates A/B and running the original `scripts/` directly | Copy to OUTDIR and adapt to the target |
| Real usernames/passwords, OTP, OAuth, and other authentication | stub/mock (see above) |
| Skipping the stub on the grounds of 'getting real data' to do privilege-escalation/injection testing | Only record outbound — that is the recon boundary |
| Irreversible operations like deleting, exporting sensitive data, or bulk writes | the same applies to coverage clicks |
| Claiming all pages and interfaces are obtained without completing runtime + dynamic enumeration | See 'Definition of done' or annotate the limitation |
| Claiming all parameters are mastered without completing the parameter-trigger matrix + diff | Phase 3b matrix + Phase 5 diff |
| Inferring required/optional from a single runtime sample | Multi-sample diff, or reverse-engineer from validation rules/errors |

---

## Two-layer model + run modes

| Layer | Output | Ceiling |
|---|---|---|
| **Static** (JS bundle) | Full set of endpoint paths, a route draft, field candidates at packing sites | No HTTP method; parameters need Phase 1b; misses runtime-concatenated URLs |
| **Runtime** (live session) | Method + body + response + dynamic URL + WS/SSE; multi-sample diff to complete parameters | The page must actually render to send a request; a single sample is not enough to fix required/optional |

| Run mode | Engine | Fits |
|---|---|---|
| **depth** | `runtime_harvest.js` (Puppeteer) | API inventory, METHOD/params/response body, WS/SSE, reproducible batch runs |
| **coverage** | browser + `preload.js` | Clicking tabs/dialogs/tables, deeper feature-point coverage |
| **both** | depth first, then coverage | Most complete, longest time |

**Parameter methodology** (no universal script): paths via harvest/regex; parameters via **anchor window expansion + UI binding chain + multi-sample diff + error reverse-inference** (grep recipes in section J of [reference.md](reference.md)).

---

## Definition of done

Recon may be claimed complete only when all are satisfied:

- [ ] **Static**: Phase 1 harvest produces `api_static.txt`, `routes.txt`, `js/`
- [ ] **Runtime**: at least one of depth or coverage; coverage/both must have **the Hook taking effect + the dynamic-enumeration loop**
- [ ] **Into the shell**: accessing a business path is not `/login` (mind hash routes)
- [ ] **Parameters**: coverage/both complete the parameter-trigger matrix + `param_samples.json`; Phase 5 merges `params_merged.json`
- [ ] **Depth** (if module pages are blank): Phase 4 reconstructs the permission tree and re-runs, until **module-level APIs** appear (not just locale/bootstrap)
- [ ] **Delivery**: the Phase 5 outputs are complete (see the Phase 5 output table); `insert_assets` writes the service and endpoint assets

---

## Scripts and gates

`scripts/` are only reference templates; it is **forbidden** to run the originals directly and take them as the final result.

**Rule**: read first → adapt to the target → write into `OUTDIR` (e.g. `recon/`) → record `CHANGES.md`; if it does not match, rewrite per the methodology, borrowing only the structure.

| Gate | When | Reference script → OUTDIR copy | Common must-change items |
|---|---|---|---|
| **A (static)** | After Phase 0, before running harvest/spider for the **first** time | `harvest_static.py` / `spider_mpa.py` | **Most sites run with the default regex directly**; only change the endpoint regex, webpack/Vite `publicPath`, MPA exclude/cookie when the manifest/dialect does not match |
| **B (runtime)** | After Phase 2, before running depth/coverage | `runtime_harvest.js` / `preload.js` + `config.json` | Cookie/localStorage keys, neutralize success values, stubs, login regex, api prefix, hash/history |

**SPA mandatory order** (not interchangeable; Phase numbering takes precedence over 'explore first, then script'):

| Step | Must | Forbidden |
|---|---|---|
| After Phase 0 is done | The next Bash = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | curl/grep/Read the main entry `index-*.js` (usually >500KB) |
| Gate A | Copy the script → make small changes as needed → **run immediately** | Manually extract APIs first, then decide whether to harvest |
| Before Phase 1 is done | `wc -l` to validate output; on 404 edit harvest and retry | Hand-write an extract script; repeatedly grep an un-downloaded URL |
| From Phase 1b | grep only `OUTDIR/js/*.js` | Use the main bundle instead of harvest |

- ✅ Copy `harvest_static.py` → (optional) edit regex → **run immediately**
- ❌ curl the main bundle → grep multiple times → write a temporary extract → harvest only at the end
- **MPA**: after Phase 0 the next Bash = `python3 OUTDIR/spider_mpa.py ...`

---

## Tool and output constraints

| Constraint | Note |
|---|---|
| Large files | an `index-*.js` >100KB **must not** be Read/grep'd into context; use OUTDIR scripts to batch-process |
| grep output | must `\| head -20` or `-m 5`; keep only path summaries in the conversation, do not paste bundle fragments |
| Validation | use `wc -l`, `ls \| wc -l`; do not Read an entire directory |
| regex first look | optional, <=1 time, only a <=50KB small chunk or HTML; the formal static pass relies on harvest |
| reference | recipes/templates/troubleshooting are in [reference.md](reference.md); do not re-inline the full text |

---

## Execution roadmap

```
Phase 0 classify + OUTDIR
  -> Gate A -> Phase 1 harvest (* run immediately *)
  -> Phase 1b parameter reverse-engineering
  -> Phase 2 three auth gates -> config.json
  -> Gate B -> Phase 3 runtime + parameter matrix
  -> Phase 4 permission tree (when needed) -> re-run Phase 3
  -> Phase 5 merge report + insert_assets bulk-inserts all discovered service and endpoint assets; under no circumstances omit an already-discovered asset when inserting
```

Check off in order; **do not enter the next Phase before the previous item is done**.

1. [ ] **Phase 0**: first look at SPA/MPA; create `OUTDIR` → [Phase 0](#phase-0--classify)
2. [ ] **Gate A + Phase 1**: copy the script → harvest **immediately** → `wc -l` validate → [Phase 1](#phase-1--static)
3. [ ] **Phase 1b**: anchor window expansion + binding layer → `param_candidates.json` → [Phase 1b](#phase-1b--parameter-reverse-engineering)
4. [ ] **Phase 2**: three auth gates → `config.json` → [Phase 2](#phase-2--three-auth-gates)
5. [ ] **Gate B**: adjust the runtime script → [Phase 3](#phase-3--runtime)
6. [ ] **Phase 3**: depth / coverage / both; confirm entry into the shell; parameter-trigger matrix → `param_samples.json`
7. [ ] **Phase 4** (if needed): permission tree → patch stubs → re-run Phase 3 → [Phase 4](#phase-4--permission-tree-reconstruction)
8. [ ] **Phase 5**: merge outputs + report + `insert_assets` → [Phase 5](#phase-5--merge-and-report)

---

## Phase 0 — classify

Fetch the entry HTML, **create `OUTDIR`** (do not edit the skill's internal `scripts/`):

- **SPA**: empty shell + `<div id=app>` + chunks → Phase 1–5
- **MPA**: SSR + `<form>`, no endpoint bundle → after Gate A:

```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```

Produces `forms.txt`, `links.txt`, `api_inline.txt`. For an SPA, if forms ≈ 0 → switch to Phase 1.

---

## Phase 1 — static

Obey [Scripts and gates](#scripts-and-gates) · [Tool and output constraints](#tool-and-output-constraints).

```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```

harvest: parse the HTML scripts → webpack/Vite manifest → download all lazy chunks → produce `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt`.

```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```

- chunk count vs manifest: on 404, edit harvest and retry; do not curl each chunk by hand
- `api_static.txt` too small → relax the endpoint regex inside OUTDIR and re-run (see reference)

### Phase 1b — parameter reverse-engineering

Paths come from Phase 1; parameter fields must be recon'd separately. grep rules are in [Tool and output constraints](#tool-and-output-constraints).

**Completion criterion**: for important interfaces you can answer — field name, transport location, type inference, whether required, sample value, confidence.

#### 1b.0 — transport form

| Form | Where the parameters are | Static priority to look at |
|---|---|---|
| REST JSON | body + query | `(params\|data\|body)\s*:\s*\{` next to the path anchor |
| GraphQL | `variables` | gql templates, `$page: Int` |
| Classic form | urlencoded | `<form>`, `FormData` |
| File upload | multipart | `FormData.append` |
| Path parameter | `/user/:id` | route table + `useParams` / `$route.params` |
| Encrypted/signed | wrapped into `sign`/`data` | Hook the crypto function's arguments (reference section D) |

Output: annotate each interface with `transport: query|json|form|graphql|encrypted`.

#### 1b.1 — anchor window expansion

Using a known path as an anchor, expand the window to find the packing object:

```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```

| Wrapper layer | Parameter clue |
|---|---|
| axios instance | `data` / `params` |
| unified request | interceptor injects global fields |
| OpenAPI client | generated method signatures |
| React Query / SWR | hook second argument |
| Vue composable | composable arguments |

Type residue: `yup`/`zod`/rules, `Form.Item name=`, embedded Swagger.

→ `param_candidates.json`: `{ path, fields[], source: "static-callsite", confidence }`

#### 1b.2 — binding layer

```
Form field -> onFinish/handleSubmit -> transform -> API payload
```

| Binding source | Technique |
|---|---|
| form submit | follow submit → transform → API |
| table search | `getFieldsValue()` → `params` |
| route | `:id` / `?tab=` |
| interceptor | global `tenantId`, pagination, sign |
| enum select | `options` → API enum values |

From the DevTools call stack, trace upward from `fetch`/`XHR.send` to the packing function.

#### 1b.3 — three packing questions (≠ Phase 2 auth gates)

| Question | What to answer |
|---|---|
| **Assembly** | where the payload is built, transform traces |
| **Validation** | required, pattern, enum |
| **Transport** | path / query / body / multipart / headers |

The interceptor gate (Phase 2) incidentally reads the globally-injected fields (Authorization, `X-Tenant-Id`, sign).

#### 1b.4 — hand-off to Phase 3

Candidate fields come from the static/binding layer; **required/optional/conditional dependencies** must be done in the Phase 3 parameter matrix + diff + Phase 5 error reverse-inference.

---

## Phase 2 — three auth gates

grep in `OUTDIR/js/` (with `head`), write into `config.json` (recipes in reference):

| Gate | Question | Keywords |
|---|---|---|
| **Render gate** | how to tell if logged in? | `isLogin`, `getToken`, Cookie/localStorage |
| **Interceptor gate** | what triggers a jump to `/login`? | `response_code`, `errno`, axios interceptor |
| **Content gate** | where do menus/permissions come from? | `menu`, `permission`, `role`, `acl`, `routes` |

Do not treat a localStorage key name as a credential — confirm it from the chunk/request chain.

**Exit = Gate B**: land the conclusions in `config.json`, and edit `OUTDIR/runtime_harvest.js` / `preload.js`.

### Phase 2b — API observation (optional)

Use `preload.js` inside OUTDIR to confirm session key names, Authorization, and nested API URLs:

| Config | Output |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | headers observation |
| `extractUrlsFromResponse: true` | sub-APIs inside the response |
| `observe.storageReads/cookieReads: true` | backfill config |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

Each coverage round exports: `__API_RECON_LOG__`, `__API_RECON_DETAIL__`, `__API_RECON_ROUTES__`, `__API_RECON_OBSERVE__`.

---

## Phase 3 — runtime

Must have passed Gate B; obey [Boundaries and prohibitions](#boundaries-and-prohibitions-agent-must-read--violating--out-of-scope) · the credential-free mock strategy.

`config.json` sets `"runtimeMode": "depth" | "coverage" | "both"` (template in reference).

### Hook and stub (shared by depth + coverage)

| Layer | Scope | Purpose |
|---|---|---|
| L1 exact | auth/permission/bootstrap stub | pass first-screen auth |
| L2 negative correction | all JSON responses | not-logged-in code → success |
| L3 fallback | `/api` etc. not matched by L1 | empty success body, prop open the UI |

- **depth**: fake auth + `forward` to change business codes + `stubs`; traverse `routes` (hash/history); produce `runtime_api.json`
- **coverage**: inject `preload.js` at **document-start** (CDP `addScriptToEvaluateOnNewDocument` or a Userscript)

Verify: `window.__API_RECON_PRELOAD__` exists; business paths do not return `/login`.

```bash
cd recon && npm install
node runtime_harvest.js config.json
```

### 3b — coverage dynamic enumeration (mandatory)

1. Main nav/sidebar — click each item, wait 1–3s for the network
2. Tabs — `role=tab`, `.ant-tabs-tab`
3. Tables — view/edit/detail on the first row
4. Toolbar — export, filter, new (**avoid irreversible deletes**)
5. On entering each module — merge APIs/routes
6. SPA — controlled `pushState` for paths in `routes.txt` not yet covered (forbidden for MPA)

**Parameter-trigger matrix** (mandatory): record once per operation type for each module, and **diff multiple samples**:

| Operation | Parameters usually added |
|---|---|
| list first screen | pagination + default filters |
| click search | keyword, filter |
| advanced filter | more optional ones |
| new/edit | full entity |
| bulk/export/sort | `ids[]`, `exportType`, `sortField` |

**Under the stub, the outbound body/headers are still real** — take the request as ground truth. Record → `scan_raw.json`, `param_samples.json`, `api_detail.json`.

- **Vue**: `neutralizeVueRouter: true` + document-start preload
- **React**: `routes.txt` + sidebar clicks + `pushState`
- **both**: 3a depth first, then 3b coverage

---

## Phase 4 — permission tree reconstruction

**Trigger**: module pages are blank / each route only has bootstrap (e.g. locale) → the content gate was not passed.

| Symptom | Meaning |
|---|---|
| entered the shell successfully | render gate + interceptor gate passed |
| missing sidebar items / clicks go blank | stub shape or permission codes incomplete |
| each route's API is identical and very few | `v-if permission` did not pass |
| `routes.txt` far fewer than the bundle | must be completed from the auth module |

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```

Typical chain: `role_permissions` (flat codes) + `permissions/all` (tree) → `getResultTree` → `userRouteAuth[CODE].url`.

```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```

Intermediate outputs: `route_map.json`, `userRouteAuth.json`, `permissions_tree.json`, `*_stub.json`, `perm_codes_all.txt`.

stub check: the outer `response_code` matches the interceptor gate; flat codes align with the tree; `routes` covers all links in `route_map`.

After updating `config.json`, **re-run Phase 3**. For large SPAs, you can tune `waitUntil`, `routeTimeout`, `perRouteMs` (see reference sections A3/I).

---

## Phase 5 — merge and report

### Output table

| File | Stage | Content |
|---|---|---|
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | static bundle and paths |
| `param_candidates.json` | 1b | static parameter field candidates |
| `config.json` | 2 | three gates + runtime config |
| `runtime_api.json` | 3a | depth detailed recording (incl. WS/SSE) |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | multi-sample, click logs, detail |
| `route_map.json` etc. | 4 | permission-tree intermediate files (if executed) |
| `params_merged.json` | 5 | merged parameter fields + confidence |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | routes, APIs, params, feature points, limitations |
| **insert_assets** | 5 | write all service and endpoint assets into the asset store |

### 5b — parameter merge

diff from `param_samples.json`; **no universal merge script**. Confidence rules are in reference J7 (high/medium/low/to-be-triggered).

### 5c — error reverse-inference

Within the authorized scope you may send incomplete requests to read 400s (**this is parameter recon, not vulnerability testing**): `field 'x' is required`, enum errors, etc. Mind the `data` wrapper, `variables`, and the pre-encryption `bizData`.

The report must note: runtimeMode, static/runtime API counts, parameter confidence, uncovered modules, and a `CHANGES.md` summary relative to the reference scripts.

Suggested `site_map.json` structure:

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

More fields and grep recipes are in [reference.md](reference.md).

---

## General notes

- **Framework-agnostic**: webpack/Vite/Angular lazy load use the same method
- **Transport**: REST/JSON, GraphQL, WebSocket, SSE; gRPC-web is out of scope
- **SSR**: client-side fetch can be recorded; RSC/Server Actions are not fully enumerable
- **Blind spots**: JSVMP, WASM, strong HMAC/mTLS validation → static + annotate the limitation
- **Parameter blind spots**: conditional linkage, hidden params, WASM packing → 'to-be-triggered' / 'unreachable'
- **Static is the safety net**: when runtime is blocked, static can still enumerate endpoints

---

## Additional resources

- Grep recipes, `config.json` templates, troubleshooting, Hook, parameter reverse-engineering section J, site_map template: **[reference.md](reference.md)**
- Reference script paths: see the [Scripts and gates](#scripts-and-gates) table
