@domain:security @area:jwt-auth @from:server/jwt
Feature: Authenticating requests with JWT bearer tokens
  Lets an operator protect a hosted domain so only clients presenting a valid, signed token with an allowed scope reach the upstream, while chosen public paths stay open.

  Background:
    Given the operator has configured a JWKS URL for the domain example.com
    And the operator allows the scope "scope1" for example.com

  @id:security.jwt-auth.001 @rule:security.jwt.valid-token-with-allowed-scope-passes @verified @source:server/jwt/jwt.go#ValidateJWT @test:server/jwt/jwt_test.go#TestValidateJWTWithScopeConfigAndScopeClaimToken @test:server/jwt/jwt_test.go#TestValidateJWTWithScopeConfigAndScpClaimToken @test:server/jwt/jwt_test.go#TestValidateJWTWithMultipleKeysInKeySet
  Scenario Outline: A signed, unexpired token carrying an allowed scope is let through
    Given the JWKS published at the configured URL contains the key that signed the token
    And the client holds an unexpired token whose "<claim>" claim lists scope1, scope2 and scope3
    When the client sends GET https://example.com/foo with the token as a Bearer Authorization header
    Then the request is forwarded to the upstream
    And the client receives the upstream's response, not a 401

    Examples:
      | claim |
      | scope |
      | scp   |

  @id:security.jwt-auth.002 @rule:security.jwt.missing-or-invalid-token-refused @verified @source:server/jwt/jwt.go#logJWTErrorAndAbort @test:server/jwt/jwt_test.go#TestValidateJWTWithoutAnyToken @test:server/jwt/jwt_test.go#TestValidateJWTWithAnExpiredToken
  Scenario Outline: A request without a valid token is refused with 401
    Given the client <token_state>
    When the client sends GET https://example.com/foo
    Then the client receives status 401 with Content-Type "application/json; charset=utf-8"
    And the body has errorCode "JsonWebTokenError"
    And the body's errorDescription contains "<reason>"
    And the request is not forwarded to the upstream

    Examples:
      | token_state                                   | reason                                                      |
      | sends no Authorization header                 | failed to find a valid token in any location of the request |
      | sends a token whose exp time has passed       | "exp" not satisfied                                         |
      | sends a token signed by a key not in the JWKS | could not verify message                                    |

  @id:security.jwt-auth.003 @rule:security.jwt.scope-not-allowed-refused @verified @source:server/jwt/jwt.go#ValidateJWT @test:server/jwt/jwt_test.go#TestValidateJWTWithoutAnyScopeInTheConfig @test:server/jwt/jwt_test.go#TestAllowedScope @questionable
  Scenario Outline: A valid token without any allowed scope is refused with 401 Invalid Scope
    Given the operator allows the scopes <allowed_scopes> for example.com
    And the client holds a valid signed token whose scope claim is <token_scopes>
    When the client sends GET https://example.com/foo with the token as a Bearer Authorization header
    Then the client receives status 401
    And the body is {"errorCode":"InvalidScope","errorDescription":"Invalid Scope"}
    And the request is not forwarded to the upstream

    Examples:
      | allowed_scopes  | token_scopes                         |
      | "admin"         | ["scope1", "scope2", "scope3"]       |
      | "admin"         | absent                               |
      | "admin"         | the string "admin" instead of a list |
      | none configured | ["scope1", "scope2", "scope3"]       |

  @id:security.jwt-auth.004 @rule:security.jwt.scp-claim-overrides-scope-claim @inferred @source:server/jwt/jwt.go#getScopes
  Scenario: When a token carries both scp and scope, only scp grants access
    Given the operator allows the scope "admin" for example.com
    And the client holds a valid signed token with scp ["read"] and scope ["admin"]
    When the client sends GET https://example.com/foo with the token as a Bearer Authorization header
    Then the client receives status 401 with errorCode "InvalidScope"

  @id:security.jwt-auth.005 @rule:security.jwt.key-set-unavailable-fails-closed @verified @source:server/jwt/jwt.go#getKeySet @test:server/jwt/jwt_test.go#TestValidateJWTWithoutAnyKeySet
  Scenario: Requests are refused when the signing keys cannot be obtained
    Given the configured JWKS URL returns an empty or unreadable response
    When a client sends GET https://example.com/foo with a Bearer token
    Then the client receives status 401 with errorCode "JsonWebTokenError"
    And the errorDescription explains the key set could not be loaded, e.g. "failed to unmarshal JWK set: EOF"
    And the request is not forwarded to the upstream

  @id:security.jwt-auth.006 @rule:security.jwt.excluded-paths-skip-authentication @verified @source:server/jwt/jwt_utils.go#IsExcluded @test:server/jwt/jwt_utils_test.go#TestIsExcluded @questionable
  Scenario Outline: Paths matching an excluded pattern are served without a token
    Given the operator excludes the path pattern "<pattern>" for example.com
    When a client without any token sends GET https://example.com<path>
    Then <outcome>

    Examples:
      | pattern          | path              | outcome                                                           |
      | ^/public/[0-9]+$ | /public/1234      | the request is forwarded to the upstream                          |
      | ^/public/[0-9]+$ | /public/1234/edit | the client receives status 401 with errorCode "JsonWebTokenError" |
      | /healthcheck     | /healthcheck      | the request is forwarded to the upstream                          |
      | /a               | /admin/a/secret   | the request is forwarded to the upstream                          |

  @id:security.jwt-auth.007 @rule:security.jwt.only-enforced-with-jwks-url @inferred @source:server/jwt/jwt.go#JWTHandler @questionable
  Scenario Outline: Domains without a JWKS URL are not token-protected
    Given the request's host <host_state>
    When a client without any token sends GET to that host
    Then no token check is applied and <outcome>

    Examples:
      | host_state                              | outcome                                             |
      | is a configured domain with no JWKS URL | the request is forwarded to the upstream            |
      | matches no configured domain            | the client receives status 501 instead of a 401     |
