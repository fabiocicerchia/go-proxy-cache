package k8s

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"k8s.io/apimachinery/pkg/labels"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// coreGroup - The empty API group, where Service and Secret live.
const coreGroup = ""

// reference - One object pointing at another, in the terms ReferenceGrant is
// written in.
type reference struct {
	FromGroup     string
	FromKind      string
	FromNamespace string

	ToGroup     string
	ToKind      string
	ToName      string
	ToNamespace string
}

// permitted - Whether a reference is allowed to cross a namespace boundary.
//
// Same-namespace references are always fine. A cross-namespace one needs a
// ReferenceGrant in the *target* namespace naming both sides, which is what
// stops a tenant pointing an HTTPRoute at a Service, or a Gateway at a TLS
// Secret, in a namespace they do not own.
func (c *Controller) permitted(ref reference) bool {
	if ref.FromNamespace == ref.ToNamespace {
		return true
	}

	if c.gatewayState == nil || c.gatewayState.grants == nil {
		return false
	}

	grants, err := c.gatewayState.grants.ReferenceGrants(ref.ToNamespace).List(labels.Everything())
	if err != nil {
		logger.GetGlobal().Warnf("Cannot list ReferenceGrants in %s: %s", ref.ToNamespace, err)

		return false
	}

	for _, grant := range grants {
		if grantAllows(grant.Spec.From, grant.Spec.To, ref) {
			return true
		}
	}

	return false
}

func grantAllows(from []gatewayv1.ReferenceGrantFrom, to []gatewayv1.ReferenceGrantTo, ref reference) bool {
	matchedFrom := false

	for i := range from {
		if string(from[i].Group) == ref.FromGroup &&
			string(from[i].Kind) == ref.FromKind &&
			string(from[i].Namespace) == ref.FromNamespace {
			matchedFrom = true

			break
		}
	}

	if !matchedFrom {
		return false
	}

	for i := range to {
		if string(to[i].Group) != ref.ToGroup || string(to[i].Kind) != ref.ToKind {
			continue
		}

		// An unset name covers every object of that kind in the namespace.
		if to[i].Name == nil || *to[i].Name == "" || string(*to[i].Name) == ref.ToName {
			return true
		}
	}

	return false
}

// refGroup - The group an optional reference field names, defaulting to core.
func refGroup(group *gatewayv1.Group) string {
	if group == nil {
		return coreGroup
	}

	return string(*group)
}

// refKind - The kind an optional reference field names, or the given default.
func refKind(kind *gatewayv1.Kind, fallback string) string {
	if kind == nil {
		return fallback
	}

	return string(*kind)
}

// refNamespace - The namespace an optional reference field names, defaulting to
// the referrer's own.
func refNamespace(namespace *gatewayv1.Namespace, fallback string) string {
	if namespace == nil {
		return fallback
	}

	return string(*namespace)
}
