Feature: Output Filename

  Scenario: Default (Single Output File)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/flatten" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |

  Scenario: Default (Many Output Files)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/flatten" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | foo.zip |

  # See https://github.com/gotenberg/gotenberg/issues/1227.
  Scenario: Path As Filename
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdfengines/flatten" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | /tmp/foo            | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |
