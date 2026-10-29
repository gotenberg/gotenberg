@pdf
@pdf-pages
Feature: /forms/pdf/{select|delete|insert}

  Scenario: POST /forms/pdf/select
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/select" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | pages                     | 3,1                  | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 1
      """

  Scenario: POST /forms/pdf/delete
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/delete" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | pages                     | odd                  | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """

  Scenario: POST /forms/pdf/insert
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/insert" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf  | file   |
      | files                     | testdata/pages_12.pdf | file   |
      | source                    | pages_12.pdf          | field  |
      | at                        | 1                     | field  |
      | sourcePages               | last                  | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 4 page(s)
    Then the "foo.pdf" PDF should have content matching "^\s*Page 1\s*$" at page 1
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 12
      """
    Then the "foo.pdf" PDF should have content matching "^\s*Page 2\s*$" at page 3

  Scenario: POST /forms/pdf/pipe (Select, Delete & Insert)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/pages_12.pdf                                                                                        | file   |
      | files                     | testdata/pages_3.pdf                                                                                         | file   |
      | stages                    | [{"op":"select","pages":"1-6"},{"op":"delete","pages":"even"},{"op":"insert","source":"pages_3.pdf","at":3}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                          | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 6 page(s)

  Scenario: POST /forms/pdf/{select|delete|insert} (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/select" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: select: "pages" must not be empty, for example "1-3,5", "odd" or "last"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/delete" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file  |
      | pages | 4                    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'delete' failed on 'pages_3.pdf': invalid page range: page 4 out of range [1, 3]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/insert" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: insert: "source" must name an uploaded PDF
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/insert" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf | file  |
      | source | pages_3.pdf          | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no PDF to process; every uploaded PDF is read by a stage
      """
