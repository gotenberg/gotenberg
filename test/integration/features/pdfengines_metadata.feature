Feature: /forms/pdfengines/{write|read}

  Scenario: POST /forms/pdfengines/metadata/{write|read} (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
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

  Scenario: POST /forms/pdfengines/metadata/{write|read} (Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files.   | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file  |
      | files    | testdata/page_2.pdf                                                                                                                                                                                                                                                                                       | file  |
      | metadata | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be 2 PDF(s) in the response
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/page_1.pdf | file |
      | files | teststore/page_2.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "page_1.pdf": {
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
        "page_2.pdf": {
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

  Scenario: POST /forms/pdfengines/metadata/write (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'metadata' is required; no form file found for extensions: [.pdf]
      """
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf | file  |
      | metadata | foo                 | field |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: form field 'metadata' is invalid (got 'foo', resulting to unmarshal metadata: invalid character 'o' in literal false (expecting 'a'))
      """

  Scenario: POST /forms/pdfengines/metadata/read (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | Gotenberg-Output-Filename | foo | header |
    Then the response status code should be 400
    Then the response header "Content-Type" should be "text/plain; charset=UTF-8"
    Then the response body should match string:
      """
      Invalid form data: no form file found for extensions: [.pdf]
      """

  Scenario: POST /forms/pdfengines/metadata/write (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file  |
      | metadata | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/metadata/read (Routes Disabled)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_DISABLE_ROUTES | true |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 404

  Scenario: POST /forms/pdfengines/metadata/write (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata        | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Trace | forms_pdfengines_metadata_write                                                                                                                                                                                                                                                                           | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_metadata_write"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_metadata_write" |

  Scenario: POST /forms/pdfengines/metadata/read (Gotenberg Trace)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files           | testdata/page_1.pdf            | file   |
      | Gotenberg-Trace | forms_pdfengines_metadata_read | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response header "Gotenberg-Trace" should be "forms_pdfengines_metadata_read"
    Then the Gotenberg container should log the following entries:
      | "trace":"forms_pdfengines_metadata_read" |

  Scenario: POST /forms/pdfengines/metadata/write (Output Filename - Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |

  Scenario: POST /forms/pdfengines/metadata/write (Output Filename - Many PDFs)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | files                     | testdata/page_2.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | foo.zip    |
      | page_1.pdf |
      | page_2.pdf |

  Scenario: POST /forms/pdfengines/metadata/write (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | downloadFrom              | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}]                                                                                                                                                                                                  | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the file request header "X-Foo" should be "bar"
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"

  Scenario: POST /forms/pdfengines/metadata/read (Download From)
    Given I have a default Gotenberg container
    Given I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/page_1.pdf","extraHttpHeaders":{"X-Foo":"bar"}}] | field |
    Then the file request header "X-Foo" should be "bar"
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"

  Scenario: POST /forms/pdfengines/metadata/write (Webhook)
    Given I have a default Gotenberg container
    Given I have a webhook server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files                       | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata                    | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Webhook-Url       | http://host.docker.internal:%d/webhook                                                                                                                                                                                                                                                                    | header |
      | Gotenberg-Webhook-Error-Url | http://host.docker.internal:%d/webhook/error                                                                                                                                                                                                                                                              | header |
      | Gotenberg-Output-Filename   | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 204
    When I wait for the asynchronous request to the webhook
    Then the webhook request header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the webhook request
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

  Scenario: POST /forms/pdfengines/metadata/write (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files    | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file  |
      | metadata | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field |
    Then the response status code should be 401

  Scenario: POST /forms/pdfengines/metadata/read (Basic Auth)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_BASIC_AUTH             | true |
      | GOTENBERG_API_BASIC_AUTH_USERNAME | foo  |
      | GOTENBERG_API_BASIC_AUTH_PASSWORD | bar  |
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 401

  Scenario: POST /foo/forms/pdfengines/metadata/{write|read} (Root Path)
    Given I have a Gotenberg container with the following environment variable(s):
      | API_ENABLE_DEBUG_ROUTE | true  |
      | API_ROOT_PATH          | /foo/ |
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/metadata/write" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                                                                                                                                                                                                                                       | file   |
      | metadata                  | {"Author":"Julien Neuhart","Copyright":"Julien Neuhart","CreateDate":"2006-09-18T16:27:50-04:00","Creator":"Gotenberg","Keywords":["first","second"],"Marked":true,"ModDate":"2006-09-18T16:27:50-04:00","PDFVersion":1.7,"Producer":"Gotenberg","Subject":"Sample","Title":"Sample","Trapped":"Unknown"} | field  |
      | Gotenberg-Output-Filename | foo                                                                                                                                                                                                                                                                                                       | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    When I make a "POST" request to Gotenberg at the "/foo/forms/pdfengines/metadata/read" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
