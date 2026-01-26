@pdfengines
@pdfengines-encrypt
@encrypt
Feature: /forms/pdfengines/encrypt

  Scenario: POST /forms/pdfengines/encrypt (default - user password only)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (default - both user and owner passwords)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | userPassword  | foo                 | field |
      | ownerPassword | bar                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (QPDF - user password only)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ENCRYPT_ENGINES | qpdf |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (QPDF - both user and owner passwords)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ENCRYPT_ENGINES | qpdf |
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | userPassword  | foo                 | field |
      | ownerPassword | bar                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (PDFtk - user password only)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ENCRYPT_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      pdftk: both 'userPassword' and 'ownerPassword' must be provided and different. Consider switching to another PDF engine if this behavior does not work with your workflow
      """

  Scenario: POST /forms/pdfengines/encrypt (PDFtk - both user and owner passwords)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ENCRYPT_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | userPassword  | foo                 | field |
      | ownerPassword | bar                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (pdfcpu - user password only)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ENCRYPT_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (pdfcpu - both user and owner passwords)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_ENCRYPT_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | userPassword  | foo                 | field |
      | ownerPassword | bar                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | files        | testdata/page_2.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'userPassword' is required
      """

  Scenario: POST /forms/pdfengines/encrypt (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/encrypt (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf      | file   |
      | userPassword    | foo                      | field  |
      | Gotenberg-Trace | forms_pdfengines_encrypt | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_encrypt"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_encrypt" |

  @download-from
  Scenario: POST /forms/pdfengines/encrypt (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | userPassword | foo                                                                                                      | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @webhook
  Scenario: POST /forms/pdfengines/encrypt (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | userPassword                | foo                                          | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/encrypt (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
