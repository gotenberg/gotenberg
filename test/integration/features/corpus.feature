@corpus
Feature: Release QA corpus
  Realistic documents with known contents, described in testdata/corpus/MANIFEST.md.

  # The page break comes from the CSS (`break-before: page` on the payment terms heading) and
  # the size from `@page { size: A4 }`.
  @corpus-chromium
  Scenario: Invoice site (2 A4 pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/invoice/index.html  | file   |
      | files                     | testdata/corpus/web/invoice/style.css   | file   |
      | files                     | testdata/corpus/web/invoice/logo.png    | file   |
      | files                     | testdata/corpus/web/invoice/header.html | file   |
      | files                     | testdata/corpus/web/invoice/footer.html | file   |
      | preferCssPageSize         | true                                    | field  |
      | printBackground           | true                                    | field  |
      | nativeBookmarks           | true                                    | field  |
      | Gotenberg-Output-Filename | foo                                     | header |
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

  @corpus-chromium
  Scenario: Invoice site from a ZIP archive
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | archives                  | testdata/corpus/web/site.zip | file   |
      | preferCssPageSize         | true                         | field  |
      | Gotenberg-Output-Filename | foo                          | header |
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

  @corpus-chromium
  Scenario: Delayed content waits for an expression
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/wait/index.html | file   |
      | waitForExpression         | window.chartReady === true          | field  |
      | Gotenberg-Output-Filename | foo                                 | header |
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
  @corpus-chromium
  Scenario: A console exception fails the conversion
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                   | testdata/corpus/web/console-error/index.html | file  |
      | failOnConsoleExceptions | true                                         | field |
    Then the response status code should be 409
    Then the response body should contain string:
      """
      Chromium console exceptions
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/console-error/index.html | file   |
      | Gotenberg-Output-Filename | foo                                          | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Console error page
      """

  # The inline script sets the title to "pwned" if it runs. The title stays the one of the
  # template and the text right after the script survives.
  @corpus-chromium
  Scenario: Markdown bundle keeps the text and the title
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/markdown" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/markdown/index.html       | file   |
      | files                     | testdata/corpus/web/markdown/release-notes.md | file   |
      | files                     | testdata/corpus/web/markdown/changelog.md     | file   |
      | files                     | testdata/corpus/web/markdown/style.css        | file   |
      | files                     | testdata/corpus/web/markdown/screenshot.png   | file   |
      | Gotenberg-Output-Filename | foo                                           | header |
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
  @corpus-chromium
  Scenario: Landing page: print media and viewport width
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/landing/index.html | file   |
      | Gotenberg-Output-Filename | foo                                    | header |
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
      | files                     | testdata/corpus/web/landing/index.html | file   |
      | paperWidth                | 14                                     | field  |
      | paperHeight               | 11                                     | field  |
      | Gotenberg-Output-Filename | foo                                    | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      WIDE-LAYOUT-MARKER
      """
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/landing/index.html | file   |
      | singlePage                | true                                   | field  |
      | Gotenberg-Output-Filename | foo                                    | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Start free trial
      """

  # The Latin marker of a right-to-left line can come out as ":AR": the patterns accept both.
  @corpus-chromium
  Scenario: Multilingual page renders every script
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/i18n/index.html | file   |
      | Gotenberg-Output-Filename | foo                                 | header |
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

  @corpus-chromium
  Scenario: Print CSS handbook gives A5 pages
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/print-css/index.html | file   |
      | files                     | testdata/corpus/web/print-css/print.css  | file   |
      | preferCssPageSize         | true                                     | field  |
      | Gotenberg-Output-Filename | foo                                      | header |
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

  @corpus-chromium
  Scenario: Wide table on a single page
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/web/wide-table/index.html | file   |
      | singlePage                | true                                      | field  |
      | landscape                 | true                                      | field  |
      | Gotenberg-Output-Filename | foo                                       | header |
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

  @corpus-libreoffice
  Scenario: Contract DOCX (10 pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/contract.docx | file   |
      | nativeBookmarks           | true                                 | field  |
      | Gotenberg-Output-Filename | foo                                  | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 10 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Master Services Agreement
      """
    Then the "foo.pdf" PDF should have the following content at page 10:
      """
      Schedule A
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Schedule A
      """
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Master Services Agreement"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Legal Department"
    Then the "foo.pdf" PDF should have the "Subject" metadata set to "Services"
    Then the "foo.pdf" PDF should have the "Keywords" metadata set to "contract, msa"
    Then the "foo.pdf" PDF should have a document outline
    Then the "foo.pdf" PDF should have content matching "Page" at page 1

  # The first column clips long cells ("Cloud host"). The Monthly sheet spans two pages; `singlePage` fits each sheet on one page.
  @corpus-libreoffice
  Scenario: Budget XLSX (3 sheets)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/budget.xlsx | file   |
      | Gotenberg-Output-Filename | foo                                | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 4 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      FY2026 Budget
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Cloud host
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/budget.xlsx | file   |
      | singlePage                | true                               | field  |
      | Gotenberg-Output-Filename | foo                                | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      FY2026 Budget
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      Budget not
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      Prepared
      """

  @corpus-libreoffice
  Scenario: Deck PPTX without the hidden slide, then with notes pages
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/deck.pptx | file   |
      | Gotenberg-Output-Filename | foo                              | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 7 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Q4 Product Review
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Agenda
      """
    Then the "foo.pdf" PDF should have the following content at page 4:
      """
      R2C2
      """
    Then the "foo.pdf" PDF should have the following content at page 7:
      """
      Thank you
      """
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 1
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 2
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 3
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 4
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 5
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 6
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 7
    Then the "foo.pdf" PDF should NOT have content matching "NOTES-MARKER" at page 2
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/deck.pptx | file   |
      | exportNotesPages          | true                             | field  |
      | Gotenberg-Output-Filename | foo                              | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 14 page(s)
    Then the "foo.pdf" PDF should have the following content at page 9:
      """
      NOTES-MARKER-2
      """
    Then the "foo.pdf" PDF should have the following content at page 14:
      """
      NOTES-MARKER-8
      """
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 7
    Then the "foo.pdf" PDF should NOT have content matching "HIDDEN-SLIDE-MARKER" at page 13

  # Fails until lok passes the CSV import options: every line lands in one cell today.
  @corpus-libreoffice
  Scenario: CSV keeps its columns
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/sales.csv | file   |
      | Gotenberg-Output-Filename | foo                              | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      customer
      """
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      id,date,customer
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Café Müller
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Straße GmbH
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Élodie & Fils
      """

  @corpus-libreoffice
  Scenario: Tracked changes keep the inserted text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/tracked.docx | file   |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      INSERTED-TEXT
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      End of sentence.
      """

  @corpus-libreoffice
  Scenario: Multilingual DOCX
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/i18n.docx | file   |
      | Gotenberg-Output-Filename | foo                              | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Multilingual document
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

  @corpus-libreoffice
  Scenario: Form ODT keeps its text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/form.odt | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      FORM-DOC-MARKER
      """

  # The manifest counted 3 pages with another LibreOffice build; the image gives 4.
  @corpus-libreoffice
  Scenario: Plain text paginates
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/readme.txt | file   |
      | Gotenberg-Output-Filename | foo                               | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 4 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Line 01:
      """
    Then the "foo.pdf" PDF should have the following content at page 4:
      """
      Line 60:
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Café résumé naïve déjà vu
      """

  @corpus-libreoffice
  Scenario: WebP is not an accepted input
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/corpus/office/images/pic.webp | file |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: no form file found for extensions:
      """

  @corpus-libreoffice
  Scenario Outline: Office formats convert with the documented pages and text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/<file> | file   |
      | Gotenberg-Output-Filename | foo                           | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have <pages> page(s)
    Then the "foo.pdf" PDF should have the following content at page <page>:
      """
      <marker>
      """

    Examples:
      | file            | pages | page | marker            |
      | memo.fodt       | 1     | 1    | MEMO-END-MARKER   |
      | memo.odt        | 1     | 1    | MEMO-END-MARKER   |
      | memo.doc        | 1     | 1    | MEMO-END-MARKER   |
      | memo.rtf        | 1     | 1    | MEMO-END-MARKER   |
      | memo-lo.docx    | 1     | 1    | MEMO-END-MARKER   |
      | memo.html       | 1     | 1    | MEMO-END-MARKER   |
      | memo.pdf        | 1     | 1    | MEMO-END-MARKER   |
      | figures.fods    | 2     | 1    | REVENUE-SH        |
      | figures.fods    | 2     | 2    | COSTS-SHEE        |
      | figures.ods     | 2     | 1    | REVENUE-SH        |
      | figures.ods     | 2     | 2    | COSTS-SHEE        |
      | figures.xls     | 2     | 1    | REVENUE-SH        |
      | figures.xls     | 2     | 2    | COSTS-SHEE        |
      | figures-lo.xlsx | 2     | 1    | REVENUE-SH        |
      | figures-lo.xlsx | 2     | 2    | COSTS-SHEE        |
      | pitch.fodp      | 4     | 1    | PITCH-1           |
      | pitch.fodp      | 4     | 4    | PITCH-4           |
      | pitch.odp       | 4     | 1    | PITCH-1           |
      | pitch.odp       | 4     | 4    | PITCH-4           |
      | pitch.ppt       | 4     | 1    | PITCH-1           |
      | pitch.ppt       | 4     | 4    | PITCH-4           |
      | pitch-lo.pptx   | 4     | 1    | PITCH-1           |
      | pitch-lo.pptx   | 4     | 4    | PITCH-4           |
      | letter.rtf      | 1     | 1    | Northwind Traders |
      | letter.rtf      | 1     | 1    | Second item       |
      | page.htm        | 1     | 1    | HTM-MARKER        |
      | data.xml        | 1     | 1    | XML-MARKER        |
      | data.xml        | 1     | 1    | Second            |

  @corpus-libreoffice
  Scenario Outline: Image formats convert to a one-page PDF
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/images/<file> | file   |
      | Gotenberg-Output-Filename | foo                                  | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have 1 image(s)

    Examples:
      | file        |
      | photo.jpg   |
      | diagram.png |
      | icon.gif    |
      | scan.tif    |
      | pic.bmp     |

  @corpus-libreoffice
  Scenario: SVG converts to a one-page PDF
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/office/images/logo.svg | file   |
      | Gotenberg-Output-Filename | foo                                    | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)

  @corpus-pdf
  Scenario: Report PDF: info and metadata
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/corpus/pdf/report.pdf | file |
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
      | files | testdata/corpus/pdf/report.pdf | file |
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
      | files                     | testdata/corpus/pdf/report.pdf | file   |
      | Gotenberg-Output-Filename | foo                            | header |
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

  @corpus-pdf
  Scenario: Redaction removes the label of page 1 only
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/redact" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/report.pdf                      | file   |
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
  @corpus-pdf
  Scenario: Merge report and annex
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/report.pdf | file   |
      | files                     | testdata/corpus/pdf/annex.pdf  | file   |
      | Gotenberg-Output-Filename | foo                            | header |
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

  @corpus-pdf
  Scenario: Stamp text and image
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-text" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/report.pdf | file   |
      | text                      | APPROVED-STAMP                 | field  |
      | pages                     | 1                              | field  |
      | x                         | 72                             | field  |
      | y                         | 200                            | field  |
      | fontSize                  | 24                             | field  |
      | Gotenberg-Output-Filename | foo                            | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      REPORT-PAGE-1
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-image" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/report.pdf | file   |
      | files                     | testdata/corpus/pdf/stamp.jpg  | file   |
      | image                     | stamp.jpg                      | field  |
      | pages                     | 1                              | field  |
      | width                     | 150                            | field  |
      | height                    | 50                             | field  |
      | Gotenberg-Output-Filename | foo                            | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have 1 image(s)

  @corpus-pdf
  Scenario: Letterhead stamped under the report
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/stamp-pdf" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/report.pdf     | file   |
      | files                     | testdata/corpus/pdf/letterhead.pdf | file   |
      | source                    | letterhead.pdf                     | field  |
      | Gotenberg-Output-Filename | foo                                | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      REPORT-PAGE-1
      """

  @corpus-pdf
  Scenario: Encrypted PDF needs its password
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/corpus/pdf/encrypted.pdf | file |
    Then the response status code should be 400
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files    | testdata/corpus/pdf/encrypted.pdf | file  |
      | password | secret                            | field |
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
      | files | testdata/corpus/pdf/encrypted.pdf | file |
    Then the response status code should be 400
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/encrypted.pdf | file   |
      | password                  | secret                            | field  |
      | Gotenberg-Output-Filename | foo                               | header |
    Then the response status code should be 200
    Then the response PDF(s) should NOT be encrypted
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      REPORT-PAGE-3
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/decrypt" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/encrypted-rc4.pdf | file   |
      | password                  | secret                                | field  |
      | Gotenberg-Output-Filename | foo                                   | header |
    Then the response status code should be 200
    Then the response PDF(s) should NOT be encrypted
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      REPORT-PAGE-2
      """

  @corpus-pdf
  Scenario: Factur-X embeds the CII invoice
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/report.pdf      | file   |
      | files                     | testdata/corpus/pdf/invoice-cii.xml | file   |
      | profile                   | Factur-X                            | field  |
      | xml                       | invoice-cii.xml                     | field  |
      | level                     | EN 16931                            | field  |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the response header "Gotenberg-Remediation-Residual" should be "0"
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Quarterly Report"
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "EN 16931"

  @corpus-pdf
  Scenario: Transparency remediates to PDF/A-2b
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | testdata/corpus/pdf/transparent.pdf | file   |
      | profile                   | PDF/A-2b                            | field  |
      | Gotenberg-Output-Filename | foo                                 | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      TRANSPARENT-MARKER
      """
    Then the response PDF(s) should be valid "PDF/A-2b" with a tolerance of 0 failed rule(s)

  @corpus-pdf
  Scenario: Color and image-heavy samples read back
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/info" endpoint with the following form data and header(s):
      | files | testdata/corpus/pdf/colors.pdf      | file |
      | files | testdata/corpus/pdf/image-heavy.pdf | file |
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
