@embed
Feature: /forms/pdfengines/embed

  Scenario: POST /forms/pdfengines/embed
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files  | testdata/page_1.pdf  | file |
      | embeds | testdata/embed_1.xml | file |
      | embeds | testdata/embed_2.xml | file |
      | embeds | testdata/page_2.pdf  | file |
    Then the response status code should be 200
    And the response header "Content-Type" should be "application/pdf"
    And there should be 1 PDF(s) in the response
    And there should be the following file(s) in the response:
      | page_1.pdf |
    And the "page_1.pdf" PDF should have the "embed_1.xml" file embedded in it
    And the "page_1.pdf" PDF should have the "embed_2.xml" file embedded in it
    And the "page_1.pdf" PDF should have the "page_2.pdf" file embedded in it

  Scenario: POST /forms/pdfengines/embed with (Download From)
    Given I have a default Gotenberg container
    And I have a static server
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/embed" endpoint with the following form data and header(s):
      | files        | testdata/page_1.pdf                                                                                                                                                              | file  |
      | downloadFrom | [{"url":"http://host.docker.internal:%d/static/testdata/embed_1.xml", "embedded": true},{"url":"http://host.docker.internal:%d/static/testdata/embed_2.xml", "embedded": false}] | field |
    Then the response status code should be 200
    And the response header "Content-Type" should be "application/pdf"
    And there should be 1 PDF(s) in the response
    And there should be the following file(s) in the response:
      | page_1.pdf |
    And the "page_1.pdf" PDF should have the "embed_1.xml" file embedded in it
    And the "page_1.pdf" PDF should NOT have the "embed_2.xml" file embedded in it
