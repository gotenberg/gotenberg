@chromium
Feature: Chromium corpus
  Realistic web documents with known contents, described in testdata/MANIFEST.md.
  Each scenario sends a site, a page or a Markdown bundle through the Chromium routes.

  # The page break comes from the CSS (`break-before: page` on the payment terms heading) and
  # the size from `@page { size: A4 }`.
  Scenario: Invoice site (2 A4 pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/invoice/index.html  | file   |
      | files                     | testdata/web/invoice/style.css   | file   |
      | files                     | testdata/web/invoice/logo.png    | file   |
      | files                     | testdata/web/invoice/header.html | file   |
      | files                     | testdata/web/invoice/footer.html | file   |
      | preferCssPageSize         | true                             | field  |
      | printBackground           | true                             | field  |
      | nativeBookmarks           | true                             | field  |
      | Gotenberg-Output-Filename | foo                              | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF page 1 should be 595.28x841.89 points
    Then the "foo.pdf" PDF page 2 should be 595.28x841.89 points
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Invoice INV-2026-0042"
    Then the "foo.pdf" PDF should have a document outline
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Northwind Traders
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Contoso Ltd
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      4,620.00 EUR
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Payment terms
      """
    Then the "foo.pdf" PDF should have content matching "Page\s*1\s*of\s*2" at page 1
    Then the "foo.pdf" PDF should have content matching "Page\s*2\s*of\s*2" at page 2
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Payment terms
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      IBAN FR76 3000 6000 0112 3456 7890 189
      """

  Scenario: Invoice site from a ZIP archive
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | archives                  | testdata/web/site.zip | file   |
      | preferCssPageSize         | true                  | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF page 1 should be 595.28x841.89 points
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Invoice INV-2026-0042
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      IBAN FR76 3000 6000 0112 3456 7890 189
      """

  Scenario: Delayed content waits for an expression
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/wait/index.html | file   |
      | waitForExpression         | window.chartReady === true   | field  |
      | Gotenberg-Output-Filename | foo                          | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Revenue: 42,000 EUR
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Loading
      """

  # The exception happens after the render: the page converts unless the flag is set.
  Scenario: A console exception fails the conversion
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                   | testdata/web/console-error/index.html | file  |
      | failOnConsoleExceptions | true                                  | field |
    Then the response status code should be 409
    Then the response body should contain string:
      """
      Chromium console exceptions
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/console-error/index.html | file   |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Console error page
      """

  # The inline script sets the title to "pwned" if it runs. The title stays the one of the
  # template and the text right after the script survives.
  Scenario: Markdown bundle keeps the text and the title
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/web/markdown/index.html       | file   |
      | files                     | testdata/web/markdown/release-notes.md | file   |
      | files                     | testdata/web/markdown/changelog.md     | file   |
      | files                     | testdata/web/markdown/style.css        | file   |
      | files                     | testdata/web/markdown/screenshot.png   | file   |
      | Gotenberg-Output-Filename | foo                                    | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Release notes 9.0.0"
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Release notes 9.0.0
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      CODE-BLOCK-MARKER
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      DONE-TASK
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      OPEN-TASK
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Removed legacy flag
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      SANITIZE-ME-MARKER
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      CHANGELOG-V9
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      CHANGELOG-V8
      """
    Then the "foo.pdf" PDF should have 1 image(s)

  # The navigation is hidden in print, and a block shows only from a 1200 px viewport.
  Scenario: Landing page: print media and viewport width
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/landing/index.html | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Start free trial
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Pricing
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      WIDE-LAYOUT-MARKER
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/landing/index.html | file   |
      | paperWidth                | 14                              | field  |
      | paperHeight               | 11                              | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      WIDE-LAYOUT-MARKER
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/landing/index.html | file   |
      | singlePage                | true                            | field  |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Start free trial
      """

  # The Latin marker of a right-to-left line can come out as ":AR": the patterns accept both.
  Scenario: Multilingual page renders every script
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/i18n/index.html | file   |
      | Gotenberg-Output-Filename | foo                          | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Multilingual
      """
    Then the "foo.pdf" PDF should have content matching "AR:|:AR" at page 1
    Then the "foo.pdf" PDF should have content matching "HE:|:HE" at page 1
    Then the "foo.pdf" PDF should have content matching "ZH:|:ZH" at page 1
    Then the "foo.pdf" PDF should have content matching "JA:|:JA" at page 1
    Then the "foo.pdf" PDF should have content matching "KO:|:KO" at page 1
    Then the "foo.pdf" PDF should have content matching "TH:|:TH" at page 1
    Then the "foo.pdf" PDF should have content matching "HI:|:HI" at page 1
    Then the "foo.pdf" PDF should have content matching "EL:|:EL" at page 1
    Then the "foo.pdf" PDF should have content matching "RU:|:RU" at page 1
    Then the "foo.pdf" PDF should have content matching "EMOJI:|:EMOJI" at page 1
    Then the "foo.pdf" PDF should have content matching "MATH:|:MATH" at page 1

  Scenario: Print CSS handbook gives A5 pages
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/print-css/index.html | file   |
      | files                     | testdata/web/print-css/print.css  | file   |
      | preferCssPageSize         | true                              | field  |
      | Gotenberg-Output-Filename | foo                               | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 4 page(s)
    Then the "foo.pdf" PDF page 1 should be 419.53x595.28 points
    Then the "foo.pdf" PDF page 4 should be 419.53x595.28 points
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Handbook
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      NAV-MARKER
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      CHAPTER-ONE
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      CHAPTER-TWO
      """
    Then the "foo.pdf" PDF should have the following content at page 4:
      """
      CHAPTER-THREE
      """

  Scenario: Wide table on a single page
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/web/wide-table/index.html | file   |
      | singlePage                | true                               | field  |
      | landscape                 | true                               | field  |
      | Gotenberg-Output-Filename | foo                                | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Wide table"
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      C01
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      r40c30
      """
