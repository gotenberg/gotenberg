Feature: /forms/pdfengines/add-password

  Scenario: POST /forms/pdfengines/add-password (default - QPDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be password protected

  Scenario: POST /forms/pdfengines/add-password with user and owner passwords (QPDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | user123             | field  |
      | ownerPassword             | owner456            | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be password protected

  Scenario: POST /forms/pdfengines/add-password (PDFtk)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be password protected

  Scenario: POST /forms/pdfengines/password (LibreOffice)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | libreoffice-pdfengine |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be password protected

  Scenario: POST /forms/pdfengines/add-password (pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | protected.pdf |
    Then the "protected.pdf" PDF should be password protected


  Scenario: POST /forms/pdfengines/add-password with multiple files
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | protected.zip |
    Then the "protected.zip" archive should contain 2 file(s)
    Then the "protected.zip" archive should contain password protected PDF file(s)

  Scenario: POST /forms/pdfengines/add-password without required userPassword field
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 400
    Then the response body should contain "userPassword"

  Scenario: POST /forms/pdfengines/add-password with password engines that don't support password protection
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_PASSWORD_ENGINES | exiftool |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/add-password" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | test123             | field  |
      | Gotenberg-Output-Filename | protected           | header |
    Then the response status code should be 500
    Then the response body should contain "password protection not supported"
