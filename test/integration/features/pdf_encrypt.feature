@pdf
@pdf-encrypt
Feature: /forms/pdf/{encrypt|decrypt}

  Scenario: POST /forms/pdf/encrypt (User Password Only)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | foo                 | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdf/encrypt (User & Owner Passwords, AES-128)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf | file  |
      | userPassword  | foo                 | field |
      | ownerPassword | bar                 | field |
      | algorithm     | aes128              | field |
    Then the response status code should be 200
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdf/encrypt (Owner Password & Restricted Permissions)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files         | testdata/page_1.pdf                   | file  |
      | ownerPassword | bar                                   | field |
      | permissions   | {"print":false,"modifyContent":false} | field |
    Then the response status code should be 200
    Then the response PDF(s) should be encrypted
    Then the response PDF(s) should NOT allow "printing"
    Then the response PDF(s) should NOT allow "modifying"
    Then the response PDF(s) should allow "copying"
    Then the response PDF(s) should allow "annotating"

  Scenario: POST /forms/pdf/encrypt (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | files        | testdata/page_2.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  Scenario: POST /forms/pdf/decrypt
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | foo                 | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files                     | teststore/foo.pdf | file   |
      | password                  | foo               | field  |
      | Gotenberg-Output-Filename | bar               | header |
    Then the response status code should be 200
    Then the response PDF(s) should NOT be encrypted

  Scenario: POST /forms/pdf/rotate (Encrypted Input)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | foo                 | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/rotate" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file  |
      | angle | 90                | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Stage 'rotate' failed on 'foo.pdf': open PDF:
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files    | teststore/foo.pdf | file  |
      | password | wrong             | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Stage 'decrypt' failed on 'foo.pdf':
      """

  Scenario: POST /forms/pdf/encrypt (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: encrypt: set "userPassword", "ownerPassword", or both
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
      | algorithm    | des                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: encrypt: "algorithm" must be aes256, aes128, rc4-128 or rc4-40, got "des"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
      | permissions  | foo                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'permissions' is invalid: must be valid JSON, got "foo"
      """

  Scenario: POST /forms/pdf/encrypt (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | userPassword | foo                 | field |
    Then the response status code should be 404
