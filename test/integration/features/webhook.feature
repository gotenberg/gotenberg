@webhook
Feature: Webhook

  Scenario: Default
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    Then the response header "Gotenberg-Version" should be "{version}"
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then the webhook request header "Gotenberg-Version" should be "{version}"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: Extra HTTP Headers
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                                | testdata/page_1.pdf                            | file   |
      | Gotenberg-Webhook-Url                | {helper}/webhook                               | header |
      | Gotenberg-Webhook-Events-Url         | {helper}/webhook/events                        | header |
      | Gotenberg-Webhook-Extra-Http-Headers | {"X-Foo":"bar","Content-Disposition":"inline"} | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then the webhook request header "X-Foo" should be "bar"
    # https://github.com/gotenberg/gotenberg/issues/1165
    Then the webhook request header "Content-Disposition" should be "inline"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: Synchronous
    Given I have a Gotenberg container with the following environment variable(s):
      | WEBHOOK_ENABLE_SYNC_MODE | true |
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: Webhook Events URL (Success)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the webhook event should match JSON:
      """
      {
        "event": "webhook.success",
        "correlationId": "ignore",
        "timestamp": "ignore"
      }
      """

  Scenario: Webhook Events URL (Synchronous)
    Given I have a Gotenberg container with the following environment variable(s):
      | WEBHOOK_ENABLE_SYNC_MODE | true |
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the webhook event should match JSON:
      """
      {
        "event": "webhook.success",
        "correlationId": "ignore",
        "timestamp": "ignore"
      }
      """

  # A failed conversion reaches the events URL as a webhook.error event.
  Scenario: Webhook Events URL (Error)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    Then the webhook event should match JSON:
      """
      {
        "event": "webhook.error",
        "correlationId": "ignore",
        "timestamp": "ignore",
        "error": {
          "status": 400,
          "message": "ignore"
        }
      }
      """

  # A failed conversion in sync mode still answers 204: the outcome goes to the
  # events URL, as in async mode.
  Scenario: Webhook Events URL (Synchronous Error)
    Given I have a Gotenberg container with the following environment variable(s):
      | WEBHOOK_ENABLE_SYNC_MODE | true |
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    Then the webhook event should match JSON:
      """
      {
        "event": "webhook.error",
        "correlationId": "ignore",
        "timestamp": "ignore",
        "error": {
          "status": 400,
          "message": "ignore"
        }
      }
      """

  # Routes answering with JSON have no output file to deliver. The asynchronous
  # case lives in pdf_metadata.feature.
  Scenario: Route Without Output File (Synchronous)
    Given I have a Gotenberg container with the following environment variable(s):
      | WEBHOOK_ENABLE_SYNC_MODE | true |
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    Then the response body should match string:
      """

      """
    Then the webhook event should match JSON:
      """
      {
        "event": "webhook.error",
        "correlationId": "ignore",
        "timestamp": "ignore",
        "error": {
          "status": 400,
          "message": "The webhook middleware can only work with multipart/form-data routes that results in output files"
        }
      }
      """

  # A graceful shutdown stops taking requests, then lets the asynchronous ones
  # deliver before the engines stop.
  Scenario: Graceful Shutdown With An Asynchronous Request In Flight
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                        | testdata/page-1-html/index.html | file   |
      | waitDelay                    | 3s                              | field  |
      | Gotenberg-Webhook-Url        | {helper}/webhook                | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events         | header |
    Then the response status code should be 204
    When I stop the Gotenberg container
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then the Gotenberg container should log the following entries:
      | api: application stopped      |
      | chromium: application stopped |

  Scenario: Webhook Method
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
      | Gotenberg-Webhook-Method     | PUT                     | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request method should be "PUT"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: Invalid Webhook Method
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
      | Gotenberg-Webhook-Method     | GET                     | header |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid 'Gotenberg-Webhook-Method' header value: expected 'POST', 'PATCH' or 'PUT', but got 'GET'
      """

  Scenario: Invalid Webhook Extra HTTP Headers
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                                | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url                | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url         | {helper}/webhook/events | header |
      | Gotenberg-Webhook-Extra-Http-Headers | {"X-Foo":               | header |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid 'Gotenberg-Webhook-Extra-Http-Headers' header value
      """

  # The outbound filter refuses a webhook URL the deny-list names, even when
  # the allow-list matches every URL. A refusal stays generic for the client.
  Scenario: Webhook URL In The Deny-List
    Given I have a webhook server
    Given I have a Gotenberg container with the following environment variable(s):
      | WEBHOOK_DENY_LIST | ^{helper}/webhook$ |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 403
    Then the response body should match string:
      """
      Forbidden
      """

  # The helper server resolves to a non-public address: a private one on Linux,
  # 0.250.250.254 on Docker Desktop.
  Scenario: Webhook URL On A Private Address
    Given I have a webhook server
    Given I have a Gotenberg container with the following environment variable(s):
      | WEBHOOK_ALLOW_LIST       |      |
      | WEBHOOK_DENY_PRIVATE_IPS | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 403
    Then the response body should match string:
      """
      Forbidden
      """

  Scenario: Missing Events URL
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                 | testdata/page_1.pdf | file   |
      | Gotenberg-Webhook-Url | {helper}/webhook    | header |
    Then the response status code should be 400
    Then the response body should match string:
      """
      The 'Gotenberg-Webhook-Events-Url' header is required with 'Gotenberg-Webhook-Url'
      """

  Scenario: Asynchronous Trace Continuity
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf                                     | file   |
      | Gotenberg-Webhook-Url        | {helper}/webhook                                        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events                                 | header |
      | traceparent                  | 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01 | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "traceparent" should carry trace id "0af7651916cd43dd8448eb211c80319c"
