@pdf
@pdf-remediate
Feature: /forms/pdf/remediate

  Scenario: POST /forms/pdf/remediate (PDF/A-1b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-1b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/remediate (PDF/A-2b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the response header "Gotenberg-Remediation-Rolled-Back" should be "0"
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/remediate (Aggressive)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files         | testdata/pages_3.pdf | file  |
      | profile       | PDF/A-2b             | field |
      | aggressive    | true                 | field |
      | figureAltText | A figure             | field |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/remediate (PDF/A-3b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-3b            | field |
    Then the response status code should be 200
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/remediate (PDF/A-4)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-4             | field |
    Then the response status code should be 200
    Then the response PDF(s) should be valid "PDF/A-4" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/pipe (Autotag & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf                                                                                             | file   |
      | stages                    | [{"op":"set-metadata","title":"Pages"},{"op":"autotag"},{"op":"remediate","profile":"PDF/UA-1","language":"en"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                              | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should be tagged
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/remediate (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | files   | testdata/page_2.pdf | file  |
      | profile | PDF/A-2b            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)

  # A Factur-X remediation embeds the invoice XML into a PDF/A-3 document.
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | EN 16931             | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "EN 16931"

  # XRECHNUNG embeds the invoice as xrechnung.xml, without a factur-x.xml
  # (Factur-X 1.07.2, 6.2 and 7.7).
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X XRECHNUNG)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | XRECHNUNG            | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "xrechnung.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should NOT have the "factur-x.xml" file embedded
    Then the response PDF(s) should declare Factur-X XMP with conformance level "XRECHNUNG"

  # MINIMUM carries less invoice data than the visual part, so the relationship
  # is Data (Factur-X 1.07.2, 6.2.2).
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X MINIMUM)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | MINIMUM              | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Data"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "MINIMUM"

  # Without the XML, Factur-X repairs the container metadata of the invoice the
  # PDF already carries.
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X Repair)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | EN 16931             | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | teststore/foo.pdf | file  |
      | profile | Factur-X          | field |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Rolled-Back" should be "0"
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | Factur-X            | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'remediate' failed on 'page_1.pdf': no hybrid invoice attachment found
      """

  @webhook
  Scenario: POST /forms/pdf/remediate (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.pdf     | file   |
      | profile                      | PDF/A-2b                | field  |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then the webhook request header "Gotenberg-Remediation-Residual" should be "0"
    Then the webhook request header "Gotenberg-Remediation-Rolled-Back" should be "0"

  Scenario: POST /forms/pdf/remediate (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/X-4             | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: remediate: "profile" must be one of PDF/A-1a, PDF/A-1b
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | Factur-X            | field |
      | level   | BASIC               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: remediate: "level" and "documentType" need "xml", the uploaded Factur-X invoice XML to embed; without "xml", the remediation repairs the invoice the PDF already carries
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
      | level   | BASIC               | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: remediate: "xml", "level" and "documentType" only apply to the Factur-X profile
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files      | testdata/page_1.pdf | file  |
      | profile    | PDF/A-2b            | field |
      | aggressive | foo                 | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'aggressive' is invalid: must be true or false, got "foo"
      """

  Scenario: POST /forms/pdf/autotag
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/autotag" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should be tagged
