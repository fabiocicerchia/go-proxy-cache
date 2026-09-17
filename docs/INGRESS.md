# Kubernetes Ingress Controller

go-proxy-cache can run as a Kubernetes ingress controller. Instead of reading
its virtual hosts from the `domains` block of `config.yml`, it watches the
cluster and builds its routing table from `Ingress` objects — and, optionally,
from the Gateway API's `Gateway` and `HTTPRoute`.

What you get over a plain ingress controller is the cache: every route is a
caching reverse proxy, with per-Ingress TTLs, negative caching, collapsed
forwarding and `PURGE`, configured through annotations.

## Which build

The controller needs the Kubernetes client, which is large enough that the
standalone proxy does not carry it: 36 MB against 91 MB. So there are two
builds of the same binary.

| | Image | Built with |
|---|---|---|
| Standalone proxy | `fabiocicerchia/go-proxy-cache:<tag>` | `make build` |
| Ingress controller | `fabiocicerchia/go-proxy-cache:<tag>-ingress` | `make build-ingress` |

The Helm chart and the kustomize overlay already select the `-ingress` image
when the controller is enabled. The `-k8s` flag exists only in that build; the
standalone binary does not accept it.

## Quick start

### Helm

```sh
helm install gpc kubernetes/helm \
  --namespace go-proxy-cache --create-namespace \
  --set controller.enabled=true
```

Then point an Ingress at it:

```sh
kubectl apply -f docs/examples/ingress/ingress.yaml
curl -H 'Host: demo.local' http://<address>/ -D-   # X-Go-Proxy-Cache-Status: MISS
curl -H 'Host: demo.local' http://<address>/ -D-   # X-Go-Proxy-Cache-Status: HIT
```

### Kustomize

```sh
kubectl apply -k kubernetes/kustomize/ingress-controller
```

### Bare binary (development, against a kubeconfig)

```sh
make build-ingress
./go-proxy-cache -k8s -kubeconfig ~/.kube/config -config config.yml
```

## How routing works

Routes are keyed on **host and path**, which the static configuration cannot
express — it matches on the `Host` header alone.

Precedence follows the Gateway API rules, which are a superset of the Ingress
"longest path wins" rule:

1. An exact hostname beats a `*.example.com` wildcard, which beats a rule with
   no hostname.
2. `Exact` path beats the longest matching `Prefix`, which beats a regular
   expression.
3. Then: most header matches, a method match, most query parameter matches.
4. Ties are broken by the oldest object, then by name.

`Prefix` matches whole path elements, so `/abc` matches `/abc` and `/abc/def`
but **not** `/abcd`. A wildcard hostname covers exactly one label, so
`*.example.com` serves `a.example.com` but neither `example.com` nor
`a.b.example.com`.

`ImplementationSpecific` paths are treated as prefixes unless the Ingress sets
`use-regex: "true"`, in which case they are compiled as regular expressions.

Unlike the static configuration, the client's `Host` header is **preserved**
when forwarding upstream. Set `preserve-host: "false"` or `upstream-host` to
change that.

## Backends

Backends are resolved from **EndpointSlices to ready pod IPs**, not through the
Service ClusterIP. That means the existing load balancing algorithms
(`round-robin`, `ip-hash`, `least-connections`, `random`) apply per pod rather
than being flattened to kube-proxy's L4 round-robin, and readiness is observed
directly. Endpoints that are unready or terminating are excluded, which is what
otherwise causes 502s during a rollout.

Two fallbacks: an `ExternalName` Service resolves to the name it points at, and
a Service whose EndpointSlices have not appeared yet resolves to its in-cluster
DNS name, so a route serves rather than 404s while the cluster catches up.

Active health checks are not used in this mode — an EndpointSlice already
reflects each pod's own readiness probe.

## TLS

Certificates come from `kubernetes.io/tls` Secrets referenced by
`Ingress.spec.tls` or a Gateway listener's `certificateRefs`, and are served
via SNI. Each certificate is indexed by the hosts the object lists, falling
back to the SANs in the certificate itself, so wildcard certificates work
without listing every subdomain.

A certificate registered under the empty host becomes the fallback served when
SNI matches nothing.

Certificates are swapped atomically as Secrets change; nothing needs a restart.

## Annotations

All annotations are prefixed with `go-proxy-cache.fabiocicerchia.it/`.

