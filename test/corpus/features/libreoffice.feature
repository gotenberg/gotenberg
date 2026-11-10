@libreoffice
Feature: LibreOffice corpus
  Realistic office documents with known contents, described in testdata/MANIFEST.md.
  Each scenario sends a contract, a budget, a deck, a text file or an image through the LibreOffice route.

  Scenario: Contract DOCX (10 pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/contract.docx | file   |
      | nativeBookmarks           | true                          | field  |
      | Gotenberg-Output-Filename | foo                           | header |
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
  Scenario: Budget XLSX (3 sheets)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/budget.xlsx | file   |
      | Gotenberg-Output-Filename | foo                         | header |
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
      | files                     | testdata/office/budget.xlsx | file   |
      | singlePage                | true                        | field  |
      | Gotenberg-Output-Filename | foo                         | header |
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

  Scenario: Deck PPTX without the hidden slide, then with notes pages
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/deck.pptx | file   |
      | Gotenberg-Output-Filename | foo                       | header |
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
      | files                     | testdata/office/deck.pptx | file   |
      | exportNotesPages          | true                      | field  |
      | Gotenberg-Output-Filename | foo                       | header |
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
  Scenario: CSV keeps its columns
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/sales.csv | file   |
      | Gotenberg-Output-Filename | foo                       | header |
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

  Scenario: Tracked changes keep the inserted text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/tracked.docx | file   |
      | Gotenberg-Output-Filename | foo                          | header |
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

  Scenario: Multilingual DOCX
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/i18n.docx | file   |
      | Gotenberg-Output-Filename | foo                       | header |
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

  Scenario: Form ODT keeps its text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/form.odt | file   |
      | Gotenberg-Output-Filename | foo                      | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      FORM-DOC-MARKER
      """

  Scenario: Plain text paginates
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/readme.txt | file   |
      | Gotenberg-Output-Filename | foo                        | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Line 01:
      """
    Then the "foo.pdf" PDF should have the following content at page 3:
      """
      Line 60:
      """
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Café résumé naïve déjà vu
      """

  Scenario: WebP is not an accepted input
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/office/images/pic.webp | file |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: no form file found for extensions:
      """

  Scenario Outline: Office formats convert with the documented pages and text
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/<file> | file   |
      | Gotenberg-Output-Filename | foo                    | header |
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

  Scenario Outline: Image formats convert to a one-page PDF
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/images/<file> | file   |
      | Gotenberg-Output-Filename | foo                           | header |
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

  Scenario: SVG converts to a one-page PDF
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/office/images/logo.svg | file   |
      | Gotenberg-Output-Filename | foo                             | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 1 page(s)
