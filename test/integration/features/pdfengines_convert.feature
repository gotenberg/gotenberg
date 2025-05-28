# TODO:
# 1. PDF/UA-2.

Feature: /forms/pdfengines/convert

  Scenario: POST /forms/pdfengines/convert (Single PDF/A-1b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | PDF/A-1b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Single PDF/A-2b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | PDF/A-2b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 1 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Single PDF/A-3b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | PDF/A-3b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 1 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Single PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfua | true                | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Single PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | PDF/A-1b            | field |
      | pdfua | true                | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | files | testdata/page_2.pdf | file  |
      | pdfa  | PDF/A-1b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: either 'pdfa' or 'pdfua' form fields must be provided
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | pdfa | PDF/A-1b | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF format, while others may have failed to convert due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfua | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'pdfua' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      """

  Scenario: POST /forms/pdfengines/convert (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf      | file   |
      | pdfa            | PDF/A-1b                 | field  |
      | Gotenberg-Trace | forms_pdfengines_convert | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_convert"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_convert" |

  Scenario: POST /forms/pdfengines/convert (Output Filename - Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | pdfa                      | PDF/A-1b            | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |

  Scenario: POST /forms/pdfengines/convert (Output Filename - Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | pdfa                      | PDF/A-1b            | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | foo.zip    |
      | page_1.pdf |
      | page_2.pdf |

  Scenario: POST /forms/pdfengines/convert (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | pdfa         | PDF/A-1b                                                                                                 | field |
    Then the file request header "X-Foo" should be "bar"
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/pdfengines/convert (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | pdfa                        | PDF/A-1b                                     | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the webhook request PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)

  Scenario: POST /forms/pdfengines/convert (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | PDF/A-1b            | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/convert (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pdfa  | PDF/A-1b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
