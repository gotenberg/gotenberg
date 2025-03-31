Feature: /forms/pdfengines/merge

  Scenario: POST /forms/pdfengines/merge (default - QPDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
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

  Scenario: POST /forms/pdfengines/merge (pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_MERGE_ENGINES | pdfcpu |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
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

  Scenario: POST /forms/pdfengines/merge (PDFtk)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_MERGE_ENGINES | pdftk |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
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

  Scenario: POST /forms/pdfengines/merge (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | files | testdata/page_2.pdf | file  |
      | pdfa  | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      At least one PDF engine cannot process the requested PDF format, while others may have failed to convert due to different issues
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file  |
      | files | testdata/page_2.pdf | file  |
      | pdfua | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'pdfua' is invalid (got 'foo', resulting to strconv.ParseBool: parsing "foo": invalid syntax)
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf | file  |
      | files    | testdata/page_2.pdf | file  |
      | metadata | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'metadata' is invalid (got 'foo', resulting to unmarshal metadata: invalid character 'o' in literal false (expecting 'a'))
      """

  Scenario: POST /forms/pdfengines/merge (PDF/A-1b & PDF/UA-1)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | pdfa                      | PDF/A-1b            | field  |
      | pdfua                     | true                | field  |
      | Gotenberg-Output-Filename | foo                 | header |
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
    Then the response PDF(s) should be valid "PDF/A-1b" with a tolerance of 1 failed rule(s)
    Then the response PDF(s) should be valid "PDF/UA-1" with a tolerance of 2 failed rule(s)

  Scenario: POST /forms/pdfengines/merge (Metadata)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | files                     | testdata/page_2.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
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

  Scenario: POST /forms/pdfengines/merge (Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | flatten                   | true                | field  |
      | Gotenberg-Output-Filename | foo                 | header |
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
    Then the response PDF(s) should be flatten

  Scenario: POST /forms/pdfengines/merge (PDF/A-1b & PDF/UA-1 & Metadata & Flatten)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | files                     | testdata/page_2.pdf                                                                                                                                                                                                                                                                                       | file   |
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
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """
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

  Scenario: POST /forms/pdfengines/merge (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
      | files | testdata/page_2.pdf | file |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/merge (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf    | file   |
      | files           | testdata/page_2.pdf    | file   |
      | Gotenberg-Trace | forms_pdfengines_merge | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_merge"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_merge" |

  Scenario: POST /forms/pdfengines/merge (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf"},{"url":"http://host.docker.internal:%d/static/testdata/page_2.pdf"}] | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/pdfengines/merge (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                          | file   |
      | files                       | testdata/page_2.pdf                          | file   |
      | Gotenberg-Output-Filename   | foo                                          | header |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook       | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
    Then there should be the following file(s) in the webhook request:
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

  Scenario: POST /forms/pdfengines/merge (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
      | files | testdata/page_2.pdf | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/merge (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/merge" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
      | files | testdata/page_2.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
