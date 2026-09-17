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

## Done since this list was written

- Multi-tenancy visibility: `-disable-catch-all` (Helm:
  `controller.disableCatchAll`), a `RouteConflict` Event on the losing object
  when two claim one host and path, and a warning when two objects claim one
  certificate hostname. `docs/INGRESS.md` has a "Sharing a cluster" section.
- `gofmt` is clean across the tree.
