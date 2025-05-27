Feature: /forms/pdfengines/encrypt

  Scenario: POST /forms/pdfengines/encrypt (default - QPDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be encrypted

  Scenario: POST /forms/pdfengines/encrypt with user and owner passwords (QPDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | user123             | field  |
      | ownerPassword             | owner456            | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (PDFtk)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be encrypted

  Scenario: POST /forms/pdfengines/encrypt with multiple files
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | protected.zip |
    Then the "protected.zip" archive should contain 2 file(s)
    Then the "protected.zip" archive should contain encrypted PDF file(s)

  Scenario: POST /forms/pdfengines/encrypt without required userPassword field
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 400
    Then the response body should contain "userPassword"

  Scenario: POST /forms/pdfengines/encrypt with password engines that don't support password protection
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | exiftool |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 500
    Then the response body should contain "password protection not supported"

  Scenario: POST /forms/pdfengines/encrypt (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be encrypted

  Scenario: POST /forms/pdfengines/encrypt (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  Scenario: POST /forms/pdfengines/encrypt (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/encrypt" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
    Then the response status code should be 404
