# TODO:
# 1. Count restarts.
# 2. Count queue size.

@prometheus-metrics
Feature: /prometheus/metrics

  Scenario: GET /prometheus/metrics (Enabled)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; version=0.0.4; charset=utf-8; escaping=underscores"
    Then the response body should contain string:
      """
      # HELP chromium_process_restarts_total Current number of Chromium restarts.
      """
    Then the response body should contain string:
      """
      # TYPE chromium_process_restarts_total counter
      """
    Then the response body should contain string:
      """
      # HELP chromium_requests_queue_size Current number of Chromium conversion requests waiting to be treated.
      """
    Then the response body should contain string:
      """
      # TYPE chromium_requests_queue_size gauge
      """
    Then the response body should contain string:
      """
      # HELP libreoffice_process_restarts_total Current number of LibreOffice restarts.
      """
    Then the response body should contain string:
      """
      # TYPE libreoffice_process_restarts_total counter
      """
    Then the response body should contain string:
      """
      # HELP libreoffice_requests_queue_size Current number of LibreOffice conversion requests waiting to be treated.
      """
    Then the response body should contain string:
      """
      # TYPE libreoffice_requests_queue_size gauge
      """
    Then the response body should contain string:
      """
      # HELP http_server_request_duration_seconds Duration of HTTP server requests.
      """
    Then the response body should contain string:
      """
      # TYPE http_server_request_duration_seconds histogram
      """
    Then the Gotenberg container should log the following entries:
      | "path":"/prometheus/metrics" |

  Scenario: GET /custom/metrics (Custom Metrics Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | PROMETHEUS_METRICS_PATH | /custom/metrics |
    When I make a "GET" request to Gotenberg at the "/custom/metrics" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; version=0.0.4; charset=utf-8; escaping=underscores"
    Then the response body should contain string:
      """
      # HELP chromium_process_restarts_total Current number of Chromium restarts.
      """
    Then the response body should contain string:
      """
      # TYPE chromium_process_restarts_total counter
      """
    Then the response body should contain string:
      """
      # HELP chromium_requests_queue_size Current number of Chromium conversion requests waiting to be treated.
      """
    Then the response body should contain string:
      """
      # TYPE chromium_requests_queue_size gauge
      """
    Then the response body should contain string:
      """
      # HELP libreoffice_process_restarts_total Current number of LibreOffice restarts.
      """
    Then the response body should contain string:
      """
      # TYPE libreoffice_process_restarts_total counter
      """
    Then the response body should contain string:
      """
      # HELP libreoffice_requests_queue_size Current number of LibreOffice conversion requests waiting to be treated.
      """
    Then the response body should contain string:
      """
      # TYPE libreoffice_requests_queue_size gauge
      """
    Then the response body should contain string:
      """
      # HELP http_server_request_duration_seconds Duration of HTTP server requests.
      """
    Then the response body should contain string:
      """
      # TYPE http_server_request_duration_seconds histogram
      """
    Then the Gotenberg container should log the following entries:
      | "path":"/custom/metrics" |

  Scenario: GET /prometheus/metrics (Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | TELEMETRY_METRIC_EXPORTER_PROTOCOLS |  |
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint
    Then the response status code should be 404

  Scenario: GET /prometheus/metrics (No Logging)
    Given I have a Gotenberg container with the following environment variable(s):
      | PROMETHEUS_DISABLE_ROUTE_LOGGING | true |
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint
    Then the response status code should be 200
    Then the Gotenberg container should NOT log the following entries:
      | "path":"/prometheus/metrics" |

  Scenario: GET /prometheus/metrics (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint with the following header(s):
      | Gotenberg-Trace | prometheus_metrics |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "prometheus_metrics"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"prometheus_metrics" |

  Scenario: GET /prometheus/metrics (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint
    Then the response status code should be 401

  Scenario: GET /foo/prometheus/metrics (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/prometheus/metrics" endpoint
    Then the response status code should be 200
