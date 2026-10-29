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

  Scenario: POST /forms/pdf/info (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 401
