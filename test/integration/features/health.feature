# TODO:
# 1. Check if down for each module.
# 2. Restarting modules do not make health check fail.

Feature: /health

  Scenario: GET /health
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/health" endpoint
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json; charset=utf-8"
    Then the response body should match JSON:
      """
      {
        "status": "up",
        "details": {
          "chromium": {
            "status": "up",
            "timestamp": "ignore"
          },
          "libreoffice": {
            "status": "up",
            "timestamp": "ignore"
          }
        }
      }
      """
    Then the Gotenberg container should log the following entries:
      | "path":"/health" |

  Scenario: GET /health (No Logging)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DISABLE_HEALTH_CHECK_LOGGING | true |
    When I make a "GET" request to Gotenberg at the "/health" endpoint
    Then the response status code should be 200
    Then the Gotenberg container should NOT log the following entries:
      | "path":"/health" |

  Scenario: GET /health (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/health" endpoint with the following header(s):
      | Gotenberg-Trace | get_health |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "get_health"
    Then the Gotenberg container should log the following entries:
      | "trace":"get_health" |

  Scenario: GET /health (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/health" endpoint
    Then the response status code should be 200

  Scenario: GET /foo/health (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ROOT_PATH | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/health" endpoint
    Then the response status code should be 200

  Scenario: HEAD /health
    Given I have a default Gotenberg container
    When I make a "HEAD" request to Gotenberg at the "/health" endpoint
    Then the response status code should be 200
    Then the response body should match string:
      """

      """
    Then the Gotenberg container should log the following entries:
      | "path":"/health" |

  Scenario: HEAD /health (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "HEAD" request to Gotenberg at the "/health" endpoint with the following header(s):
      | Gotenberg-Trace | head_health |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "head_health"
    Then the Gotenberg container should log the following entries:
      | "trace":"head_health" |

  Scenario: HEAD /health (No Logging)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DISABLE_HEALTH_CHECK_LOGGING | true |
    When I make a "HEAD" request to Gotenberg at the "/health" endpoint
    Then the response status code should be 200
    Then the Gotenberg container should NOT log the following entries:
      | "path":"/health" |

  Scenario: HEAD /health (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "HEAD" request to Gotenberg at the "/health" endpoint
    Then the response status code should be 200

  Scenario: HEAD /foo/health (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ROOT_PATH | /foo/ |
    When I make a "HEAD" request to Gotenberg at the "/foo/health" endpoint
    Then the response status code should be 200


# TODO:
# 1. Check if down for each module.
# 2. Restarting modules do not make health check fail.