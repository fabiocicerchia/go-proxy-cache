package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	random "math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func GenerateTestJWT(key jwk.Key, scope string, isExpired bool) (string, error) {
	claims := jwt.New()
	claims.Set(scope, []string{"scope1", "scope2", "scope3"})
	if isExpired {
		claims.Set(jwt.ExpirationKey, time.Now())
	} else {
		claims.Set(jwt.ExpirationKey, time.Now().Add(1*time.Hour))
	}
	claims.Set(jwt.IssuerKey, "issuer")
	claims.Set(jwt.AudienceKey, "audience")
	claims.Set(jwt.NotBeforeKey, time.Now().Add(-1*time.Minute))
	claims.Set(jwt.IssuedAtKey, time.Now())
	claims.Set(jwt.JwtIDKey, "jti")

	token, err := jwt.Sign(claims, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return "", err
	}

	return string(token), nil
}

func generateTestJWKSingleKey(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, keyID string) (jwk.Key, jwk.Set, error) {
	jwkKeySingle, _ := jwk.FromRaw(privateKey)
	jwkKeySingle.Set("kid", keyID)
	key, err := jwk.FromRaw(publicKey)
	if err != nil {
		return jwkKeySingle, nil, err
	}
	key.Set(jwk.KeyIDKey, keyID)
	key.Set(jwk.KeyUsageKey, jwk.ForSignature)
	key.Set(jwk.AlgorithmKey, "RS256")
	jwks := jwk.NewSet()
	err = jwks.AddKey(key)
	if err != nil {
		return jwkKeySingle, nil, err
	}

	return jwkKeySingle, jwks, nil
}

func generateTestJWKMultipleKeys(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, keyID string, numberOfMultiples int) (jwk.Key, jwk.Set, error) {
	jwkKeyMultiple, jwks, _ := generateTestJWKSingleKey(privateKey, publicKey, keyID)
	for i := 0; i < numberOfMultiples; i++ {
		newKey, err := jwk.FromRaw(publicKey)
		if err != nil {
			return jwkKeyMultiple, nil, err
		}
		newKey.Set(jwk.KeyIDKey, keyID+"."+strconv.Itoa(random.Intn(1000)))
		newKey.Set(jwk.KeyUsageKey, jwk.ForSignature)
		newKey.Set(jwk.AlgorithmKey, "RS256")
		err = jwks.AddKey(newKey)
		if err != nil {
			return jwkKeyMultiple, nil, err
		}
	}

	return jwkKeyMultiple, jwks, nil
}

func GenerateTestKeysAndKeySets() (jwk.Key, jwk.Key, []byte, []byte) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	jwkKeySingle, jwkKeySetSingle, _ := generateTestJWKSingleKey(privateKey, publicKey, "key-id-single")
	jwkKeyMultiple, jwkKeySetMultiple, _ := generateTestJWKMultipleKeys(privateKey, publicKey, "key-id-multiple", 1)
	jsonJWKKeySetSingle, _ := json.Marshal(jwkKeySetSingle)
	jsonJWKKeySetMultiple, _ := json.Marshal(jwkKeySetMultiple)

	return jwkKeySingle, jwkKeyMultiple, jsonJWKKeySetSingle, jsonJWKKeySetMultiple
}

func CreateTestServer(t *testing.T, jsonJWKKeySetSingle []byte, jsonJWKKeySetMultiple []byte, port int) *httptest.Server {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("Got connection!")
		switch r.URL.String() {
		case "/.well-known-multiple/jwks.json":
			w.Write([]byte(jsonJWKKeySetMultiple))
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			break
		case "/.well-known-single/jwks.json":
			w.Write([]byte(jsonJWKKeySetSingle))
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			break

		case "/.bad-known/jwks.json":
			break
		default:
			t.Fatalf("Unknown request:" + r.URL.String())
		}
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Log("Error while configuring the server listener")
	}
	ts.Listener = listener
	ts.Start()

	return ts
}
