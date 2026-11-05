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

  # A value the form's own font cannot draw, outside Windows-1252, is drawn
  # with a font installed in the image, and an empty field's border is drawn
  # too, so nothing of the form outlives the flatten.
  Scenario: POST /forms/pdf/pipe (Set Form, Then Flatten Other Scripts)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/form.pdf                                                              | file   |
      | stages                    | [{"op":"set-form","fields":{"name":"Иван Петров 山田太郎"}},{"op":"flatten-form"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                            | header |
    Then the response status code should be 200
    Then the response PDF(s) should be flatten
    Then the "foo.pdf" PDF should have content matching "Иван Петров 山田太郎" at page 1
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/form.pdf | file   |
      | Gotenberg-Output-Filename | foo               | header |
    Then the response status code should be 200
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

  # A value the field cannot hold is the client's mistake: the answer says what
  # the field takes. A list box that selects several takes an array.
  Scenario: POST /forms/pdf/set-form (Field Values)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files  | testdata/fields.pdf | file  |
      | fields | {"agree":"true"}    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'set-form' failed on 'fields.pdf': value the form field cannot hold: "true" for field "agree": no widget shows that state; it takes Off or one of Yes
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files  | testdata/fields.pdf | file  |
      | fields | {"color":"Blue"}    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'set-form' failed on 'fields.pdf': value the form field cannot hold: "Blue" for field "color": "Blue" is none of its options: Red, Green
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-form" endpoint with the following form data and header(s):
      | files                     | testdata/multi.pdf | file   |
      | fields                    | {"tags":["b","c"]} | field  |
      | Gotenberg-Output-Filename | foo                | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the form field "tags" set to "b, c"

  # Setting a radio button group draws the state on its widget and Off on the
  # others: the S, M and L widgets of multi.pdf sit on page 2, at x 72, 112 and
  # 152 points, and fill their center when on.
  Scenario: POST /forms/pdf/pipe (Set A Radio Button Group)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/multi.pdf                                                             | file  |
      | stages | [{"op":"set-form","fields":{"size":"M"}},{"op":"render","pages":"2","dpi":72}] | field |
    Then the response status code should be 200
    Then there should be the following file(s) in the response:
      | multi.pdf   |
      | multi_2.png |
    Then the "multi_2.png" image pixel at 82,182 should be "#ffffff"
    Then the "multi_2.png" image pixel at 122,182 should be "#000000"
    Then the "multi_2.png" image pixel at 162,182 should be "#ffffff"

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
