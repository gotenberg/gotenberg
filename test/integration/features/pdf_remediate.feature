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
      | files                     | testdata/page_1.pdf          | file   |
      | files                     | testdata/invoice-en16931.xml | file   |
      | profile                   | Factur-X                     | field  |
      | xml                       | invoice-en16931.xml          | field  |
      | level                     | EN 16931                     | field  |
      | Gotenberg-Output-Filename | foo                          | header |
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
      | files                     | testdata/page_1.pdf            | file   |
      | files                     | testdata/invoice-xrechnung.xml | file   |
      | profile                   | Factur-X                       | field  |
      | xml                       | invoice-xrechnung.xml          | field  |
      | level                     | XRECHNUNG                      | field  |
      | Gotenberg-Output-Filename | foo                            | header |
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
      | files                     | testdata/page_1.pdf          | file   |
      | files                     | testdata/invoice-minimum.xml | file   |
      | profile                   | Factur-X                     | field  |
      | xml                       | invoice-minimum.xml          | field  |
      | level                     | MINIMUM                      | field  |
      | Gotenberg-Output-Filename | foo                          | header |
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
      | files                     | testdata/page_1.pdf          | file   |
      | files                     | testdata/invoice-en16931.xml | file   |
      | profile                   | Factur-X                     | field  |
      | xml                       | invoice-en16931.xml          | field  |
      | level                     | EN 16931                     | field  |
      | Gotenberg-Output-Filename | foo                          | header |
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

  # The invoice XML must be a Cross-Industry Invoice of the declared level.
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X Invalid Invoice)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf          | file  |
      | files   | testdata/invoice-en16931.xml | file  |
      | profile | Factur-X                     | field |
      | xml     | invoice-en16931.xml          | field |
      | level   | MINIMUM                      | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'remediate' failed on 'page_1.pdf': factur-x conformance level does not match the invoice XML: level "MINIMUM" given, but the invoice XML declares "EN 16931" (guideline ID "urn:cen.eu:en16931:2017")
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf  | file  |
      | files   | testdata/embed_1.xml | file  |
      | profile | Factur-X             | field |
      | xml     | embed_1.xml          | field |
      | level   | EN 16931             | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'remediate' failed on 'page_1.pdf': factur-x invoice XML is not a Cross-Industry Invoice: the root element is not CrossIndustryInvoice in namespace "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
      """

  # A pair of profiles, the field repeated, writes a PDF that conforms to both,
  # and the residual header counts what either leaves.
  Scenario: POST /forms/pdf/remediate (PDF/A-2b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
      | profile | PDF/UA-1            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the response header "Gotenberg-Remediation-Rolled-Back" should be "0"
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 0 failed rule(s)

  Scenario: POST /forms/pdf/pipe (PDF/A-4 & PDF/UA-2)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.pdf                                  | file  |
      | stages | [{"op":"remediate","profile":["PDF/A-4","PDF/UA-2"]}] | field |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the response PDF(s) should be valid "PDF/A-4" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-2" with a tolerance of 0 failed rule(s)

  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf          | file   |
      | files                     | testdata/invoice-en16931.xml | file   |
      | profile                   | Factur-X                     | field  |
      | profile                   | PDF/UA-1                     | field  |
      | xml                       | invoice-en16931.xml          | field  |
      | level                     | EN 16931                     | field  |
      | Gotenberg-Output-Filename | foo                          | header |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "EN 16931"

  # A PDF/A profile pairs with the PDF/UA profile of its PDF version alone, and
  # a list remediates once.
  Scenario: POST /forms/pdf/remediate (Impossible Pair)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | PDF/A-2b            | field |
      | profile | PDF/UA-2            | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: remediate: "profile" pairs a PDF/A profile with the PDF/UA profile of its PDF version: PDF/A-1a, PDF/A-1b, PDF/A-2a, PDF/A-2b, PDF/A-2u, PDF/A-3a, PDF/A-3b, PDF/A-3u or Factur-X with PDF/UA-1, and PDF/A-4, PDF/A-4e or PDF/A-4f with PDF/UA-2; got ["PDF/A-2b","PDF/UA-2"]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/pipe" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf                                                               | file  |
      | stages | [{"op":"remediate","profile":"PDF/A-2b"},{"op":"remediate","profile":"PDF/UA-1"}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 1: only one "remediate" stage is allowed; to remediate to a PDF/A profile and a PDF/UA one together, give one stage both, such as {"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]}
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
      Invalid form data: form field 'aggressive' is invalid (got 'foo', want true or false)
      """

  Scenario: POST /forms/pdf/autotag
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/autotag" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should be tagged
