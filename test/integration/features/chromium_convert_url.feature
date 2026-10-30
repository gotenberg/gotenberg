# TODO:
# 1. JavaScript disabled on some feature.

@chromium
@chromium-convert-url
Feature: /forms/chromium/convert/url

  Scenario: POST /forms/chromium/convert/url (Default)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
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

  Scenario: POST /forms/chromium/convert/url (Paper Size)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | paperSize                 | A4                                            | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF first page should be 595.28x841.89 points

  Scenario: POST /forms/chromium/convert/url (Paper Size With Width Override)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | paperSize                 | A4                                            | field  |
      | paperWidth                | 100mm                                         | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF first page should be 283.46x841.89 points

  # Each conversion runs in its own browser context, so localStorage never
  # carries over.
  Scenario: POST /forms/chromium/convert/url (localStorage isolated per conversion)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/local-storage-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "localStorageCount=1" at page 1
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/local-storage-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "localStorageCount=1" at page 1

  # With --chromium-shared-browser-context, localStorage is per-origin and
  # shared by the long-lived browser, so without clearing it accumulates
  # across same-origin conversions.
  # See https://github.com/gotenberg/gotenberg/issues/919.
  Scenario: POST /forms/chromium/convert/url (localStorage leaks in a shared browser context)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_SHARED_BROWSER_CONTEXT | true |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/local-storage-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "localStorageCount=1" at page 1
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/local-storage-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "localStorageCount=2" at page 1

  # With --chromium-clear-storage every conversion in a shared browser context
  # clears the origin's localStorage first, so the counter never carries over.
  # See https://github.com/gotenberg/gotenberg/issues/919.
  Scenario: POST /forms/chromium/convert/url (Clear Storage)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_SHARED_BROWSER_CONTEXT | true |
      | CHROMIUM_CLEAR_STORAGE          | true |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/local-storage-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "localStorageCount=1" at page 1
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/local-storage-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have content matching "localStorageCount=1" at page 1

  Scenario: POST /forms/chromium/convert/url (Single Page)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-12-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                             | header |
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
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-12-html/index.html | field  |
      | singlePage                | true                                            | field  |
      | Gotenberg-Output-Filename | foo                                             | header |
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

  Scenario: POST /forms/chromium/convert/url (Landscape)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should NOT be set to landscape orientation
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | landscape                 | true                                          | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should be set to landscape orientation

  Scenario: POST /forms/chromium/convert/url (Native Page Ranges)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-12-html/index.html | field  |
      | nativePageRanges          | 2-3                                             | field  |
      | Gotenberg-Output-Filename | foo                                             | header |
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

  Scenario: POST /forms/chromium/convert/url (Header & Footer)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-12-html/index.html | field  |
      | files                     | testdata/header-footer-html/header.html         | file   |
      | files                     | testdata/header-footer-html/footer.html         | file   |
      | Gotenberg-Output-Filename | foo                                             | header |
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

  Scenario: POST /forms/chromium/convert/url (Custom HTTP Headers)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html                                                                                                                                                                | field  |
      | userAgent                 | Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko)                                                                                                                                       | field  |
      | extraHttpHeaders          | {"X-Header":"foo","X-Scoped-Header-1":"bar;scope=https?:\\/\\/([a-zA-Z0-9-]+\\\\.)*domain\\\\.com\\/.*","X-Scoped-Header-2":"baz;scope=https?:\\/\\/([a-zA-Z0-9-]+\\\\.)*docker\\\\.internal:(\\\\d+)\\/.*"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                          | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the server request header "User-Agent" should be "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko)"
    Then the server request header "X-Header" should be "foo"
    Then the server request header "X-Scoped-Header-1" should be ""
    Then the server request header "X-Scoped-Header-2" should be "baz"

  Scenario: POST /forms/chromium/convert/url (Cookies)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html                                                                        | field  |
      | cookies                   | [{"name":"cookie_1","value":"foo","domain":"{helper-host}"},{"name":"cookie_2","value":"bar","domain":"domain.com"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the server request cookie "cookie_1" should be "foo"
    Then the server request cookie "cookie_2" should be ""

  # A cookie set by one conversion must not reach the next one.
  Scenario: POST /forms/chromium/convert/url (Cookies isolated per conversion)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url     | {helper}/html/testdata/page-1-html/index.html                | field |
      | cookies | [{"name":"cookie_1","value":"foo","domain":"{helper-host}"}] | field |
    Then the response status code should be 200
    Then the server request cookie "cookie_1" should be "foo"
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 200
    Then the server request cookie "cookie_1" should be ""

  # With --chromium-shared-browser-context, cookies carry over between
  # conversions.
  Scenario: POST /forms/chromium/convert/url (Cookies in a shared browser context)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_SHARED_BROWSER_CONTEXT | true |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url     | {helper}/html/testdata/page-1-html/index.html                | field |
      | cookies | [{"name":"cookie_1","value":"foo","domain":"{helper-host}"}] | field |
    Then the response status code should be 200
    Then the server request cookie "cookie_1" should be "foo"
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 200
    Then the server request cookie "cookie_1" should be "foo"

  # See https://github.com/gotenberg/gotenberg/issues/1130.
  Scenario: POST /forms/chromium/convert/url (case-insensitive sameSite)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html                                                                                         | field  |
      | cookies                   | [{"name":"cookie_1","value":"foo","domain":"{helper-host}"},{"name":"cookie_2","value":"bar","domain":"domain.com","sameSite":"lax"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the server request cookie "cookie_1" should be "foo"
    Then the server request cookie "cookie_2" should be ""

  Scenario: POST /forms/chromium/convert/url (Wait Delay)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | waitDelay                 | 2.5s                                                       | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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

  Scenario: POST /forms/chromium/convert/url (Wait For Expression)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | waitForExpression         | window.globalVar === 'ready'                               | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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

  Scenario: POST /forms/chromium/convert/url (Wait For Selector)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Wait on selector returns true.
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | waitForSelector           | #wait-selector                                             | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Wait on selector returns true.
      """

  Scenario: POST /forms/chromium/convert/url (Emulated Media Type)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | emulatedMediaType         | print                                                      | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | emulatedMediaType         | screen                                                     | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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

  Scenario: POST /forms/chromium/convert/url (Emulated Media Features)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Prefers reduced motion.
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | emulatedMediaFeatures     | {"prefers-reduced-motion":"reduce"}                        | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Prefers reduced motion.
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | emulatedMediaType         | screen                                                     | field  |
      | emulatedMediaFeatures     | {"prefers-reduced-motion":"reduce"}                        | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Prefers reduced motion.
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Emulated media type is 'print'.
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | emulatedMediaType         | print                                                      | field  |
      | emulatedMediaFeatures     | {"prefers-reduced-motion":"reduce"}                        | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Prefers reduced motion.
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Emulated media type is 'screen'.
      """

  Scenario: POST /forms/chromium/convert/url (Default Allow / Deny Lists)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the Gotenberg container should NOT log the following entries:
      # Modern browsers block file URIs from being loaded into iframes when the parent page is served over HTTP/HTTPS.
      | 'file:///etc/passwd' matches the expression from the denied list |

  Scenario: POST /forms/chromium/convert/url (Main URL does NOT match allowed list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^file:(?!//\\/tmp/).* |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: POST /forms/chromium/convert/url (Main URL does match denied list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |         |
      | CHROMIUM_DENY_LIST  | ^http.* |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: POST /forms/chromium/convert/url (Request does not match the allowed list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^.* |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the Gotenberg container should NOT log the following entries:
      # Modern browsers block file URIs from being loaded into iframes when the parent page is served over HTTP/HTTPS.
      | 'file:///etc/passwd' does not match any expression from the allowed list |

  Scenario: POST /forms/chromium/convert/url (file:// scheme rejected at route layer)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | file:///tmp/foo/index.html | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      file:// URLs are not accepted on this route. Use the /convert/html or /convert/markdown routes to render local HTML
      """

  # devtools:// pages expose privileged bindings such as InspectorFrontendHost,
  # so evaluating waitForExpression there could read local files. chrome://
  # pages can crash or restart the browser. Only http(s) URLs are accepted.
  Scenario Outline: POST /forms/chromium/convert/url (non-http(s) scheme rejected at route layer)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url               | <url>                    | field |
      | waitForExpression | window.foo === undefined | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'url' must be an http:// or https:// URL
      """

    Examples:
      | url                                           |
      | devtools://devtools/bundled/devtools_app.html |
      | DEVTOOLS://devtools/bundled/devtools_app.html |
      | chrome://version                              |
      | chrome-untrusted://terminal/                  |
      | view-source:https://example.com               |
      | javascript:alert(1)                           |
      | data:text/html,foo                            |

  Scenario: POST /forms/chromium/convert/url (Main URL resolves to a non-public IP, permissive default)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |  |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/url (Main URL is a non-public IP literal, deny-private-ips on)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | http://127.0.0.1/ | field |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  # IPv6 loopback literal is parsed as an IP and rejected by the IP-class check,
  # like the IPv4 loopback literal above.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/url (Main URL is an IPv6 loopback literal, deny-private-ips on)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | http://[::1]/ | field |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  # An alternate IP encoding (decimal for 127.0.0.1) that Chromium would read
  # as loopback but the resolver rejects as a hostname. It must fail closed as
  # filtered (a generic 403), not surface as a 500.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/url (Main URL is a decimal-encoded loopback IP, deny-private-ips on)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | http://2130706433/ | field |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  # A classic SSRF vector: an allow-listed URL that redirects to an internal
  # address. The redirected request must not inherit the initial URL's
  # allow-list pass. listenForEventRequestPaused re-validates it; it does not
  # match the allow-list, so it is blocked (Chromium reports ERR_ACCESS_DENIED
  # and the conversion renders the resulting error page).
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/url (Redirect to a non-allow-listed address is re-filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^https?://host\\.docker\\.internal(:[0-9]+)?/ |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/redirect-to-private | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'http://127.0.0.1:9999/redirected' does not match any expression from the allowed list |

  Scenario: POST /forms/chromium/convert/url (Main URL resolves to a non-public IP, deny-private-ips on with allow-list bypass)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       | .+   |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/chromium/convert/url (JavaScript Enabled)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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

  Scenario: POST /forms/chromium/convert/url (JavaScript Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DISABLE_JAVASCRIPT | true |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field  |
      | Gotenberg-Output-Filename | foo                                                        | header |
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

  Scenario: POST /forms/chromium/convert/url (Fail On Resource HTTP Status Codes)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                           | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
      | failOnResourceHttpStatusCodes | [499,599]                                                  | field |
      | skipNetworkIdleEvent          | false                                                      | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Invalid HTTP status code from resources:
      """
    Then the response body should contain string:
      """
      /favicon.ico - 404: Not Found
      """

  Scenario: POST /forms/chromium/convert/url (Fail On Resource Loading Failed)
    # A configured allow-list routes loopback through the pinning proxy, which
    # reports a refused connection as a reset. See the convert/html
    # scenario.
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |  |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                         | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
      | failOnResourceLoadingFailed | true                                                       | field |
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

  Scenario: POST /forms/chromium/convert/url (Fail On Console Exceptions)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                     | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
      | failOnConsoleExceptions | true                                                       | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Chromium console exceptions
      """
    Then the response body should contain string:
      """
      Error: Exception 1
      """
    Then the response body should contain string:
      """
      Error: Exception 2
      """

  Scenario: POST /forms/chromium/convert/url (Fail On Console Message Types)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
      | failOnConsoleMessageTypes | ["error","warning"]                                        | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium console messages:
      [warning] a warning message
      [error] an error message
      """

  Scenario: POST /forms/chromium/convert/url (Fail On Console Message Types - Unknown Type)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/feature-rich-html-remote/index.html | field |
      | failOnConsoleMessageTypes | ["warn"]                                                   | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'failOnConsoleMessageTypes' is invalid (got '["warn"]', resulting to unknown console message type 'warn', expected one or more of: log, debug, info, warning, error, assert)
      """

  Scenario: POST /forms/chromium/convert/url (Bad Request)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
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
    Then the response body should match string:
      """
      Invalid form data: form field 'skipNetworkIdleEvent' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'skipNetworkAlmostIdleEvent' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'failOnHttpStatusCodes' is invalid (got 'foo', resulting to unmarshal failOnHttpStatusCodes: invalid character 'o' in literal false (expecting 'a'))
      form field 'failOnResourceHttpStatusCodes' is invalid (got 'foo', resulting to unmarshal failOnResourceHttpStatusCodes: invalid character 'o' in literal false (expecting 'a'))
      form field 'failOnResourceLoadingFailed' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'failOnConsoleExceptions' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'failOnConsoleMessageTypes' is invalid (got 'foo', resulting to unmarshal failOnConsoleMessageTypes: invalid character 'o' in literal false (expecting 'a'))
      form field 'waitDelay' is invalid (got 'foo', resulting to time: invalid duration "foo")
      form field 'emulatedMediaType' is invalid (got 'foo', resulting to wrong value, expected either 'screen', 'print' or empty)
      form field 'omitBackground' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'landscape' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'printBackground' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'scale' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'singlePage' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'paperWidth' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'paperHeight' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'marginTop' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'marginBottom' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'marginLeft' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'marginRight' is invalid (got 'foo', resulting to strconv.ParseFloat: parsing "foo": invalid syntax)
      form field 'preferCssPageSize' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'nativeBookmarks' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'nativeTaggedPdf' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'url' is required
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url            | {helper}/html/testdata/page-1-html/index.html | field |
      | omitBackground | true                                          | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      omitBackground requires printBackground set to true
      """
    # Does not seems to happen on amd architectures anymore since Chromium 137.
    # See: https://github.com/gotenberg/gotenberg/actions/runs/15384321883/job/43280184372.
    #    Given I have a static server
    #    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
    #      | url          | {helper}/html/testdata/page-1-html/index.html | field |
    #      | paperWidth   | 0                                                                   | field |
    #      | paperHeight  | 0                                                                   | field |
    #      | marginTop    | 1000000                                                             | field |
    #      | marginBottom | 1000000                                                             | field |
    #      | marginLeft   | 1000000                                                             | field |
    #      | marginRight  | 1000000                                                             | field |
    #    Then the response status code should be 400
    #    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    #    Then the response body should match string:
    #      """
    #      Chromium does not handle the provided settings; please check for aberrant form values
    #      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url              | {helper}/html/testdata/page-1-html/index.html | field |
      | nativePageRanges | foo                                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium does not handle the page ranges 'foo' (nativePageRanges) syntax
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url              | {helper}/html/testdata/page-1-html/index.html | field |
      | nativePageRanges | 2-3                                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The page ranges '2-3' (nativePageRanges) exceeds the page count
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url               | {helper}/html/testdata/page-1-html/index.html | field |
      | waitForExpression | undefined                                     | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The expression 'undefined' (waitForExpression) returned an exception or undefined
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url     | {helper}/html/testdata/page-1-html/index.html | field |
      | cookies | foo                                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'cookies' is invalid (got 'foo', resulting to unmarshal cookies: invalid character 'o' in literal false (expecting 'a'))
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url     | {helper}/html/testdata/page-1-html/index.html | field |
      | cookies | [{"name":"yummy_cookie","value":"choco"}]     | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'cookies' is invalid (got '[{"name":"yummy_cookie","value":"choco"}]', resulting to cookie 0 must have its name, value and domain set)
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url              | {helper}/html/testdata/page-1-html/index.html | field |
      | extraHttpHeaders | foo                                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got 'foo', resulting to unmarshal extraHttpHeaders: invalid character 'o' in literal false (expecting 'a'))
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url              | {helper}/html/testdata/page-1-html/index.html | field |
      | extraHttpHeaders | {"foo":"bar;scope;;"}                         | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got '{"foo":"bar;scope;;"}', resulting to invalid scope '' for header 'foo')
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url              | {helper}/html/testdata/page-1-html/index.html | field |
      | extraHttpHeaders | {"foo":"bar;scope=*."}                        | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got '{"foo":"bar;scope=*."}', resulting to invalid scope regex pattern for header 'foo': error parsing regexp: missing argument to repetition operator in `*.`)
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                   | {helper}/html/testdata/page-1-html/index.html | field |
      | emulatedMediaFeatures | foo                                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'emulatedMediaFeatures' is invalid (got 'foo', resulting to unmarshal emulatedMediaFeatures: invalid character 'o' in literal false (expecting 'a'))
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | foo                                           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stages must be a JSON array of objects: invalid character 'o' in literal false (expecting 'a')
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"shred"}]                              | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: unknown operation "shred"
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"split"}]                              | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: split: set "every" or "ranges"
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"remediate","profile":"foo"}]          | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: remediate: "profile" must be one of
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html                        | field |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "rotate" cannot follow "remediate": only encrypt, optimize, sign and timestamp may
      """
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | stages                    | [{"op":"rotate","angle":90,"pages":"5"}]      | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Stage 'rotate' failed on 'foo.pdf':
      """

  @pdf-split
  Scenario: POST /forms/chromium/convert/url (Split Intervals)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/pages-3-html/index.html | field |
      | stages | [{"op":"split","every":2}]                     | field |
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

  # See https://github.com/gotenberg/gotenberg/issues/1130.
  @pdf-split
  @output-filename
  Scenario: POST /forms/chromium/convert/url (Split Output Filename)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-3-html/index.html | field  |
      | stages                    | [{"op":"split","every":2}]                     | field  |
      | Gotenberg-Output-Filename | foo                                            | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.zip   |
      | foo_0.pdf |
      | foo_1.pdf |
    Then the "foo_0.pdf" PDF should have 2 page(s)
    Then the "foo_1.pdf" PDF should have 1 page(s)
    Then the "foo_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "foo_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  @pdf-split
  Scenario: POST /forms/chromium/convert/url (Split Pages)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/pages-3-html/index.html | field |
      | stages | [{"op":"split","ranges":"2,3"}]                | field |
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

  @pdf-split
  Scenario: POST /forms/chromium/convert/url (Split Pages & Unify)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-3-html/index.html | field  |
      | stages                    | [{"op":"select","pages":"2-3"}]                | field  |
      | Gotenberg-Output-Filename | foo                                            | header |
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

  @pdf-split
  Scenario: POST /forms/chromium/convert/url (Split Many PDFs - Lot of Pages)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/pages-12-html/index.html | field |
      | stages | [{"op":"split","every":1}]                      | field |
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

  @pdf-remediate
  Scenario: POST /forms/chromium/convert/url (PDF/A-1b)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"remediate","profile":"PDF/A-1b"}]     | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  @pdf-remediate
  Scenario: POST /forms/chromium/convert/url (PDF/UA-1)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url             | {helper}/html/testdata/page-1-html/index.html | field |
      | nativeTaggedPdf | true                                          | field |
      | stages          | [{"op":"remediate","profile":"PDF/UA-1"}]     | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/chromium/convert/url (Split & PDF/A-1b)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/pages-3-html/index.html                     | field |
      | stages | [{"op":"split","every":2},{"op":"remediate","profile":"PDF/A-1b"}] | field |
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
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  # See https://github.com/gotenberg/gotenberg/issues/1130.
  @pdf-remediate
  @pdf-split
  @output-filename
  Scenario: POST /forms/chromium/convert/url (Split & PDF/A-1b & Output Filename)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/pages-3-html/index.html                     | field  |
      | stages                    | [{"op":"split","every":2},{"op":"remediate","profile":"PDF/A-1b"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.zip   |
      | foo_0.pdf |
      | foo_1.pdf |
    Then the "foo_0.pdf" PDF should have 2 page(s)
    Then the "foo_1.pdf" PDF should have 1 page(s)
    Then the "foo_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "foo_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  @pdf-metadata
  Scenario: POST /forms/chromium/convert/url (Metadata)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html                                                                                                                 | field  |
      | stages                    | [{"op":"set-metadata","title":"Sample","author":"Julien Neuhart","subject":"Sample","creator":"Gotenberg","producer":"Gotenberg","keywords":"first, second"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Julien Neuhart"
    Then the "foo.pdf" PDF should have the "Subject" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Creator" metadata set to "Gotenberg"
    Then the "foo.pdf" PDF should have the "Producer" metadata set to "Gotenberg"

  @pdf-form
  Scenario: POST /forms/chromium/convert/url (Flatten)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"flatten-form"}]                       | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be flatten

  @pdf-encrypt
  Scenario: POST /forms/chromium/convert/url (Encrypt - user password only)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"encrypt","userPassword":"foo"}]       | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @pdf-encrypt
  Scenario: POST /forms/chromium/convert/url (Encrypt - both user and owner passwords)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html                 | field |
      | stages | [{"op":"encrypt","userPassword":"foo","ownerPassword":"bar"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @pdf-attach
  Scenario: POST /foo/forms/chromium/convert/url (Attach)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html                               | field  |
      | files                     | testdata/embed_1.xml                                                        | file   |
      | files                     | testdata/embed_2.xml                                                        | file   |
      | stages                    | [{"op":"attach","file":"embed_1.xml"},{"op":"attach","file":"embed_2.xml"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                         | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should have the "embed_2.xml" file embedded

  # Attaching files to a PDF/A-3 document needs a relationship, here Data.
  @pdf-remediate
  @pdf-metadata
  @pdf-form
  @pdf-attach
  Scenario: POST /forms/chromium/convert/url (PDF/A-3b & Metadata & Flatten & Attach)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html                                                                                                                                                                                                                                                                                                     | field  |
      | files                     | testdata/embed_1.xml                                                                                                                                                                                                                                                                                                                              | file   |
      | files                     | testdata/embed_2.xml                                                                                                                                                                                                                                                                                                                              | file   |
      | stages                    | [{"op":"set-metadata","title":"Sample","author":"Julien Neuhart","subject":"Sample","creator":"Gotenberg","producer":"Gotenberg","keywords":"first, second"},{"op":"flatten-form"},{"op":"attach","file":"embed_1.xml","relationship":"Data"},{"op":"attach","file":"embed_2.xml","relationship":"Data"},{"op":"remediate","profile":"PDF/A-3b"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                                                               | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should be flatten
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should have the "embed_2.xml" file embedded
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Julien Neuhart"
    Then the "foo.pdf" PDF should have the "Subject" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Creator" metadata set to "Gotenberg"
    Then the "foo.pdf" PDF should have the "Producer" metadata set to "Gotenberg"

  Scenario: POST /forms/chromium/convert/url (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DISABLE_ROUTES | true |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 404

  Scenario: POST /forms/chromium/convert/url (Gotenberg Trace)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url             | {helper}/html/testdata/page-1-html/index.html | field  |
      | Gotenberg-Trace | forms_chromium_convert_url                    | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_chromium_convert_url"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"forms_chromium_convert_url" |

  @webhook
  Scenario: POST /forms/chromium/convert/url (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                          | {helper}/html/testdata/page-1-html/index.html | field  |
      | Gotenberg-Output-Filename    | foo                                           | header |
      | Gotenberg-Webhook-Url        | {helper}/webhook                              | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events                       | header |
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

  Scenario: POST /forms/chromium/convert/url (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/chromium/convert/url (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/foo/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url | {helper}/html/testdata/page-1-html/index.html | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/chromium/convert/url (Stage file not uploaded => 400)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url    | {helper}/html/testdata/page-1-html/index.html | field |
      | stages | [{"op":"stamp-pdf","pdf":"/etc/hostname"}]    | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form file '/etc/hostname' is required
      """

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/chromium/convert/url (Long Filename)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/url" endpoint with the following form data and header(s):
      | url                       | {helper}/html/testdata/page-1-html/index.html | field  |
      | Gotenberg-Output-Filename | foo                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
