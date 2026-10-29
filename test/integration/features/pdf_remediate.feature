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
      Invalid form data: remediate: "xml" must name the uploaded Factur-X invoice XML
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
