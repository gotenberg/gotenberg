@watermark
Feature: /forms/pdfengines/watermark
  Scenario: POST /forms/pdfengines/watermark (default - without params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | myfile.pdf        | testdata/page_1.pdf              | file  |
      | watermark.png     | testdata/watermark/watermark.png | file  |
      | watermarkFilename | watermark.png                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark


  Scenario: POST /forms/pdfengines/watermark (default - with params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | myfile.pdf        | testdata/page_1.pdf                                       | file  |
      | watermark.png     | testdata/watermark/watermark.png                          | file  |
      | watermarkFilename | watermark.png                                             | field |
      | params            | rotation:-180, position:tl, scalefactor:0.20, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files             | testdata/page_1.pdf              | file  |
      | files             | testdata/page_2.pdf              | file  |
      | watermark.png     | testdata/watermark/watermark.png | file  |
      | watermarkFilename | watermark.png                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/watermark (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files             | testdata/page_1.pdf              | file   |
      | watermark.png     | testdata/watermark/watermark.png | file   |
      | watermarkFilename | watermark.png                    | field  |
      | Gotenberg-Trace   | forms_pdfengines_add_watermark   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_add_watermark"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_add_watermark" |

  Scenario: POST /forms/pdfengines/watermark (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | downloadFrom      | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | watermark.png     | testdata/watermark/watermark.png                                                                         | file  |
      | watermarkFilename | watermark.png                                                                                            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | watermark.png               | testdata/watermark/watermark.png             | file   |
      | watermarkFilename           | watermark.png                                | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/watermark (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files             | testdata/page_1.pdf              | file  |
      | watermark.png     | testdata/watermark/watermark.png | file  |
      | watermarkFilename | watermark.png                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Text Mode - default params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | myfile.pdf    | testdata/page_1.pdf | file  |
      | watermarkMode | text                | field |
      | watermarkText | Confidential        | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Text Mode - with params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | myfile.pdf    | testdata/page_1.pdf      | file  |
      | watermarkMode | text                     | field |
      | watermarkText | DRAFT DOCUMENT           | field |
      | params        | position:tl, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Text Mode - Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf      | file  |
      | files         | testdata/page_2.pdf      | file  |
      | watermarkMode | text                     | field |
      | watermarkText | CONFIDENTIAL             | field |
      | params        | position:tl, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should have a watermark

  Scenario: POST /forms/pdfengines/watermark (Text Mode - Bad Request - Missing watermarkText)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | watermarkMode | text                | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'watermarkMode' is invalid (got 'text', resulting to watermarkText is required for text mode)
      """

  Scenario: POST /forms/pdfengines/watermark (Image Mode - Bad Request - Missing watermarkFilename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | watermarkMode | image               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'watermarkMode' is invalid (got 'image', resulting to watermarkFilename is required for image mode)
      """

  Scenario: POST /forms/pdfengines/watermark (Text Mode - Page numbers)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/watermark" endpoint with the following form data and header(s):
      | myfile.pdf    | testdata/page_1.pdf      | file  |
      | watermarkMode | text                     | field |
      | watermarkText | Page %p of %P            | field |
      | params        | position:tl, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a watermark