| Annotation | Effect |
| --- | --- |
| `cache-enabled` | `false` disables caching for the route |
| `cache-ttl` | Fallback TTL in seconds when the response carries no cache headers |
| `cache-allowed-statuses` | Comma separated status codes to cache, e.g. `200,301,404` |
| `cache-allowed-methods` | Comma separated methods to cache, e.g. `GET,HEAD` |
| `cache-negative-ttl` | Per-status TTL overrides, e.g. `404=30,502=10` |
| `gzip` | Compress responses |
| `collapsed-forwarding` | Coalesce concurrent identical misses into one upstream request |
| `backend-protocol` | `HTTP`, `HTTPS`, `WS` or `WSS` |
| `insecure-bridge` | Skip upstream certificate verification |
| `balancing-algorithm` | `round-robin`, `ip-hash`, `least-connections`, `random` |
| `preserve-host` | Forward the client `Host` upstream (default `true`) |
| `upstream-host` | Force a specific `Host` header upstream |
| `http-to-https` | Redirect HTTP to HTTPS |
| `redirect-status-code` | Status code used for that redirect |
| `hsts-enabled`, `hsts-max-age`, `hsts-include-subdomains`, `hsts-preload` | `Strict-Transport-Security` |
| `purge-allowed-ips` | Comma separated IPs/CIDRs allowed to issue `PURGE` |
| `jwt-jwks-url`, `jwt-allowed-scopes`, `jwt-excluded-paths` | JWT validation |
| `rewrite-target` | Rewrite the matched path before forwarding |
| `use-regex` | Treat `ImplementationSpecific` paths as regular expressions |

A malformed annotation value is logged and ignored rather than taking the route
out of service.

## Gateway API

Install the CRDs, then start the controller with `-gateway-api` (Helm:
`--set controller.gatewayAPI.enabled=true`):

```sh
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/standard-install.yaml
```

Supported: `GatewayClass`, `Gateway` (HTTP and HTTPS listeners, hostnames,
`certificateRefs`, `allowedRoutes` namespace policies) and `HTTPRoute`
(hostname intersection with the listener, path/method/header/query matches,
weighted `backendRefs`, and the `RequestHeaderModifier`,
`ResponseHeaderModifier`, `RequestRedirect` and `URLRewrite` filters).

Not yet supported: `GRPCRoute`, `TCPRoute`/`TLSRoute`, `BackendTLSPolicy`,
selector-based `allowedRoutes` policies (a listener using one admits no
cross-namespace routes rather than guessing), and session persistence.

### Inference Extension (partial)

An `HTTPRoute` may name an `InferencePool`
(`inference.networking.k8s.io/v1`) as a `backendRef`. The pool's
`selector` and `targetPorts` are resolved to the ready Pods behind it, and
traffic is load balanced across them like any other backend — with the cache in
front of it.

```yaml
rules:
  - backendRefs:
      - group: inference.networking.k8s.io
        kind: InferencePool
        name: vllm-pool
```

**The endpoint picker is not implemented.** `endpointPickerRef` is ignored, so
endpoints are chosen by the configured load-balancing algorithm rather than by
KV-cache utilisation or queue depth, and `failureMode` has nothing to act on.
A pool declaring several `targetPorts` uses the first and logs the rest, since
choosing between them is the picker's job. Model-aware routing on the request
body is not implemented either.

In other words: an InferencePool works as a pool of Pods, not as an inference
gateway. If you need the scheduling behaviour the extension exists for, this is
not yet a substitute for an implementation that speaks ext-proc to the picker.

Cross-namespace `backendRefs` to a pool need a `ReferenceGrant`, the same as
any other kind.

See `docs/examples/ingress/httproute.yaml` for a weighted canary split.

### Sharing a cluster

Nothing in the Ingress API expresses hostname ownership, so any namespace that
can create an Ingress can name any hostname. That is the same position every
ingress controller is in, and admission control (Kyverno, OPA/Gatekeeper) or
`-watch-namespace` is the real answer. Two things make the situation visible
rather than silent:

- **Collisions are reported.** When two objects claim the same host, path and
  path type, the one that loses gets a `RouteConflict` warning Event naming the
  winner, so `kubectl describe ingress` says why a route is not serving. The
  winner is decided by the same precedence the router applies: path specificity
  first, then the older object.
- **Duplicate certificate claims are logged**, naming both objects, rather than
  one silently replacing the other in the SNI store.

`-disable-catch-all` (Helm: `controller.disableCatchAll`) ignores hostless
Ingress rules and `defaultBackend`s. One of those otherwise serves every path
no other route claims, on every hostname in the cluster.

### Cross-namespace references

A `backendRef` or `certificateRef` that names another namespace is only
honoured when that namespace has published a `ReferenceGrant` permitting it,
as the Gateway API requires. Without one the reference is skipped and the
object reports `ResolvedRefs=False` with reason `RefNotPermitted`.

