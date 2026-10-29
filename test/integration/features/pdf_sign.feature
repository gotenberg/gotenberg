@pdf
@pdf-sign
Feature: /forms/pdf/sign

  # No timestamp authority is reachable from the tests, so the timestamp stage
  # and the timestampUrl option of the sign stage are not covered here.
  Scenario: POST /forms/pdf/sign (PEM Certificate & Key)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf   | file   |
      | files                     | testdata/pem/cert.pem | file   |
      | files                     | testdata/pem/key.pem  | file   |
      | certificate               | cert.pem              | field  |
      | key                       | key.pem               | field  |
      | reason                    | Approved              | field  |
      | location                  | Paris                 | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be signed

  Scenario: POST /forms/pdf/sign (PAdES)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf   | file  |
      | files       | testdata/pem/cert.pem | file  |
      | files       | testdata/pem/key.pem  | file  |
      | certificate | cert.pem              | field |
      | key         | key.pem               | field |
      | pades       | true                  | field |
    Then the response status code should be 200
    Then the response PDF(s) should be signed

  Scenario: POST /forms/pdf/flatten-form (Unsigned)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 200
    Then the response PDF(s) should NOT be signed

  Scenario: POST /forms/pdf/sign (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: sign: "certificate" must name an uploaded PKCS#12 file, or a PEM certificate with "key"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf   | file  |
      | files       | testdata/pem/cert.pem | file  |
      | certificate | cert.pem              | field |
      | key         | key.pem               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form file 'key.pem' is required
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf   | file  |
      | files        | testdata/pem/cert.pem | file  |
      | files        | testdata/pem/key.pem  | file  |
      | certificate  | cert.pem              | field |
      | key          | key.pem               | field |
      | timestampUrl | file:///etc/passwd    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: sign: "timestampUrl" must be an http:// or https:// URL
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                            | file  |
      | files  | testdata/pem/cert.pem                                                                          | file  |
      | files  | testdata/pem/key.pem                                                                           | file  |
      | stages | [{"op":"sign","certificate":"cert.pem","key":"key.pem"},{"op":"encrypt","userPassword":"foo"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "encrypt" cannot follow "sign", which must come last
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/timestamp" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | url   | ftp://tsa.example   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: timestamp: "url" must be an http:// or https:// URL
      """
