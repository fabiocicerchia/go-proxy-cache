package router

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import "math/rand"

// SelectBackend - Picks one of the route's backends, honouring the weights an
// HTTPRoute can assign to them. Ingress rules only ever produce one backend,
// in which case this is a constant-time first-element pick.
//
// A backend whose endpoints all disappeared is skipped: routing to an empty
// backend would 502 every request that the weighted draw happened to send its
// way, even though a sibling backend is healthy.
func (r *Route) SelectBackend() (int, bool) {
	if len(r.Backends) == 0 {
		return 0, false
	}

	if len(r.Backends) == 1 {
		if len(r.Backends[0].Endpoints) == 0 {
			return 0, false
		}

		return 0, true
	}

	total := int32(0)

	for i := range r.Backends {
		if len(r.Backends[i].Endpoints) == 0 {
			continue
		}

		total += r.Backends[i].Weight
	}

	if total <= 0 {
		// Every weight is zero (or every backend is empty). Fall back to the
		// first backend that can actually serve.
		for i := range r.Backends {
			if len(r.Backends[i].Endpoints) > 0 {
				return i, true
			}
		}

		return 0, false
	}

	// #nosec G404 -- backend selection is load distribution, not a security
	// decision, so the cheaper non-cryptographic source is the right one.
	draw := rand.Int31n(total)

	for i := range r.Backends {
		if len(r.Backends[i].Endpoints) == 0 {
			continue
		}

		draw -= r.Backends[i].Weight
		if draw < 0 {
			return i, true
		}
	}

	return 0, true
}
