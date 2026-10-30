@pdf
@pdf-redact
Feature: /forms/pdf/redact

  Scenario: POST /forms/pdf/redact
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/redact" endpoint with the following form data and header(s):
      | files                     | testdata/pages_3.pdf                                | file   |
      | regions                   | [{"page":1,"x":0,"y":0,"width":2000,"height":2000}] | field  |
      | Gotenberg-Output-Filename | foo                                                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then the "foo.pdf" PDF should have 3 page(s)
    Then the "foo.pdf" PDF should NOT have the following content at page 1:
      """
      Page 1
      """
    Then the "foo.pdf" PDF should have the following content at page 2:
      """
      Page 2
      """

  Scenario: POST /forms/pdf/redact (Bad Request)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/redact" endpoint with the following form data and header(s):
      | files   | testdata/pages_3.pdf                            | file  |
      | regions | [{"page":0,"x":0,"y":0,"width":10,"height":10}] | field |
    Then the response status code should be 400
    Then the response body should contain string:
      """
      region 0 needs a "page" of 1 or more and a positive "width" and "height"
      """
