@api-limits
Feature: API limits

  Scenario: Body Limit
    Given I have a Gotenberg container with the following environment variable(s):
      | API_BODY_LIMIT | 1KB |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 413
    Then the response body should match string:
      """
      The request body exceeds the configured size limit. Increase it with --api-body-limit, or send a smaller request.
      """

  Scenario: Request Timeout
    Given I have a Gotenberg container with the following environment variable(s):
      | API_TIMEOUT | 5s |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/page-1-html/index.html | file  |
      | waitDelay | 10s                             | field |
    Then the response status code should be 503
    Then the response body should match string:
      """
      The request exceeded the time limit. Increase it with --api-timeout, or reduce the workload.
      """
