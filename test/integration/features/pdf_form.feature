@pdf
@pdf-form
Feature: /forms/pdf/{flatten-form|set-form}

  Scenario: POST /forms/pdf/flatten-form (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be flatten

  Scenario: POST /forms/pdf/flatten-form (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
      | files | testdata/page_2.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should be flatten

  Scenario: POST /forms/pdf/set-form
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files                     | testdata/form.pdf    | file   |
      | fields                    | {"name":"Gotenberg"} | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have the form field "name" set to "Gotenberg"

  Scenario: POST /forms/pdf/set-form (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: set-form: "fields" must map at least one field name to a value
      """

  Scenario: POST /forms/pdf/flatten-form (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404
