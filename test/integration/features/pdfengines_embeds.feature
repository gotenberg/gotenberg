Feature: /forms/pdfengines/merge

  Scenario: Embed multiple files (pdfcpu)
    Given I have a Gotenberg container with the following environment variable(s):
      | PDFENGINES_MERGE_ENGINES | pdfcpu |

    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html-embeds/index.html       | file   |
      | embeds                    | testdata/page-1-html-embeds/embed_1.xml      | file   |
      | embeds                    | testdata/page-1-html-embeds/embed_2.xml      | file   |
      | Gotenberg-Output-Filename | foo                                          | header |
    Then the response status code should be 200
    And the response header "Content-Type" should be "application/pdf"
    And there should be 1 PDF(s) in the response
    And there should be the following file(s) in the response:
      | foo.pdf |
    And the "foo.pdf" PDF should have 1 page(s)
    And the "foo.pdf" PDF should have the following content at page 1:
      """
      Page 1 with embeds
      """
    And the "foo.pdf" PDF should have the "embed_1.xml" file embedded in it
    And the "foo.pdf" PDF should have the "embed_2.xml" file embedded in it