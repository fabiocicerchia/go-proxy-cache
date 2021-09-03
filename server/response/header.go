package response

// CacheStatusHeader - HTTP Header for showing cache status.
const CacheStatusHeader = "X-Go-Proxy-Cache-Status"

// CacheStatusHeader - HTTP Header for showing cache status.
const CacheBypassHeader = "X-Go-Proxy-Cache-Force-Fresh"

// CacheStatusHeaderHit - Cache status HIT for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderHit = "HIT"

// CacheStatusHeaderMiss - Cache status MISS for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderMiss = "MISS"

// CacheStatusHeaderStale - Cache status STALE for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderStale = "STALE"
