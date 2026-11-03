@pdf
@pdf-rotate
Feature: /forms/pdf/rotate

  Scenario: POST /forms/pdf/rotate (All Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | angle                     | 90                   | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF page 1 should be rotated by 90 degrees
    Then the "foo.pdf" PDF page 3 should be rotated by 90 degrees

  Scenario: POST /forms/pdf/rotate (Specific Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | angle                     | 270                  | field  |
      | pages                     | 2-3                  | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF page 1 should be rotated by 0 degrees
    Then the "foo.pdf" PDF page 2 should be rotated by 270 degrees
    Then the "foo.pdf" PDF page 3 should be rotated by 270 degrees

  Scenario: POST /forms/pdf/rotate (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | files | testdata/page_2.pdf | file  |
      | angle | 180                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the "page_1.pdf" PDF page 1 should be rotated by 180 degrees
    Then the "page_2.pdf" PDF page 1 should be rotated by 180 degrees

  Scenario: POST /forms/pdf/rotate (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | angle | 45                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: rotate: "angle" must be a multiple of 90, got 45
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | angle | ninety              | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'angle' is invalid (got 'ninety', want an integer)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | angle | 90 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | angle | 90                  | field |
      | pages | 2                   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'rotate' failed on 'page_1.pdf': invalid page range: page 2 out of range [1, 1]
      """

  # A Gotenberg 8 watermark file is not one more PDF to rotate, and a field the
  # route does not read answers 400.
  Scenario: POST /forms/pdf/rotate (Unknown Form Fields)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | watermark   | testdata/page_2.pdf | file  |
      | angle       | 90                  | field |
      | rotateAngle | 90                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'rotateAngle' is unknown to this route; Gotenberg 9 moved it to a stage: add {"op":"rotate","angle":90,"pages":"1"} to 'stages'
      form file field 'watermark' is unknown to this route; Gotenberg 9 reads every file from 'files': upload it there, and add {"op":"stamp-pdf","source":"annex.pdf"} or {"op":"stamp-image","image":"logo.png"} to 'stages'
      """

  Scenario: POST /forms/pdf/rotate (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf | file   |
      | angle           | 90                  | field  |
      | Gotenberg-Trace | forms_pdf_rotate    | header |
    Then the response status code should be 200
    Then the response header "Gotenberg-Trace" should be "forms_pdf_rotate"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"forms_pdf_rotate" |

  Scenario: POST /forms/pdf/rotate (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | angle | 90                  | field |
    Then the response status code should be 404

  @webhook
  Scenario: POST /forms/pdf/rotate (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | angle                        | 90                      | field  |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: POST /forms/pdf/rotate (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | angle | 90                  | field |
    Then the response status code should be 401

  Scenario: POST /forms/pdf/rotate (Truncated PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | testdata/truncated.pdf | file  |
      | angle | 90                     | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Stage 'rotate' failed on 'truncated.pdf': open PDF:
      """
