@api-archives
Feature: Archives

  # Each entry of a zip archive sent under "archives" counts as an uploaded file.
  @chromium
  Scenario: POST /forms/chromium/convert/html (Archive)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | archives                  | testdata/pages-3-html.zip | file   |
      | Gotenberg-Output-Filename | foo                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdf/merge (Archive)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | archives                  | testdata/pages_1_2.zip | file   |
      | files                     | testdata/pages_3.pdf   | file   |
      | Gotenberg-Output-Filename | foo                    | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 5 page(s)

  # A zip archive under "files" is a file like any other.
  Scenario: POST /forms/pdf/attach (Archive As A File)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf    | file   |
      | files                     | testdata/pages_1_2.zip | file   |
      | file                      | pages_1_2.zip          | field  |
      | Gotenberg-Output-Filename | foo                    | header |
    Then the response status code should be 200
    Then the response PDF(s) should have the "pages_1_2.zip" file embedded

  Scenario: POST /forms/pdf/merge (Archive Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | archives | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'archives' is invalid: 'page_1.pdf' is not a zip archive
      """

  Scenario: POST /forms/pdf/merge (Archive Bomb)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | archives | testdata/bomb.zip | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'archives' is invalid: 'bomb.zip' unpacks to more than 100 times its size
      """
