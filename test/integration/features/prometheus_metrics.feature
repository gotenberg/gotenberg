# TODO:
# 1. Count restarts.
# 2. Count queue size.

Feature: /prometheus/metrics

  Scenario: GET /prometheus/metrics (Enabled)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; version=0.0.4; charset=utf-8; escaping=underscores"
    Then the response body should match string:
      """
      # HELP gotenberg_chromium_requests_queue_size Current number of Chromium conversion requests waiting to be treated.
      # TYPE gotenberg_chromium_requests_queue_size gauge
      gotenberg_chromium_requests_queue_size 0
      # HELP gotenberg_chromium_restarts_count Current number of Chromium restarts.
      # TYPE gotenberg_chromium_restarts_count gauge
      gotenberg_chromium_restarts_count 0
      # HELP gotenberg_libreoffice_requests_queue_size Current number of LibreOffice conversion requests waiting to be treated.
      # TYPE gotenberg_libreoffice_requests_queue_size gauge
      gotenberg_libreoffice_requests_queue_size 0
      # HELP gotenberg_libreoffice_restarts_count Current number of LibreOffice restarts.
      # TYPE gotenberg_libreoffice_restarts_count gauge
      gotenberg_libreoffice_restarts_count 0

      """
    Then the Gotenberg container should log the following entries:
      | "path":"/prometheus/metrics" |

  Scenario: GET /prometheus/metrics (Custom Namespace)
    Given I have a Gotenberg container with the following environment variable(s):
      | PROMETHEUS_NAMESPACE | foo |
    When I make a "GET" request to Gotenberg at the "/prometheus/metrics" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; version=0.0.4; charset=utf-8; escaping=underscores"
    Then the response body should match string:
      """
      # HELP foo_chromium_requests_queue_size Current number of Chromium conversion requests waiting to be treated.
      # TYPE foo_chromium_requests_queue_size gauge
      foo_chromium_requests_queue_size 0
      # HELP foo_chromium_restarts_count Current number of Chromium restarts.
      # TYPE foo_chromium_restarts_count gauge
      foo_chromium_restarts_count 0
      # HELP foo_libreoffice_requests_queue_size Current number of LibreOffice conversion requests waiting to be treated.
      # TYPE foo_libreoffice_requests_queue_size gauge
      foo_libreoffice_requests_queue_size 0
      # HELP foo_libreoffice_restarts_count Current number of LibreOffice restarts.
      # TYPE foo_libreoffice_restarts_count gauge
      foo_libreoffice_restarts_count 0

      """

  Scenario: GET /prometheus/metrics (Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PROMETHEUS_DISABLE_COLLECT | true |
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
      | "trace":"prometheus_metrics" |

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
