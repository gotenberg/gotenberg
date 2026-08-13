@pdfengines
@pdfengines-optimize
@optimize
Feature: /forms/pdfengines/optimize

  # image-heavy.pdf is a ~710 KB PDF whose single image Chromium embedded
  # losslessly (FlateDecode). Re-encoding it to JPEG shrinks the file well
  # below this threshold while leaving the structure intact.
  # See https://github.com/gotenberg/gotenberg/issues/359.
  Scenario: POST /forms/pdfengines/optimize (Image-heavy PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/optimize" endpoint with the following form data and header(s):
      | files                     | testdata/image-heavy.pdf | file   |
      | Gotenberg-Output-Filename | foo                      | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" file size should be less than 300 KB

  Scenario: POST /forms/pdfengines/optimize (Custom Image Quality)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/optimize" endpoint with the following form data and header(s):
      | files                     | testdata/image-heavy.pdf | file   |
      | imageQuality              | 40                       | field  |
      | Gotenberg-Output-Filename | foo                      | header |
    Then the response status code should be 200
    Then the "foo.pdf" file size should be less than 300 KB

  Scenario: POST /forms/pdfengines/optimize (PDF Without Images)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/optimize" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/pdfengines/optimize (Invalid Image Quality)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/optimize" endpoint with the following form data and header(s):
      | files        | testdata/image-heavy.pdf | file  |
      | imageQuality | 200                      | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"

  Scenario: POST /forms/pdfengines/optimize (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/optimize" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdfengines/optimize (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/optimize" endpoint with the following form data and header(s):
      | files | testdata/image-heavy.pdf | file |
    Then the response status code should be 404
