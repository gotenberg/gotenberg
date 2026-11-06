@pdf
@pdf-extract-text
Feature: /forms/pdf/extract-text

  # The pages are joined by a form feed, which \f stands for.
  Scenario: POST /forms/pdf/extract-text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should contain "text/plain"
    Then there should be 0 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.txt |
    Then the "pages_3.txt" file should match string:
      """
      Page 1\fPage 2\fPage 3
      """

  Scenario: POST /forms/pdf/extract-text (Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | pages                     | 2                    | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then there should be the following file(s) in the response:
      | foo.txt |
    Then the "foo.txt" file should match string:
      """
      Page 2
      """

  # page_1.pdf shows its text in a FreeText annotation only: the page itself
  # reads as empty.
  Scenario: POST /forms/pdf/extract-text (Annotations)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 200
    Then the "page_1.txt" file should match string:
      """

      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | annotations | true                | field |
    Then the response status code should be 200
    Then the "page_1.txt" file should match string:
      """
      Page 1
      """

  # Gotenberg recognizes no text in an image: a scanned page reads as empty
  # rather than failing the request.
  Scenario: POST /forms/pdf/extract-text (Scanned Page)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files | testdata/scanned.pdf | file |
    Then the response status code should be 200
    Then the "scanned.txt" file should match string:
      """

      """

  Scenario: POST /forms/pdf/extract-text (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf  | file  |
      | files       | testdata/pages_3.pdf | file  |
      | annotations | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 0 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.txt  |
      | pages_3.txt |
    Then the "page_1.txt" file should match string:
      """
      Page 1
      """

  # stamp-text marks its stamp as an artifact, which only artifacts reads.
  Scenario: POST /forms/pdf/pipe (Extract Text)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf                                                                                                  | file  |
      | stages | [{"op":"select","pages":"1"},{"op":"stamp-text","text":"DRAFT","x":72,"y":72},{"op":"extract-text","artifacts":true}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.pdf |
      | pages_3.txt |
    Then the "pages_3.pdf" PDF should have 1 page(s)
    Then the "pages_3.txt" file should match string:
      """
      Page 1
      DRAFT
      """

  Scenario: POST /forms/pdf/extract-text (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pages | 2                   | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Stage 'extract-text' failed on 'page_1.pdf'
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-text" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf | file  |
      | annotations | yes                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'annotations' is invalid (got 'yes', want true or false)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf                               | file  |
      | stages | [{"op":"extract-text"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: "extract-text" must be the last stage
      """
