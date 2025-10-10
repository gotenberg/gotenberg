Feature: /forms/libreoffice/convert/txt

  Scenario: POST /forms/libreoffice/convert/txt (Single Document)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; charset=utf-8"

  Scenario: POST /forms/libreoffice/convert/txt (Many Documents)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | files                     | testdata/page_2.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"

  # See:
  # https://github.com/gotenberg/gotenberg/issues/104
  # https://github.com/gotenberg/gotenberg/issues/730
  Scenario: POST /forms/libreoffice/convert/txt (Non-basic Latin Characters)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files                     | testdata/Special_Chars_ß.docx | file   |
      | Gotenberg-Output-Filename | foo                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; charset=utf-8"

  Scenario: POST /forms/libreoffice/convert/txt (Protected)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files | testdata/protected_page_1.docx | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      LibreOffice failed to process a document: a password may be required, or, if one has been given, it is invalid. In any case, the exact cause is uncertain.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files    | testdata/protected_page_1.docx | file  |
      | password | foo                            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; charset=utf-8"

  Scenario: POST /forms/libreoffice/convert/txt (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | password | foo | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions:
      """

  Scenario: POST /forms/libreoffice/convert/txt (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | LIBREOFFICE_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 404

  Scenario: POST /forms/libreoffice/convert/txt (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files           | testdata/page_1.docx              | file   |
      | Gotenberg-Trace | forms_libreoffice_convert_txt | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; charset=utf-8"
    Then the response header "Gotenberg-Trace" should be "forms_libreoffice_convert_txt"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_libreoffice_convert_txt" |

  Scenario: POST /forms/libreoffice/convert/txt (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.docx","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "text/plain; charset=utf-8"

  Scenario: POST /forms/libreoffice/convert/txt (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.docx                         | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "text/plain; charset=utf-8"

  Scenario: POST /forms/libreoffice/convert/txt (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/libreoffice/convert/txt (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/libreoffice/convert/txt" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "text/plain; charset=utf-8"

