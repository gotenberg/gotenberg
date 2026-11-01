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
    Then the response body should match string:
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
      | files  | testdata/page_1.pdf           | file  |
      | stages | [{"op":"rotate","angle":"x"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: rotate: "angle" is invalid (got "x", want an integer)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/form.pdf                           | file  |
      | stages | [{"op":"set-form","fields":{"agree":true}}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: set-form: "fields" values must be strings; a check box takes its state name, such as "Yes" or "Off"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                  | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "rotate" cannot follow "remediate": only encrypt, optimize, sign, timestamp and render may
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
      | files  | testdata/page_1.pdf                                                           | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"},{"op":"encrypt","userPassword":"bar"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: only one "encrypt" stage is allowed
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                               | file  |
      | stages | [{"op":"timestamp","timestampUrl":"https://tsa.example"},{"op":"timestamp","timestampUrl":"https://tsa.example"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: only one "timestamp" stage is allowed
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                         | file  |
      | stages | [{"op":"decrypt","password":"foo"},{"op":"timestamp","timestampUrl":"https://tsa.example"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: "decrypt" cannot be combined with "sign" or "timestamp": litho cannot sign a PDF it reads encrypted; decrypt it in a request of its own, then sign the result
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                             | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"},{"op":"timestamp","timestampUrl":"https://tsa.example"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: "encrypt" cannot be combined with "sign" or "timestamp": a signature covers the written bytes
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "rotate" cannot follow "encrypt": only optimize and render may
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

  # litho applies a stamp when a later stage or the write runs: the failure
  # names the stage that asked for it.
  Scenario: POST /forms/pdf/pipe (Failed Edit Applied By A Later Stage)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                  | file  |
      | files  | testdata/page_2.pdf                                                                  | file  |
      | stages | [{"op":"stamp-pdf","source":"page_2.pdf","sourcePage":9},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'stamp-pdf' failed on 'page_1.pdf': pdf stamp: source page 9: page number out of bounds (source has 1 pages)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                                                 | file  |
      | files  | testdata/page_2.pdf                                                                                                                 | file  |
      | stages | [{"op":"stamp-pdf","source":"page_2.pdf","sourcePage":9},{"op":"redact","regions":[{"page":1,"x":0,"y":0,"width":10,"height":10}]}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'stamp-pdf' failed on 'page_1.pdf': pdf stamp: source page 9: page number out of bounds (source has 1 pages)
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

  # A field the route does not read answers 400, and a Gotenberg 8 one says what
  # replaced it.
  Scenario: POST /forms/pdf/pipe (Unknown Form Fields)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf          | file  |
      | files  | testdata/page_2.pdf          | file  |
      | stages | [{"op":"rotate","angle":90}] | field |
      | merge  | true                         | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'merge' is unknown to this route; Gotenberg 9 moved it to a stage: add {"op":"merge"} to 'stages'
      """

  # The Gotenberg 8 routes answer 404 with the route that replaced each.
  Scenario: POST /forms/pdfengines/* (Gotenberg 8 Routes)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404
    Then the response body should match string:
      """
      Gotenberg 9 replaced /forms/pdfengines/merge with /forms/pdf/merge
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404
    Then the response body should match string:
      """
      Gotenberg 9 replaced /forms/pdfengines/metadata/write with /forms/pdf/set-metadata
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/foo" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404
    Then the response body should match string:
      """
      Gotenberg 9 replaced the /forms/pdfengines routes with the /forms/pdf routes
      """
