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

  Scenario: GET /version (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "GET" request to Gotenberg at the "/version" endpoint with the following header(s):
      | Gotenberg-Trace | version |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "version"
    Then the Gotenberg container should log the following entries:
      | "trace":"version" |

  Scenario: GET /version (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "GET" request to Gotenberg at the "/version" endpoint
    Then the response status code should be 401

  Scenario: GET /foo/version (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "GET" request to Gotenberg at the "/foo/version" endpoint
    Then the response status code should be 200
