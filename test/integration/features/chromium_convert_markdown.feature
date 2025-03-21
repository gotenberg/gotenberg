# TODO:
# 1. JavaScript disabled on some feature.

Feature: /forms/chromium/convert/markdown

  Scenario: POST /forms/chromium/convert/markdown (Default)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html | file   |
      | files                     | testdata/page-1-markdown/page_1.md  | file   |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  Scenario: POST /forms/chromium/convert/markdown (Single Page)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-markdown/index.html | file   |
      | files                     | testdata/pages-12-markdown/page_1.md  | file   |
      | files                     | testdata/pages-12-markdown/page_2.md  | file   |
      | files                     | testdata/pages-12-markdown/page_3.md  | file   |
      | files                     | testdata/pages-12-markdown/page_4.md  | file   |
      | files                     | testdata/pages-12-markdown/page_5.md  | file   |
      | files                     | testdata/pages-12-markdown/page_6.md  | file   |
      | files                     | testdata/pages-12-markdown/page_7.md  | file   |
      | files                     | testdata/pages-12-markdown/page_8.md  | file   |
      | files                     | testdata/pages-12-markdown/page_9.md  | file   |
      | files                     | testdata/pages-12-markdown/page_10.md | file   |
      | files                     | testdata/pages-12-markdown/page_11.md | file   |
      | files                     | testdata/pages-12-markdown/page_12.md | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 12 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 12:
      """
      Page 12
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-markdown/index.html | file   |
      | files                     | testdata/pages-12-markdown/page_1.md  | file   |
      | files                     | testdata/pages-12-markdown/page_2.md  | file   |
      | files                     | testdata/pages-12-markdown/page_3.md  | file   |
      | files                     | testdata/pages-12-markdown/page_4.md  | file   |
      | files                     | testdata/pages-12-markdown/page_5.md  | file   |
      | files                     | testdata/pages-12-markdown/page_6.md  | file   |
      | files                     | testdata/pages-12-markdown/page_7.md  | file   |
      | files                     | testdata/pages-12-markdown/page_8.md  | file   |
      | files                     | testdata/pages-12-markdown/page_9.md  | file   |
      | files                     | testdata/pages-12-markdown/page_10.md | file   |
      | files                     | testdata/pages-12-markdown/page_11.md | file   |
      | files                     | testdata/pages-12-markdown/page_12.md | file   |
      | singlePage                | true                                  | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      # page-break-after: always; tells the browser's print engine to force a page break after each element,
      # even when calculating a large enough paper height, Chromium's PDF rendering will still honor those page break
      # directives.
      """
      Page 12
      """

  Scenario: POST /forms/chromium/convert/markdown (Landscape)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html | file   |
      | files                     | testdata/page-1-markdown/page_1.md  | file   |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should NOT be set to landscape orientation
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html | file   |
      | files                     | testdata/page-1-markdown/page_1.md  | file   |
      | landscape                 | true                                | field  |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should be set to landscape orientation

  Scenario: POST /forms/chromium/convert/markdown (Native Page Ranges)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-markdown/index.html | file   |
      | files                     | testdata/pages-12-markdown/page_1.md  | file   |
      | files                     | testdata/pages-12-markdown/page_2.md  | file   |
      | files                     | testdata/pages-12-markdown/page_3.md  | file   |
      | files                     | testdata/pages-12-markdown/page_4.md  | file   |
      | files                     | testdata/pages-12-markdown/page_5.md  | file   |
      | files                     | testdata/pages-12-markdown/page_6.md  | file   |
      | files                     | testdata/pages-12-markdown/page_7.md  | file   |
      | files                     | testdata/pages-12-markdown/page_8.md  | file   |
      | files                     | testdata/pages-12-markdown/page_9.md  | file   |
      | files                     | testdata/pages-12-markdown/page_10.md | file   |
      | files                     | testdata/pages-12-markdown/page_11.md | file   |
      | files                     | testdata/pages-12-markdown/page_12.md | file   |
      | nativePageRanges          | 2-3                                   | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/chromium/convert/markdown (Header & Footer)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-markdown/index.html   | file   |
      | files                     | testdata/pages-12-markdown/page_1.md    | file   |
      | files                     | testdata/pages-12-markdown/page_2.md    | file   |
      | files                     | testdata/pages-12-markdown/page_3.md    | file   |
      | files                     | testdata/pages-12-markdown/page_4.md    | file   |
      | files                     | testdata/pages-12-markdown/page_5.md    | file   |
      | files                     | testdata/pages-12-markdown/page_6.md    | file   |
      | files                     | testdata/pages-12-markdown/page_7.md    | file   |
      | files                     | testdata/pages-12-markdown/page_8.md    | file   |
      | files                     | testdata/pages-12-markdown/page_9.md    | file   |
      | files                     | testdata/pages-12-markdown/page_10.md   | file   |
      | files                     | testdata/pages-12-markdown/page_11.md   | file   |
      | files                     | testdata/pages-12-markdown/page_12.md   | file   |
      | files                     | testdata/header-footer-html/header.html | file   |
      | files                     | testdata/header-footer-html/footer.html | file   |
      | Gotenberg-Output-Filename | foo                                     | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 12 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Pages 12
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      1 of 12
      """
    Then the "foo.pdf" PDF should have the following content at page 12:
      """
      Pages 12
      """
    Then the "foo.pdf" PDF should have the following content at page 12:
      """
      12 of 12
      """

  Scenario: POST /forms/chromium/convert/markdown (Wait Delay)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Wait delay > 2 seconds or expression window globalVar === 'ready' returns true.
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | waitDelay                 | 2.5s                                      | field  |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Wait delay > 2 seconds or expression window globalVar === 'ready' returns true.
      """

  Scenario: POST /forms/chromium/convert/markdown (Wait For Expression)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Wait delay > 2 seconds or expression window globalVar === 'ready' returns true.
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | waitForExpression         | window.globalVar === 'ready'              | field  |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Wait delay > 2 seconds or expression window globalVar === 'ready' returns true.
      """

  Scenario: POST /forms/chromium/convert/markdown (Emulated Media Type)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Emulated media type is 'print'.
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Emulated media type is 'screen'.
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | emulatedMediaType         | print                                     | field  |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Emulated media type is 'print'.
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Emulated media type is 'screen'.
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | emulatedMediaType         | screen                                    | field  |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Emulated media type is 'screen'.
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Emulated media type is 'print'.
      """

  Scenario: POST /forms/chromium/convert/markdown (Default Allow / Deny Lists)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-markdown/index.html | file |
      | files | testdata/feature-rich-markdown/table.md   | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the Gotenberg container should log the following entries:
      | 'file:///etc/passwd' matches the expression from the denied list |

  Scenario: POST /forms/chromium/convert/markdown (Main URL does NOT match allowed list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^file:(?!//\\/tmp/).* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-markdown/index.html | file |
      | files | testdata/feature-rich-markdown/table.md   | file |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: POST /forms/chromium/convert/markdown (Main URL does match denied list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |                |
      | CHROMIUM_DENY_LIST  | ^file:///tmp.* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-markdown/index.html | file |
      | files | testdata/feature-rich-markdown/table.md   | file |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: POST /forms/chromium/convert/markdown (Request does not match the allowed list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^file:///tmp.* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-markdown/index.html | file |
      | files | testdata/feature-rich-markdown/table.md   | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the Gotenberg container should log the following entries:
      | 'file:///etc/passwd' does not match the expression from the allowed list |

  Scenario: POST /forms/chromium/convert/markdown (JavaScript Enabled)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      JavaScript is enabled.
      """

  Scenario: POST /forms/chromium/convert/markdown (JavaScript Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DISABLE_JAVASCRIPT | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-markdown/index.html | file   |
      | files                     | testdata/feature-rich-markdown/table.md   | file   |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      JavaScript is enabled.
      """

  Scenario: POST /forms/chromium/convert/markdown (Fail On Resource HTTP Status Codes)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                         | testdata/feature-rich-markdown/index.html | file  |
      | files                         | testdata/feature-rich-markdown/table.md   | file  |
      | failOnResourceHttpStatusCodes | [499,599]                                 | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid HTTP status code from resources:
      https://httpstat.us/400 - 400: Bad Request
      """

  Scenario: POST /forms/chromium/convert/markdown (Fail On Resource Loading Failed)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                       | testdata/feature-rich-markdown/index.html | file  |
      | files                       | testdata/feature-rich-markdown/table.md   | file  |
      | failOnResourceLoadingFailed | true                                      | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Chromium failed to load resources
      """
    Then the response body should contain string:
      """
      resource Stylesheet: net::ERR_CONNECTION_REFUSED
      """
    Then the response body should contain string:
      """
      resource Stylesheet: net::ERR_FILE_NOT_FOUND
      """

  Scenario: POST /forms/chromium/convert/markdown (Fail On Console Exceptions)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                   | testdata/feature-rich-markdown/index.html | file  |
      | files                   | testdata/feature-rich-markdown/table.md   | file  |
      | failOnConsoleExceptions | true                                      | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Chromium console exceptions
      """
    Then the response body should contain string:
      """
      exception "Uncaught" (93:12): Error: Exception 1
      """
    Then the response body should contain string:
      """
      exception "Uncaught" (97:12): Error: Exception 2
      """

  Scenario: POST /forms/chromium/convert/markdown (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/pages-3-markdown/index.html | file |
      | files | testdata/pages-3-markdown/page_1.md  | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Markdown file(s) not found: 'page_2.md'; 'page_3.md'
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | singlePage                    | foo | field |
      | paperWidth                    | foo | field |
      | paperHeight                   | foo | field |
      | marginTop                     | foo | field |
      | marginBottom                  | foo | field |
      | marginLeft                    | foo | field |
      | marginRight                   | foo | field |
      | preferCssPageSize             | foo | field |
      | generateDocumentOutline       | foo | field |
      | printBackground               | foo | field |
      | omitBackground                | foo | field |
      | landscape                     | foo | field |
      | scale                         | foo | field |
      | waitDelay                     | foo | field |
      | emulatedMediaType             | foo | field |
      | failOnHttpStatusCodes         | foo | field |
      | failOnResourceHttpStatusCodes | foo | field |
      | failOnResourceLoadingFailed   | foo | field |
      | failOnConsoleExceptions       | foo | field |
      | skipNetworkIdleEvent          | foo | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'skipNetworkIdleEvent' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'failOnHttpStatusCodes' is invalid (got 'foo', resulting to unmarshal failOnHttpStatusCodes: invalid character 'o' in literal false (expecting 'a')); form field 'failOnResourceHttpStatusCodes' is invalid (got 'foo', resulting to unmarshal failOnResourceHttpStatusCodes: invalid character 'o' in literal false (expecting 'a')); form field 'failOnResourceLoadingFailed' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'failOnConsoleExceptions' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'waitDelay' is invalid (got 'foo', resulting to time: invalid duration "foo"); form field 'emulatedMediaType' is invalid (got 'foo', resulting to wrong value, expected either 'screen', 'print' or empty); form field 'omitBackground' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'landscape' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'printBackground' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'scale' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'singlePage' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'paperWidth' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'paperHeight' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'marginTop' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'marginBottom' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'marginLeft' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'marginRight' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax); form field 'preferCssPageSize' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'generateDocumentOutline' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form file 'index.html' is required; no form file found for extensions: [.md]
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files          | testdata/page-1-markdown/index.html | file  |
      | files          | testdata/page-1-markdown/page_1.md  | file  |
      | omitBackground | true                                | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      omitBackground requires printBackground set to true
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files        | testdata/page-1-markdown/index.html | file  |
      | files        | testdata/page-1-markdown/page_1.md  | file  |
      | paperWidth   | 0                                   | field |
      | paperHeight  | 0                                   | field |
      | marginTop    | 1000000                             | field |
      | marginBottom | 1000000                             | field |
      | marginLeft   | 1000000                             | field |
      | marginRight  | 1000000                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium does not handle the provided settings; please check for aberrant form values
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files            | testdata/page-1-markdown/index.html | file  |
      | files            | testdata/page-1-markdown/page_1.md  | file  |
      | nativePageRanges | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium does not handle the page ranges 'foo' (nativePageRanges)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files             | testdata/page-1-markdown/index.html | file  |
      | files             | testdata/page-1-markdown/page_1.md  | file  |
      | waitForExpression | undefined                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The expression 'undefined' (waitForExpression) returned an exception or undefined
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files   | testdata/page-1-markdown/index.html | file  |
      | files   | testdata/page-1-markdown/page_1.md  | file  |
      | cookies | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'cookies' is invalid (got 'foo', resulting to unmarshal cookies: invalid character 'o' in literal false (expecting 'a'))
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files   | testdata/page-1-markdown/index.html       | file  |
      | files   | testdata/page-1-markdown/page_1.md        | file  |
      | cookies | [{"name":"yummy_cookie","value":"choco"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'cookies' is invalid (got '[{"name":"yummy_cookie","value":"choco"}]', resulting to cookie 0 must have its name, value and domain set)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files            | testdata/page-1-markdown/index.html | file  |
      | files            | testdata/page-1-markdown/page_1.md  | file  |
      | extraHttpHeaders | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got 'foo', resulting to unmarshal extraHttpHeaders: invalid character 'o' in literal false (expecting 'a'))
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files            | testdata/page-1-markdown/index.html | file  |
      | files            | testdata/page-1-markdown/page_1.md  | file  |
      | extraHttpHeaders | {"foo":"bar;scope;;"}               | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got '{"foo":"bar;scope;;"}', resulting to invalid scope '' for header 'foo')
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files            | testdata/page-1-markdown/index.html | file  |
      | files            | testdata/page-1-markdown/page_1.md  | file  |
      | extraHttpHeaders | {"foo":"bar;scope=*."}              | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got '{"foo":"bar;scope=*."}', resulting to invalid scope regex pattern for header 'foo': error parsing regexp: missing argument to repetition operator in `*.`)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/page-1-markdown/index.html | file  |
      | files     | testdata/page-1-markdown/page_1.md  | file  |
      | splitMode | foo                                 | field |
      | splitSpan | 2                                   | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitMode' is invalid (got 'foo', resulting to wrong value, expected either 'intervals' or 'pages')
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/page-1-markdown/index.html | file  |
      | files     | testdata/page-1-markdown/page_1.md  | file  |
      | splitMode | intervals                           | field |
      | splitSpan | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitSpan' is invalid (got 'foo', resulting to strconv.Atoi: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/page-1-markdown/index.html | file  |
      | files     | testdata/page-1-markdown/page_1.md  | file  |
      | splitMode | pages                               | field |
      | splitSpan | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF split mode, while others may have failed to split due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file  |
      | files | testdata/page-1-markdown/page_1.md  | file  |
      | pdfa  | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF format, while others may have failed to convert due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file  |
      | files | testdata/page-1-markdown/page_1.md  | file  |
      | pdfua | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'pdfua' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files    | testdata/page-1-markdown/index.html | file  |
      | files    | testdata/page-1-markdown/page_1.md  | file  |
      | metadata | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'metadata' is invalid (got 'foo', resulting to unmarshal metadata: invalid character 'o' in literal false (expecting 'a'))
      """

  Scenario: POST /forms/chromium/convert/markdown (Split Intervals)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/pages-3-markdown/index.html | file  |
      | files     | testdata/pages-3-markdown/page_1.md  | file  |
      | files     | testdata/pages-3-markdown/page_2.md  | file  |
      | files     | testdata/pages-3-markdown/page_3.md  | file  |
      | splitMode | intervals                            | field |
      | splitSpan | 2                                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | *_0.pdf |
      | *_1.pdf |
    Then the "*_0.pdf" PDF should have 2 page(s)
    Then the "*_1.pdf" PDF should have 1 page(s)
    Then the "*_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "*_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "*_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  Scenario: POST /forms/chromium/convert/markdown (Split Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/pages-3-markdown/index.html | file  |
      | files     | testdata/pages-3-markdown/page_1.md  | file  |
      | files     | testdata/pages-3-markdown/page_2.md  | file  |
      | files     | testdata/pages-3-markdown/page_3.md  | file  |
      | splitMode | pages                                | field |
      | splitSpan | 2-                                   | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | *_0.pdf |
      | *_1.pdf |
    Then the "*_0.pdf" PDF should have 1 page(s)
    Then the "*_1.pdf" PDF should have 1 page(s)
    Then the "*_0.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "*_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  Scenario: POST /forms/chromium/convert/markdown (Split Pages & Unify)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-markdown/index.html | file   |
      | files                     | testdata/pages-3-markdown/page_1.md  | file   |
      | files                     | testdata/pages-3-markdown/page_2.md  | file   |
      | files                     | testdata/pages-3-markdown/page_3.md  | file   |
      | splitMode                 | pages                                | field  |
      | splitSpan                 | 2-                                   | field  |
      | splitUnify                | true                                 | field  |
      | Gotenberg-Output-Filename | foo                                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/chromium/convert/markdown (Split Many PDFs - Lot of Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/pages-12-markdown/index.html | file  |
      | files     | testdata/pages-12-markdown/page_1.md  | file  |
      | files     | testdata/pages-12-markdown/page_2.md  | file  |
      | files     | testdata/pages-12-markdown/page_3.md  | file  |
      | files     | testdata/pages-12-markdown/page_4.md  | file  |
      | files     | testdata/pages-12-markdown/page_5.md  | file  |
      | files     | testdata/pages-12-markdown/page_6.md  | file  |
      | files     | testdata/pages-12-markdown/page_7.md  | file  |
      | files     | testdata/pages-12-markdown/page_8.md  | file  |
      | files     | testdata/pages-12-markdown/page_9.md  | file  |
      | files     | testdata/pages-12-markdown/page_10.md | file  |
      | files     | testdata/pages-12-markdown/page_11.md | file  |
      | files     | testdata/pages-12-markdown/page_12.md | file  |
      | splitMode | intervals                             | field |
      | splitSpan | 1                                     | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 12 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | *_0.pdf  |
      | *_1.pdf  |
      | *_2.pdf  |
      | *_3.pdf  |
      | *_4.pdf  |
      | *_5.pdf  |
      | *_6.pdf  |
      | *_7.pdf  |
      | *_8.pdf  |
      | *_9.pdf  |
      | *_10.pdf |
      | *_11.pdf |
    Then the "*_0.pdf" PDF should have 1 page(s)
    Then the "*_11.pdf" PDF should have 1 page(s)
    Then the "*_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "*_11.pdf" PDF should have the following content at page 1:
      """
      Page 12
      """

  Scenario: POST /forms/chromium/convert/markdown (PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file  |
      | files | testdata/page-1-markdown/page_1.md  | file  |
      | pdfa  | PDF/A-1b                            | field |
      | pdfua | true                                | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/chromium/convert/markdown (Split & PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files     | testdata/pages-3-markdown/index.html | file  |
      | files     | testdata/pages-3-markdown/page_1.md  | file  |
      | files     | testdata/pages-3-markdown/page_2.md  | file  |
      | files     | testdata/pages-3-markdown/page_3.md  | file  |
      | splitMode | intervals                            | field |
      | splitSpan | 2                                    | field |
      | pdfa      | PDF/A-1b                             | field |
      | pdfua     | true                                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | *_0.pdf |
      | *_1.pdf |
    Then the "*_0.pdf" PDF should have 2 page(s)
    Then the "*_1.pdf" PDF should have 1 page(s)
    Then the "*_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "*_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "*_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/chromium/convert/markdown (Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html                                                                                                                                                                                                                                                                       | file   |
      | files                     | testdata/page-1-markdown/page_1.md                                                                                                                                                                                                                                                                        | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "Author": "Julien Neuhart",
          "Copyright": "Julien Neuhart",
          "CreateDate": "2006:09:18 16:27:50-04:00",
          "Creator": "Gotenberg",
          "Keywords": ["first", "second"],
          "Marked": true,
          "ModDate": "2006:09:18 16:27:50-04:00",
          "PDFVersion": 1.7,
          "Producer": "Gotenberg",
          "Subject": "Sample",
          "Title": "Sample",
          "Trapped": "Unknown"
        }
      }
      """

  Scenario: POST /forms/chromium/convert/markdown (Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files   | testdata/page-1-markdown/index.html | file  |
      | files   | testdata/page-1-markdown/page_1.md  | file  |
      | flatten | true                                | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be flatten

  Scenario: POST /forms/chromium/convert/markdown (PDF/A-1b & PDF/UA-1 & Metadata & Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html                                                                                                                                                                                                                                                                       | file   |
      | files                     | testdata/page-1-markdown/page_1.md                                                                                                                                                                                                                                                                        | file   |
      | pdfa                      | PDF/A-1b                                                                                                                                                                                                                                                                                                  | field  |
      | pdfua                     | true                                                                                                                                                                                                                                                                                                      | field  |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | flatten                   | true                                                                                                                                                                                                                                                                                                      | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 7 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)
    Then the response PDF(s) should be flatten
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "Author": "Julien Neuhart",
          "Copyright": "Julien Neuhart",
          "CreateDate": "2006:09:18 16:27:50-04:00",
          "Creator": "Gotenberg",
          "Keywords": ["first", "second"],
          "Marked": true,
          "ModDate": "2006:09:18 16:27:50-04:00",
          "PDFVersion": 1.7,
          "Producer": "Gotenberg",
          "Subject": "Sample",
          "Title": "Sample",
          "Trapped": "Unknown"
        }
      }
      """

  Scenario: POST /forms/chromium/convert/markdown (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file |
      | files | testdata/page-1-markdown/page_1.md  | file |
    Then the response status code should be 404

  Scenario: POST /forms/chromium/convert/markdown (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files           | testdata/page-1-markdown/index.html | file   |
      | files           | testdata/page-1-markdown/page_1.md  | file   |
      | Gotenberg-Trace | forms_chromium_convert_html         | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_chromium_convert_html"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_chromium_convert_html" |

  Scenario: POST /forms/chromium/convert/markdown (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page-1-markdown/index.html","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | files        | testdata/page-1-markdown/page_1.md                                                                                       | file  |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/chromium/convert/markdown (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                       | testdata/page-1-markdown/index.html          | file   |
      | files                       | testdata/page-1-markdown/page_1.md           | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  Scenario: POST /forms/chromium/convert/markdown (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file |
      | files | testdata/page-1-markdown/page_1.md  | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/chromium/convert/markdown (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file |
      | files | testdata/page-1-markdown/page_1.md  | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
