@libreoffice
@libreoffice-convert
Feature: /forms/libreoffice/convert

  Scenario: POST /forms/libreoffice/convert (Single Document)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Version" should be "{version}"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  # LibreOffice detects OOXML PowerPoint shows from content and converts them
  # like .pptx; only Gotenberg's extension allow list gated them out.
  # See https://github.com/gotenberg/gotenberg/pull/1626.
  Scenario: POST /forms/libreoffice/convert (PowerPoint Show .ppsx)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/slideshow.ppsx | file   |
      | Gotenberg-Output-Filename | foo                     | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)

  Scenario: POST /forms/libreoffice/convert (PowerPoint Show .ppsm)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/slideshow.ppsm | file   |
      | Gotenberg-Output-Filename | foo                     | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)

  # A CSV becomes a single Calc sheet named after the input file, and Calc's
  # default page style prints that sheet name as a centered header. Uploads are
  # stored under a UUID-based filename, so the UUID must not leak into the PDF.
  # See https://github.com/gotenberg/gotenberg/issues/1568.
  Scenario: POST /forms/libreoffice/convert (CSV Without Sheet Name Header)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/sheet.csv | file   |
      | Gotenberg-Output-Filename | foo                | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Alice
      """
    Then the "foo.pdf" PDF should NOT have content matching "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" at page 1

  Scenario: POST /forms/libreoffice/convert (Many Documents)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | files                     | testdata/page_2.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.zip         |
      | page_1.docx.pdf |
      | page_2.docx.pdf |
    Then the "page_1.docx.pdf" PDF should have 1 page(s)
    Then the "page_2.docx.pdf" PDF should have 1 page(s)
    Then the "page_1.docx.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "page_2.docx.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """

  # See:
  # https://github.com/gotenberg/gotenberg/issues/104
  # https://github.com/gotenberg/gotenberg/issues/730
  Scenario: POST /forms/libreoffice/convert (Non-basic Latin Characters)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/Special_Chars_ß.docx | file   |
      | Gotenberg-Output-Filename | foo                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  Scenario: POST /forms/libreoffice/convert (Protected)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/protected_page_1.docx | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The document 'protected_page_1.docx' is password-protected. Provide its password in the 'password' form field.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files    | testdata/protected_page_1.docx | file  |
      | password | foo                            | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/libreoffice/convert (Landscape)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should NOT be set to landscape orientation
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | landscape                 | true                 | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should be set to landscape orientation

  Scenario: POST /forms/libreoffice/convert (Tagged PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should NOT be tagged
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | nativeTaggedPdf           | true                 | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should be tagged

  Scenario: POST /forms/libreoffice/convert (Native Page Ranges - Single Document)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.docx | file   |
      | nativePages               | 2-3                   | field  |
      | Gotenberg-Output-Filename | foo                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/libreoffice/convert (Native Page Ranges - Many Documents)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.docx  | file   |
      | files                     | testdata/pages_12.docx | file   |
      | nativePages               | 2-3                    | field  |
      | Gotenberg-Output-Filename | foo                    | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.zip           |
      | pages_3.docx.pdf  |
      | pages_12.docx.pdf |
    Then the "pages_3.docx.pdf" PDF should have 2 page(s)
    Then the "pages_12.docx.pdf" PDF should have 2 page(s)
    Then the "pages_3.docx.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3.docx.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """
    Then the "pages_12.docx.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_12.docx.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/libreoffice/convert (Paper Size)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | paperSize                 | A4                   | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF first page should be 595.3x841.9 points
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | paperSize                 | A5                   | field  |
      | landscape                 | true                 | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF first page should be 595.3x419.5 points
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | paperWidth                | 4in                  | field  |
      | paperHeight               | 6in                  | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF first page should be 288x432 points

  Scenario: POST /forms/libreoffice/convert (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | landscape                       | foo | field |
      | exportFormFields                | foo | field |
      | allowDuplicateFieldNames        | foo | field |
      | nativeBookmarks                 | foo | field |
      | exportBookmarksToPdfDestination | foo | field |
      | exportPlaceholders              | foo | field |
      | exportNotes                     | foo | field |
      | exportNotesPages                | foo | field |
      | exportOnlyNotesPages            | foo | field |
      | exportNotesInMargin             | foo | field |
      | convertOooTargetToPdfTarget     | foo | field |
      | exportLinksRelativeFsys         | foo | field |
      | exportHiddenSlides              | foo | field |
      | skipEmptyPages                  | foo | field |
      | addOriginalDocumentAsStream     | foo | field |
      | singlePage                      | foo | field |
      | nativeTaggedPdf                 | foo | field |
      | initialView                     | foo | field |
      | initialPage                     | -1  | field |
      | magnification                   | foo | field |
      | zoom                            | -1  | field |
      | pageLayout                      | foo | field |
      | firstPageOnLeft                 | foo | field |
      | resizeWindowToInitialPage       | foo | field |
      | centerWindow                    | foo | field |
      | openInFullScreenMode            | foo | field |
      | displayPDFDocumentTitle         | foo | field |
      | hideViewerMenubar               | foo | field |
      | hideViewerToolbar               | foo | field |
      | hideViewerWindowControls        | foo | field |
      | useTransitionEffects            | foo | field |
      | openBookmarkLevels              | 15  | field |
      | nativeLosslessImageCompression  | foo | field |
      | nativeQuality                   | -1  | field |
      | nativeReduceImageResolution     | foo | field |
      | nativeMaxImageResolution        | 10  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response header "Gotenberg-Version" should be "{version}"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.123 .602 .abw .bib .bmp .cdr .cgm .cmx .csv .cwk .dbf .dif .doc .docm .docx .dot .dotm .dotx .dxf .emf .eps .epub .fodg .fodp .fods .fodt .fopd .gif .htm .html .hwp .jpeg .jpg .key .ltx .lwp .mcw .met .mml .mw .numbers .odd .odg .odm .odp .ods .odt .otg .oth .otp .ots .ott .pages .pbm .pcd .pct .pcx .pdb .pdf .pgm .png .pot .potm .potx .ppm .pps .ppsm .ppsx .ppt .pptm .pptx .psd .psw .pub .pwp .pxl .ras .rtf .sda .sdc .sdd .sdp .sdw .sgl .slk .smf .stc .std .sti .stw .svg .svm .swf .sxc .sxd .sxg .sxi .sxm .sxw .tga .tif .tiff .txt .uof .uop .uos .uot .vdx .vor .vsd .vsdm .vsdx .wb2 .wk1 .wks .wmf .wpd .wpg .wps .xbm .xhtml .xls .xlsb .xlsm .xlsx .xlt .xltm .xltx .xlw .xml .xpm .zabw]
      form field 'landscape' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportFormFields' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'allowDuplicateFieldNames' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'nativeBookmarks' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportBookmarksToPdfDestination' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportPlaceholders' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportNotes' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportNotesPages' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportOnlyNotesPages' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportNotesInMargin' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'convertOooTargetToPdfTarget' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportLinksRelativeFsys' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'exportHiddenSlides' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'skipEmptyPages' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'addOriginalDocumentAsStream' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'singlePage' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'nativeTaggedPdf' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'initialView' is invalid (got 'foo', resulting to must be one of default, outline, thumbnails, got "foo")
      form field 'initialPage' is invalid (got '-1', resulting to value is inferior to 1)
      form field 'magnification' is invalid (got 'foo', resulting to must be one of default, fitWindow, fitWidth, fitVisible, zoom, got "foo")
      form field 'zoom' is invalid (got '-1', resulting to value is inferior to 1)
      form field 'pageLayout' is invalid (got 'foo', resulting to must be one of default, singlePage, oneColumn, twoColumnLeft, twoColumnRight, got "foo")
      form field 'firstPageOnLeft' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'resizeWindowToInitialPage' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'centerWindow' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'openInFullScreenMode' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'displayPDFDocumentTitle' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'hideViewerMenubar' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'hideViewerToolbar' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'hideViewerWindowControls' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'useTransitionEffects' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'openBookmarkLevels' is invalid (got '15', resulting to value is not -1 or between 1 and 10)
      form field 'nativeLosslessImageCompression' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'nativeQuality' is invalid (got '-1', resulting to value is inferior to 1)
      form field 'nativeReduceImageResolution' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      form field 'nativeMaxImageResolution' is invalid (got '10', resulting to value is not 75, 150, 300, 600 or 1200)
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files       | testdata/page_1.docx | file  |
      | nativePages | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      LibreOffice could not apply the pages 'foo' to the document 'page_1.docx'. Check the 'nativePages' form field; valid values look like '1-4', '2' or '1,3,5-7'.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files    | testdata/protected_page_1.docx | file  |
      | password | bar                            | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      The password for the document 'protected_page_1.docx' is incorrect. Check the 'password' form field.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files      | testdata/page_1.docx | file  |
      | nativePdfa | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'nativePdfa' is invalid (got 'foo', want PDF/A-1b, PDF/A-2b, PDF/A-3b or PDF/A-4).
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files       | testdata/page_1.docx | file  |
      | nativePdfua | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'nativePdfua' is invalid (got 'foo', resulting to must be PDF/UA-1, got "foo")
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files      | testdata/page_1.docx | file  |
      | paperWidth | 8in                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: set 'paperSize', or both 'paperWidth' and 'paperHeight'
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/page_1.docx | file  |
      | paperSize | A0                   | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'paperSize' is invalid (got 'A0', want A3, A4, A5, B4, B5, Letter, Legal or Tabloid)
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx | file  |
      | stages | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stages must be a JSON array of objects: invalid character 'o' in literal false (expecting 'a')
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx | file  |
      | stages | [{"op":"foo"}]       | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: unknown operation "foo"
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.docx | file  |
      | stages | [{"op":"split"}]      | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: split: set "every" or "ranges"
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                     | file  |
      | stages | [{"op":"set-metadata","Title":"Sample"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'stages' is invalid: stage 0: set-metadata: unknown option "Title"
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                         | file  |
      | stages | [{"op":"stamp-image","image":"missing.png"}] | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form file 'missing.png' is required
      """

  # lok ignores a password given for a document that has none.
  Scenario: POST /forms/libreoffice/convert (Password On Unprotected Document)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files    | testdata/page_1.docx | file  |
      | password | foo                  | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  @pdf-merge
  Scenario: POST /forms/libreoffice/convert (Merge)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | files                     | testdata/page_2.docx | file   |
      | stages                    | [{"op":"merge"}]     | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """

  @pdf-merge
  @pdf-split
  Scenario: POST /forms/libreoffice/convert (Merge & Split)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                      | file  |
      | files  | testdata/page_2.docx                      | file  |
      | stages | [{"op":"merge"},{"op":"split","every":1}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.docx_0.pdf |
      | page_1.docx_1.pdf |
    Then the "page_1.docx_0.pdf" PDF should have 1 page(s)
    Then the "page_1.docx_1.pdf" PDF should have 1 page(s)
    Then the "page_1.docx_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "page_1.docx_1.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """

  @pdf-split
  Scenario: POST /forms/libreoffice/convert (Split Every)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.docx      | file  |
      | stages | [{"op":"split","every":2}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.docx_0.pdf |
      | pages_3.docx_1.pdf |
    Then the "pages_3.docx_0.pdf" PDF should have 2 page(s)
    Then the "pages_3.docx_1.pdf" PDF should have 1 page(s)
    Then the "pages_3.docx_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3.docx_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3.docx_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  @pdf-split
  Scenario: POST /forms/libreoffice/convert (Split Ranges)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.docx               | file  |
      | stages | [{"op":"split","ranges":["2","3"]}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.docx_0.pdf |
      | pages_3.docx_1.pdf |
    Then the "pages_3.docx_0.pdf" PDF should have 1 page(s)
    Then the "pages_3.docx_1.pdf" PDF should have 1 page(s)
    Then the "pages_3.docx_0.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3.docx_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  @pdf-pages
  Scenario: POST /forms/libreoffice/convert (Select)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.docx           | file  |
      | stages | [{"op":"select","pages":"2-3"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.docx.pdf |
    Then the "pages_3.docx.pdf" PDF should have 2 page(s)
    Then the "pages_3.docx.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3.docx.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  @pdf-split
  Scenario: POST /forms/libreoffice/convert (Split Many PDFs - Lot of Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/pages_12.docx     | file  |
      | files  | testdata/pages_3.docx      | file  |
      | stages | [{"op":"split","every":1}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 15 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.docx_0.pdf   |
      | pages_3.docx_1.pdf   |
      | pages_3.docx_2.pdf   |
      | pages_12.docx_0.pdf  |
      | pages_12.docx_1.pdf  |
      | pages_12.docx_2.pdf  |
      | pages_12.docx_3.pdf  |
      | pages_12.docx_4.pdf  |
      | pages_12.docx_5.pdf  |
      | pages_12.docx_6.pdf  |
      | pages_12.docx_7.pdf  |
      | pages_12.docx_8.pdf  |
      | pages_12.docx_9.pdf  |
      | pages_12.docx_10.pdf |
      | pages_12.docx_11.pdf |
    Then the "pages_3.docx_0.pdf" PDF should have 1 page(s)
    Then the "pages_3.docx_2.pdf" PDF should have 1 page(s)
    Then the "pages_12.docx_0.pdf" PDF should have 1 page(s)
    Then the "pages_12.docx_11.pdf" PDF should have 1 page(s)
    Then the "pages_3.docx_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3.docx_2.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the "pages_12.docx_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_12.docx_11.pdf" PDF should have the following content at page 1:
      """
      Page 12
      """

  Scenario: POST /forms/libreoffice/convert (Native PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files       | testdata/page_1.docx | file  |
      | nativePdfa  | PDF/A-1b             | field |
      | nativePdfua | PDF/UA-1             | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 3 failed rule(s)

  Scenario: POST /forms/libreoffice/convert (Native PDF/A-4)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files      | testdata/page_1.docx | file  |
      | nativePdfa | PDF/A-4              | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-4" with a tolerance of 0 failed rule(s)

  # Merging builds a new document, which loses the conformance of each one:
  # remediate the merged document instead.
  @pdf-merge
  @pdf-remediate
  Scenario: POST /forms/libreoffice/convert (Merge & Remediate PDF/A-1b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                                     | file  |
      | files  | testdata/page_2.docx                                     | file  |
      | stages | [{"op":"merge"},{"op":"remediate","profile":"PDF/A-1b"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  # Stages after a split run on every part.
  @pdf-split
  @pdf-remediate
  Scenario: POST /forms/libreoffice/convert (Split & Remediate PDF/A-1b)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/pages_3.docx                                              | file  |
      | stages | [{"op":"split","every":2},{"op":"remediate","profile":"PDF/A-1b"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.docx_0.pdf |
      | pages_3.docx_1.pdf |
    Then the "pages_3.docx_0.pdf" PDF should have 2 page(s)
    Then the "pages_3.docx_1.pdf" PDF should have 1 page(s)
    Then the "pages_3.docx_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3.docx_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 0 failed rule(s)

  @pdf-metadata
  Scenario: POST /forms/libreoffice/convert (Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx                                                                                                                                          | file   |
      | stages                    | [{"op":"set-metadata","title":"Sample","author":"Julien Neuhart","subject":"Sample","keywords":"first, second","creator":"Gotenberg","producer":"Gotenberg"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Julien Neuhart"
    Then the "foo.pdf" PDF should have the "Subject" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Keywords" metadata set to "first, second"
    Then the "foo.pdf" PDF should have the "Creator" metadata set to "Gotenberg"
    Then the "foo.pdf" PDF should have the "Producer" metadata set to "Gotenberg"

  @pdf-form
  Scenario: POST /forms/libreoffice/convert (Flatten Form)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx    | file  |
      | stages | [{"op":"flatten-form"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be flatten

  @pdf-encrypt
  Scenario: POST /forms/libreoffice/convert (Encrypt - user password only)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                    | file  |
      | stages | [{"op":"encrypt","userPassword":"foo"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @pdf-encrypt
  Scenario: POST /forms/libreoffice/convert (Encrypt - both user and owner passwords)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                                          | file  |
      | stages | [{"op":"encrypt","userPassword":"foo","ownerPassword":"bar"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be encrypted

  @pdf-stamp
  Scenario: POST /forms/libreoffice/convert (Stamp Text)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx                               | file  |
      | stages | [{"op":"stamp-text","text":"DRAFT","opacity":0.5}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  @pdf-rotate
  Scenario: POST /forms/libreoffice/convert (Rotate 90)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx         | file   |
      | stages                    | [{"op":"rotate","angle":90}] | field  |
      | Gotenberg-Output-Filename | foo                          | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF page 1 should be rotated by 90 degrees

  Scenario: POST /forms/libreoffice/convert (Native Watermark - Text)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | nativeWatermarkText       | CONFIDENTIAL         | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      CONFIDENTIAL
      """

  Scenario: POST /forms/libreoffice/convert (Native Watermark - Text with Options)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                   | testdata/page_1.docx | file  |
      | nativeWatermarkText     | DRAFT                | field |
      | nativeWatermarkColor    | #ff0000              | field |
      | nativeWatermarkFontSize | 48                   | field |
      | nativeWatermarkRotation | 45                   | field |
      | nativeWatermarkFont     | Courier              | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  Scenario: POST /forms/libreoffice/convert (Native Watermark - Tiled)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                    | testdata/page_1.docx | file  |
      | nativeTiledWatermarkText | CONFIDENTIAL         | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response

  # Files a stage reads are uploaded like the documents, and are not converted.
  @pdf-attach
  Scenario: POST /forms/libreoffice/convert (Attach)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx                                                        | file   |
      | files                     | testdata/embed_1.xml                                                        | file   |
      | files                     | testdata/embed_2.xml                                                        | file   |
      | stages                    | [{"op":"attach","file":"embed_1.xml"},{"op":"attach","file":"embed_2.xml"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                         | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the response PDF(s) should have the "embed_1.xml" file embedded
    Then the response PDF(s) should have the "embed_2.xml" file embedded

  # The Factur-X remediation turns the document into PDF/A-3 and embeds the
  # invoice XML as factur-x.xml, with the Alternative relationship.
  @pdf-remediate
  @factur-x
  Scenario: POST /forms/libreoffice/convert (Factur-X / ZUGFeRD)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx                                                             | file   |
      | files                     | testdata/embed_1.xml                                                             | file   |
      | stages                    | [{"op":"remediate","profile":"Factur-X","xml":"embed_1.xml","level":"EN 16931"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                              | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "EN 16931"

  # The base PDF is already PDF/A-3b.
  @pdf-remediate
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X, already PDF/A-3)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | nativePdfa                | PDF/A-3b             | field  |
      | Gotenberg-Output-Filename | base                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | teststore/base.pdf   | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | BASIC                | field  |
      | documentType              | ORDER                | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "BASIC"

  # The base PDF is not PDF/A: the remediation converts it to PDF/A-3b.
  @pdf-remediate
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X, converts non-PDF/A input)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | plain                | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | teststore/plain.pdf  | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | EN 16931             | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "factur-x.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should declare Factur-X XMP with conformance level "EN 16931"

  @pdf-remediate
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X XRECHNUNG)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | nativePdfa                | PDF/A-3b             | field  |
      | Gotenberg-Output-Filename | base                 | header |
    Then the response status code should be 200
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files                     | teststore/base.pdf   | file   |
      | files                     | testdata/embed_1.xml | file   |
      | profile                   | Factur-X             | field  |
      | xml                       | embed_1.xml          | field  |
      | level                     | XRECHNUNG            | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should have the "xrechnung.xml" file embedded with relationship "Alternative"
    Then the response PDF(s) should NOT have the "factur-x.xml" file embedded
    Then the response PDF(s) should declare Factur-X XMP with conformance level "XRECHNUNG"

  @pdf-render
  Scenario: POST /forms/libreoffice/convert (Render)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files  | testdata/page_1.docx              | file  |
      | stages | [{"op":"render","format":"jpeg"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | page_1.docx.pdf    |
      | page_1.docx_1.jpeg |

  # A Factur-X level without the invoice XML is a half-specified request:
  # without the XML, the remediation repairs an existing invoice instead.
  @pdf-remediate
  @factur-x
  Scenario: POST /forms/pdf/remediate (Factur-X Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/remediate" endpoint with the following form data and header(s):
      | files   | testdata/page_1.pdf | file  |
      | profile | Factur-X            | field |
      | level   | EN 16931            | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: remediate: "level" and "documentType" need "xml", the uploaded Factur-X invoice XML to embed; without "xml", the remediation repairs the invoice the PDF already carries
      """

  # Edits run in order on the native PDF/A-3b document, and the remediation
  # comes last, so the output stays PDF/A-3b.
  @pdf-metadata
  @pdf-stamp
  @pdf-rotate
  @pdf-form
  @pdf-attach
  @pdf-remediate
  Scenario: POST /forms/libreoffice/convert (Native PDF/A-3b & Metadata & Stamps & Rotate & Flatten & Attach & Remediate)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx                                                                                                                                                                                                                                                                                                                                                                                                                                | file   |
      | files                     | testdata/embed_1.xml                                                                                                                                                                                                                                                                                                                                                                                                                                | file   |
      | files                     | testdata/embed_2.xml                                                                                                                                                                                                                                                                                                                                                                                                                                | file   |
      | nativePdfa                | PDF/A-3b                                                                                                                                                                                                                                                                                                                                                                                                                                            | field  |
      | stages                    | [{"op":"set-metadata","title":"Sample","author":"Julien Neuhart","subject":"Sample","creator":"Gotenberg","producer":"Gotenberg"},{"op":"stamp-text","text":"CONFIDENTIAL","opacity":0.3},{"op":"stamp-text","text":"DRAFT","y":50},{"op":"rotate","angle":90},{"op":"flatten-form"},{"op":"attach","file":"embed_1.xml","relationship":"Data"},{"op":"attach","file":"embed_2.xml","relationship":"Data"},{"op":"remediate","profile":"PDF/A-3b"}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                                                                                                                                                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the response PDF(s) should be valid "PDF/A-3b" with a tolerance of 0 failed rule(s)
    Then the response PDF(s) should be flatten
    Then the response PDF(s) should have the "embed_1.xml" file embedded with relationship "Data"
    Then the response PDF(s) should have the "embed_2.xml" file embedded with relationship "Data"
    Then the "foo.pdf" PDF page 1 should be rotated by 90 degrees
    Then the "foo.pdf" PDF should have the "Title" metadata set to "Sample"
    Then the "foo.pdf" PDF should have the "Author" metadata set to "Julien Neuhart"

  Scenario: POST /forms/libreoffice/convert (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | LIBREOFFICE_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 404

  Scenario: POST /forms/libreoffice/convert (Enable Macros)
    Given I have a Gotenberg container with the following environment variable(s):
      | LIBREOFFICE_ENABLE_MACROS | true |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the Gotenberg container should log the following entries:
      | --libreoffice-enable-macros (LIBREOFFICE_ENABLE_MACROS) runs the macros of every converted document |

  Scenario: POST /forms/libreoffice/convert (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files           | testdata/page_1.docx      | file   |
      | Gotenberg-Trace | forms_libreoffice_convert | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_libreoffice_convert"
    Then the Gotenberg container should log the following entries:
      | "correlation_id":"forms_libreoffice_convert" |

  @download-from
  Scenario: POST /forms/libreoffice/convert (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"{helper}/static/testdata/page_1.docx","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "application/pdf"

  # The outbound filter refuses a download URL the deny-list names, even when
  # the allow-list matches every URL. A refusal stays generic for the client.
  @download-from
  Scenario: POST /forms/libreoffice/convert (Download From In The Deny-List)
    Given I have a static server
    Given I have a Gotenberg container with the following environment variable(s):
      | API_DOWNLOAD_FROM_DENY_LIST | ^{helper}/static/ |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"{helper}/static/testdata/page_1.docx"}] | field |
    Then the response status code should be 403
    Then the response body should match string:
      """
      Forbidden
      """

  @webhook
  Scenario: POST /forms/libreoffice/convert (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                        | testdata/page_1.docx    | file   |
      | Gotenberg-Output-Filename    | foo                     | header |
      | Gotenberg-Webhook-Url        | {helper}/webhook        | header |
      | Gotenberg-Webhook-Events-Url | {helper}/webhook/events | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then there should be the following file(s) in the webhook request:
      | foo.pdf |
    Then the "foo.pdf" PDF should have 1 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """

  Scenario: POST /forms/libreoffice/convert (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/libreoffice/convert (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  # See: https://github.com/gotenberg/gotenberg/issues/1500.
  Scenario: POST /forms/libreoffice/convert (Long Filename)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/Longitudinell_jämförelse_mellan_laserkirurgi_och_strålbehandling_gällande_röstkvalitet_och_självskattad_kommunikation_upp_till_två_år_efter_tidig_stämbandscancer_i_ett_randomiserat_kontrollerat_försök.docx | file   |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                    | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 page(s)

  # An embedded image is stored inside the document, not linked, so blocking
  # untrusted linked content leaves it untouched. Guards against over-blocking.
  @libreoffice-ssrf
  Scenario: POST /forms/libreoffice/convert (Embedded Image Survives)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/libreoffice-embedded-image.fodt | file   |
      | Gotenberg-Output-Filename | foo                                      | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 1 image(s)

  # An uploaded document always loads from an untrusted location, so LibreOffice
  # refuses to resolve any content it links (absolute file:// path or external
  # URL). Closes the SSRF and local-file-read vector.
  @libreoffice-ssrf
  Scenario: POST /forms/libreoffice/convert (Linked External Resource Blocked)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/libreoffice-linked-external.fodt | file   |
      | Gotenberg-Output-Filename | foo                                       | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have 0 image(s)

  # The workbook was saved scrolled down to row 37. Without the topLeftCell
  # reset, singlePage would start the page there and drop the header rows,
  # so the "Meteor" column header only appears when the whole sheet is rendered.
  # See https://github.com/gotenberg/gotenberg/issues/1222.
  Scenario: POST /forms/libreoffice/convert (singlePage renders a scrolled workbook in full)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/singlepagesheets-scrolled.xlsx | file   |
      | singlePage                | true                                    | field  |
      | Gotenberg-Output-Filename | foo                                     | header |
    Then the response status code should be 200
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have content matching "Meteor" at page 1
