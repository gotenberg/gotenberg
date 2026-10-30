@pdf
@pdf-merge
Feature: /forms/pdf/merge

  Scenario: POST /forms/pdf/merge (Default)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """

  # Inputs merge in the order of their filenames, numbers compared as numbers:
  # pages_3.pdf comes before pages_12.pdf.
  Scenario: POST /forms/pdf/merge (Order)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/pages_12.pdf | file   |
      | files                     | testdata/pages_3.pdf  | file   |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 15 page(s)
    Then the "foo.pdf" PDF should have content matching "^\s*Page 3\s*$" at page 3
    Then the "foo.pdf" PDF should have content matching "^\s*Page 1\s*$" at page 4
    Then the "foo.pdf" PDF should have the following content at page 15:
      """
      Page 12
      """

  # The merged document is named after the first input.
  Scenario: POST /forms/pdf/merge (Output Name)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf  | file |
      | files | testdata/pages_12.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | pages_3.pdf |

  Scenario: POST /forms/pdf/merge (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)

  Scenario: POST /forms/pdf/merge (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdf/merge (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
      | files | testdata/page_2.pdf | file |
    Then the response status code should be 404

  @download-from
  Scenario: POST /forms/pdf/merge (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom              | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf"},{"url":"http://host.docker.internal:%d/static/testdata/page_2.pdf"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 2 page(s)

  @webhook
  Scenario: POST /forms/pdf/merge (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | files                       | testdata/page_2.pdf                          | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)
