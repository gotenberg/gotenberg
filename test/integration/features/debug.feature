@debug
Feature: /debug

  Scenario: GET /debug (Disabled)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/debug" endpoint
    Then the response status code should be 404

  Scenario: GET /debug (Enabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true |
    When I make a "GET" request to Gotenberg at the "/debug" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "version": "{version}",
        "timezone": "UTC",
        "architecture": "ignore",
        "modules": [
          "api",
          "chromium",
          "exiftool",
          "libreoffice",
          "libreoffice-api",
          "libreoffice-pdfengine",
          "pdfcpu",
          "pdfengines",
          "pdftk",
          "prometheus",
          "qpdf",
          "webhook"
        ],
        "modules_additional_data": {
          "chromium": {
            "version": "ignore"
          },
          "exiftool": {
            "version": "ignore"
          },
          "libreoffice-api": {
            "version": "ignore"
          },
          "pdfcpu": {
            "version": "ignore"
          },
          "pdftk": {
            "version": "ignore"
          },
          "qpdf": {
            "version": "ignore"
          }
        },
        "flags": {
          "api-bind-ip": "",
          "api-body-limit": "",
          "api-correlation-id-header": "Gotenberg-Trace",
          "api-disable-download-from": "false",
          "api-disable-health-check-logging": "false",
          "api-disable-debug-route-telemetry": "true",
          "api-disable-health-check-route-telemetry": "true",
          "api-disable-root-route-telemetry": "true",
          "api-disable-version-route-telemetry": "true",
          "api-download-from-allow-list": "[]",
          "api-download-from-deny-list": "[^https?://(10\\.|172\\.(1[6-9]|2[0-9]|3[01])\\.|192\\.168\\.|169\\.254\\.|0\\.0\\.0\\.0|127\\.|localhost|\\[::1\\]|\\[fd)]",
          "api-download-from-max-retry": "4",
          "api-enable-basic-auth": "false",
          "api-enable-debug-route": "true",
          "api-port": "3000",
          "api-port-from-env": "",
          "api-root-path": "/",
          "api-start-timeout": "30s",
          "api-timeout": "30s",
          "api-tls-cert-file": "",
          "api-tls-key-file": "",
          "api-trace-header": "Gotenberg-Trace",
          "chromium-allow-file-access-from-files": "false",
          "chromium-allow-insecure-localhost": "false",
          "chromium-allow-list": "[]",
          "chromium-auto-start": "false",
          "chromium-clear-cache": "false",
          "chromium-clear-cookies": "false",
          "chromium-deny-list": "[^file:(?!//\\/tmp/).*]",
          "chromium-disable-javascript": "false",
          "chromium-disable-routes": "false",
          "chromium-disable-web-security": "false",
          "chromium-host-resolver-rules": "",
          "chromium-ignore-certificate-errors": "false",
          "chromium-idle-shutdown-timeout": "0s",
          "chromium-incognito": "false",
          "chromium-max-concurrency": "6",
          "chromium-max-queue-size": "0",
          "chromium-proxy-server": "",
          "chromium-restart-after": "100",
          "chromium-start-timeout": "20s",
          "gotenberg-build-debug-data": "true",
          "gotenberg-graceful-shutdown-duration": "30s",
          "libreoffice-auto-start": "false",
          "libreoffice-disable-routes": "false",
          "libreoffice-idle-shutdown-timeout": "0s",
          "libreoffice-max-queue-size": "0",
          "libreoffice-restart-after": "10",
          "libreoffice-start-timeout": "20s",
          "log-enable-gcp-fields": "false",
          "log-fields-prefix": "",
          "log-format": "auto",
          "log-level": "info",
          "log-std-enable-gcp-fields": "false",
          "log-std-format": "auto",
          "pdfengines-convert-engines": "[libreoffice-pdfengine]",
          "pdfengines-disable-routes": "false",
          "pdfengines-engines": "[]",
          "pdfengines-flatten-engines": "[qpdf]",
          "pdfengines-merge-engines": "[qpdf,pdfcpu,pdftk]",
          "pdfengines-read-bookmarks-engines": "[pdfcpu]",
          "pdfengines-read-metadata-engines": "[exiftool]",
          "pdfengines-split-engines": "[pdfcpu,qpdf,pdftk]",
          "pdfengines-write-bookmarks-engines": "[pdfcpu]",
          "pdfengines-write-metadata-engines": "[exiftool]",
          "prometheus-collect-interval": "1s",
          "prometheus-disable-collect": "false",
          "prometheus-disable-route-logging": "false",
          "prometheus-disable-route-telemetry": "true",
          "prometheus-namespace": "gotenberg",
          "prometheus-metrics-path": "/prometheus/metrics",
          "webhook-allow-list": "[]",
          "webhook-client-timeout": "30s",
          "webhook-deny-list": "[^https?://(10\\.|172\\.(1[6-9]|2[0-9]|3[01])\\.|192\\.168\\.|169\\.254\\.|0\\.0\\.0\\.0|127\\.|localhost|\\[::1\\]|\\[fd)]",
          "webhook-disable": "false",
          "webhook-error-allow-list": "[]",
          "webhook-error-deny-list": "[]",
          "webhook-max-retry": "4",
          "webhook-retry-max-wait": "30s",
          "webhook-retry-min-wait": "1s"
        }
      }
      """

  Scenario: GET /debug (Environment based timezone)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true             |
      | TZ                     | America/New_York |
    When I make a "GET" request to Gotenberg at the "/debug" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "version": "{version}",
        "timezone": "America/New_York",
        "architecture": "ignore",
        "modules": [
          "api",
          "chromium",
          "exiftool",
          "libreoffice",
          "libreoffice-api",
          "libreoffice-pdfengine",
          "pdfcpu",
          "pdfengines",
          "pdftk",
          "prometheus",
          "qpdf",
          "webhook"
        ],
        "modules_additional_data": {
          "chromium": {
            "version": "ignore"
          },
          "exiftool": {
            "version": "ignore"
          },
          "libreoffice-api": {
            "version": "ignore"
          },
          "pdfcpu": {
            "version": "ignore"
          },
          "pdftk": {
            "version": "ignore"
          },
          "qpdf": {
            "version": "ignore"
          }
        },
        "flags": {
          "api-bind-ip": "",
          "api-body-limit": "",
          "api-correlation-id-header": "Gotenberg-Trace",
          "api-disable-download-from": "false",
          "api-disable-health-check-logging": "false",
          "api-disable-debug-route-telemetry": "true",
          "api-disable-health-check-route-telemetry": "true",
          "api-disable-root-route-telemetry": "true",
          "api-disable-version-route-telemetry": "true",
          "api-download-from-allow-list": "[]",
          "api-download-from-deny-list": "[^https?://(10\\.|172\\.(1[6-9]|2[0-9]|3[01])\\.|192\\.168\\.|169\\.254\\.|0\\.0\\.0\\.0|127\\.|localhost|\\[::1\\]|\\[fd)]",
          "api-download-from-max-retry": "4",
          "api-enable-basic-auth": "false",
          "api-enable-debug-route": "true",
          "api-port": "3000",
          "api-port-from-env": "",
          "api-root-path": "/",
          "api-start-timeout": "30s",
          "api-timeout": "30s",
          "api-tls-cert-file": "",
          "api-tls-key-file": "",
          "api-trace-header": "Gotenberg-Trace",
          "chromium-allow-file-access-from-files": "false",
          "chromium-allow-insecure-localhost": "false",
          "chromium-allow-list": "[]",
          "chromium-auto-start": "false",
          "chromium-clear-cache": "false",
          "chromium-clear-cookies": "false",
          "chromium-deny-list": "[^file:(?!//\\/tmp/).*]",
          "chromium-disable-javascript": "false",
          "chromium-disable-routes": "false",
          "chromium-disable-web-security": "false",
          "chromium-host-resolver-rules": "",
          "chromium-ignore-certificate-errors": "false",
          "chromium-idle-shutdown-timeout": "0s",
          "chromium-incognito": "false",
          "chromium-max-queue-size": "0",
          "chromium-max-concurrency": "6",
          "chromium-proxy-server": "",
          "chromium-restart-after": "100",
          "chromium-start-timeout": "20s",
          "gotenberg-build-debug-data": "true",
          "gotenberg-graceful-shutdown-duration": "30s",
          "libreoffice-auto-start": "false",
          "libreoffice-disable-routes": "false",
          "libreoffice-idle-shutdown-timeout": "0s",
          "libreoffice-max-queue-size": "0",
          "libreoffice-restart-after": "10",
          "libreoffice-start-timeout": "20s",
          "log-enable-gcp-fields": "false",
          "log-fields-prefix": "",
          "log-format": "auto",
          "log-level": "info",
          "log-std-enable-gcp-fields": "false",
          "log-std-format": "auto",
          "pdfengines-convert-engines": "[libreoffice-pdfengine]",
          "pdfengines-disable-routes": "false",
          "pdfengines-engines": "[]",
          "pdfengines-flatten-engines": "[qpdf]",
          "pdfengines-merge-engines": "[qpdf,pdfcpu,pdftk]",
          "pdfengines-read-bookmarks-engines": "[pdfcpu]",
          "pdfengines-read-metadata-engines": "[exiftool]",
          "pdfengines-split-engines": "[pdfcpu,qpdf,pdftk]",
          "pdfengines-write-bookmarks-engines": "[pdfcpu]",
          "pdfengines-write-metadata-engines": "[exiftool]",
          "prometheus-collect-interval": "1s",
          "prometheus-disable-collect": "false",
          "prometheus-disable-route-logging": "false",
          "prometheus-disable-route-telemetry": "true",
          "prometheus-namespace": "gotenberg",
          "prometheus-metrics-path": "/prometheus/metrics",
          "webhook-allow-list": "[]",
          "webhook-client-timeout": "30s",
          "webhook-deny-list": "[^https?://(10\\.|172\\.(1[6-9]|2[0-9]|3[01])\\.|192\\.168\\.|169\\.254\\.|0\\.0\\.0\\.0|127\\.|localhost|\\[::1\\]|\\[fd)]",
          "webhook-disable": "false",
          "webhook-error-allow-list": "[]",
          "webhook-error-deny-list": "[]",
          "webhook-max-retry": "4",
          "webhook-retry-max-wait": "30s",
          "webhook-retry-min-wait": "1s"
        }
      }
      """

  Scenario: GET /debug (No Debug Data)
    Given I have a Gotenberg container with the following environment variable(s):
      | GOTENBERG_BUILD_DEBUG_DATA | false |
      | API_ENABLE_DEBUG_ROUTE     | true  |
    When I make a "GET" request to Gotenberg at the "/debug" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "version": "",
        "timezone": "",
        "architecture": "",
        "modules": null,
        "modules_additional_data": null,
        "flags": null
      }
      """

  Scenario: GET /debug (Gotenberg Trace)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE            | true  |
      | API_DISABLE_DEBUG_ROUTE_TELEMETRY | false |
    When I make a "GET" request to Gotenberg at the "/debug" endpoint with the following header(s):
      | Gotenberg-Trace | debug |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "debug"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"debug" |

  Scenario: GET /debug (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE            | true |
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/debug" endpoint
    Then the response status code should be 401

  Scenario: GET /foo/debug (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/debug" endpoint
    Then the response status code should be 200
