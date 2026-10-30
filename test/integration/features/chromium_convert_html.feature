# TODO:
# 1. JavaScript disabled on some feature.

@chromium
@chromium-convert-html
Feature: /forms/chromium/convert/html

  Scenario: POST /forms/chromium/convert/html (Default)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
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

  Scenario: POST /forms/chromium/convert/html (Paper Size)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | paperSize                 | A4                              | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF first page should be 595.28x841.89 points

  Scenario: POST /forms/chromium/convert/html (Paper Size With Width Override)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | paperSize                 | A4                              | field  |
      | paperWidth                | 100mm                           | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF first page should be 283.46x841.89 points

  Scenario: POST /forms/chromium/convert/html (Single Page)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                               | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-html/index.html | file   |
      | singlePage                | true                              | field  |
      | Gotenberg-Output-Filename | foo                               | header |
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

  # A wide table on one landscape page: the page must expand its width to the
  # content, so the rightmost column is not truncated and the page comes out
  # landscape instead of the narrow-tall strip the height-only expansion used
  # to produce. See https://github.com/gotenberg/gotenberg/issues/1390.
  Scenario: POST /forms/chromium/convert/html (Single Page Landscape)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/wide-table-html/index.html | file   |
      | singlePage                | true                                | field  |
      | landscape                 | true                                | field  |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should be set to landscape orientation
    Then the "foo.pdf" PDF should have content matching "Column 12" at page 1

  Scenario: POST /forms/chromium/convert/html (Landscape)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should NOT be set to landscape orientation
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | landscape                 | true                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should be set to landscape orientation

  Scenario: POST /forms/chromium/convert/html (Native Page Ranges)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-html/index.html | file   |
      | nativePageRanges          | 2-3                               | field  |
      | Gotenberg-Output-Filename | foo                               | header |
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

  Scenario: POST /forms/chromium/convert/html (Header & Footer)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-12-html/index.html       | file   |
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

  Scenario: POST /forms/chromium/convert/html (Wait Delay)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | waitDelay                 | 2.5s                                  | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (Wait For Expression)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | waitForExpression         | window.globalVar === 'ready'          | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (Wait Window Status)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | waitWindowStatus          | ready                                 | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Wait delay > 2 seconds or expression window globalVar === 'ready' returns true.
      """

  # A thenable (async) expression is awaited and its resolved value gates the
  # print. Without awaiting, the Promise object never evaluates to true and the
  # request would time out. See https://github.com/gotenberg/gotenberg/pull/1617.
  Scenario: POST /forms/chromium/convert/html (Wait For Thenable Expression)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html        | file   |
      | waitForExpression         | (async () => window.globalVar === 'ready')() | field  |
      | Gotenberg-Output-Filename | foo                                          | header |
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

  Scenario: POST /forms/chromium/convert/html (rAF / ResizeObserver / IntersectionObserver fire with waitForExpression)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/paint-callbacks-html/index.html       | file   |
      | waitForExpression         | !!document.body.getAttribute('data-pdf-ready') | field  |
      | Gotenberg-Output-Filename | foo                                            | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      raf-fired
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      ro-fired
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      io-fired
      """

  Scenario: POST /forms/chromium/convert/html (rAF / ResizeObserver / IntersectionObserver fire with waitDelay and emulatedMediaType=print)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/paint-callbacks-html/index.html | file   |
      | waitDelay                 | 3s                                       | field  |
      | emulatedMediaType         | print                                    | field  |
      | Gotenberg-Output-Filename | foo                                      | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      raf-fired
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      ro-fired
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      io-fired
      """

  Scenario: POST /forms/chromium/convert/html (Wait For Selector)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | waitForSelector           | #wait-selector                        | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (Emulated Media Type)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | emulatedMediaType         | print                                 | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | emulatedMediaType         | screen                                | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (Emulated Media Features)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | emulatedMediaFeatures     | {"prefers-reduced-motion":"reduce"}   | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | emulatedMediaType         | screen                                | field  |
      | emulatedMediaFeatures     | {"prefers-reduced-motion":"reduce"}   | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | emulatedMediaType         | print                                 | field  |
      | emulatedMediaFeatures     | {"prefers-reduced-motion":"reduce"}   | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (Default Allow / Deny Lists)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-html/index.html | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the Gotenberg container should log the following entries:
      | 'file:///etc/passwd' matches the expression from the denied list |

  # Control for the WebSocket scenario below. An ordinary fetch to a loopback
  # address is surfaced as a Fetch.requestPaused event, so it is blocked by
  # CHROMIUM_DENY_PRIVATE_IPS and the block is logged. The allow-list is
  # cleared because a matching allow-list entry bypasses the IP-based check.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (Fetch to a non-public address is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-fetch-html/index.html | file  |
      | waitDelay | 1s                                  | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'http://127.0.0.1:9999/ssrf-fetch' targets a non-public address |

  # A WebSocket handshake is never surfaced as a Fetch.requestPaused event, so
  # it escapes the filter in listenForEventRequestPaused. The page opens
  # WebSockets to two non-public addresses (loopback and the link-local cloud
  # metadata IP). listenForEventWebSocketCreated logs each disallowed handshake
  # with its full ws:// URL (detection), and the pinning proxy severs the
  # connection now that the implicit loopback bypass is removed (enforcement).
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (WebSocket to a non-public address is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-html/index.html | file  |
      | waitDelay | 1s                                      | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'ws://127.0.0.1:9999/ssrf-websocket' targets a non-public address |
      | CONNECT blocked for '127.0.0.1:9999'                              |

  # A Web Worker is a separate CDP target, so its WebSocket handshake is not
  # observed by listenForEventWebSocketCreated. Enforcement must not depend on
  # that listener: the pinning proxy sees the handshake and severs it whatever
  # the originating context. Only the proxy's block is asserted, since no
  # detection log is produced for the worker target.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (WebSocket from a Web Worker is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-worker-html/index.html | file  |
      | waitDelay | 1s                                             | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | CONNECT blocked for '127.0.0.1:9999' |

  # wss:// (TLS) handshakes tunnel through the proxy via CONNECT, the same path
  # as ws://, and must be filtered identically.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (Secure WebSocket to a non-public address is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-tls-html/index.html | file  |
      | waitDelay | 1s                                          | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'wss://127.0.0.1:9999/wss-test' targets a non-public address |
      | CONNECT blocked for '127.0.0.1:9999'                         |

  # An allow-list is enforced on its own, without the IP-class flags. Chromium
  # skips the proxy for loopback and link-local destinations unless told
  # otherwise, so a WebSocket the allow-list rejects must still reach the
  # pinning proxy to be severed.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (WebSocket not in the allow-list is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^file:///tmp/.* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-html/index.html | file  |
      | waitDelay | 1s                                      | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'ws://127.0.0.1:9999/ssrf-websocket' does not match any expression from the allowed list |
      | CONNECT blocked for '127.0.0.1:9999'                                                     |

  # The pinning proxy sees only the host and port of a CONNECT tunnel, so it
  # matches the deny-list against the root URL of every scheme a tunnel may
  # carry. A rule anchored on the wss:// scheme and terminated on "host/"
  # therefore severs the tunnel.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (WebSocket denied by a scheme-anchored rule is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DENY_LIST | ^wss://127\\.0\\.0\\.1:9999/ |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-tls-html/index.html | file  |
      | waitDelay | 1s                                          | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'wss://127.0.0.1:9999/' matches the expression from the denied list |
      | CONNECT blocked for '127.0.0.1:9999'                                |

  # A rule that names a path cannot match a tunnel, so the proxy would admit
  # it. The handshake cannot be aborted through CDP either, so the conversion
  # fails with a generic 403 instead.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (WebSocket denied by a path rule fails the conversion)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DENY_LIST | ^wss://127\\.0\\.0\\.1:9999/wss-test |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-tls-html/index.html | file  |
      | waitDelay | 1s                                          | field |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """
    Then the Gotenberg container should log the following entries:
      | 'wss://127.0.0.1:9999/wss-test' matches the expression from the denied list |

  # EventSource issues an ordinary HTTP GET, so unlike a WebSocket it IS surfaced
  # as a fetch.EventRequestPaused and blocked by listenForEventRequestPaused.
  @chromium-ssrf
  Scenario: POST /forms/chromium/convert/html (EventSource to a non-public address is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-eventsource-html/index.html | file  |
      | waitDelay | 1s                                        | field |
    Then the response status code should be 200
    Then the Gotenberg container should log the following entries:
      | 'http://127.0.0.1:9999/sse' targets a non-public address |

  Scenario: POST /forms/chromium/convert/html (Main URL does NOT match allowed list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^file:(?!//\\/tmp/).* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-html/index.html | file |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: POST /forms/chromium/convert/html (Main URL does match denied list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |                |
      | CHROMIUM_DENY_LIST  | ^file:///tmp.* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-html/index.html | file |
    Then the response status code should be 403
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: POST /forms/chromium/convert/html (Request does not match the allowed list)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST | ^file:///tmp.* |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/feature-rich-html/index.html | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the Gotenberg container should log the following entries:
      | 'file:///etc/passwd' does not match any expression from the allowed list |

  Scenario: POST /forms/chromium/convert/html (JavaScript Enabled)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (JavaScript Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DISABLE_JAVASCRIPT | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
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

  Scenario: POST /forms/chromium/convert/html (Fail On Resource HTTP Status Codes)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                         | testdata/feature-rich-html/index.html | file  |
      | failOnResourceHttpStatusCodes | [499,599]                             | field |
      | skipNetworkIdleEvent          | false                                 | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid HTTP status code from resources:
      http://localhost:3000/no-such-stylesheet.css - 404: Not Found
      """

  Scenario: POST /forms/chromium/convert/html (Fail On Resource HTTP Status Codes - Ignore Domains)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                           | testdata/feature-rich-html/index.html | file  |
      | failOnResourceHttpStatusCodes   | [499,599]                             | field |
      | ignoreResourceHttpStatusDomains | ["localhost"]                         | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  # Through the pinning proxy, an unreachable resource must still fail the
  # conversion. The proxy resets the connection instead of answering 502, which
  # Chromium would treat as an ordinary HTTP error.
  Scenario: POST /forms/chromium/convert/html (Fail On Resource Loading Failed through the pinning proxy)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                       | testdata/feature-rich-html/index.html | file  |
      | failOnResourceLoadingFailed | true                                  | field |
    Then the response status code should be 409
    Then the response body should contain string:
      """
      resource Stylesheet: net::ERR_CONNECTION_RESET
      """

  Scenario: POST /forms/chromium/convert/html (Fail On Resource Loading Failed)
    # A configured allow-list routes loopback through the pinning proxy, which
    # reports a refused connection as a reset. See the scenario below.
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |  |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                       | testdata/feature-rich-html/index.html | file  |
      | failOnResourceLoadingFailed | true                                  | field |
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

  # The URL filter resolves the host itself, so a failed lookup must still
  # surface as Chromium's own name resolution error.
  Scenario: POST /forms/chromium/convert/html (Fail On Resource Loading Failed with an unresolvable host)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST |  |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                       | testdata/unresolvable-resource-html/index.html | file  |
      | failOnResourceLoadingFailed | true                                           | field |
    Then the response status code should be 409
    Then the response body should contain string:
      """
      resource Stylesheet: net::ERR_NAME_NOT_RESOLVED
      """

  Scenario: POST /forms/chromium/convert/html (Fail On Console Exceptions)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                   | testdata/feature-rich-html/index.html | file  |
      | failOnConsoleExceptions | true                                  | field |
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

  Scenario: POST /forms/chromium/convert/html (Fail On Console Message Types)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file  |
      | failOnConsoleMessageTypes | ["error","warning"]                   | field |
    Then the response status code should be 409
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium console messages:
      [warning] a warning message
      [error] an error message
      """

  Scenario: POST /forms/chromium/convert/html (Fail On Console Message Types - Unknown Type)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/feature-rich-html/index.html | file  |
      | failOnConsoleMessageTypes | ["warn"]                              | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'failOnConsoleMessageTypes' is invalid (got '["warn"]', resulting to unknown console message type 'warn', expected one or more of: log, debug, info, warning, error, assert)
      """

  Scenario: POST /forms/chromium/convert/html (Document Outline)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-outline-html/index.html | file   |
      | nativeBookmarks           | true                                  | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    # nativeTaggedPdf is left unset, yet the outline must still be embedded:
    # Gotenberg enables tagged PDF automatically because Chromium derives the
    # outline from the structure tree. See issue #1579.
    Then the "foo.pdf" PDF should have a document outline
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-outline-html/index.html | file   |
      | Gotenberg-Output-Filename | bar                                   | header |
    Then the response status code should be 200
    Then the "bar.pdf" PDF should NOT have a document outline

  Scenario: POST /forms/chromium/convert/html (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
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
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files          | testdata/page-1-html/index.html | file  |
      | omitBackground | true                            | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      omitBackground requires printBackground set to true
      """
    # Does not seems to happen on amd architectures anymore since Chromium 137.
    # See: https://github.com/gotenberg/gotenberg/actions/runs/15384321883/job/43280184372.
    #    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
    #      | files        | testdata/page-1-html/index.html | file  |
    #      | paperWidth   | 0                               | field |
    #      | paperHeight  | 0                               | field |
    #      | marginTop    | 1000000                         | field |
    #      | marginBottom | 1000000                         | field |
    #      | marginLeft   | 1000000                         | field |
    #      | marginRight  | 1000000                         | field |
    #    Then the response status code should be 400
    #    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    #    Then the response body should match string:
    #      """
    #      Chromium does not handle the provided settings; please check for aberrant form values
    #      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files            | testdata/page-1-html/index.html | file  |
      | nativePageRanges | foo                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Chromium does not handle the page ranges 'foo' (nativePageRanges) syntax
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files            | testdata/page-1-html/index.html | file  |
      | nativePageRanges | 2-3                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The page ranges '2-3' (nativePageRanges) exceeds the page count
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files             | testdata/page-1-html/index.html | file  |
      | waitForExpression | undefined                       | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The expression 'undefined' (waitForExpression) returned an exception or undefined
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files   | testdata/page-1-html/index.html | file  |
      | cookies | foo                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'cookies' is invalid (got 'foo', resulting to unmarshal cookies: invalid character 'o' in literal false (expecting 'a'))
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files   | testdata/page-1-html/index.html           | file  |
      | cookies | [{"name":"yummy_cookie","value":"choco"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'cookies' is invalid (got '[{"name":"yummy_cookie","value":"choco"}]', resulting to cookie 0 must have its name, value and domain set)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files            | testdata/page-1-html/index.html | file  |
      | extraHttpHeaders | foo                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got 'foo', resulting to unmarshal extraHttpHeaders: invalid character 'o' in literal false (expecting 'a'))
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files            | testdata/page-1-html/index.html | file  |
      | extraHttpHeaders | {"foo":"bar;scope;;"}           | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got '{"foo":"bar;scope;;"}', resulting to invalid scope '' for header 'foo')
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files            | testdata/page-1-html/index.html | file  |
      | extraHttpHeaders | {"foo":"bar;scope=*."}          | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'extraHttpHeaders' is invalid (got '{"foo":"bar;scope=*."}', resulting to invalid scope regex pattern for header 'foo': error parsing regexp: missing argument to repetition operator in `*.`)
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                 | testdata/page-1-html/index.html | file  |
      | emulatedMediaFeatures | foo                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'emulatedMediaFeatures' is invalid (got 'foo', resulting to unmarshal emulatedMediaFeatures: invalid character 'o' in literal false (expecting 'a'))
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html | file  |
      | stages | foo                             | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stages must be a JSON array of objects: invalid character 'o' in literal false (expecting 'a')
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html | file  |
      | stages | [{"op":"shred"}]                | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: unknown operation "shred"
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html | file  |
      | stages | [{"op":"split"}]                | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: split: set "every" or "ranges"
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html      | file  |
      | stages | [{"op":"remediate","profile":"foo"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: remediate: "profile" must be one of
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html                                      | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: "rotate" cannot follow "remediate": only encrypt, optimize, sign and timestamp may
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html          | file   |
      | stages                    | [{"op":"rotate","angle":90,"pages":"5"}] | field  |
      | Gotenberg-Output-Filename | foo                                      | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should contain string:
      """
      Stage 'rotate' failed on 'foo.pdf':
      """

  @pdf-split
  Scenario: POST /forms/chromium/convert/html (Split Intervals)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/pages-3-html/index.html | file  |
      | stages | [{"op":"split","every":2}]       | field |
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
  # A backslash is not a path separator on Linux, so filepath.Base leaves it in
  # place and it reaches the archive entry names. See GHSA-hwc4-gmrw-5222.
  @pdf-split
  @output-filename
  Scenario: POST /forms/chromium/convert/html (Split Windows Path As Output Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html   | file   |
      | stages                    | [{"op":"split","every":2}]         | field  |
      | Gotenberg-Output-Filename | ..\\..\\..\\Windows\\System32\\foo | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.zip   |
      | foo_0.pdf |
      | foo_1.pdf |

  Scenario: POST /forms/chromium/convert/html (Split Rooted Windows Path As Output Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html | file   |
      | stages                    | [{"op":"split","every":2}]       | field  |
      | Gotenberg-Output-Filename | C:\\Windows\\Temp\\foo           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.zip   |
      | foo_0.pdf |
      | foo_1.pdf |

  Scenario: POST /forms/chromium/convert/html (Split Output Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html | file   |
      | stages                    | [{"op":"split","every":2}]       | field  |
      | Gotenberg-Output-Filename | foo                              | header |
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
  Scenario: POST /forms/chromium/convert/html (Split Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/pages-3-html/index.html | file  |
      | stages | [{"op":"split","ranges":"2,3"}]  | field |
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
  Scenario: POST /forms/chromium/convert/html (Split Pages & Unify)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html | file   |
      | stages                    | [{"op":"select","pages":"2-3"}]  | field  |
      | Gotenberg-Output-Filename | foo                              | header |
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
  Scenario: POST /forms/chromium/convert/html (Split Many PDFs - Lot of Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/pages-12-html/index.html | file  |
      | stages | [{"op":"split","every":1}]        | field |
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
  Scenario: POST /forms/chromium/convert/html (PDF/A-1b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html           | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-1b"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  @pdf-remediate
  Scenario: POST /forms/chromium/convert/html (PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files           | testdata/page-1-html/index.html           | file  |
      | nativeTaggedPdf | true                                      | field |
      | stages          | [{"op":"remediate","profile":"PDF/UA-1"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 0 failed rule(s)

  @pdf-remediate
  @pdf-split
  Scenario: POST /forms/chromium/convert/html (Split & PDF/A-1b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/pages-3-html/index.html                                   | file  |
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
  Scenario: POST /forms/chromium/convert/html (Split & PDF/A-1b & Output Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html                                   | file   |
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
  Scenario: POST /forms/chromium/convert/html (Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html                                                                                                                               | file   |
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
  Scenario: POST /forms/chromium/convert/html (Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html | file  |
      | stages | [{"op":"flatten-form"}]         | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be flatten

  # The optimize stage recompresses images losslessly and deduplicates streams.
  # litho does not re-encode to JPEG, so only success is asserted.
  # See https://github.com/gotenberg/gotenberg/issues/359.
  @pdf-optimize
  Scenario: POST /forms/chromium/convert/html (Optimize Images)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/optimize-image-html/index.html | file   |
      | files                     | testdata/optimize-image-html/image.png  | file   |
      | Gotenberg-Output-Filename | foo                                     | header |
    Then the response status code should be 200
    Then the "foo.pdf" file size should be greater than 300 KB
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/optimize-image-html/index.html | file   |
      | files                     | testdata/optimize-image-html/image.png  | file   |
      | stages                    | [{"op":"optimize"}]                     | field  |
      | Gotenberg-Output-Filename | foo                                     | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  @pdf-encrypt
  Scenario: POST /forms/chromium/convert/html (Encrypt - user password only)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html         | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @pdf-encrypt
  Scenario: POST /forms/chromium/convert/html (Encrypt - both user and owner passwords)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html                               | file  |
      | stages | [{"op":"encrypt","userPassword":"foo","ownerPassword":"bar"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @pdf-stamp
  Scenario: POST /forms/chromium/convert/html (Watermark - Text)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html                           | file  |
      | stages | [{"op":"stamp-text","text":"CONFIDENTIAL","opacity":0.3}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  @pdf-stamp
  Scenario: POST /forms/chromium/convert/html (Stamp - Text)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html      | file  |
      | stages | [{"op":"stamp-text","text":"DRAFT"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  @pdf-rotate
  Scenario: POST /forms/chromium/convert/html (Rotate 90)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | stages                    | [{"op":"rotate","angle":90}]    | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF page 1 should be rotated by 90 degrees

  @pdf-attach
  Scenario: POST /forms/chromium/convert/html (Attach)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html                                             | file   |
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

  # Attaching files to a PDF/A-3 document needs a relationship, here Data.
  @pdf-remediate
  @pdf-metadata
  @pdf-stamp
  @pdf-form
  @pdf-attach
  Scenario: POST /forms/chromium/convert/html (PDF/A-3b & Metadata & Flatten & Attach)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html                                                                                                                                                                                                                                                                                                                   | file   |
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

  Scenario: POST /forms/chromium/convert/html (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then the response status code should be 404

  Scenario: POST /forms/chromium/convert/html (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files           | testdata/page-1-html/index.html | file   |
      | Gotenberg-Trace | forms_chromium_convert_html     | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_chromium_convert_html"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"forms_chromium_convert_html" |

  @download-from
  Scenario: POST /forms/chromium/convert/html (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"{helper}/static/testdata/page-1-html/index.html","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "application/pdf"

  @webhook
  Scenario: POST /forms/chromium/convert/html (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                        | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename    | foo                             | header |
      | Gotenberg-Webhook-Url        | {helper}/webhook                | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events         | header |
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

  Scenario: POST /forms/chromium/convert/html (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/chromium/convert/html (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  # See: https://github.com/gotenberg/gotenberg/issues/1505.
  Scenario: POST /forms/chromium/convert/html (Asset)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/html-with-asset/index.html | file   |
      | files                     | testdata/html-with-asset/image.png  | file   |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have 1 image(s)

  Scenario: POST /forms/chromium/convert/html (Stage file not uploaded => 400)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files  | testdata/page-1-html/index.html            | file  |
      | stages | [{"op":"stamp-pdf","pdf":"/etc/hostname"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form file '/etc/hostname' is required
      """

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/chromium/convert/html (Long Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
