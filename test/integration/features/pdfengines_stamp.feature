@stamp
Feature: /forms/pdfengines/stamp
  Scenario: POST /forms/pdfengines/stamp (default - without params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | myfile.pdf        | testdata/page_1.pdf              | file  |
      | stamp.png     | testdata/stamp/stamp.png | file  |
      | stampFilename | stamp.png                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp


  Scenario: POST /forms/pdfengines/stamp (default - with params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | myfile.pdf        | testdata/page_1.pdf                                       | file  |
      | stamp.png     | testdata/stamp/stamp.png                          | file  |
      | stampFilename | stamp.png                                             | field |
      | params            | rotation:-180, position:tl, scalefactor:0.20, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files             | testdata/page_1.pdf              | file  |
      | files             | testdata/page_2.pdf              | file  |
      | stamp.png     | testdata/stamp/stamp.png | file  |
      | stampFilename | stamp.png                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/stamp (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files             | testdata/page_1.pdf              | file   |
      | stamp.png     | testdata/stamp/stamp.png | file   |
      | stampFilename | stamp.png                    | field  |
      | Gotenberg-Trace   | forms_pdfengines_add_stamp   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_add_stamp"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_add_stamp" |

  Scenario: POST /forms/pdfengines/stamp (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | downloadFrom      | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | stamp.png     | testdata/stamp/stamp.png                                                                         | file  |
      | stampFilename | stamp.png                                                                                            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | stamp.png               | testdata/stamp/stamp.png             | file   |
      | stampFilename           | stamp.png                                | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/stamp (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files             | testdata/page_1.pdf              | file  |
      | stamp.png     | testdata/stamp/stamp.png | file  |
      | stampFilename | stamp.png                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Text Mode - default params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | myfile.pdf    | testdata/page_1.pdf | file  |
      | stampMode | text                | field |
      | stampText | Confidential        | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Text Mode - with params)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | myfile.pdf    | testdata/page_1.pdf      | file  |
      | stampMode | text                     | field |
      | stampText | DRAFT DOCUMENT           | field |
      | params        | position:tl, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Text Mode - Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf      | file  |
      | files         | testdata/page_2.pdf      | file  |
      | stampMode | text                     | field |
      | stampText | CONFIDENTIAL             | field |
      | params        | position:tl, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should have a stamp

  Scenario: POST /forms/pdfengines/stamp (Text Mode - Bad Request - Missing stampText)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | stampMode | text                | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stampMode' is invalid (got 'text', resulting to stampText is required for text mode)
      """

  Scenario: POST /forms/pdfengines/stamp (Image Mode - Bad Request - Missing stampFilename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | stampMode | image               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stampMode' is invalid (got 'image', resulting to stampFilename is required for image mode)
      """

  Scenario: POST /forms/pdfengines/stamp (Text Mode - Page numbers)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp" endpoint with the following form data and header(s):
      | myfile.pdf    | testdata/page_1.pdf      | file  |
      | stampMode | text                     | field |
      | stampText | Page %p of %P            | field |
      | params        | position:tl, opacity:0.4 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have a stamp
