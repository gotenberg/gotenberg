@pdf
@pdf-stamp
Feature: /forms/pdf/{stamp-text|stamp-image|stamp-pdf}

  Scenario: POST /forms/pdf/stamp-text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | text                      | DRAFT                | field  |
      | pages                     | 1-2                  | field  |
      | x                         | 50                   | field  |
      | y                         | 50                   | field  |
      | fontSize                  | 48                   | field  |
      | font                      | Helvetica-Bold       | field  |
      | color                     | #ff0000              | field  |
      | opacity                   | 0.5                  | field  |
      | rotation                  | 45                   | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdf/stamp-image
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf    | file   |
      | files                     | testdata/watermark.png | file   |
      | image                     | watermark.png          | field  |
      | width                     | 100                    | field  |
      | height                    | 100                    | field  |
      | Gotenberg-Output-Filename | foo                    | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 image(s)

  Scenario: POST /forms/pdf/stamp-pdf
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | pdf                       | page_2.pdf          | field  |
      | scale                     | 0.5                 | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)

  Scenario: POST /forms/pdf/pipe (Many Stamps)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                           | file   |
      | files                     | testdata/watermark.png                                                                                                        | file   |
      | stages                    | [{"op":"stamp-text","text":"CONFIDENTIAL","opacity":0.3},{"op":"stamp-image","image":"watermark.png","width":40,"height":40}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                           | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 image(s)

  Scenario: POST /forms/pdf/{stamp-text|stamp-image|stamp-pdf} (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-text: "text" must not be empty
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | text  | DRAFT               | field |
      | color | red                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-text: "color" must be a #rrggbb hexadecimal color, got "red"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | text  | DRAFT               | field |
      | font  | Arial               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-text: "font" must be one of the 14 standard PDF fonts, such as Helvetica or Times-Roman, got "Arial"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | image | watermark.png       | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form file 'watermark.png' is required
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | image | logo.gif            | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-image: "image" must be a .png, .jpg or .jpeg file, got "logo.gif"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdf   | page_1.pdf          | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no PDF to process; every uploaded PDF is read by a stage
      """

  @download-from
  Scenario: POST /forms/pdf/stamp-image (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                                                      | file  |
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/watermark.png"}] | field |
      | image        | watermark.png                                                            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
