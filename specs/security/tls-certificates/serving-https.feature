@domain:security @area:tls-certificates @from:server/tls
Feature: Serving HTTPS for hosted domains
  Lets operators serve each hosted domain over HTTPS with their own certificate files or with certificates issued automatically by an ACME authority such as Let's Encrypt, under a secure TLS policy.

  Background:
    Given the proxy hosts the domain "www.example.com" on HTTPS port 443

  @id:security.tls-certificates.002 @rule:security.tls.certificate-by-server-name @inferred @source:server/tls/tls.go#returnCert
  Scenario Outline: Each domain is served with its own certificate
    Given the domain "www.example.com" is configured with its own certificate and key files
    And the domain "shop.example.com" is configured with its own certificate and key files on the same HTTPS port
    When a client opens a TLS connection asking for the server name "<server name>"
    Then the proxy presents the certificate configured for "<server name>"

    Examples:
      | server name      |
      | www.example.com  |
      | shop.example.com |

  @id:security.tls-certificates.003 @rule:security.tls.unknown-server-name-refused @inferred @source:server/tls/tls.go#returnCert @questionable
  Scenario Outline: A connection for a server name with no configured certificate is refused
    Given only "www.example.com" has a certificate configured
    When a client opens a TLS connection asking for <requested name>
    Then the TLS handshake fails
    And no certificate is presented to the client

    Examples:
      | requested name                                         |
      | the server name "unknown.example.net"                  |
      | no server name (connecting by IP address 203.0.113.10) |

  @id:security.tls-certificates.004 @rule:security.tls.https-requires-certificate-pair @inferred @source:server/tls/tls.go#Config
  Scenario Outline: A domain without a usable certificate and key gets no HTTPS listener
    Given automatic certificates are disabled for "www.example.com"
    And <certificate setup>
    When the operator starts the proxy
    Then the log shows "Skipping 'www.example.com' TLS server configuration: <reason>"
    And the log shows "No HTTPS server will be listening on 'www.example.com'"
    And "www.example.com" is still served over plain HTTP

    Examples:
      | certificate setup                      | reason                                     |
      | no certificate file is configured      | missing certificate file and/or key file   |
      | no key file is configured              | missing certificate file and/or key file   |
      | the certificate file does not exist    | open <path>: no such file or directory     |
      | the key does not match the certificate | tls: private key does not match public key |

  @id:security.tls-certificates.005 @rule:security.tls.auto-certificates-only-for-configured-domains @inferred @source:server/tls/tls.go#InitCertManager
  Scenario Outline: Automatic certificates are issued only for configured domains
    Given automatic certificates are enabled for "www.example.com" with contact email "ops@example.com"
    When a client opens a TLS connection asking for the server name "<server name>"
    Then <outcome>

    Examples:
      | server name      | outcome                                                                                                                                                       |
      | www.example.com  | a certificate for "www.example.com" is obtained from the ACME authority, registered under "ops@example.com" with its terms of service accepted, and presented |
      | evil.example.net | no certificate is requested and the handshake fails                                                                                                           |

  @id:security.tls-certificates.006 @rule:security.tls.auto-certificates-persist-across-restarts @inferred @source:server/tls/tls.go#InitCertManager
  Scenario Outline: Automatically issued certificates survive a restart
    Given automatic certificates are enabled for "www.example.com" with certificate cache directory "<cache dir>"
    And a certificate for "www.example.com" was already issued
    When the operator restarts the proxy
    Then the previously issued certificate is presented again without a new issuance
    And the certificates are kept in "<stored in>", accessible only to the proxy's user

    Examples:
      | cache dir                     | stored in                             |
      | /var/lib/go-proxy-cache/certs | /var/lib/go-proxy-cache/certs         |
      | (not set)                     | <OS temp dir>/go-proxy-cache-autocert |

  @id:security.tls-certificates.007 @rule:security.tls.protocol-version-and-cipher-policy @inferred @source:server/tls/tls.go#newDefaultTLSConfig
  Scenario Outline: HTTPS connections follow the configured TLS version and cipher policy
    Given the global TLS override allows versions <min> to <max>
    And the global TLS override lists the permitted cipher suites
    When a client offering only <client offer> connects to "www.example.com" over HTTPS
    Then <outcome>

    Examples:
      | min               | max               | client offer                                       | outcome                                           |
      | TLS 1.2 (default) | TLS 1.3 (default) | TLS 1.1                                            | the handshake fails with a protocol version alert |
      | TLS 1.2 (default) | TLS 1.3 (default) | TLS 1.2 with TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 | the connection is established with TLS 1.2        |
      | TLS 1.2 (default) | TLS 1.3 (default) | TLS 1.2 with only TLS_RSA_WITH_AES_128_CBC_SHA     | the handshake fails                               |

  @id:security.tls-certificates.008 @rule:security.tls.per-domain-tls-policy-ignored @inferred @source:server/tls/tls.go#newDefaultTLSConfig @questionable
  Scenario: A domain's own TLS version policy is not applied; the global policy is used
    Given the global TLS override sets the minimum version to TLS 1.2
    And the "www.example.com" domain settings set its TLS override minimum version to TLS 1.3
    When a client offering only TLS 1.2 connects to "www.example.com" over HTTPS
    Then the connection is established with TLS 1.2
