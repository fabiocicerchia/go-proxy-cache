package response

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

// CacheStatusHeader - HTTP Header for showing cache status.
const CacheStatusHeader = "X-Go-Proxy-Cache-Status"

// CacheBypassHeader - HTTP Header for showing cache status.
const CacheBypassHeader = "X-Go-Proxy-Cache-Force-Fresh" //#nosec G101

// CacheStatusHeaderHit - Cache status HIT for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderHit = "HIT"

// CacheStatusHeaderMiss - Cache status MISS for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderMiss = "MISS"

// CacheStatusHeaderStale - Cache status STALE for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderStale = "STALE"
