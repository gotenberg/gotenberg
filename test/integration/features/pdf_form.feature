@pdf
@pdf-form
Feature: /forms/pdf/{flatten-form|set-form|form}

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

  # A field the PDF does not have is the client's mistake: the answer lists the
  # fields it has.
  Scenario: POST /forms/pdf/set-form (Unknown Field)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files  | testdata/form.pdf | file  |
      | fields | {"nope":"x"}      | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Stage 'set-form' failed on 'form.pdf': form field "nope" not found; the PDF's fields are: name
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | fields | {"nope":"x"}        | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'set-form' failed on 'page_1.pdf': form field "nope" not found; the PDF has no form fields
      """

  Scenario: POST /forms/pdf/set-form (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: set-form: "fields" must map at least one field name to a value
      """

  # Each name and value feeds set-form as it is.
  Scenario: POST /forms/pdf/form
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/form" endpoint with the following form data and header(s):
      | files | testdata/form.pdf   | file |
      | files | testdata/multi.pdf  | file |
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "form.pdf": [{ "name": "name", "type": "text", "value": "", "pages": [1] }],
        "multi.pdf": [
          { "name": "total", "type": "text", "value": "42", "pages": [1, 2], "readOnly": true },
          {
            "name": "size",
            "type": "radio",
            "value": "Off",
            "pages": [2],
            "states": ["S", "M", "L"],
            "required": true
          },
          { "name": "calc", "type": "text", "value": "7", "pages": [] },
          { "name": "country", "type": "combo", "value": "FR", "pages": [], "options": ["FR", "DE"] },
          { "name": "sig", "type": "signature", "value": "", "pages": [2] },
          {
            "name": "tags",
            "type": "list",
            "value": ["a", "c"],
            "pages": [1],
            "options": ["a", "b", "c"],
            "multiSelect": true
          }
        ],
        "page_1.pdf": []
      }
      """

  Scenario: POST /forms/pdf/form (Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/form" endpoint with the following form data and header(s):
      | files | testdata/multi.pdf | file  |
      | pages | 2                  | field |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "multi.pdf": [
          { "name": "total", "type": "text", "value": "42", "pages": [1, 2], "readOnly": true },
          {
            "name": "size",
            "type": "radio",
            "value": "Off",
            "pages": [2],
            "states": ["S", "M", "L"],
            "required": true
          },
          { "name": "sig", "type": "signature", "value": "", "pages": [2] }
        ]
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/form" endpoint with the following form data and header(s):
      | files | testdata/multi.pdf | file  |
      | pages | 3                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'form' failed on 'multi.pdf': form field 'pages': invalid page range: page 3 out of range [1, 2]
      """

  Scenario: POST /forms/pdf/flatten-form (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404
