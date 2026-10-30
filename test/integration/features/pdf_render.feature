@pdf
@pdf-render
Feature: /forms/pdf/render

  Scenario: POST /forms/pdf/render (Every Page)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/render" endpoint with the following form data and header(s):
      | files | testdata/pages_3.pdf | file  |
      | dpi   | 36                   | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 0 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_1.png |
      | pages_3_2.png |
      | pages_3_3.png |
    Then the "pages_3_1.png" image should be 224x316 pixels
    Then the "pages_3_1.png" image pixel at 0,0 should be "#ffffff"

  Scenario: POST /forms/pdf/render (One Page)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/render" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | pages                     | 2                    | field  |
      | format                    | jpeg                 | field  |
      | quality                   | 80                   | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "image/jpeg"
    Then there should be the following file(s) in the response:
      | foo.jpeg |
    Then the "foo.jpeg" image should be 930x1316 pixels

  Scenario: POST /forms/pdf/pipe (Render)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                                | file  |
      | files  | testdata/pages_3.pdf                                                               | file  |
      | stages | [{"op":"merge"},{"op":"encrypt","userPassword":"u"},{"op":"render","pages":"1,4"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.pdf   |
      | page_1_1.png |
      | page_1_4.png |

  Scenario: POST /forms/pdf/render (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/render" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf | file  |
      | format | webp                | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: render: "format" must be png or jpeg, got "webp"
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/render" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | pages | 2                   | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Stage 'render' failed on 'page_1.pdf'
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                          | file  |
      | stages | [{"op":"render"},{"op":"rotate","angle":90}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: "render" must be the last stage
      """

  @webhook
  Scenario: POST /forms/pdf/render (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/render" endpoint with the following form data and header(s):
      | files                        | testdata/pages_3.pdf    | file   |
      | dpi                          | 36                      | field  |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/zip"
