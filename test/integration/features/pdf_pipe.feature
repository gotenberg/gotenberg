@pdf
@pdf-pipe
Feature: /forms/pdf/pipe

  Scenario: POST /forms/pdf/pipe (Merge, Rotate & Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/pages_12.pdf                                                                                               | file   |
      | files                     | testdata/pages_3.pdf                                                                                                | file   |
      | stages                    | [{"op":"merge"},{"op":"rotate","angle":90,"pages":"1"},{"op":"set-metadata","title":"Merged","author":"Gotenberg"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 15 page(s)
    Then the "foo.pdf" PDF page 1 should be rotated by 90 degrees
    Then the "foo.pdf" PDF page 2 should be rotated by 0 degrees
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Merged"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Gotenberg"

  Scenario: POST /forms/pdf/pipe (Split Then Edit Each Part)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf                                            | file  |
      | stages | [{"op":"split","every":2},{"op":"set-metadata","title":"Part"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the "Title" metadata set to "Part"
    Then the "pages_3_1.pdf" PDF should have the "Title" metadata set to "Part"

  Scenario: POST /forms/pdf/pipe (Many PDFs Without Merge)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf           | file  |
      | files  | testdata/page_2.pdf           | file  |
      | stages | [{"op":"rotate","angle":180}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf |
      | page_2.pdf |
    Then the "page_1.pdf" PDF page 1 should be rotated by 180 degrees
    Then the "page_2.pdf" PDF page 1 should be rotated by 180 degrees

  Scenario: POST /forms/pdf/pipe (Remediate, Then Sign)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                  | file   |
      | files                     | testdata/pem/cert.pem                                                                                                | file   |
      | files                     | testdata/pem/key.pem                                                                                                 | file   |
      | stages                    | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"sign","certificate":"cert.pem","key":"key.pem","reason":"Approved"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response PDF(s) should be signed
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/pipe (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is required, for example [{"op":"merge"}]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | stages | {"op":"merge"}      | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: form field 'stages' is invalid: stages must be a JSON array of objects
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | stages | [{"op":"shred"}]    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: unknown operation "shred"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                       | file  |
      | stages | [{"op":"rotate","angle":90,"degrees":90}] | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: rotate: unknown option "degrees"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                  | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "rotate" cannot follow "remediate": only encrypt, optimize, sign and timestamp may
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                             | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"encrypt","userPassword":"foo"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: "PDF/A-2b" cannot be encrypted: PDF/A forbids encryption
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                            | file  |
      | stages | [{"op":"stamp-image","image":"watermark.png"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form file 'watermark.png' is required
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | stages | [{"op":"merge"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                      | file  |
      | stages | [{"op":"rotate","angle":90,"pages":"3"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'rotate' failed on 'page_1.pdf': invalid page range: page 3 out of range [1, 1]
      """

  Scenario: POST /forms/pdf/pipe (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | stages | [{"op":"merge"}]    | field |
    Then the response status code should be 404

  Scenario: POST /forms/pdf/pipe (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf          | file   |
      | stages          | [{"op":"rotate","angle":90}] | field  |
      | Gotenberg-Trace | forms_pdf_pipe               | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdf_pipe"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"forms_pdf_pipe" |

  @download-from
  Scenario: POST /forms/pdf/pipe (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | downloadFrom              | [{"url":"{helper}/static/testdata/pages_3.pdf"},{"url":"{helper}/static/testdata/watermark.png"}] | field  |
      | stages                    | [{"op":"stamp-image","image":"watermark.png","width":50,"height":50}]                             | field  |
      | Gotenberg-Output-Filename | foo                                                                                               | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 3 page(s)

  @webhook
  Scenario: POST /forms/pdf/pipe (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | files                        | testdata/page_2.pdf     | file   |
      | stages                       | [{"op":"merge"}]        | field  |
      | Gotenberg-Output-Filename    | foo                     | header |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)

  Scenario: POST /forms/pdf/pipe (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | stages | [{"op":"merge"}]    | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdf/pipe (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | stages | [{"op":"merge"}]    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/pdf/pipe (Long Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/Longitudinell_jämförelse_mellan_laserkirurgi_och_strålbehandling_gällande_röstkvalitet_och_självskattad_kommunikation_upp_till_två_år_efter_tidig_stämbandscancer_i_ett_randomiserat_kontrollerat_försök.pdf | file   |
      | stages                    | [{"op":"rotate","angle":90}]                                                                                                                                                                                          | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
