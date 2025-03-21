Feature: /forms/pdfengines/split

  Scenario: POST /forms/pdfengines/split (Intervals - Default - pdfcpu)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  Scenario: POST /forms/pdfengines/split (Pages - Default - pdfcpu)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | pages                | field |
      | splitSpan | 2-                   | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 1 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  Scenario: POST /forms/pdfengines/split (Pages & Unify - Default - pdfcpu)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf | file  |
      | splitMode  | pages                | field |
      | splitSpan  | 2-                   | field |
      | splitUnify | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.pdf |
    Then the "pages_3.pdf" PDF should have 2 page(s)
    Then the "pages_3.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/pdfengines/split (Pages & Unify - QPDF)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_SPLIT_ENGINES | qpdf |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf | file  |
      | splitMode  | pages                | field |
      | splitSpan  | 2-z                  | field |
      | splitUnify | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.pdf |
    Then the "pages_3.pdf" PDF should have 2 page(s)
    Then the "pages_3.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/pdfengines/split (Pages & Unify - PDFtk)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_SPLIT_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files      | testdata/pages_3.pdf | file  |
      | splitMode  | pages                | field |
      | splitSpan  | 2-end                | field |
      | splitUnify | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3.pdf |
    Then the "pages_3.pdf" PDF should have 2 page(s)
    Then the "pages_3.pdf" PDF should have the following content at page 1:
      """
      Page 2
      """
    Then the "pages_3.pdf" PDF should have the following content at page 2:
      """
      Page 3
      """

  Scenario: POST /forms/pdfengines/split (Many PDFs - Lot of Pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_12.pdf | file  |
      | files     | testdata/pages_3.pdf  | file  |
      | splitMode | intervals             | field |
      | splitSpan | 1                     | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 15 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf   |
      | pages_3_1.pdf   |
      | pages_3_2.pdf   |
      | pages_12_0.pdf  |
      | pages_12_1.pdf  |
      | pages_12_2.pdf  |
      | pages_12_3.pdf  |
      | pages_12_4.pdf  |
      | pages_12_5.pdf  |
      | pages_12_6.pdf  |
      | pages_12_7.pdf  |
      | pages_12_8.pdf  |
      | pages_12_9.pdf  |
      | pages_12_10.pdf |
      | pages_12_11.pdf |
    Then the "pages_3_0.pdf" PDF should have 1 page(s)
    Then the "pages_3_2.pdf" PDF should have 1 page(s)
    Then the "pages_12_0.pdf" PDF should have 1 page(s)
    Then the "pages_12_11.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_2.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the "pages_12_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_12_11.pdf" PDF should have the following content at page 1:
      """
      Page 12
      """

  Scenario: POST /forms/pdfengines/split (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitMode' is required; form field 'splitSpan' is required; no form file found for extensions: [.pdf]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | foo                  | field |
      | splitSpan | 2                    | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitMode' is invalid (got 'foo', resulting to wrong value, expected either 'intervals' or 'pages')
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'splitSpan' is invalid (got 'foo', resulting to strconv.Atoi: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | pages                | field |
      | splitSpan | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF split mode, while others may have failed to split due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
      | pdfa      | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF format, while others may have failed to convert due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
      | pdfua     | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'pdfua' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
      | metadata  | foo                  | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'metadata' is invalid (got 'foo', resulting to unmarshal metadata: invalid character 'o' in literal false (expecting 'a'))
      """

  Scenario: POST /forms/pdfengines/split (PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
      | pdfa      | PDF/A-1b             | field |
      | pdfua     | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/pdfengines/split (Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf                                                                                                                                                                                                                                                                                      | file  |
      | splitMode | intervals                                                                                                                                                                                                                                                                                                 | field |
      | splitSpan | 2                                                                                                                                                                                                                                                                                                         | field |
      | metadata  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/pages_3_0.pdf | file |
      | files | teststore/pages_3_1.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "pages_3_0.pdf": {
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
        },
        "pages_3_1.pdf": {
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

  Scenario: POST /forms/pdfengines/split (Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
      | flatten   | true                 | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the response PDF(s) should be flatten

  Scenario: POST /forms/pdfengines/split (PDF/A-1b & PDF/UA-1 & Metadata & Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf                                                                                                                                                                                                                                                                                      | file  |
      | splitMode | intervals                                                                                                                                                                                                                                                                                                 | field |
      | splitSpan | 2                                                                                                                                                                                                                                                                                                         | field |
      | pdfa      | PDF/A-1b                                                                                                                                                                                                                                                                                                  | field |
      | pdfua     | true                                                                                                                                                                                                                                                                                                      | field |
      | metadata  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field |
      | flatten   | true                                                                                                                                                                                                                                                                                                      | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    Then there should be the following file(s) in the response:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 7 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)
    Then the response PDF(s) should be flatten
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/pages_3_0.pdf | file |
      | files | teststore/pages_3_1.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "pages_3_0.pdf": {
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
        },
        "pages_3_1.pdf": {
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

  Scenario: POST /forms/pdfengines/split (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/split (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files           | testdata/pages_3.pdf   | file   |
      | splitMode       | intervals              | field  |
      | splitSpan       | 2                      | field  |
      | Gotenberg-Trace | forms_pdfengines_split | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_split"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_split" |

  Scenario: POST /forms/pdfengines/split (Output Filename - Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | splitMode                 | pages                | field  |
      | splitSpan                 | 2-                   | field  |
      | splitUnify                | true                 | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |

  Scenario: POST /forms/pdfengines/split (Output Filename - Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf | file   |
      | splitMode                 | intervals            | field  |
      | splitSpan                 | 2                    | field  |
      | Gotenberg-Output-Filename | foo                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | foo.zip       |
      | pages_3_0.pdf |
      | pages_3_1.pdf |

  Scenario: POST /forms/pdfengines/split (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/pages_3.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
      | splitMode    | intervals                                                                                                 | field |
      | splitSpan    | 2                                                                                                         | field |
    Then the response status code should be 200
    Then the file request header "X-Foo" should be "bar"
    Then the response header "Content-Type" should be "application/zip"

  Scenario: POST /forms/pdfengines/split (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files                       | testdata/pages_3.pdf                         | file   |
      | splitMode                   | intervals                                    | field  |
      | splitSpan                   | 2                                            | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the webhook request
    Then there should be the following file(s) in the webhook request:
      | pages_3_0.pdf |
      | pages_3_1.pdf |
    Then the "pages_3_0.pdf" PDF should have 2 page(s)
    Then the "pages_3_1.pdf" PDF should have 1 page(s)
    Then the "pages_3_0.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "pages_3_0.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
    Then the "pages_3_1.pdf" PDF should have the following content at page 1:
      """
      Page 3
      """

  Scenario: POST /forms/pdfengines/split (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/split (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/split" endpoint with the following form data and header(s):
      | files     | testdata/pages_3.pdf | file  |
      | splitMode | intervals            | field |
      | splitSpan | 2                    | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
