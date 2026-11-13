@domain:operations @area:container-image @from:docker/dockerfile
Feature: Running the proxy from its container image
  Operators deploy go-proxy-cache as a Docker image and rely on it to start safely with sensible defaults and to report its own health to the container runtime.

  Background:
    Given the operator runs the go-proxy-cache container image

  @id:operations.container-image.001 @rule:operations.container-image.runs-unprivileged-on-standard-ports @inferred @source:docker/Dockerfile#USER gpc
  Scenario: The container runs as an unprivileged user yet still serves on ports 80 and 443
    Given no user override is set for the container
    When the operator starts the container
    Then the proxy process runs as user "gpc" with uid 10001, not as root
    And the proxy can still bind the privileged ports 80 (HTTP) and 443 (HTTPS, once a certificate or auto TLS is configured)

  @id:operations.container-image.002 @rule:operations.container-image.health-follows-healthcheck-endpoint @inferred @source:docker/Dockerfile#HEALTHCHECK
  Scenario Outline: The container is reported unhealthy when the proxy's health check stops answering
    Given the container has been up for longer than its 10 second start period
    And the proxy's /healthcheck endpoint <state>
    When the container runtime probes the container
    Then the container status becomes "<status>"

    Examples:
      | state                                            | status    |
      | answers on 127.0.0.1:52021                       | healthy   |
      | fails 3 probes in a row, 30 seconds apart        | unhealthy |
      | takes longer than 5 seconds on 3 probes in a row | unhealthy |

  @id:operations.container-image.003 @rule:operations.container-image.health-probe-fixed-port @inferred @source:docker/Dockerfile#HEALTHCHECK @questionable
  Scenario: The container health probe ignores a changed internal port
    Given the operator configures the internal listening port as 52022
    And the proxy is up and answers /healthcheck on 127.0.0.1:52022
    When the container runtime probes the container
    Then the probe calls http://127.0.0.1:52021/healthcheck and gets no answer
    And after 3 failed probes the container status becomes "unhealthy"
