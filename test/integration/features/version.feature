@version
Feature: /version

  Scenario: GET /version
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/version" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      {version}
      """

  @telemetry
  Scenario: GET /version (Telemetry)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/version" endpoint with the following header(s):
      | Gotenberg-Trace | version                                                 |
      | traceparent     | 00-12345678901234567890123456789012-1234567890123456-01 |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "version"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"version"                    |
      | "trace_id":"12345678901234567890123456789012" |

  @telemetry
  Scenario: GET /version (No Telemetry)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DISABLE_VERSION_ROUTE_TELEMETRY | true |
    When I make a "GET" request to Gotenberg at the "/version" endpoint with the following header(s):
      | Gotenberg-Trace | version_no_telemetry                                    |
      | traceparent     | 00-12345678901234567890123456789012-1234567890123456-01 |
    Then the response status code should be 200
    Then the Gotenberg container should NOT log the following entries:
      | "correlation_id":"version_no_telemetry"       |
      | "trace_id":"12345678901234567890123456789012" |

  Scenario: GET /version (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/version" endpoint
    Then the response status code should be 401

  Scenario: GET /foo/version (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ROOT_PATH | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/version" endpoint
    Then the response status code should be 200
