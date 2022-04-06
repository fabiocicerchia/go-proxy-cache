package cache

// StatusHit - Value for HIT.
const StatusHit = 1

// StatusMiss - Value for MISS.
const StatusMiss = 0

// StatusStale - Value for STALE.
const StatusStale = -1

// StatusNA - Value for Not Applicable.
const StatusNA = -2

// StatusLabel - Labels used for displaying HIT/MISS based on cache usage.
var StatusLabel = map[int]string{
	StatusHit:   "HIT",
	StatusMiss:  "MISS",
	StatusStale: "STALE",
	StatusNA:    "-",
}
