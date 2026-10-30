@pdfengines
@pdfengines-convert-image
Feature: /forms/pdfengines/convert/image

  Scenario: POST /forms/pdfengines/convert/image (Default PNG)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert/image" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"
    Then there should be the following file(s) in the response:
      | page_1-1.png |

  Scenario: POST /forms/pdfengines/convert/image (JPEG)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert/image" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | format | jpeg                | field |
      | dpi    | 100                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/jpeg"
    Then there should be the following file(s) in the response:
      | page_1-1.jpeg |

  Scenario: POST /forms/pdfengines/convert/image (Multi-page archive)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert/image" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | pages_3-1.png |
      | pages_3-2.png |
      | pages_3-3.png |

  Scenario: POST /forms/pdfengines/convert/image (Page range)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert/image" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | firstPage | 2                    | field |
      | lastPage  | 3                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | pages_3-1.png |
      | pages_3-2.png |

  Scenario: POST /forms/pdfengines/convert/image (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert/image" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | format | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'format' is invalid (got 'foo', resulting to wrong value, expected either 'png', 'jpeg' or 'tiff')
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert/image" endpoint with the following form data and header(s):
      | format | png | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """
