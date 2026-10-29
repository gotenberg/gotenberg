@pdf
@pdf-metadata
Feature: /forms/pdf/{set-metadata|metadata}

  Scenario: POST /forms/pdf/{set-metadata|metadata} (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-metadata" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | title                     | Sample              | field  |
      | author                    | Julien Neuhart      | field  |
      | subject                   | Sample              | field  |
      | keywords                  | first, second       | field  |
      | creator                   | Gotenberg           | field  |
      | producer                  | Gotenberg           | field  |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Julien Neuhart"
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "Title": "Sample",
          "Author": "Julien Neuhart",
          "Subject": "Sample",
          "Keywords": "first, second",
          "Creator": "Gotenberg",
          "Producer": "Gotenberg"
        }
      }
      """

  Scenario: POST /forms/pdf/metadata (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files | testdata/titled_alpha.pdf | file |
      | files | testdata/titled_bravo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "titled_alpha.pdf": {
          "Title": "Alpha"
        },
        "titled_bravo.pdf": "ignore"
      }
      """

  Scenario: POST /forms/pdf/pipe (Metadata After Merge)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/titled_alpha.pdf                             | file   |
      | files                     | testdata/titled_bravo.pdf                             | file   |
      | stages                    | [{"op":"merge"},{"op":"set-metadata","title":"Both"}] | field  |
      | Gotenberg-Output-Filename | foo                                                   | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Both"

  Scenario: POST /forms/pdf/set-metadata (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                  | file  |
      | stages | [{"op":"set-metadata","Copyright":"Julien Neuhart"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: set-metadata: unknown option "Copyright"
      """

  Scenario: POST /forms/pdf/metadata (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdf/metadata (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDF_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404

  @webhook
  Scenario: POST /forms/pdf/metadata (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files                       | testdata/titled_alpha.pdf                    | file   |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/json"
    Then the webhook request body should match JSON:
      """
      {
        "status": 400,
        "message": "The webhook middleware can only work with multipart/form-data routes that results in output files"
      }
      """

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/pdf/metadata (Long Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files | testdata/Longitudinell_jämförelse_mellan_laserkirurgi_och_strålbehandling_gällande_röstkvalitet_och_självskattad_kommunikation_upp_till_två_år_efter_tidig_stämbandscancer_i_ett_randomiserat_kontrollerat_försök.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
