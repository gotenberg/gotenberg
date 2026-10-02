@chromium
@chromium-convert-html
@chromium-image-deduplication
Feature: Chromium header and footer image deduplication

  # See: https://github.com/gotenberg/gotenberg/issues/1077.
  # See: https://github.com/gotenberg/gotenberg/issues/1669.
  # The PNG must appear on every page, but its image XObject must be shared.
  Scenario: POST /forms/chromium/convert/html (Header image reused across pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html          | file   |
      | files                     | testdata/header-footer-image/header.html | file   |
      | marginTop                 | 1                                        | field  |
      | marginBottom              | 1                                        | field  |
      | Gotenberg-Output-Filename | header                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "header.pdf" PDF should have 1 page(s)
    Then the "header.pdf" PDF should reuse 1 image XObject(s) across 1 page(s)
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html         | file   |
      | files                     | testdata/header-footer-image/header.html | file   |
      | marginTop                 | 1                                        | field  |
      | marginBottom              | 1                                        | field  |
      | Gotenberg-Output-Filename | header                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "header.pdf" PDF should have 3 page(s)
    Then the "header.pdf" PDF should reuse 1 image XObject(s) across 3 page(s)

  Scenario: POST /forms/chromium/convert/html (Footer image reused across pages)
    Given I have a default Gotenberg container
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/page-1-html/index.html          | file   |
      | files                     | testdata/header-footer-image/footer.html | file   |
      | marginTop                 | 1                                        | field  |
      | marginBottom              | 1                                        | field  |
      | Gotenberg-Output-Filename | footer                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "footer.pdf" PDF should have 1 page(s)
    Then the "footer.pdf" PDF should reuse 1 image XObject(s) across 1 page(s)
    When I make a "POST" request to Gotenberg at the "/forms/chromium/convert/html" endpoint with the following form data and header(s):
      | files                     | testdata/pages-3-html/index.html         | file   |
      | files                     | testdata/header-footer-image/footer.html | file   |
      | marginTop                 | 1                                        | field  |
      | marginBottom              | 1                                        | field  |
      | Gotenberg-Output-Filename | footer                                   | header |
    Then the response status code should be 200
    Then the response header "Content-Type" should be "application/pdf"
    Then there should be 1 PDF(s) in the response
    Then the "footer.pdf" PDF should have 3 page(s)
    Then the "footer.pdf" PDF should reuse 1 image XObject(s) across 3 page(s)
