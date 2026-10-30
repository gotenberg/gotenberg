@pdf
@pdf-attach
Feature: /forms/pdf/{attach|detach|attachments}

  Scenario: POST /forms/pdf/{attach|attachments}
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | file                      | embed_1.xml          | field  |
      | mimeType                  | application/xml      | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attachments" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": [
          {
            "name": "embed_1.xml",
            "mimeType": "application/xml"
          }
        ]
      }
      """

  Scenario: POST /forms/pdf/attach (Name & Relationship)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | file                      | embed_1.xml          | field  |
      | name                      | data.xml             | field  |
      | relationship              | Data                 | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response PDF(s) should have the "data.xml" file embedded with relationship "Data"
    Then the response PDF(s) should NOT have the "embed_1.xml" file embedded

  Scenario: POST /forms/pdf/pipe (Many Attachments)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                         | file   |
      | files                     | testdata/embed_1.xml                                                        | file   |
      | files                     | testdata/embed_2.xml                                                        | file   |
      | stages                    | [{"op":"attach","file":"embed_1.xml"},{"op":"attach","file":"embed_2.xml"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                         | header |
    Then the response status code should be 200
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should have the "embed_2.xml" file embedded

  Scenario: POST /forms/pdf/detach
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | file                      | embed_1.xml          | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/detach" endpoint with the following form data and header(s):
      | files                     | teststore/foo.pdf | file   |
      | name                      | embed_1.xml       | field  |
      | Gotenberg-Output-Filename | bar               | header |
    Then the response status code should be 200
    Then the response PDF(s) should NOT have the "embed_1.xml" file embedded

  Scenario: POST /forms/pdf/{attach|detach} (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: attach: "file" must name an uploaded file
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf  | file  |
      | files        | testdata/embed_1.xml | file  |
      | file         | embed_1.xml          | field |
      | relationship | Owner                | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: attach: "relationship" must be Source, Data, Alternative, Supplement or Unspecified, got "Owner"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/detach" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: detach: "name" must name an embedded file
      """

  @download-from
  Scenario: POST /forms/pdf/attach (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                              | file  |
      | downloadFrom | [{"url":"{helper}/static/testdata/embed_1.xml"}] | field |
      | file         | embed_1.xml                                      | field |
    Then the response status code should be 200
    Then the response PDF(s) should have the "embed_1.xml" file embedded
