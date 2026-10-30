# TODO:
# 1. JavaScript disabled on some feature.

@chromium
@chromium-convert-markdown
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
      | 'file:///etc/passwd' does not match any expression from the allowed list |

  Scenario: POST /forms/chromium/convert/markdown (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/pages-3-markdown/index.html | file |
      | files | testdata/pages-3-markdown/page_1.md  | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Markdown file(s) not found: 'page_2.md'
      'page_3.md'
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
      | nativeBookmarks               | foo | field |
      | nativeTaggedPdf               | foo | field |
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
      | failOnConsoleMessageTypes     | foo | field |
      | skipNetworkIdleEvent          | foo | field |
      | skipNetworkAlmostIdleEvent    | foo | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain the following line(s):
      """
      Invalid form data:
      form field 'skipNetworkIdleEvent' is invalid (got 'foo'
      form field 'skipNetworkAlmostIdleEvent' is invalid (got 'foo'
      form field 'failOnHttpStatusCodes' is invalid (got 'foo'
      form field 'failOnResourceHttpStatusCodes' is invalid (got 'foo'
      form field 'failOnResourceLoadingFailed' is invalid (got 'foo'
      form field 'failOnConsoleExceptions' is invalid (got 'foo'
      form field 'failOnConsoleMessageTypes' is invalid (got 'foo'
      form field 'waitDelay' is invalid (got 'foo'
      form field 'emulatedMediaType' is invalid (got 'foo', resulting to wrong value, expected either 'screen', 'print' or empty)
      form field 'omitBackground' is invalid (got 'foo'
      form field 'landscape' is invalid (got 'foo'
      form field 'printBackground' is invalid (got 'foo'
      form field 'scale' is invalid (got 'foo'
      form field 'singlePage' is invalid (got 'foo'
      form field 'paperWidth' is invalid (got 'foo'
      form field 'paperHeight' is invalid (got 'foo'
      form field 'marginTop' is invalid (got 'foo'
      form field 'marginBottom' is invalid (got 'foo'
      form field 'marginLeft' is invalid (got 'foo'
      form field 'marginRight' is invalid (got 'foo'
      form field 'preferCssPageSize' is invalid (got 'foo'
      form field 'nativeBookmarks' is invalid (got 'foo'
      form field 'nativeTaggedPdf' is invalid (got 'foo'
      form file 'index.html' is required
      no form file found for extensions: [.md]
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
    # Does not seems to happen on amd architectures anymore since Chromium 137.
    # See: https://github.com/gotenberg/gotenberg/actions/runs/15384321883/job/43280184372.
    #    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
    #      | files        | testdata/page-1-markdown/index.html | file  |
    #      | files        | testdata/page-1-markdown/page_1.md  | file  |
    #      | paperWidth   | 0                                   | field |
    #      | paperHeight  | 0                                   | field |
    #      | marginTop    | 1000000                             | field |
    #      | marginBottom | 1000000                             | field |
    #      | marginLeft   | 1000000                             | field |
    #      | marginRight  | 1000000                             | field |
    #    Then the response status code should be 400
    #    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    #    Then the response body should match string:
    #      """
    #      Chromium does not handle the provided settings; please check for aberrant form values
    #      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files            | testdata/page-1-markdown/index.html | file  |
      | files            | testdata/page-1-markdown/page_1.md  | file  |
      | nativePageRanges | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium does not handle the page ranges 'foo' (nativePageRanges) syntax
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files            | testdata/page-1-markdown/index.html | file  |
      | files            | testdata/page-1-markdown/page_1.md  | file  |
      | nativePageRanges | 2-3                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The page ranges '2-3' (nativePageRanges) exceeds the page count
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
      | files  | testdata/page-1-markdown/index.html | file  |
      | files  | testdata/page-1-markdown/page_1.md  | file  |
      | stages | foo                                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stages must be a JSON array of objects: invalid character 'o' in literal false (expecting 'a')
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files  | testdata/page-1-markdown/index.html | file  |
      | files  | testdata/page-1-markdown/page_1.md  | file  |
      | stages | [{"op":"shred"}]                    | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: unknown operation "shred"
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files  | testdata/page-1-markdown/index.html | file  |
      | files  | testdata/page-1-markdown/page_1.md  | file  |
      | stages | [{"op":"split"}]                    | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: split: set "every" or "ranges"
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files  | testdata/page-1-markdown/index.html  | file  |
      | files  | testdata/page-1-markdown/page_1.md   | file  |
      | stages | [{"op":"remediate","profile":"foo"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: remediate: "profile" must be one of
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files  | testdata/page-1-markdown/index.html                                  | file  |
      | files  | testdata/page-1-markdown/page_1.md                                   | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "rotate" cannot follow "remediate": only encrypt, optimize, sign and timestamp may
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html      | file   |
      | files                     | testdata/page-1-markdown/page_1.md       | file   |
      | stages                    | [{"op":"rotate","angle":90,"pages":"5"}] | field  |
      | Gotenberg-Output-Filename | foo                                      | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Stage 'rotate' failed on 'foo.pdf':
      """

  @pdf-attach
  Scenario: POST /forms/chromium/convert/markdown (Attach)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html                                         | file   |
      | files                     | testdata/page-1-markdown/page_1.md                                          | file   |
      | files                     | testdata/embed_1.xml                                                        | file   |
      | files                     | testdata/embed_2.xml                                                        | file   |
      | stages                    | [{"op":"attach","file":"embed_1.xml"},{"op":"attach","file":"embed_2.xml"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                         | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should have the "embed_2.xml" file embedded

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
      | "correlation_id":"forms_chromium_convert_html" |

  @download-from
  Scenario: POST /forms/chromium/convert/markdown (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"{helper}/static/testdata/page-1-markdown/index.html","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | files        | testdata/page-1-markdown/page_1.md                                                                 | file  |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "application/pdf"

  @webhook
  Scenario: POST /forms/chromium/convert/markdown (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                        | testdata/page-1-markdown/index.html | file   |
      | files                        | testdata/page-1-markdown/page_1.md  | file   |
      | Gotenberg-Output-Filename    | foo                                 | header |
      | Gotenberg-Webhook-Url        | {helper}/webhook                    | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events             | header |
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

  Scenario: POST /foo/forms/chromium/convert/markdown (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files | testdata/page-1-markdown/index.html | file |
      | files | testdata/page-1-markdown/page_1.md  | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/chromium/convert/markdown (Long Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-markdown/index.html | file   |
      | files                     | testdata/page-1-markdown/page_1.md  | file   |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
