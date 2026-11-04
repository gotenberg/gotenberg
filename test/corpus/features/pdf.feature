@pdf
Feature: PDF corpus
  Realistic PDF files with known contents, described in testdata/MANIFEST.md.
  Each scenario sends a report, an annex, an invoice or an encrypted copy through the PDF routes.

  Scenario: Report PDF: info and metadata
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/pdf/report.pdf | file |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "report.pdf": {
          "pages": 3,
          "version": "ignore",
          "encrypted": false,
          "tagged": false,
          "linearized": false,
          "formFields": false
        }
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/metadata" endpoint with the following form data and header(s):
      | files | testdata/pdf/report.pdf | file |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "report.pdf": {
          "title": "Quarterly Report",
          "author": "Finance"
        }
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf | file   |
      | Gotenberg-Output-Filename | foo                     | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF page 1 should be 612x792 points
    Then the "foo.pdf" PDF page 3 should be 612x792 points
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Quarterly Report"
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Body line 1 of page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      REPORT-PAGE-3
      """

  Scenario: Redaction removes the label of page 1 only
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/redact" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf                             | file   |
      | regions                   | [{"page":1,"x":70,"y":388,"width":300,"height":42}] | field  |
      | Gotenberg-Output-Filename | foo                                                 | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      REPORT-PAGE-1
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Quarterly Report
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Body line 1 of page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      REPORT-PAGE-2
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      REPORT-PAGE-3
      """
    Then the "foo.pdf" PDF page 1 should be 612x792 points

  # Inputs merge in the order of their filenames, so the annex comes first.
  Scenario: Merge report and annex
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf | file   |
      | files                     | testdata/pdf/annex.pdf  | file   |
      | Gotenberg-Output-Filename | foo                     | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 5 page(s)
    Then the "foo.pdf" PDF page 5 should be 612x792 points
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      ANNEX-PAGE-1
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      ANNEX-PAGE-2
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      REPORT-PAGE-1
      """
    Then the "foo.pdf" PDF should have the following content at page 4:
      """
      REPORT-PAGE-2
      """
    Then the "foo.pdf" PDF should have the following content at page 5:
      """
      REPORT-PAGE-3
      """

  Scenario: Stamp text and image
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf | file   |
      | text                      | APPROVED-STAMP          | field  |
      | pages                     | 1                       | field  |
      | x                         | 72                      | field  |
      | y                         | 200                     | field  |
      | fontSize                  | 24                      | field  |
      | Gotenberg-Output-Filename | foo                     | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      REPORT-PAGE-1
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf | file   |
      | files                     | testdata/pdf/stamp.jpg  | file   |
      | image                     | stamp.jpg               | field  |
      | pages                     | 1                       | field  |
      | width                     | 150                     | field  |
      | height                    | 50                      | field  |
      | Gotenberg-Output-Filename | foo                     | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have 1 image(s)

  Scenario: Letterhead stamped under the report
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf     | file   |
      | files                     | testdata/pdf/letterhead.pdf | file   |
      | source                    | letterhead.pdf              | field  |
      | Gotenberg-Output-Filename | foo                         | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      REPORT-PAGE-1
      """

  Scenario: Encrypted PDF needs its password
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/pdf/encrypted.pdf | file |
    Then the response status code should be 400
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files    | testdata/pdf/encrypted.pdf | file  |
      | password | secret                     | field |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "encrypted.pdf": {
          "pages": 3,
          "encrypted": true
        }
      }
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files | testdata/pdf/encrypted.pdf | file |
    Then the response status code should be 400
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/encrypted.pdf | file   |
      | password                  | secret                     | field  |
      | Gotenberg-Output-Filename | foo                        | header |
    Then the response status code should be 200
    Then the response PDF(s) should NOT be encrypted
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      REPORT-PAGE-3
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/encrypted-rc4.pdf | file   |
      | password                  | secret                         | field  |
      | Gotenberg-Output-Filename | foo                            | header |
    Then the response status code should be 200
    Then the response PDF(s) should NOT be encrypted
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      REPORT-PAGE-2
      """

  Scenario: Factur-X embeds the CII invoice
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/report.pdf      | file   |
      | files                     | testdata/pdf/invoice-cii.xml | file   |
      | profile                   | Factur-X                     | field  |
      | xml                       | invoice-cii.xml              | field  |
      | level                     | EN 16931                     | field  |
      | Gotenberg-Output-Filename | foo                          | header |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Quarterly Report"
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "EN 16931"

  Scenario: Transparency remediates to PDF/A-2b
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/pdf/transparent.pdf | file   |
      | profile                   | PDF/A-2b                     | field  |
      | Gotenberg-Output-Filename | foo                          | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      TRANSPARENT-MARKER
      """
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)

  Scenario: Color and image-heavy samples read back
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/pdf/colors.pdf      | file |
      | files | testdata/pdf/image-heavy.pdf | file |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "colors.pdf": {
          "pages": 1
        },
        "image-heavy.pdf": {
          "pages": 3
        }
      }
      """
