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

## Enterprise Metrics

### Build metrics

Name | Type | Description | Labels
----|----|----|----|
`gpcee_build_info` | Gauge | Shows the exporter build information. | `env`, `hostname`, `gitCommit`, `version` |
`gpcee_up` | Gauge | Shows the status of the last metric scrape: `1` for a successful scrape and `0` for a failed one | `env`, `hostname` |

### HTTP

Name | Type | Description | Labels
----|----|----|----|
`gpcee_http_request` | Counter | Distribution by Request. | `env`, `hostname`, `req_id`, `url`, `host`, `scheme`, `method`, `protocol`, `content_length` |
`gpcee_http_response` | Counter | Distribution by Response. | `env`, `hostname`, `req_id`, `url`, `host`, `scheme`, `method`, `protocol`, `content_length` |
`gpcee_http_requests_total` | Counter | Total http requests. | `env`, `hostname` |

#### HTTP Upstreams

Name | Type | Description | Labels
----|----|----|----|
`gpcee_upstream_server_requests` | Counter | Total client requests. | `env`, `hostname`, `server`, `upstream` |
`gpcee_upstream_server_responses` | Counter | Total responses sent to clients. | `code` (the response status code. The values are: `1xx`, `2xx`, `3xx`, `4xx` and `5xx`), `env`, `hostname`, "code", `server`, `upstream` |
`gpcee_upstream_server_sent` | Counter | Bytes sent to this server. | `env`, `hostname`, `server`, `upstream` |
`gpcee_upstream_server_received` | Counter | Bytes received to this server. | `env`, `hostname`, `server`, `upstream` |
`gpcee_upstream_server_response_time` | Gauge | Average time to get the full response from the server. | `env`, `hostname`, `server`, `upstream` |
`gpcee_upstream_server_health_checks_checks` | Counter | Total health check requests. | `env`, `hostname`, `server`, `upstream` |
`gpcee_upstream_server_health_checks_fails` | Counter | Failed health checks. | `env`, `hostname`, `server`, `upstream` |
`gpcee_upstream_server_health_checks_unhealthy` | Counter | How many times the server became unhealthy (state 'unhealthy'). | `env`, `hostname`, `server`, `upstream` |
