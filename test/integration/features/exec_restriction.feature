@exec-restriction
Feature: --gotenberg-enable-exec-restriction

  # Landlock requires Linux 5.13 or later with Landlock enabled, which the
  # Docker hosts of the CI and Docker Desktop provide. Without it, Gotenberg
  # refuses to start and these scenarios fail.
  @chromium
  Scenario: POST /forms/chromium/convert/html (Exec Restriction)
    Given I have a Gotenberg container with the following environment variable(s):
      | GOTENBERG_ENABLE_EXEC_RESTRICTION | true |
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the Gotenberg container should log the following entries:
      | exec restriction enabled |
      | /usr/lib/chromium        |

  @libreoffice
  Scenario: POST /forms/libreoffice/convert (Exec Restriction)
    Given I have a Gotenberg container with the following environment variable(s):
      | GOTENBERG_ENABLE_EXEC_RESTRICTION | true |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the Gotenberg container should log the following entries:
      | exec restriction enabled     |
      | /usr/lib/libreoffice/program |
