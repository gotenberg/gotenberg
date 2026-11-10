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

  Scenario: POST /forms/pdf/stamp-text (Exclusions)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | text                      | DRAFT                | field  |
      | pages                     | !1                   | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should NOT have content matching "DRAFT" at page 1, artifacts included
    Then the "foo.pdf" PDF should have content matching "DRAFT" at page 2, artifacts included
    Then the "foo.pdf" PDF should have content matching "DRAFT" at page 3, artifacts included

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
      | source                    | page_2.pdf          | field  |
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

  # page_1.pdf is 446.25 by 631.5 points, so its center shows at 223,315 at 72
  # dpi. A 100-point square centered there spans 173 to 273 across and 265 to
  # 365 down.
  Scenario: POST /forms/pdf/stamp-image (Position)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                                        | file  |
      | files  | testdata/watermark.png                                                                                                     | file  |
      | stages | [{"op":"stamp-image","image":"watermark.png","width":100,"height":100,"position":"center"},{"op":"render","pages":"1","dpi":72}] | field |
    Then the response status code should be 200
    Then there should be the following file(s) in the response:
      | page_1.pdf   |
      | page_1_1.png |
    Then the "page_1_1.png" image pixel at 223,315 should be "#ff0000"
    Then the "page_1_1.png" image pixel at 223,270 should be "#ff0000"
    Then the "page_1_1.png" image pixel at 223,260 should be "#ffffff"
    Then the "page_1_1.png" image pixel at 280,315 should be "#ffffff"

  # Turned a quarter, page_1.pdf shows 631.5 points wide and 446.25 tall. The
  # stamp sits in the top-left corner a viewer sees, the same way up: 100
  # points across and 50 down.
  Scenario: POST /forms/pdf/pipe (Stamp Position On A Rotated Page)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                                                                        | file  |
      | files  | testdata/watermark.png                                                                                                                                     | file  |
      | stages | [{"op":"rotate","angle":90},{"op":"stamp-image","image":"watermark.png","width":100,"height":50,"position":"top-left"},{"op":"render","pages":"1","dpi":72}] | field |
    Then the response status code should be 200
    Then the "page_1_1.png" image should be 632x447 pixels
    Then the "page_1_1.png" image pixel at 50,25 should be "#ff0000"
    Then the "page_1_1.png" image pixel at 95,45 should be "#ff0000"
    Then the "page_1_1.png" image pixel at 110,25 should be "#ffffff"
    Then the "page_1_1.png" image pixel at 50,60 should be "#ffffff"

  # The text's ink centers on the page, so the stem of a lone I covers the
  # center, turned or not.
  Scenario: POST /forms/pdf/stamp-text (Position)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                                                         | file  |
      | stages | [{"op":"stamp-text","text":"I","fontSize":200,"color":"#ff0000","position":"center","rotation":45},{"op":"render","pages":"1","dpi":72}] | field |
    Then the response status code should be 200
    Then the "page_1_1.png" image pixel at 223,315 should be "#ff0000"
    Then the "page_1_1.png" image pixel at 223,200 should be "#ffffff"
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | text                      | DRAFT                | field  |
      | position                  | bottom-right         | field  |
      | x                         | -20                  | field  |
      | y                         | 20                   | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "DRAFT" at page 3, artifacts included

  Scenario: POST /forms/pdf/stamp-pdf (Position)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | source                    | page_2.pdf          | field  |
      | scale                     | 0.25                | field  |
      | position                  | top-right           | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)

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
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf | file  |
      | text     | DRAFT               | field |
      | position | middle              | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-text: "position" must be center, top-left, top, top-right, left, right, bottom-left, bottom or bottom-right, got "middle"
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
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | files | testdata/broken.png | file  |
      | image | broken.png          | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'stamp-image' failed on 'page_1.pdf': invalid image data: png: invalid format: not a PNG file; send a valid PNG or JPEG image, its extension naming its format (.png, .jpg or .jpeg)
      """
    # The fonts installed in the image draw the text a standard font cannot.
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | text  | 日本 Кириллица        | field |
    Then the response status code should be 200
    Then the "page_1.pdf" PDF should have content matching "日本 Кириллица" at page 1, artifacts included
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | text  | ͸                   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'stamp-text' failed on 'page_1.pdf': no font covers the text: no font has '\u0378'; the fonts installed in the image draw Latin, Greek, Cyrillic, Chinese, Japanese and Korean text, among others: install a font that has these characters, see https://gotenberg.dev/docs/troubleshooting#install-fonts
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | source | page_1.pdf          | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no PDF to process; every uploaded PDF is read by a stage
      """
    # litho reports the failure from its write: the message keeps its cause
    # only.
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files      | testdata/page_1.pdf | file  |
      | files      | testdata/page_2.pdf | file  |
      | source     | page_2.pdf          | field |
      | sourcePage | 9                   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'stamp-pdf' failed on 'page_1.pdf': pdf stamp: source page 9: page number out of bounds (source has 1 pages)
      """

  @download-from
  Scenario: POST /forms/pdf/stamp-image (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                                | file  |
      | downloadFrom | [{"url":"{helper}/static/testdata/watermark.png"}] | field |
      | image        | watermark.png                                      | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/pdf/{stamp-text|stamp-pdf} (Invalid Numbers)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf | file  |
      | text     | DRAFT               | field |
      | fontSize | -4                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-text: "fontSize" must be greater than 0, got -4
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf | file  |
      | text     | DRAFT               | field |
      | fontSize | 0                   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-text: "fontSize" must be greater than 0, got 0
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | files  | testdata/page_2.pdf | file  |
      | source | page_2.pdf          | field |
      | scale  | -1                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: stamp-pdf: "scale" must be greater than 0, got -1
      """
