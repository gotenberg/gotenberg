@output-filename
Feature: Output Filename

  Scenario: Default (Single Output File)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |

  Scenario: Default (Many Output Files)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | files                     | testdata/page_2.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/zip"
    Then there should be the following file(s) in the response:
      | foo.zip |

  # RFC 6266 and RFC 8187: a non-ASCII name travels in filename*, beside an
  # ASCII fallback in filename.
  Scenario: Non-ASCII Filename
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | é日本               | header |
    Then the response status code should be 200
    Then the response header "Content-Disposition" should contain "___.pdf"
    Then the response header "Content-Disposition" should contain "filename*=UTF-8''%C3%A9%E6%97%A5%E6%9C%AC.pdf"
    Then there should be the following file(s) in the response:
      | é日本.pdf |

  Scenario: ASCII Filename
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | foo                 | header |
    Then the response status code should be 200
    Then the response header "Content-Disposition" should contain "foo.pdf"

  # The extension of the name is kept: Gotenberg adds the one of the output.
  Scenario: Filename With An Extension
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | report.pdf          | header |
    Then the response status code should be 200
    Then there should be the following file(s) in the response:
      | report.pdf.pdf |

  # See GHSA-hwc4-gmrw-5222.
  Scenario: Windows Path As Filename
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf    | file   |
      | Gotenberg-Output-Filename | C:\\Windows\\Temp\\foo | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |

  # See https://github.com/gotenberg/gotenberg/issues/1227.
  Scenario: Path As Filename
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/pdf/flatten-form" endpoint with the following form data and header(s):
      | files                     | testdata/page_1.pdf | file   |
      | Gotenberg-Output-Filename | /tmp/foo            | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be the following file(s) in the response:
      | foo.pdf |
