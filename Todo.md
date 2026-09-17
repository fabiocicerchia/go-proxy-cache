# Todo

Outstanding items on the Kubernetes ingress controller ([PR #272]).

[PR #272]: https://github.com/fabiocicerchia/go-proxy-cache/pull/272

## Gateway API Inference Extension

`InferencePool` backends and the Endpoint Picker are both implemented. What is
left:

- [ ] **Body-aware routing.** Choosing a pool by the `model` field of an
      OpenAI-style JSON body. `server/router/router.go` matches on host, path,
      method, headers and query, and never reads the body.
- [ ] A pool declaring several `targetPorts` uses the first and logs the rest.
      The picker protocol carries one endpoint per answer, so a second port
      needs a way to say which the answer refers to.
- [ ] The picker is reached over plaintext gRPC. The extension does not define
      a TLS story for this hop and the reference picker serves h2c, so this
      matches it, but it is worth revisiting if the spec grows one.
- [ ] Not exercised against a real Endpoint Picker. The ext-proc wire format is
      covered by tests against an in-process ext-proc server, which is as far
      as CI can go without a cluster.

## Done since this list was written

- Multi-tenancy visibility: `-disable-catch-all` (Helm:
  `controller.disableCatchAll`), a `RouteConflict` Event on the losing object
  when two claim one host and path, and a warning when two objects claim one
  certificate hostname. `docs/INGRESS.md` has a "Sharing a cluster" section.
- `gofmt` is clean across the tree.
