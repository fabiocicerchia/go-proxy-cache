@domain:operations @area:build-release @from:bin/build-multiarch
Feature: Publishing prebuilt release binaries
  Operators download a ready-to-run go-proxy-cache binary for their operating system and CPU from each release instead of compiling it themselves.

  Background:
    Given a tagged go-proxy-cache release is being built

  @id:operations.build-release.001 @rule:release.binaries.supported-platforms @inferred @source:bin/build-multiarch.sh#platforms
  Scenario Outline: Each release offers a binary for every supported operating system and CPU
    Given the source builds for every supported platform
    When the maintainer builds the release binaries
    Then the release assets include a binary named "<asset>"
    And that binary runs on <os> with a <arch> CPU
    And no Windows binary is offered

    Examples:
      | asset                        | os      | arch  |
      | go-proxy-cache-freebsd-amd64 | FreeBSD | amd64 |
      | go-proxy-cache-linux-amd64   | Linux   | amd64 |
      | go-proxy-cache-linux-arm64   | Linux   | arm64 |
      | go-proxy-cache-darwin-amd64  | macOS   | amd64 |
      | go-proxy-cache-darwin-arm64  | macOS   | arm64 |

  @id:operations.build-release.002 @rule:release.binaries.all-or-nothing @inferred @source:bin/build-multiarch.sh#platforms
  Scenario: A release is not published when any platform binary fails to build
    Given the source fails to build for one supported platform
    When the maintainer builds the release binaries
    Then the build reports "An error has occurred! Aborting..."
    And the build stops with a failure exit status
    And no release is published with a partial set of binaries
