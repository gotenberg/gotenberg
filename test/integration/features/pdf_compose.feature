@pdf
@pdf-compose
Feature: /forms/pdf/compose

  # A 6 by 5 inch label: page 1 of pages_3.pdf on the top half, page 2 of
  # pages_12.pdf on the bottom half, each at a quarter of its size.
  Scenario: POST /forms/pdf/compose (Label)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf                                                                                                    | file   |
      | files                     | testdata/pages_12.pdf                                                                                                   | file   |
      | paperWidth                | 6in                                                                                                                     | field  |
      | paperHeight               | 5in                                                                                                                     | field  |
      | placements                | [{"source":"pages_3.pdf","x":0,"y":180,"scale":0.25},{"source":"pages_12.pdf","sourcePage":2,"x":0,"y":0,"scale":0.25}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                     | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF first page should be 432x360 points
    Then the "foo.pdf" PDF should have content matching "Page 1" at page 1
    Then the "foo.pdf" PDF should have content matching "Page 2" at page 1

  # Without a source, a placement draws from the only PDF. A placement's page
  # sets the composed page it lands on.
  Scenario: POST /forms/pdf/compose (Paper Size & Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf                                                          | file   |
      | paperSize                 | A4                                                                            | field  |
      | placements                | [{"x":0,"y":0,"scale":0.5},{"sourcePage":2,"page":2,"x":0,"y":0,"scale":0.5}] | field  |
      | Gotenberg-Output-Filename | foo                                                                           | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF page 1 should be 595x842 points
    Then the "foo.pdf" PDF page 2 should be 595x842 points

  Scenario: POST /forms/pdf/pipe (Compose)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf                                                                                                                                                              | file   |
      | stages                    | [{"op":"compose","paperWidth":"432pt","paperHeight":"360pt","placements":[{"source":"pages_3.pdf","scale":0.25}]},{"op":"stamp-text","text":"DRAFT","x":36,"y":36,"fontSize":18}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                               | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have content matching "DRAFT" at page 1, artifacts included

  # Inputs come in the order of their filenames, numbers compared as numbers:
  # pages_3.pdf comes before pages_12.pdf, and names the stage's failures.
  Scenario: POST /forms/pdf/compose (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf | file  |
      | placements | [{}]                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: compose: set "paperSize", or "paperWidth" and "paperHeight"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf | file  |
      | paperSize  | A4                   | field |
      | placements | []                   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: compose: "placements" must list at least one placement
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf    | file  |
      | paperSize  | A4                      | field |
      | placements | [{"source":"nope.pdf"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'compose' failed on 'pages_3.pdf': "source" must name an uploaded PDF, got "nope.pdf"; the uploaded PDFs are pages_3.pdf
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf       | file  |
      | files      | testdata/pages_12.pdf      | file  |
      | paperSize  | A4                         | field |
      | placements | [{"source":"pages_3.pdf"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'compose' failed on 'pages_3.pdf': "placements" must place every uploaded PDF: "pages_12.pdf" is not placed
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/compose" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf | file  |
      | paperSize  | A4                   | field |
      | placements | [{"sourcePage":9}]   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'compose' failed on 'pages_3.pdf': invalid placement: placement 0: page 9 lies outside its source's 3 pages
      """
