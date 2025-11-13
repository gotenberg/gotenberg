@pdfengines
@pdfengines-embed
@embed
Feature: /forms/pdfengines/embed

  Scenario: POST /forms/pdfengines/embed
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf  | file |
      | embeds | testdata/embed_1.xml | file |
      | embeds | testdata/embed_2.xml | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf |
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should have the "embed_2.xml" file embedded

  @download-from
  Scenario: POST /forms/pdfengines/embed with (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                                                                                                                                                            | file  |
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/embed_1.xml","embedded": true},{"url":"http://host.docker.internal:%d/static/testdata/embed_2.xml","embedded": false}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf |
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should NOT have the "embed_2.xml" file embedded

  @webhook
  Scenario: POST /forms/pdfengines/embed (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | embeds                      | testdata/embed_1.xml                         | file   |
      | embeds                      | testdata/embed_2.xml                         | file   |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the webhook request PDF(s) should have the "embed_1.xml" file embedded
    Then the webhook request PDF(s) should have the "embed_2.xml" file embedded

  Scenario: POST /forms/pdfengines/embed (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf  | file |
      | embeds | testdata/embed_1.xml | file |
      | embeds | testdata/embed_2.xml | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/embed (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf  | file |
      | embeds | testdata/embed_1.xml | file |
      | embeds | testdata/embed_2.xml | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
