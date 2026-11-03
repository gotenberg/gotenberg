@pdf
@pdf-sign
Feature: /forms/pdf/sign

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

  # A stage that rewrites a signed PDF would drop its signatures, even after an
  # incremental edit; an incremental edit alone keeps them.
  Scenario: POST /forms/pdf/pipe (Signed PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf   | file   |
      | files                     | testdata/pem/cert.pem | file   |
      | files                     | testdata/pem/key.pem  | file   |
      | certificate               | cert.pem              | field  |
      | key                       | key.pem               | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | teststore/foo.pdf                                                | file  |
      | stages | [{"op":"stamp-text","text":"DRAFT"},{"op":"select","pages":"1"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'select' failed on 'foo.pdf': the PDF is signed, and this stage rewrites it, which would remove its signatures
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | teststore/foo.pdf | file  |
      | profile | PDF/A-2b          | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'remediate' failed on 'foo.pdf': the PDF is signed, and this stage rewrites it, which would remove its signatures
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | teststore/foo.pdf                    | file   |
      | stages                    | [{"op":"stamp-text","text":"DRAFT"}] | field  |
      | Gotenberg-Output-Filename | bar                                  | header |
    Then the response status code should be 200
    Then the response PDF(s) should be signed

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
      Invalid form data: form field 'stages' is invalid: "encrypt" cannot be combined with "sign" or "timestamp": a signature covers the written bytes
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                                                                | file  |
      | files  | testdata/pem/cert.pem                                                                                                                              | file  |
      | files  | testdata/pem/key.pem                                                                                                                               | file  |
      | stages | [{"op":"sign","certificate":"cert.pem","key":"key.pem","reason":"First"},{"op":"sign","certificate":"cert.pem","key":"key.pem","reason":"Second"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: only one "sign" stage is allowed
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                                                                                | file  |
      | files  | testdata/pem/cert.pem                                                                                                                              | file  |
      | files  | testdata/pem/key.pem                                                                                                                               | file  |
      | stages | [{"op":"decrypt","password":"foo"},{"op":"sign","certificate":"cert.pem","key":"key.pem"},{"op":"timestamp","timestampUrl":"https://tsa.example"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: "decrypt" cannot be combined with "sign" or "timestamp": a PDF that was encrypted cannot be signed in the request that decrypts it; decrypt it in a request of its own, then sign the result
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/timestamp" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | timestampUrl | ftp://tsa.example   | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: timestamp: "timestampUrl" must be an http:// or https:// URL
      """

  # The helper server answers as an RFC 3161 timestamp authority.
  Scenario: POST /forms/pdf/sign (Timestamp Authority)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf   | file  |
      | files        | testdata/pem/cert.pem | file  |
      | files        | testdata/pem/key.pem  | file  |
      | certificate  | cert.pem              | field |
      | key          | key.pem               | field |
      | timestampUrl | {helper}/tsa          | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be signed

  Scenario: POST /forms/pdf/timestamp
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/timestamp" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf | file  |
      | timestampUrl | {helper}/tsa        | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be signed

  Scenario: POST /forms/pdf/sign (PKCS #12 Store)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files               | testdata/page_1.pdf   | file  |
      | files               | testdata/pem/cert.p12 | file  |
      | certificate         | cert.p12              | field |
      | certificatePassword | secret                | field |
    Then the response status code should be 200
    Then the response PDF(s) should be signed

  Scenario: POST /forms/pdf/sign (PKCS #12 Wrong Password)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files               | testdata/page_1.pdf   | file  |
      | files               | testdata/pem/cert.p12 | file  |
      | certificate         | cert.p12              | field |
      | certificatePassword | nope                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'sign' failed on 'page_1.pdf': the PKCS #12 certificate "cert.p12" did not open: "certificatePassword" is missing or wrong
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf   | file  |
      | files       | testdata/pem/cert.p12 | file  |
      | certificate | cert.p12              | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'sign' failed on 'page_1.pdf': the PKCS #12 certificate "cert.p12" did not open: "certificatePassword" is missing or wrong
      """

  Scenario: POST /forms/pdf/sign (PEM Certificate Without Key)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/sign" endpoint with the following form data and header(s):
      | files       | testdata/page_1.pdf   | file  |
      | files       | testdata/pem/cert.pem | file  |
      | certificate | cert.pem              | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'sign' failed on 'page_1.pdf': "certificate" "cert.pem" is a PEM file: "key" is required, the PEM private key that goes with it
      """
