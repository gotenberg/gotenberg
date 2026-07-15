@pdfengines
@pdfengines-stamp
@stamp
@bulk-stamp
Feature: /forms/pdfengines/stamp/bulk

  Scenario: POST /forms/pdfengines/stamp/bulk (Text and Image - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp/bulk" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf    | file  |
      | stamp  | testdata/watermark.png | file  |
      | stamps | [{"source":"text","expression":"CONFIDENTIAL","options":{"rot":"45","scale":"0.5 abs"}},{"source":"image","file":"watermark.png","options":{"pos":"br","scale":"0.2 abs"}}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp/bulk (Multiple Images - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp/bulk" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf              | file  |
      | stamp  | testdata/watermark.png           | file  |
      | stamp  | testdata/html-with-asset/image.png | file  |
      | stamps | [{"source":"image","file":"watermark.png","options":{"pos":"br","scale":"0.2 abs"}},{"source":"image","file":"image.png","options":{"pos":"tl","scale":"0.15 abs"}}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  @download-from
  Scenario: POST /forms/pdfengines/stamp/bulk (PDF via Download From - pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_STAMP_ENGINES | pdfcpu |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp/bulk" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                                                                   | file  |
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_2.pdf","field":"stamp"}] | field |
      | stamps       | [{"source":"pdf","file":"page_2.pdf","pages":"1"}]                                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/stamp/bulk (Bad Request - Unknown Uploaded File for Image Source)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/stamp/bulk" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                       | file  |
      | stamps | [{"source":"image","file":"watermark.png","pages":"1"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: bulk stamp entry 0 references unknown uploaded stamp file 'watermark.png'
      """
