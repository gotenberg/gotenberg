@chromium
@chromium-concurrent
Feature: Chromium concurrent conversions

  Scenario: Concurrent HTML to PDF conversions with max concurrency 3
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_MAX_CONCURRENCY | 3 |
    When I make 3 concurrent "POST" requests to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then all concurrent response status codes should be 200
    Then all concurrent responses should have 1 PDF(s)

  Scenario: Concurrent conversions exceeding restart-after limit
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_MAX_CONCURRENCY | 3 |
      | CHROMIUM_RESTART_AFTER   | 5 |
    When I make 10 concurrent "POST" requests to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then all concurrent response status codes should be 200
    Then all concurrent responses should have 1 PDF(s)
