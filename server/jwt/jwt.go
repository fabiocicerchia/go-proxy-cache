package jwt

import (
	"encoding/json"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var co *config.Jwt

func errorJson(resp http.ResponseWriter, statuscode int, error *config.JwtError) {
	resp.WriteHeader(statuscode)
	resp.Header().Add("Content-Type", "application/json; charset=utf-8")
	json_error, _ := json.Marshal(error)
	resp.Write(json_error)
}

func logJWTErrorAndAbort(w http.ResponseWriter, err error) error {
	co.Logger.Info("Error jwt:", err)
	errorJson(w, http.StatusUnauthorized, &config.JwtError{ErrorCode: "JsonWebTokenError", ErrorDescription: err.Error()})

	return http.ErrAbortHandler
}

func ValidateJWT(w http.ResponseWriter, r *http.Request, keySet jwk.Set) error {
	token, err := jwt.ParseRequest(r,
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithTypedClaim("scope", json.RawMessage{}),
		jwt.WithTypedClaim("scp", json.RawMessage{}),
	)
	if err != nil {
		return logJWTErrorAndAbort(w, err)
	}
	if err := jwt.Validate(token); err != nil {
		return logJWTErrorAndAbort(w, err)
	}
	scopes := getScopes(token)
	haveAllowedScope := haveAllowedScope(scopes, co.AllowedScopes)
	if !haveAllowedScope {
		errorJson(w, http.StatusUnauthorized, &config.JwtError{ErrorCode: "InvalidScope", ErrorDescription: "Invalid Scope"})
		return http.ErrAbortHandler
	}

	return nil
}

func getKeySet(w http.ResponseWriter) (jwk.Set, error) {
	keySet, err := co.JwkCache.Get(co.Context, co.JwksUrl)
	if err != nil {
		return keySet, logJWTErrorAndAbort(w, err)
	}

	return keySet, nil
}

func JWTHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := handler.NewRequestCall(w, r)
		domainConfig, isDomainFound := config.DomainConf(r.Host, rc.GetScheme())
		if !isDomainFound {
			next.ServeHTTP(w, r)
			return
		}
		if !IsExcluded(domainConfig.Jwt.ExcludedPaths, r.URL.Path) {
			co = &domainConfig.Jwt
			keySet, err := getKeySet(w)
			if err != nil {
				return
			}
			err = ValidateJWT(w, r, keySet)
			if err != nil {
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func haveAllowedScope(scopes []string, allowedScopes []string) bool {
	if allowedScopes != nil {
		for _, s := range allowedScopes {
			isAllowed := slice.ContainsString(scopes, s)
			if isAllowed {
				return true
			}
		}
	}

	return false
}

func getScopes(token jwt.Token) []string {
	_, isScp := token.Get("scp")
	if isScp {
		scpInterface := token.PrivateClaims()["scp"]
		return extractScopes(scpInterface)
	}
	scopeInterface := token.PrivateClaims()["scope"]

	return extractScopes(scopeInterface)
}

func extractScopes(scopesInterface interface{}) []string {
	scpRaw, _ := scopesInterface.(json.RawMessage)
	scopes := []string{}
	json.Unmarshal(scpRaw, &scopes)

	return scopes
}
