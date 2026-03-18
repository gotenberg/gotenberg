@pdfengines
@pdfengines-stamp
@stamp
Feature: /forms/pdfengines/stamp

  Scenario: POST /forms/pdfengines/stamp (Text - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file  |
      | stampSource     | text                | field |
      | stampExpression | CONFIDENTIAL        | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "page_1.pdf" PDF should have 1 page(s)

  Scenario: POST /forms/pdfengines/stamp (Text with Pages - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/pages_3.pdf | file  |
      | stampSource     | text                 | field |
      | stampExpression | DRAFT                | field |
      | stampPages      | 1-2                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "pages_3.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdfengines/stamp (Text with Options - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf                                  | file  |
      | stampSource     | text                                                 | field |
      | stampExpression | SAMPLE                                               | field |
      | stampOptions    | {"scale":"0.5 abs","rot":"45","fillcolor":"#FF0000"} | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp (Image - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf    | file  |
      | stamps      | testdata/watermark.png | file  |
      | stampSource | image                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp (PDF - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | stamps      | testdata/page_2.pdf | file  |
      | stampSource | pdf                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp (PDF - pdftk)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | stamps      | testdata/page_2.pdf | file  |
      | stampSource | pdf                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp (Text - pdftk unsupported)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file  |
      | stampSource     | text                | field |
      | stampExpression | CONFIDENTIAL        | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested stamp source type, while others may have failed due to different issues
      """

  Scenario: POST /forms/pdfengines/stamp (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file  |
      | files           | testdata/page_2.pdf | file  |
      | stampSource     | text                | field |
      | stampExpression | DRAFT               | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp (Bad Request - No Source)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stampSource' is required
      """

  Scenario: POST /forms/pdfengines/stamp (Bad Request - Invalid Source)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | stampSource | foo                 | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: form field 'stampSource' is invalid
      """

  Scenario: POST /forms/pdfengines/stamp (Bad Request - Missing File for Image Source)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | stampSource | image               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: a stamp file is required for image or pdf source
      """

  Scenario: POST /forms/pdfengines/stamp (Bad Request - Missing File for PDF Source)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | stampSource | pdf                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: a stamp file is required for image or pdf source
      """

  Scenario: POST /forms/pdfengines/stamp (Bad Request - No PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | stampSource     | text         | field |
      | stampExpression | CONFIDENTIAL | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdfengines/stamp (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file  |
      | stampSource     | text                | field |
      | stampExpression | CONFIDENTIAL        | field |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/stamp (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf    | file   |
      | stampSource     | text                   | field  |
      | stampExpression | CONFIDENTIAL           | field  |
      | Gotenberg-Trace | forms_pdfengines_stamp | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_stamp"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_stamp" |

  @webhook
  Scenario: POST /forms/pdfengines/stamp (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | stampSource                 | text                                         | field  |
      | stampExpression             | CONFIDENTIAL                                 | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: POST /forms/pdfengines/stamp (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file  |
      | stampSource     | text                | field |
      | stampExpression | CONFIDENTIAL        | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/stamp (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file  |
      | stampSource     | text                | field |
      | stampExpression | CONFIDENTIAL        | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
