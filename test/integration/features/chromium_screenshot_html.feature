@chromium
@chromium-screenshot-html
Feature: /forms/chromium/screenshot/html

  # Route parity: the WebSocket outbound filter lives in the shared browser
  # code path, so the screenshot route enforces it exactly like conversion.
  @chromium-ssrf
  Scenario: POST /forms/chromium/screenshot/html (WebSocket to a non-public address is filtered)
    Given I have a Gotenberg container with the following environment variable(s):
      | CHROMIUM_ALLOW_LIST       |      |
      | CHROMIUM_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files     | testdata/ssrf-websocket-html/index.html | file  |
      | waitDelay | 1s                                      | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"
    Then the Gotenberg container should log the following entries:
      | 'ws://127.0.0.1:9999/ssrf-websocket' targets a non-public address |
      | CONNECT blocked for '127.0.0.1:9999'                              |

  Scenario: POST /forms/chromium/screenshot/html (Default)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"
    Then there should be the following file(s) in the response:
      | foo.png |

  Scenario: POST /forms/chromium/screenshot/html (JPEG)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | format                    | jpeg                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/jpeg"
    Then there should be the following file(s) in the response:
      | foo.jpeg |

  Scenario: POST /forms/chromium/screenshot/html (WebP)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | format                    | webp                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/webp"
    Then there should be the following file(s) in the response:
      | foo.webp |

  Scenario: POST /forms/chromium/screenshot/html (Custom Dimensions)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | width                     | 1920                            | field  |
      | height                    | 1080                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  Scenario: POST /forms/chromium/screenshot/html (Clip)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | clip                      | true                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  # The target element is 300x200 and sits below a spacer, so a correct clip
  # proves both the element size and its page offset. See issue #947.
  Scenario: POST /forms/chromium/screenshot/html (Selector)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/screenshot-selector-html/index.html | file   |
      | selector                  | #target                                      | field  |
      | Gotenberg-Output-Filename | foo                                          | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"
    Then the "foo.png" image should be 300x200 pixels
    # A centered red pixel proves the clip landed on the element, not on the
    # white spacer above it, i.e. the page offset was applied.
    Then the "foo.png" image pixel at 150,100 should be "#ff0000"

  Scenario: POST /forms/chromium/screenshot/html (Selector Not Found)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files    | testdata/screenshot-selector-html/index.html | file  |
      | selector | #does-not-exist                              | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      The selector '#does-not-exist' (selector) matched no element with a visible box
      """

  Scenario: POST /forms/chromium/screenshot/html (Quality)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | format                    | jpeg                            | field  |
      | quality                   | 50                              | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/jpeg"

  Scenario: POST /forms/chromium/screenshot/html (Optimize for Speed)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | optimizeForSpeed          | true                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  Scenario: POST /forms/chromium/screenshot/html (Device Scale Factor)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | deviceScaleFactor         | 1.0                             | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  Scenario: POST /forms/chromium/screenshot/html (Omit Background)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | omitBackground            | true                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  Scenario: POST /forms/chromium/screenshot/html (Emulated Media Type)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | emulatedMediaType         | screen                          | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  Scenario: POST /forms/chromium/screenshot/html (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | skipNetworkIdleEvent        | foo | field |
      | skipNetworkAlmostIdleEvent  | foo | field |
      | failOnResourceLoadingFailed | foo | field |
      | failOnConsoleExceptions     | foo | field |
      | emulatedMediaType           | foo | field |
      | omitBackground              | foo | field |
      | width                       | foo | field |
      | height                      | foo | field |
      | clip                        | foo | field |
      | format                      | bar | field |
      | quality                     | -1  | field |
      | optimizeForSpeed            | foo | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"

  @webhook
  Scenario: POST /forms/chromium/screenshot/html (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                       | testdata/page-1-html/index.html              | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "image/png"
    Then there should be the following file(s) in the webhook request:
      | foo.png |

  Scenario: POST /forms/chromium/screenshot/html (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/chromium/screenshot/html (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files | testdata/page-1-html/index.html | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/chromium/screenshot/html (Long Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/screenshot/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/png"
