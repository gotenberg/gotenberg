# TODO:
# 1. Other HTTP Methods
# 2. Errors

Feature: Webhook

  Scenario: Default
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/flatten" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request

  Scenario: Extra HTTP Headers
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/flatten" endpoint with the following form data and header(s):
      | files                                | testdata/page_1.pdf                            | file   |
      | Gotenberg-Webhook-Url                | http://host.docker.internal:%d/webhook         | header |
      | Gotenberg-Webhook-Error-Url          | http://host.docker.internal:%d/webhook/error   | header |
      | Gotenberg-Webhook-Extra-Http-Headers | {"X-Foo":"bar","Content-Disposition":"inline"} | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then the webhook request header "X-Foo" should be "bar"
    # https://github.com/gotenberg/gotenberg/issues/1165
    Then the webhook request header "Content-Disposition" should be "inline"
    Then there should be 1 PDF(s) in the webhook request
