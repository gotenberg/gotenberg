@pdf
@pdf-colors
Feature: /forms/pdf/{convert-colors|flatten-transparency}

  Scenario: POST /forms/pdf/convert-colors (Gray)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/convert-colors" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | target                    | gray                 | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """

  Scenario: POST /forms/pdf/convert-colors (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/convert-colors" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf | file  |
      | target | foo                  | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      "target" must be cmyk, rgb, gray or output-intent, got "foo"
      """

  Scenario: POST /forms/pdf/flatten-transparency
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-transparency" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      Page 3
      """
