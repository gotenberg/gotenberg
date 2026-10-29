@pdf
@pdf-optimize
Feature: /forms/pdf/optimize

  Scenario: POST /forms/pdf/optimize (Image-heavy PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files                     | testdata/image-heavy.pdf | file   |
      | Gotenberg-Output-Filename | foo                      | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" file size should be less than 712 KB

  Scenario: POST /forms/pdf/optimize (Selected Options)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files            | testdata/pages_12.pdf | file  |
      | recompressImages | false                 | field |
      | subsetFonts      | false                 | field |
    Then the response status code should be 200
    Then the "pages_12.pdf" PDF should have 12 page(s)

  Scenario: POST /forms/pdf/pipe (Encrypt & Optimize)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                       | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"},{"op":"optimize"}] | field |
    Then the response status code should be 200
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdf/optimize (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | dedup | maybe               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'dedup' is invalid: must be true or false, got "maybe"
      """
