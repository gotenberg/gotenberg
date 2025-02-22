Feature: /forms/pdfengines/convert

  Scenario: POST /forms/pdfengines/convert (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfa  | PDF/A-1b             | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s)
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfa  | PDF/A-2b             | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s)
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 1 failed rule(s)
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfa  | PDF/A-3b             | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s)
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 1 failed rule(s)
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfua | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfa  | PDF/A-1b             | field |
      | pdfua | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s)
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf  | file  |
      | files | testdata/blank_20.pdf | file  |
      | pdfa  | PDF/A-1b              | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s)
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: either 'pdfa' or 'pdfua' form fields must be provided
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | pdfa  | PDF/A-1b | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdfengines/convert (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files                     | testdata/blank_1.pdf | file   |
      | pdfa                      | PDF/A-1b             | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then there should be the following file(s):
     | foo.pdf |
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files                     | testdata/blank_1.pdf  | file   |
      | files                     | testdata/blank_20.pdf | file   |
      | pdfa                      | PDF/A-1b              | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    Then there should be the following file(s):
      | foo.zip      |
      | blank_1.pdf  |
      | blank_20.pdf |

  Scenario: POST /forms/pdfengines/convert (Basic Auth)
    Given I have a Gotenberg container with the following environment variables:
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfa  | PDF/A-1b             | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/convert (Root Path)
    Given I have a Gotenberg container with the following environment variables:
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/convert" endpoint with the following form data and headers:
      | files | testdata/blank_1.pdf | file  |
      | pdfa  | PDF/A-1b             | field |
    Then the response status code should be 200

# TODO:
# 1. PDF/UA-2
