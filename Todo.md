# Todo

Outstanding items on the Kubernetes ingress controller ([PR #272]).

[PR #272]: https://github.com/fabiocicerchia/go-proxy-cache/pull/272

## Gateway API Inference Extension — only partially supported

An `HTTPRoute` can name an `InferencePool` (`inference.networking.k8s.io/v1`) as
a `backendRef`, and it resolves to the pool's ready Pods. **The endpoint picker
is not implemented**, so the useful part of the extension is absent: an
InferencePool currently works as a pool of Pods, not as an inference gateway.

- [ ] **Tier 2 — the Endpoint Picker.** `endpointPickerRef` is ignored, so
      endpoints are chosen by the ordinary load-balancing algorithm rather than
      by KV-cache utilisation or queue depth, and `failureMode`
      (`FailOpen`/`FailClose`) has nothing to act on. Needs an ext-proc gRPC
      client (`envoy.service.ext_proc.v3`) and request buffering.
      **Verify the wire contract against the upstream repo before building
      it** — what is currently in `docs/INGRESS.md` came from secondary
      sources, not the spec, because the spec site was unreachable.
- [ ] **Tier 3 — body-aware routing.** Choosing a backend by the `model` field
      of an OpenAI-style JSON body. `server/router/router.go` never reads the
      body.
- [ ] A pool declaring several `targetPorts` uses the first and logs the rest;
      choosing between them is the picker's job, so this resolves with Tier 2.

## Multi-tenancy hardening

Not vulnerabilities — this matches the accepted baseline for ingress
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

## Housekeeping

- [ ] `gofmt` fails on three files, none of them new:
      `cache/engine/client/client_cluster_test.go` and `client_wildcard.go`
      predate the branch; `client.go` was reformatted by `a13f470`. CI runs
      these under `sca` with `continue-on-error: true`, so nothing is red
      today.
