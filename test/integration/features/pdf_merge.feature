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
      | downloadFrom              | [{"url":"{helper}/static/testdata/page_1.pdf"},{"url":"{helper}/static/testdata/page_2.pdf"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 2 page(s)

  # A host that does not resolve is the client's mistake, and says nothing of
  # the filters. An allow-list match skips the lookup: none here.
  @download-from
  Scenario: POST /forms/pdf/merge (Download From An Unresolvable Host)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DOWNLOAD_FROM_ALLOW_LIST |  |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                        | file  |
      | downloadFrom | [{"url":"http://nope.invalid/page_2.pdf"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'downloadFrom' entry 0 is invalid (the host 'nope.invalid' does not resolve)
      """

  @download-from
  Scenario: POST /forms/pdf/merge (Download From A Non-HTTP URL)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                 | file  |
      | downloadFrom | [{"url":"ftp://example.com/a.pdf"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'downloadFrom' entry 0 is invalid (want an absolute URL with an http or https scheme and a host, such as https://example.com/file.pdf)
      """

  @download-from
  Scenario: POST /forms/pdf/merge (Download From Invalid Value)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"   "}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'downloadFrom' entry 0 is invalid (URL must be set)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom | [{"url": | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'downloadFrom' is invalid (want a JSON array such as [{"url":"https://example.com/file.pdf"}]: unexpected EOF)
      """

  # Object stores such as S3 often serve files without a Content-Disposition
  # header: the URL path names the file, or the entry does.
  @download-from
  Scenario: POST /forms/pdf/merge (Download From A Response Without Content-Disposition)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom              | [{"url":"{helper}/raw/testdata/page_1.pdf"},{"url":"{helper}/unnamed?path=testdata/page_2.pdf","filename":"page_2.pdf"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 2 page(s)
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"{helper}/unnamed?path=testdata/page_1.pdf"}] | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      nothing names the file. Set 'filename' on the entry, such as [{"url":"https://example.com/file","filename":"file.pdf"}], or make the server send a 'Content-Disposition' header with a filename
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"{helper}/raw/testdata/page_1.pdf","filename":" "}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'downloadFrom' entry 0 is invalid (filename must hold a name, such as 'file.pdf', or be omitted)
      """

  # The message names no address: the one the container resolved is internal.
  @download-from
  Scenario: POST /forms/pdf/merge (Download From An Unreachable Host)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DOWNLOAD_FROM_MAX_RETRY | 0 |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://127.0.0.1:1/a.pdf"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Unable to download file from 'http://127.0.0.1:1/a.pdf': the host could not be reached
      """

  @webhook
  Scenario: POST /forms/pdf/merge (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | files                        | testdata/page_2.pdf     | file   |
      | Gotenberg-Output-Filename    | foo                     | header |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)

  Scenario: POST /forms/pdf/merge (Truncated PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files | testdata/truncated.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'merge' failed on 'truncated.pdf': not a valid PDF document: trailer missing /Root
      """
