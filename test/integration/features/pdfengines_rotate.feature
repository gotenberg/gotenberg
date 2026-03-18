@pdfengines
@pdfengines-rotate
@rotate
Feature: /forms/pdfengines/rotate

  Scenario: POST /forms/pdfengines/rotate (90 - All Pages - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ROTATE_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/pages_3.pdf | file  |
      | rotateAngle | 90                   | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "pages_3.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdfengines/rotate (180 - All Pages - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ROTATE_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 180                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "page_1.pdf" PDF should have 1 page(s)

  Scenario: POST /forms/pdfengines/rotate (270 - All Pages - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ROTATE_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 270                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "page_1.pdf" PDF should have 1 page(s)

  Scenario: POST /forms/pdfengines/rotate (90 - Specific Pages - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ROTATE_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/pages_3.pdf | file  |
      | rotateAngle | 90                   | field |
      | rotatePages | 1,3                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "pages_3.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdfengines/rotate (90 - All Pages - pdftk)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ROTATE_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 90                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "page_1.pdf" PDF should have 1 page(s)

  Scenario: POST /forms/pdfengines/rotate (Specific Pages - pdftk unsupported)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ROTATE_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/pages_3.pdf | file  |
      | rotateAngle | 90                   | field |
      | rotatePages | 1,3                  | field |
    Then the response status code should be 500

  Scenario: POST /forms/pdfengines/rotate (Bad Request - Invalid Angle)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 45                  | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: form field 'rotateAngle' is invalid
      """

  Scenario: POST /forms/pdfengines/rotate (Bad Request - Missing Angle)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'rotateAngle' is required
      """

  Scenario: POST /forms/pdfengines/rotate (Bad Request - No PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | rotateAngle | 90 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdfengines/rotate (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | files       | testdata/page_2.pdf | file  |
      | rotateAngle | 90                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response

  Scenario: POST /forms/pdfengines/rotate (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 90                  | field |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/rotate (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf     | file   |
      | rotateAngle     | 90                      | field  |
      | Gotenberg-Trace | forms_pdfengines_rotate | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_rotate"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_rotate" |

  @webhook
  Scenario: POST /forms/pdfengines/rotate (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | rotateAngle                 | 90                                           | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: POST /forms/pdfengines/rotate (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 90                  | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/rotate (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | rotateAngle | 90                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
