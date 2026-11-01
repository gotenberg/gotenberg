@pdf
@pdf-optimize
Feature: /forms/pdf/optimize

  Scenario: POST /forms/pdf/optimize (Image-heavy PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files                     | testdata/image-heavy.pdf | file   |
      | Gotenberg-Output-Filename | foo                      | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" file size should be less than 100 KB

  Scenario: POST /forms/pdf/optimize (Selected Options)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files            | testdata/pages_12.pdf | file  |
      | recompressImages | false                 | field |
      | subsetFonts      | false                 | field |
    Then the response status code should be 200
    Then the "pages_12.pdf" PDF should have 12 page(s)

  Scenario: POST /forms/pdf/pipe (Encrypt & Optimize)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                       | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"},{"op":"optimize"}] | field |
    Then the response status code should be 200
    Then the response PDF(s) should be encrypted

  # A rewrite cannot keep a signature, and writes the document decrypted:
  # optimize refuses both unless the stages say what to do.
  Scenario: POST /forms/pdf/optimize (Signed or Encrypted PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf   | file   |
      | files                     | testdata/pem/cert.pem | file   |
      | files                     | testdata/pem/key.pem  | file   |
      | certificate               | cert.pem              | field  |
      | key                       | key.pem               | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'optimize' failed on 'foo.pdf': the PDF is signed, and optimizing would remove its signatures
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | ownerPassword             | owner               | field  |
      | Gotenberg-Output-Filename | bar                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files | teststore/bar.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'optimize' failed on 'bar.pdf': the PDF is encrypted, and optimizing would write it decrypted: on /forms/pdf/pipe, add a "decrypt" stage before "optimize" to remove the encryption, or an "encrypt" stage to keep one
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | teststore/bar.pdf                                                                  | file  |
      | stages | [{"op":"optimize"},{"op":"encrypt","ownerPassword":"owner","userPassword":"user"}] | field |
    Then the response status code should be 200
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdf/optimize (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/optimize" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | dedup | maybe               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'dedup' is invalid (got 'maybe', want true or false)
      """
