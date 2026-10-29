@pdf
@pdf-split
Feature: /forms/pdf/split

  Scenario: POST /forms/pdf/split (Every)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file  |
      | every | 2                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  Scenario: POST /forms/pdf/split (Ranges)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf | file  |
      | ranges | 2-3,1                | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  Scenario: POST /forms/pdf/split (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf  | file  |
      | files | testdata/pages_12.pdf | file  |
      | every | 10                    | field |
    Then the response status code should be 200
    Then there should be 3 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_12_0.pdf |
      | pages_12_1.pdf |
      | pages_3_0.pdf  |
    Then the "pages_12_0.pdf" PDF should have 10 page(s)
    Then the "pages_12_1.pdf" PDF should have 2 page(s)
    Then the "pages_3_0.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdf/split (Output Filename - Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | every                     | 2                    | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | foo.zip       |
      | pages_3_0.pdf |
      | pages_3_1.pdf |

  Scenario: POST /forms/pdf/split (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: split: set "every" or "ranges"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf | file  |
      | every  | 2                    | field |
      | ranges | 1                    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: split: set "every" or "ranges", not both
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file  |
      | every | foo                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'every' is invalid: must be a number, got "foo"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf | file  |
      | ranges | 1,9                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'split' failed on 'pages_3.pdf': invalid page range: page 9 out of range [1, 3]
      """

  @webhook
  Scenario: POST /forms/pdf/split (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/split" endpoint with the following form data and header(s):
      | files                       | testdata/pages_3.pdf                         | file   |
      | every                       | 2                                            | field  |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the webhook request
