@root
Feature: /

  Scenario: GET /
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/html; charset=UTF-8"
    Then the response body should match string:
      """
      Hey, Gotenberg has no UI, it's an API. Head to the <a href="https://gotenberg.dev">documentation</a> to learn how to interact with it 🚀
      """

  @telemetry
  Scenario: GET / (Telemetry)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/" endpoint with the following header(s):
      | Gotenberg-Trace | root                                                    |
      | traceparent     | 00-12345678901234567890123456789012-1234567890123456-01 |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "root"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"root"                       |
      | "trace_id":"12345678901234567890123456789012" |

  @telemetry
  Scenario: GET / (No Telemetry)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DISABLE_ROOT_ROUTE_TELEMETRY | true |
    When I make a "GET" request to Gotenberg at the "/" endpoint with the following header(s):
      | Gotenberg-Trace | root_no_telemetry                                       |
      | traceparent     | 00-12345678901234567890123456789012-1234567890123456-01 |
    Then the response status code should be 200
    Then the Gotenberg container should NOT log the following entries:
      | "correlation_id":"root_no_telemetry"          |
      | "trace_id":"12345678901234567890123456789012" |

  Scenario: GET / (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/" endpoint
    Then the response status code should be 401

  Scenario: GET /foo/ (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ROOT_PATH | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/" endpoint
    Then the response status code should be 200

  Scenario: GET /favicon.ico
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/favicon.ico" endpoint
    Then the response status code should be 204

  @telemetry
  Scenario: GET /favicon.ico (No Telemetry)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/favicon.ico" endpoint with the following header(s):
      | Gotenberg-Trace | favicon_no_telemetry                                    |
      | traceparent     | 00-12345678901234567890123456789012-1234567890123456-01 |
    Then the response status code should be 204
    Then the Gotenberg container should NOT log the following entries:
      | "correlation_id":"favicon_no_telemetry"       |
      | "trace_id":"12345678901234567890123456789012" |

  Scenario: GET /favicon.ico (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/favicon.ico" endpoint
    Then the response status code should be 401

  Scenario: GET /foo/favicon.ico (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ROOT_PATH | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/favicon.ico" endpoint
    Then the response status code should be 204
