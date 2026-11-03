@pdf
@pdf-read
Feature: /forms/pdf/{info|validate}

  Scenario: POST /forms/pdf/info
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/pages_12.pdf | file |
      | files | testdata/page_1.pdf   | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "pages_12.pdf": {
          "pages": 12,
          "version": "ignore",
          "encrypted": false,
          "tagged": false,
          "linearized": false,
          "formFields": false
        },
        "page_1.pdf": {
          "pages": 1,
          "encrypted": false
        }
      }
      """

  Scenario: POST /forms/pdf/info (Encrypted PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/encrypt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | userPassword              | foo                 | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files    | teststore/foo.pdf | file  |
      | password | foo               | field |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Reading 'foo.pdf' failed: the PDF is encrypted, and the 'password' form field is missing or wrong
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files    | teststore/foo.pdf | file  |
      | password | foo               | field |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "pages": 1,
          "encrypted": true
        }
      }
      """

  Scenario: POST /forms/pdf/validate (Not Compliant, Then Remediated)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "page_1.pdf": {
          "profile": "PDF/A-2b",
          "compliant": false,
          "violations": "ignore"
        }
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | profile                   | PDF/A-2b            | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | teststore/foo.pdf | file  |
      | profile | PDF/A-2b          | field |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "profile": "PDF/A-2b",
          "compliant": true,
          "violations": []
        }
      }
      """

  # A pair of profiles, the field repeated, answers one object per file:
  # compliant when the PDF conforms to both, with the violations of both.
  Scenario: POST /forms/pdf/validate (PDF/A-2b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
      | profile | PDF/UA-1            | field |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "page_1.pdf": {
          "profile": ["PDF/A-2b", "PDF/UA-1"],
          "compliant": false,
          "violations": "ignore"
        }
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | profile                   | PDF/A-2b            | field  |
      | profile                   | PDF/UA-1            | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | teststore/foo.pdf | file  |
      | profile | PDF/A-2b          | field |
      | profile | PDF/UA-1          | field |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "profile": ["PDF/A-2b", "PDF/UA-1"],
          "compliant": true,
          "violations": []
        }
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-4             | field |
      | profile | PDF/UA-1            | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'profile' pairs a PDF/A profile with the PDF/UA profile of its PDF version: PDF/A-1a, PDF/A-1b, PDF/A-2a, PDF/A-2b, PDF/A-2u, PDF/A-3a, PDF/A-3b, PDF/A-3u or Factur-X with PDF/UA-1, and PDF/A-4, PDF/A-4e or PDF/A-4f with PDF/UA-2; got ["PDF/A-4","PDF/UA-1"]
      """

  Scenario: POST /forms/pdf/validate (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'profile' is required
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/X-4             | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: form field 'profile' must be one of
      """

  Scenario: POST /forms/pdf/info (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  # A field the route does not read answers 400; the validate route reads its
  # profile.
  Scenario: POST /forms/pdf/{info|validate} (Unknown Form Fields)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf     | file  |
      | stages | [{"op":"flatten-form"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is unknown to this route; this route takes no 'stages': send them to /forms/pdf/pipe
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/validate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
      | strict  | true                | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'strict' is unknown to this route
      """

  Scenario: POST /forms/pdf/info (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 401

  Scenario: POST /forms/pdf/info (Truncated PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/truncated.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'info' failed on 'truncated.pdf': not a valid PDF document: trailer missing /Root
      """
