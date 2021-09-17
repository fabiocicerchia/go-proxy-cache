# Metrics

Go Proxy Cache supports application metrics via Prometheus endpoint, `/metrics`.

For every request these are the metrics collected:

- `gpc_status_codes_total`  
   Distribution by status codes  
   Labels: hostname, env, code

- `gpc_request_host_total`  
   Distribution by Request Host  
   Labels: hostname, env, host

- `gpc_http_methods_total`  
   Distribution by HTTP methods  
   Labels: hostname, env, method

- `gpc_url_scheme_total`  
   Distribution by URL scheme  
   Labels: hostname, env, scheme

- `gpc_request_sum_total`  
   Total number of sent requests  
   Labels: hostname, env

- `gpc_request_1xx_total`  
   Total number of sent 1xx requests  
   Labels: hostname, env

- `gpc_request_2xx_total`  
   Total number of sent 2xx requests  
   Labels: hostname, env

- `gpc_request_3xx_total`  
   Total number of sent 3xx requests  
   Labels: hostname, env

- `gpc_request_4xx_total`  
   Total number of sent 4xx requests  
   Labels: hostname, env

- `gpc_request_5xx_total`  
   Total number of sent 5xx requests  
   Labels: hostname, env

- `gpc_host_healthy`  
   Health state of hosts  
   Labels: hostname, env

- `gpc_host_unhealthy`  
   Health state of hosts  
   Labels: hostname, env

- `gpc_cache_hits_total`  
   The amount of cache hits  
   Labels: hostname, env

- `gpc_cache_miss_total`  
   The amount of cache misses  
   Labels: hostname, env

- `gpc_cache_stale_total`  
   The amount of cache misses  
   Labels: hostname, env
