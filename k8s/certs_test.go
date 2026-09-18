//go:build all || unit
// +build all unit

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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateSelfSigned - A throwaway certificate and key in PEM form.
func generateSelfSigned(t *testing.T, hosts ...string) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("cannot generate key: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: hosts[0]},
		DNSNames:     hosts,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cannot create certificate: %s", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certPEM, keyPEM
}

func tlsSecret(certPEM []byte, keyPEM []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "tls"},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPEM,
			corev1.TLSPrivateKeyKey: keyPEM,
		},
	}
}

func TestCertificateFromSecretReadsSANs(t *testing.T) {
	certPEM, keyPEM := generateSelfSigned(t, "a.example.com", "*.example.org")

	cert, hosts, err := certificateFromSecret(tlsSecret(certPEM, keyPEM))

	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotNil(t, cert.Leaf, "the parsed leaf is kept so handshakes do not re-parse it")
	assert.Equal(t, []string{"a.example.com", "*.example.org"}, hosts)
}

func TestCertificateFromSecretMissingKeys(t *testing.T) {
	certPEM, keyPEM := generateSelfSigned(t, "a.example.com")

	noCert := tlsSecret(certPEM, keyPEM)
	delete(noCert.Data, corev1.TLSCertKey)
	_, _, err := certificateFromSecret(noCert)
	assert.Error(t, err)

	noKey := tlsSecret(certPEM, keyPEM)
	delete(noKey.Data, corev1.TLSPrivateKeyKey)
	_, _, err = certificateFromSecret(noKey)
	assert.Error(t, err)
}

func TestCertificateFromSecretRejectsGarbage(t *testing.T) {
	_, _, err := certificateFromSecret(tlsSecret([]byte("not a certificate"), []byte("not a key")))

	assert.Error(t, err)
}
