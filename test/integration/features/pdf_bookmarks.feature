@pdf
@pdf-bookmarks
Feature: /forms/pdf/{set-bookmarks|bookmarks}

  Scenario: POST /forms/pdf/{set-bookmarks|bookmarks} (Single PDF)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf                                                                  | file   |
      | bookmarks                 | [{"title":"Index","pageNumber":1,"children":[{"title":"Sub-index","pageNumber":1}]}] | field  |
      | Gotenberg-Output-Filename | foo                                                                                  | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "foo.pdf" PDF should have a document outline
    When I make a "POST" request to Gotenberg at the "/forms/pdf/bookmarks" endpoint with the following form data and header(s):
      | files | teststore/foo.pdf | file |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/json"
    Then the response body should match JSON:
      """
      {
        "foo.pdf": [
          {
            "title": "Index",
            "pageNumber": 1,
            "children": [
              {
                "title": "Sub-index",
                "pageNumber": 1
              }
            ]
          }
        ]
      }
      """

  Scenario: POST /forms/pdf/bookmarks (Existing Outline)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/bookmarks" endpoint with the following form data and header(s):
      | files | testdata/page_1_with_bookmarks.pdf | file |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "page_1_with_bookmarks.pdf": [
          {
            "title": "Page 1",
            "pageNumber": 1
          }
        ]
      }
      """

  Scenario: POST /forms/pdf/bookmarks (Empty List)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/bookmarks" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 200
    Then the response body should match JSON:
      """
      {
        "page_1.pdf": []
      }
      """

  # Merging keeps the outline of each document.
  Scenario: POST /forms/pdf/merge (Bookmarks)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/merge" endpoint with the following form data and header(s):
      | files                     | testdata/page_1_with_bookmarks.pdf | file   |
      | files                     | testdata/page_2_with_bookmarks.pdf | file   |
      | Gotenberg-Output-Filename | foo                                | header |
    Then the response status code should be 200
    Then the "foo.pdf" PDF should have 2 page(s)
    Then the "foo.pdf" PDF should have a document outline

  Scenario: POST /forms/pdf/set-bookmarks (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files     | testdata/page_1.pdf | file  |
      | bookmarks | foo                 | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      Invalid form data: form field 'bookmarks' is invalid (got 'foo', want a JSON array)
      """

  Scenario: POST /forms/pdf/set-bookmarks (Invalid Bookmarks)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files | testdata/page_1.pdf | file |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: set-bookmarks: "bookmarks" must list at least one bookmark, such as [{"title":"Intro","pageNumber":1}]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files     | testdata/page_1.pdf | file  |
      | bookmarks | []                  | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: set-bookmarks: "bookmarks" must list at least one bookmark, such as [{"title":"Intro","pageNumber":1}]
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files     | testdata/page_1.pdf                                              | file  |
      | bookmarks | [{"title":"Index","pageNumber":1,"children":[{"pageNumber":1}]}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Invalid form data: set-bookmarks: bookmarks[0].children[0].title is required
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files     | testdata/page_1.pdf                | file  |
      | bookmarks | [{"title":"Index","pageNumber":4}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'set-bookmarks' failed on 'page_1.pdf': bookmarks[0].pageNumber 4 is past the last page 1
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files     | testdata/page_1.pdf                | file  |
      | bookmarks | [{"title":"Index","pageNumber":0}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'set-bookmarks' failed on 'page_1.pdf': bookmarks[0].pageNumber must be a page from 1 to 1, got 0
      """
    When I make a "POST" request to Gotenberg at the "/forms/pdf/set-bookmarks" endpoint with the following form data and header(s):
      | files     | testdata/page_1.pdf                                                            | file  |
      | bookmarks | [{"title":"Index","pageNumber":1,"children":[{"title":"Sub","pageNumber":9}]}] | field |
    Then the response status code should be 400
    Then the response body should match string:
      """
      Stage 'set-bookmarks' failed on 'page_1.pdf': bookmarks[0].children[0].pageNumber 9 is past the last page 1
      """
