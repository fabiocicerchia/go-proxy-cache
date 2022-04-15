# Metrics

Go Proxy Cache supports application metrics via Prometheus endpoint, `/metrics`.

Name | Type | Description | Labels
----|----|----|----|
`gpc_status_codes_total` | Counter | Distribution by status codes. | `env`, `hostname`, `code` |
`gpc_request_host_total` | Counter | Distribution by Request Host. | `env`, `hostname`, `host` |
`gpc_http_methods_total` | Counter | Distribution by HTTP methods. | `env`, `hostname`, `method` |
`gpc_url_scheme_total` | Counter | Distribution by URL scheme. | `env`, `hostname`, `scheme` |
`gpc_request_sum_total` | Counter | Total number of sent requests. | `env`, `hostname` |
`gpc_request_1xx_total` | Counter | Total number of sent 1xx requests. | `env`, `hostname` |
`gpc_request_2xx_total` | Counter | Total number of sent 2xx requests. | `env`, `hostname` |
`gpc_request_3xx_total` | Counter | Total number of sent 3xx requests. | `env`, `hostname` |
`gpc_request_4xx_total` | Counter | Total number of sent 4xx requests. | `env`, `hostname` |
`gpc_request_5xx_total` | Counter | Total number of sent 5xx requests. | `env`, `hostname` |
`gpc_host_healthy` | Gauge | Health state of hosts. | `env`, `hostname` |
`gpc_host_unhealthy` | Gauge | Health state of hosts. | `env`, `hostname` |
`gpc_cache_hits_total` | Counter | The amount of cache hits. | `env`, `hostname` |
`gpc_cache_miss_total` | Counter | The amount of cache misses. | `env`, `hostname` |
`gpc_cache_stale_total` | Counter | The amount of cache misses. | `env`, `hostname` |

