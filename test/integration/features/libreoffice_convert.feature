Feature: /forms/libreoffice/convert

  Scenario: POST /forms/libreoffice/convert (Single Document)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | Gotenberg-Output-Filename | foo                  | header |
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

  Scenario: POST /forms/libreoffice/convert (Protected)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/protected_page_1.docx | file |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      LibreOffice failed to process a document: a password may be required, or, if one has been given, it is invalid. In any case, the exact cause is uncertain.
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

  Scenario: POST /forms/libreoffice/convert (Native Page Ranges - Single Document)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.docx | file   |
      | nativePageRanges          | 2-3                   | field  |
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
      | nativePageRanges          | 2-3                    | field  |
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

  Scenario: POST /forms/libreoffice/convert (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | landscape                       | foo | field |
      | exportFormFields                | foo | field |
      | allowDuplicateFieldNames        | foo | field |
      | exportBookmarks                 | foo | field |
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
      | singlePageSheets                | foo | field |
      | losslessImageCompression        | foo | field |
      | quality                         | -1  | field |
      | reduceImageResolution           | foo | field |
      | maxImageResolution              | 10  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.123 .602 .abw .bib .bmp .cdr .cgm .cmx .csv .cwk .dbf .dif .doc .docm .docx .dot .dotm .dotx .dxf .emf .eps .epub .fodg .fodp .fods .fodt .fopd .gif .htm .html .hwp .jpeg .jpg .key .ltx .lwp .mcw .met .mml .mw .numbers .odd .odg .odm .odp .ods .odt .otg .oth .otp .ots .ott .pages .pbm .pcd .pct .pcx .pdb .pdf .pgm .png .pot .potm .potx .ppm .pps .ppt .pptm .pptx .psd .psw .pub .pwp .pxl .ras .rtf .sda .sdc .sdd .sdp .sdw .sgl .slk .smf .stc .std .sti .stw .svg .svm .swf .sxc .sxd .sxg .sxi .sxm .sxw .tga .tif .tiff .txt .uof .uop .uos .uot .vdx .vor .vsd .vsdm .vsdx .wb2 .wk1 .wks .wmf .wpd .wpg .wps .xbm .xhtml .xls .xlsb .xlsm .xlsx .xlt .xltm .xltx .xlw .xml .xpm .zabw]; form field 'landscape' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportFormFields' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'allowDuplicateFieldNames' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportBookmarks' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportBookmarksToPdfDestination' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportPlaceholders' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportNotes' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportNotesPages' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportOnlyNotesPages' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportNotesInMargin' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'convertOooTargetToPdfTarget' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportLinksRelativeFsys' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'exportHiddenSlides' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'skipEmptyPages' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'addOriginalDocumentAsStream' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'singlePageSheets' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'losslessImageCompression' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'quality' is invalid (got '-1', resulting to value is inferior to 1); form field 'reduceImageResolution' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax); form field 'maxImageResolution' is invalid (got '10', resulting to value is not 75, 150, 300, 600 or 1200)
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files            | testdata/page_1.docx | file  |
      | nativePageRanges | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      LibreOffice failed to process a document: possible causes include malformed page ranges 'foo' (nativePageRanges), or, if a password has been provided, it may not be required. In any case, the exact cause is uncertain.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files    | testdata/page_1.docx | file  |
      | password | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      LibreOffice failed to process a document: possible causes include malformed page ranges '' (nativePageRanges), or, if a password has been provided, it may not be required. In any case, the exact cause is uncertain.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files    | testdata/protected_page_1.docx | file  |
      | password | bar                            | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      LibreOffice failed to process a document: a password may be required, or, if one has been given, it is invalid. In any case, the exact cause is uncertain.
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file  |
      | files | testdata/page_2.docx | file  |
      | merge | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'merge' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.docx | file  |
      | splitMode | foo                   | field |
      | splitSpan | 2                     | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitMode' is invalid (got 'foo', resulting to wrong value, expected either 'intervals' or 'pages')
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.docx | file  |
      | splitMode | intervals             | field |
      | splitSpan | foo                   | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitSpan' is invalid (got 'foo', resulting to strconv.Atoi: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.docx | file  |
      | splitMode | pages                 | field |
      | splitSpan | foo                   | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF split mode, while others may have failed to split due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/pages_3.docx | file  |
      | pdfa  | foo                   | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      A PDF format in '{PdfA:foo PdfUa:false}' is not supported
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file  |
      | pdfua | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'pdfua' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files    | testdata/page_1.docx | file  |
      | metadata | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'metadata' is invalid (got 'foo', resulting to unmarshal metadata: invalid character 'o' in literal false (expecting 'a'))
      """

  Scenario: POST /forms/libreoffice/convert (Merge)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx | file   |
      | files                     | testdata/page_2.docx | file   |
      | merge                     | true                 | field  |
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

  Scenario: POST /forms/libreoffice/convert (Merge & Split)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/page_1.docx | file  |
      | files     | testdata/page_2.docx | file  |
      | merge     | true                 | field |
      | splitMode | intervals            | field |
      | splitSpan | 1                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | *_0.pdf |
      | *_1.pdf |
    Then the "*_0.pdf" PDF should have 1 page(s)
    Then the "*_1.pdf" PDF should have 1 page(s)
    Then the "*_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "*_1.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """

  Scenario: POST /forms/libreoffice/convert (Split Intervals)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.docx | file  |
      | splitMode | intervals             | field |
      | splitSpan | 2                     | field |
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

  Scenario: POST /forms/libreoffice/convert (Split Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.docx | file  |
      | splitMode | pages                 | field |
      | splitSpan | 2-                    | field |
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

  Scenario: POST /forms/libreoffice/convert (Split Pages & Unify)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.docx | file  |
      | splitMode  | pages                 | field |
      | splitSpan  | 2-                    | field |
      | splitUnify | true                  | field |
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

  Scenario: POST /forms/libreoffice/convert (Split Many PDFs - Lot of Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_12.docx | file  |
      | files     | testdata/pages_3.docx  | file  |
      | splitMode | intervals              | field |
      | splitSpan | 1                      | field |
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

  Scenario: POST /forms/libreoffice/convert (PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file  |
      | pdfa  | PDF/A-1b             | field |
      | pdfua | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/libreoffice/convert (Split & PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.docx | file  |
      | splitMode | intervals             | field |
      | splitSpan | 2                     | field |
      | pdfa      | PDF/A-1b              | field |
      | pdfua     | true                  | field |
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
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/libreoffice/convert (Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx                                                                                                                                                                                                                                                                                      | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "Author": "Julien Neuhart",
          "Copyright": "Julien Neuhart",
          "CreateDate": "2006:09:18 16:27:50-04:00",
          "Creator": "Gotenberg",
          "Keywords": ["first", "second"],
          "Marked": true,
          "ModDate": "2006:09:18 16:27:50-04:00",
          "PDFVersion": 1.7,
          "Producer": "Gotenberg",
          "Subject": "Sample",
          "Title": "Sample",
          "Trapped": "Unknown"
        }
      }
      """

  Scenario: POST /forms/libreoffice/convert (Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files   | testdata/page_1.docx | file  |
      | flatten | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the response PDF(s) should be flatten

  Scenario: POST /forms/libreoffice/convert (PDF/A-1b & PDF/UA-1 & Metadata & Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.docx                                                                                                                                                                                                                                                                                      | file   |
      | pdfa                      | PDF/A-1b                                                                                                                                                                                                                                                                                                  | field  |
      | pdfua                     | true                                                                                                                                                                                                                                                                                                      | field  |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | flatten                   | true                                                                                                                                                                                                                                                                                                      | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | foo.pdf |
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 7 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)
    Then the response PDF(s) should be flatten
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": {
          "Author": "Julien Neuhart",
          "Copyright": "Julien Neuhart",
          "CreateDate": "2006:09:18 16:27:50-04:00",
          "Creator": "Gotenberg",
          "Keywords": ["first", "second"],
          "Marked": true,
          "ModDate": "2006:09:18 16:27:50-04:00",
          "PDFVersion": 1.7,
          "Producer": "Gotenberg",
          "Subject": "Sample",
          "Title": "Sample",
          "Trapped": "Unknown"
        }
      }
      """

  Scenario: POST /forms/libreoffice/convert (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | LIBREOFFICE_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files | testdata/page_1.docx | file |
    Then the response status code should be 404

  Scenario: POST /forms/libreoffice/convert (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files           | testdata/page_1.docx      | file   |
      | Gotenberg-Trace | forms_libreoffice_convert | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_libreoffice_convert"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_libreoffice_convert" |

  Scenario: POST /forms/libreoffice/convert (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.docx","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/libreoffice/convert (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/libreoffice/convert" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.docx                         | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
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
