@pdf
@pdf-extract-attachments
Feature: /forms/pdf/extract-attachments

  Scenario: POST /forms/pdf/extract-attachments
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/attach" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | file                      | embed_1.xml          | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-attachments" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file  |
      | name  | embed_1.xml       | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should contain "xml"
    Then there should be 0 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo_embed_1.xml |
    Then the "foo_embed_1.xml" file should match string:
      """
      <xml>
          <test>test 1.1</test>
          <test>test 1.2</test>
          <test>test 1.3</test>
      </xml>
      """

  # Every attachment of every PDF, prefixed by the PDF's name so that two PDFs
  # embedding the same name do not collide in the ZIP.
  Scenario: POST /forms/pdf/extract-attachments (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                         | file   |
      | files                     | testdata/embed_1.xml                                                        | file   |
      | files                     | testdata/embed_2.xml                                                        | file   |
      | stages                    | [{"op":"attach","file":"embed_1.xml"},{"op":"attach","file":"embed_2.xml"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                         | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-attachments" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf                   | file |
      | files | testdata/page_1_with_attachment.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 0 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo_embed_1.xml                    |
      | foo_embed_2.xml                    |
      | page_1_with_attachment_embed_1.xml |
    Then the "foo_embed_2.xml" file should match string:
      """
      <xml>
          <test>test 2.1</test>
          <test>test 2.2</test>
          <test>test 2.3</test>
      </xml>
      """

  Scenario: POST /forms/pdf/pipe (Extract Attachments)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                 | file  |
      | files  | testdata/embed_1.xml                                                | file  |
      | stages | [{"op":"attach","file":"embed_1.xml"},{"op":"extract-attachments"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf         |
      | page_1_embed_1.xml |
    Then the "page_1.pdf" PDF should have 1 page(s)

  # A name the PDF does not embed, or a PDF without attachments, is the
  # client's mistake, not an empty answer.
  Scenario: POST /forms/pdf/extract-attachments (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-attachments" endpoint with the following form data and header(s):
      | files | testdata/page_1_with_attachment.pdf | file  |
      | name  | nope.xml                            | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'extract-attachments' failed on 'page_1_with_attachment.pdf': no attachment named "nope.xml"; the PDF's attachments are: embed_1.xml
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/extract-attachments" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'extract-attachments' failed on 'page_1.pdf': the PDF has no attachments
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1_with_attachment.pdf                       | file  |
      | stages | [{"op":"extract-attachments"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: "extract-attachments" must be the last stage
      """
