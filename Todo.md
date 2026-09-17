# Todo

Bugs found while reviewing the Kubernetes ingress controller branch
(`claude/go-proxy-k8s-ingress-compat-vfpn8v`, [PR #272]) and what is left open.

Four review passes were run over `origin/main...HEAD`: `/code-review` and
`/security-review` on the first round, then both again after the fixes landed.
The second round found four regressions the first round's fixes had introduced,
which is why the fixed list below is longer than the review counts.

[PR #272]: https://github.com/fabiocicerchia/go-proxy-cache/pull/272

---

## Open

### Gateway API Inference Extension — only partially supported

An `HTTPRoute` can name an `InferencePool` (`inference.networking.k8s.io/v1`) as
a `backendRef`, and it resolves to the pool's ready Pods. **The endpoint picker
is not implemented**, so the useful part of the extension is absent.

- [ ] **Tier 2 — the Endpoint Picker.** `endpointPickerRef` is ignored, so
      endpoints are chosen by the ordinary load-balancing algorithm rather than
      by KV-cache utilisation or queue depth, and `failureMode`
      (`FailOpen`/`FailClose`) has nothing to act on. Needs an ext-proc gRPC
      client (`envoy.service.ext_proc.v3`) and request buffering.
      **Verify the wire contract against the upstream repo before building
      it** — the details currently in `docs/INGRESS.md` came from secondary
      sources, not the spec, because the spec site was unreachable.
- [ ] **Tier 3 — body-aware routing.** Choosing a backend by the `model` field
      of an OpenAI-style JSON body. `server/router/router.go` never reads the
      body.
- [ ] A pool declaring several `targetPorts` uses the first and logs the rest;
      choosing between them is the picker's job, so this resolves with Tier 2.

### Multi-tenancy hardening

Not vulnerabilities — verified as the accepted baseline for ingress
controllers, which document admission control as the mitigation — but each is
something other controllers grew.

- [ ] No equivalent of ingress-nginx's `--disable-catch-all`. A hostless
      Ingress rule or `defaultBackend` from any namespace captures every
      unmatched path of every other tenant's hostname.
- [ ] A cross-namespace `host`+`path` conflict is silently deprioritised rather
      than rejected with a status condition and an Event. The routing outcome
      matches ingress-nginx; the operator visibility does not.
- [ ] The certificate map is last-writer-wins across Ingresses in
      namespace/name order, with no warning when two objects claim one
      hostname.

### Housekeeping

- [ ] `gofmt` fails on three files, none of them new in this session:
      `cache/engine/client/client_cluster_test.go` and `client_wildcard.go`
      predate the branch; `client.go` was reformatted by `a13f470`. CI runs
      these under `sca` with `continue-on-error: true`, so nothing is red today.

---

## Known limitations (by design, not bugs)

- **An escaped separator cannot survive a path rewrite.** `URL.Path` is decoded
  before any filter runs, so `%2F` in a request path becomes a real separator
  once `ReplacePrefixMatch` or `ReplaceFullPath` touches it. Inherent to
  rewriting `Path`; nothing in `RawPath` recovers it. Redirect `Location`
  headers *do* preserve it when the path is passed through untouched.
- **`routedDomains` keeps one entry per host.** Where several objects serve one
  hostname, the first by (namespace, name) supplies the domain-level settings
  used by the health endpoint. Anything that must honour a specific route's
  settings reads them from the matched route instead — which is what the JWT
  fix below did.

---

## Fixed

Two rounds. Commit refs are on the branch.

### Round 1 — the branch as it stood

| | Fix |
|---|---|
| `server/balancer/balancer.go` | `DoHealthCheck` dereferenced the nil `url.Parse("10.1.2.3:8080")` returns. Killed the process ~30s after the first sync, from a bare goroutine. `fd55b40` |
| `server/router/model.go` | Health check inherited the global `https` default, so plain-HTTP pods all went unhealthy and traffic pinned to `Endpoints[0]`. `fd55b40` |
| `k8s/annotations.go` | `InitJWT` on every translation pass leaked a `jwk.Cache` goroutine set per jwt-annotated object per sync. `fd55b40` |
| `k8s/election.go` | `RunOrDie` returns on lease *loss*, not only ctx cancel, so a replica that blipped never stood again and status write-back stopped for good. `fd55b40` |
| `k8s/gateway.go` | Cross-namespace `backendRef` honoured with no `ReferenceGrant`: any tenant could publish any Service in the cluster. RBAC already granted `referencegrants`; no code read them. `fbaef96` |
| `k8s/gateway_status.go` | Same gap for `certificateRef` — another namespace's TLS private key, served under a hostname of the referrer's choosing. `fbaef96` |
| `k8s/controller.go` | JWT resolved by Host from the first-sorting route. Exploitable, but also fires accidentally: splitting one host across two Ingresses silently disabled authentication. `fbaef96` |
| `cache/cache.go` | Two routes differing only by a header match shared one cache entry and served each other's bodies. `f4bf432` |

### Round 2 — regressions from the round-1 fixes

| | Fix |
|---|---|
| `k8s/controller.go` | **The InferencePool cache sync was waited on unconditionally.** With the CRDs absent the reflector never syncs, so `Run` blocked before publishing any routing table and the proxy 404'd everything. Hit anyone using Gateway API without the Inference Extension. `860a073` |
| `cache/cache.go` | The `META` key never got the route variant the storage key gained, so two routes clobbered each other's `Vary` list — a permanent miss, worse than the bug the variant fixed. `860a073` |
| `server/router/model.go` | Copying the backend scheme into the health check fixed https-probing-http and broke `ws`/`wss`, which an HTTP client refuses outright. `860a073` |
| `.env.dist` | `METRICS_PER_REQUEST_SERIES=` empty-but-set fails `ParseBool` and reaches `log.Fatal`. Three pre-existing keys had the same defect. `860a073` |

### Round 2 — pre-existing branch defects

| | Fix |
|---|---|
| `server/handler/realip.go` | `GetScheme()` read the trusted-proxy list off `DomainConfig`, but is called *to resolve* it. Behind a TLS-terminating LB the domain lookup always ran as http, so a host with separate http/https entries got the wrong block — purge allowlist and JWT settings included. `074c871` |
| `server/handler/filters.go` | Redirect `Location` concatenated from the decoded `URL.Path`, emitting a raw space for `%20`. `074c871` |
| `k8s/gateway.go` | `ResolvedRefs` judged per route, so one working rule masked two broken ones. `074c871` |
| `k8s/gateway_status.go` | `AttachedRoutes` counted listener bindings, and the Gateway-wide total was written onto every listener. `074c871` |
| `server/handler/healthcheck.go` | Ready before the first routing table: a restarting replica joined the Service while still 404ing everything. `074c871` |

---

## Refuted

Recorded so they are not re-raised. Each was verified against the source, and
the first two by running the library rather than reasoning about it.

- **`applyRewrite` leaves a stale `URL.RawPath`.** Go's `EscapedPath()` already
  checks `RawPath` against `Path` and re-encodes when they disagree, which after
  a rewrite they always do. Clearing it is a no-op there, and would discard a
  valid encoding in the one case it still describes `Path`. The genuine form of
  this — a `Location` header built by concatenation, where nothing escapes
  anything — was a separate bug and is fixed above.
- **`X-Forwarded-Proto` read from the left-most hop.** Left-most is the correct
  reading for XFP (it is what nginx, Envoy and Traefik do); unlike XFF there is
  no per-hop chain to walk. Gated on `isFromTrustedProxy()` and inert with the
  default empty `trusted_proxies`. The cache-poisoning chain does not close:
  `GetScheme()` reduces the header to a boolean and returns one of four
  constants, and the forged scheme drives both the bucket choice and the
  upstream fetch, so an attacker only moves their own request between buckets.
- **Hostname hijacking across namespaces.** Exact-over-Prefix precedence is
  mandated by the Gateway API spec, the same attack works against ingress-nginx,
  and the oldest-wins tie-break the finding said was missing exists at
  `server/router/router.go:298-302`. Hardening items extracted above.

---

## Not verified locally

Both run in CI on the PR; neither can run in the dev container.

- Functional and end-to-end suites — need Redis.
- `make helm-template` / `make kustomize-build` — need the `helm` and
  `kubectl` binaries.
