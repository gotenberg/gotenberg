@pdfengines
@pdfengines-attachments
@attachments
Feature: /forms/pdfengines/attachments/add

  Scenario: POST /forms/pdfengines/attachments/add
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/attachments/add" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf       | file |
      | attachments | testdata/attachment_1.xml | file |
      | attachments | testdata/attachment_2.xml | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf |
    Then the response PDF(s) should have the "attachment_1.xml" file attached
    Then the response PDF(s) should have the "attachment_2.xml" file attached

  @download-from
  Scenario: POST /forms/pdfengines/attachments/add (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/attachments/add" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                                                                                                                                                                          | file  |
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/attachment_1.xml","attachment": true},{"url":"http://host.docker.internal:%d/static/testdata/attachment_2.xml","attachment": false}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf |
    Then the response PDF(s) should have the "attachment_1.xml" file attached
    Then the response PDF(s) should NOT have the "attachment_2.xml" file attached

  @webhook
  Scenario: POST /forms/pdfengines/attachments/add (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/attachments/add" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | attachments                 | testdata/attachment_1.xml                    | file   |
      | attachments                 | testdata/attachment_2.xml                    | file   |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the webhook request PDF(s) should have the "attachment_1.xml" file attached
    Then the webhook request PDF(s) should have the "attachment_2.xml" file attached

  Scenario: POST /forms/pdfengines/attachments/add (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/attachments/add" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf       | file |
      | attachments | testdata/attachment_1.xml | file |
      | attachments | testdata/attachment_2.xml | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/attachments/add (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/attachments/add" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf       | file |
      | attachments | testdata/attachment_1.xml | file |
      | attachments | testdata/attachment_2.xml | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
