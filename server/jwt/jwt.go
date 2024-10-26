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

func errorJson(resp http.ResponseWriter, statuscode int, error *config.JwtError) {
	resp.WriteHeader(statuscode)
	resp.Header().Add("Content-Type", "application/json; charset=utf-8")
	json_error, _ := json.Marshal(error)
	resp.Write(json_error)
}

func logJWTErrorAndAbort(w http.ResponseWriter, err error, jwtConfig *config.Jwt) error {
	jwtConfig.Logger.Info("Error jwt:", err)
	errorJson(w, http.StatusUnauthorized, &config.JwtError{ErrorCode: "JsonWebTokenError", ErrorDescription: err.Error()})

	return http.ErrAbortHandler
}

func ValidateJWT(w http.ResponseWriter, r *http.Request, keySet jwk.Set, jwtConfig *config.Jwt) error {
	token, err := jwt.ParseRequest(r,
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithTypedClaim("scope", json.RawMessage{}),
		jwt.WithTypedClaim("scp", json.RawMessage{}),
	)
	if err != nil {
		return logJWTErrorAndAbort(w, err, jwtConfig)
	}

	if err := jwt.Validate(token); err != nil {
		return logJWTErrorAndAbort(w, err, jwtConfig)
	}

	scopes := getScopes(token)
	haveAllowedScope := haveAllowedScope(scopes, jwtConfig.AllowedScopes)
	if !haveAllowedScope {
		errorJson(w, http.StatusUnauthorized, &config.JwtError{ErrorCode: "InvalidScope", ErrorDescription: "Invalid Scope"})
		return http.ErrAbortHandler
	}

	return nil
}

func getKeySet(w http.ResponseWriter, jwtConfig *config.Jwt) (jwk.Set, error) {
	keySet, err := jwtConfig.JwkCache.Get(jwtConfig.Context, jwtConfig.JwksUrl)
	if err != nil {
		return keySet, logJWTErrorAndAbort(w, err, jwtConfig)
	}

	return keySet, nil
}

func JWTHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := handler.NewRequestCall(w, r)
		domainConfig, isDomainFound := config.DomainConf(r.Host, rc.GetScheme())

		if isDomainFound && !IsExcluded(domainConfig.Jwt.ExcludedPaths, r.URL.Path) {
			keySet, err := getKeySet(w, &domainConfig.Jwt)
			if err != nil {
				return
			}

			err = ValidateJWT(w, r, keySet, &domainConfig.Jwt)
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
