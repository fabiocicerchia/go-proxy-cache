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
	"crypto/tls"
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// certificateFromSecret - Parses a kubernetes.io/tls Secret into a usable
// certificate, along with every hostname it is valid for.
//
// The names are read from the leaf certificate's SANs (falling back to the
// Common Name for certificates old enough to lack them) so an Ingress that
// lists no hosts under spec.tls still gets its certificate served for the
// right SNI values -- including wildcards, which the store indexes separately.
func certificateFromSecret(secret *corev1.Secret) (*tls.Certificate, []string, error) {
	certPEM, ok := secret.Data[corev1.TLSCertKey]
	if !ok {
		return nil, nil, fmt.Errorf("secret has no %q key", corev1.TLSCertKey)
	}

	keyPEM, ok := secret.Data[corev1.TLSPrivateKeyKey]
	if !ok {
		return nil, nil, fmt.Errorf("secret has no %q key", corev1.TLSPrivateKeyKey)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, err
	}

	if len(cert.Certificate) == 0 {
		return nil, nil, fmt.Errorf("secret contains no certificate")
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	// Keeping the parsed leaf lets crypto/tls skip re-parsing it on every
	// handshake.
	cert.Leaf = leaf

	hosts := leaf.DNSNames
	if len(hosts) == 0 && leaf.Subject.CommonName != "" {
		hosts = []string{leaf.Subject.CommonName}
	}

	return &cert, hosts, nil
}