This matters because the controller reads Services and Secrets cluster-wide:
without the check, anyone able to create an `HTTPRoute` could publish any
Service in the cluster, and anyone able to create a `Gateway` could serve
another namespace's TLS private key under a hostname of their choosing.

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: allow-tenant-a
  namespace: platform          # the namespace being referenced
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      namespace: tenant-a      # the namespace doing the referencing
  to:
    - group: ""
      kind: Service
      name: shared-api         # omit to cover every Service in the namespace
```

Use `kind: Gateway` in `from` and `kind: Secret` in `to` to grant a
`certificateRef`. Same-namespace references never need a grant.

## High availability

Every replica watches the cluster and serves traffic. Only the replica holding
a `coordination.k8s.io` Lease writes status back onto the objects, so replicas
do not fight over `.status.loadBalancer`; losing the lease never takes a pod
out of the data path.

The address published in `kubectl get ingress`'s `ADDRESS` column comes from
`-publish-service <namespace>/<name>` (the controller's own Service by default)
or from an explicit `-publish-status-address`. Nothing is published when
neither resolves — an empty column is more honest than a guessed address that
would mislead external-dns and cert-manager.

Redis is shared by all replicas, so the cache and `PURGE` stay consistent
across the fleet.

## Flags

| Flag | Environment variable | Default |
| --- | --- | --- |
| `-k8s` | `INGRESS_CONTROLLER_ENABLED` | off |
| `-gateway-api` | `GATEWAY_API_ENABLED` | off |
| `-ingress-class` | `INGRESS_CLASS` | `go-proxy-cache` |
| `-controller-name` | `INGRESS_CONTROLLER_NAME` | `fabiocicerchia.it/go-proxy-cache` |
| `-watch-namespace` | `WATCH_NAMESPACE` | whole cluster |
| `-publish-service` | `PUBLISH_SERVICE` | — |
| `-publish-status-address` | `PUBLISH_STATUS_ADDRESS` | — |
| `-election-id` | `ELECTION_ID` | `go-proxy-cache-ingress-controller` |
| `-disable-status-updates` | `DISABLE_STATUS_UPDATES` | off |
| `-kubeconfig` | `KUBECONFIG` | in-cluster |

Leader election also reads `POD_NAMESPACE` and `POD_NAME`; the shipped
manifests inject both from the downward API.

## RBAC

The controller needs `get`/`list`/`watch` on `ingresses`, `ingressclasses`,
`services`, `secrets`, `namespaces` and `endpointslices` (plus the Gateway API
resources when enabled), `update`/`patch` on the corresponding `/status`
subresources, and namespaced access to `leases` for the election. Both the Helm
chart and the kustomize overlay ship this; run with
`-disable-status-updates` if you would rather grant read-only access, at the
cost of an empty `ADDRESS` column.

## Running behind a cloud load balancer

When another proxy sits in front — a cloud L7 load balancer, or a second
ingress — the connection go-proxy-cache sees is from that proxy, not the
client. List it under `server.trusted_proxies` (or `TRUSTED_PROXIES`):

```yaml
server:
  trusted_proxies:
    - 10.0.0.0/8
```

Only then are `X-Forwarded-Proto` and `X-Forwarded-For` believed. Without it a
TLS-terminating load balancer makes every request look like plain HTTP, which
suppresses HSTS and, with `http-to-https` on, loops the redirect; and the
`PURGE` IP allowlist would be matching the load balancer's address rather than
the client's.

Both headers are forgeable by anything that can reach the listener, so the
default is to trust nothing. The client address is taken as the right-most
`X-Forwarded-For` entry that is not itself a trusted proxy.

An alternative on a `LoadBalancer` Service is
`externalTrafficPolicy: Local`, which preserves the client source IP at the
connection level. The shipped kustomize overlay sets it.

## Metrics

`gpcee_http_request` and `gpcee_http_response` carry `req_id`, `url`, `size`
and `duration` as *labels*, so their cardinality grows with the number of
distinct URLs served. Serving a whole cluster's traffic, that has no bound
anyone has agreed to, so ingress mode does not record them. Set
`metrics.per_request_series: true` (or `METRICS_PER_REQUEST_SERIES=true`) to
opt back in.

Every aggregate metric — by host, method, scheme, status code, cache
hit/miss/stale, upstream health — is bounded and always recorded.

## What stays in config.yml

Ingress mode still reads `config.yml` for everything that is global to the
controller rather than per-route: listening ports, timeouts, the Redis backend,
TLS cipher settings, logging, tracing and the circuit breaker. The `domains`
block is unused — routing comes from the cluster.
