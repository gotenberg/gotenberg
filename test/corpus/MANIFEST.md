# Gotenberg v9 QA corpus manifest

Root: `test/corpus/testdata`; the file paths below are relative to it. Generators in `generators/` (see its README). Page counts marked "(LO)" were measured by converting with the LibreOffice in the gotenberg:snapshot image and `pdfinfo`; "expected" counts were not measured.

## web/ (Chromium routes)

| file | format | what it contains | assertable facts |
|---|---|---|---|
| web/invoice/index.html (+ style.css, logo.png, header.html, footer.html) | HTML | Invoice, `lang="en"`, `<title>Invoice INV-2026-0042</title>` | Northwind Traders (seller, h3), "Bill to: Contoso Ltd" (h3), 12 line items in a `<thead>` table (th background `#143272`, so printBackground matters), subtotal 3,850.00 EUR, VAT 20% 770.00 EUR, total "4,620.00 EUR"; 7 headings: 1 h1 ("Invoice INV-2026-0042"), 3 h2 (Line items, Totals, Payment terms), 3 h3 (Northwind Traders, Bill to, Bank details); `<img alt="Northwind logo">`; CSS `@page{size:A4;margin:15mm}` and `h2.terms{break-before:page}` -> expected 2 A4 pages (A4 = 595.28 x 841.89 pt); page 2 starts with "Payment terms" and contains "IBAN FR76 3000 6000 0112 3456 7890 189" |
| web/invoice/logo.png | PNG | 240x80 PIL, text NORTHWIND | 240x80, 3063 bytes |
| web/invoice/header.html | HTML | Chromium header template, font-size 9px, classes `title`, `date` | title text = "Invoice INV-2026-0042" |
| web/invoice/footer.html | HTML | footer, classes `pageNumber`, `totalPages` | prints "Page 1 of 2", "Page 2 of 2" |
| web/wait/index.html | HTML | JS replaces content after 1000 ms | initial "Loading…"; after 1 s `#chart` = "Revenue: 42,000 EUR", gets class `ready`; `window.chartReady===true`, `window.status==="ready"` |
| web/console-error/index.html | HTML | renders then throws | text "Console error page"; uncaught `Error("boom")` after render |
| web/markdown/index.html (+ release-notes.md, changelog.md, style.css, screenshot.png) | HTML template + MD | `{{ toHTML "release-notes.md" }}` and `{{ toHTML "changelog.md" }}`, `<title>Release notes 9.0.0</title>` | release-notes.md: h1 "Release notes 9.0.0" + h2 Highlights/Code sample/Checklist/Glossary/Links; table (Feature/Status, "Chromium routes", "PDF remediation"); fenced go block containing "CODE-BLOCK-MARKER"; strikethrough "Removed legacy flag"; task list "DONE-TASK" (checked), "OPEN-TASK" (unchecked); definition list (Gotenberg / LibreOffice); bare URL https://gotenberg.dev; image `![Screenshot](screenshot.png)` (640x360); inline `<script>document.title='pwned'</script>` immediately followed by "SANITIZE-ME-MARKER" (script should be stripped/not run, text kept; title must NOT become "pwned"). changelog.md: h1 "Changelog", markers "CHANGELOG-V9", "CHANGELOG-V8" |
| web/landing/index.html | HTML | long landing page, `<title>Landing</title>` | height > 3000 px (8 sections of min-height 320px + 600px hero); `<nav>` ("Home \| Pricing \| Contact") hidden in print; hero linear-gradient; `#cta` button "Start free trial"; "WIDE-LAYOUT-MARKER" visible only at viewport width >= 1200px; body background transparent, only `.hero` and `.box` painted |
| web/i18n/index.html | HTML | multilingual paragraphs, each starts with a Latin marker | markers `AR:` `HE:` (both `dir="rtl"`), `ZH:` `JA:` `KO:` `TH:` `HI:` `EL:` `RU:` `EMOJI:` `MATH:`; MATH line "∑ ∫ ≠ ∞ √ π ≤ ≥"; EMOJI line "😀 🚀 🎉 ❤️ 🌍"; h1 "Multilingual". Note: pdftotext of RTL lines may reorder the marker visually (e.g. ":AR"), search for the Latin marker letters |
| web/print-css/index.html + print.css | HTML + CSS | A5 handbook | `@page{size:A5;margin:15mm}` (A5 = 419.53 x 595.28 pt), `h2{break-before:page}`; `<nav>` (contains "NAV-MARKER") hidden in print; h1 "Handbook"; h2 chapters with markers CHAPTER-ONE, CHAPTER-TWO, CHAPTER-THREE -> expected 4 pages (intro + one per chapter; chapters may overflow to more) |
| web/wide-table/index.html | HTML | table | `<title>Wide table</title>`, 30 columns C01..C30, 40 body rows, cells "r{row}c{col}" (e.g. r40c30), no-wrap so it overflows page width |
| web/site.zip | ZIP | invoice assets | exactly 3 entries, no folders: index.html, style.css, logo.png |

## office/ (LibreOffice route)

| file | format | what it contains | assertable facts |
|---|---|---|---|
| office/contract.docx | DOCX | Master Services Agreement | core props: title "Master Services Agreement", author "Legal Department", subject "Services", keywords "contract, msa"; Title paragraph + 9 Heading 1 (8 numbered sections "1. Definitions" … "8. Governing Law", plus "Schedule A" after a page break), 16 Heading 2 (2 per section, e.g. "1.1 Definitions - sub-clause 1"); 3-column definitions table (Term/Meaning/Section; "Confidential Information"); bulleted list (Consulting/Hosting/Support services); header text "Master Services Agreement"; footer "Page " + PAGE field; logo.png embedded once; 10 pages (LO) |
| office/budget.xlsx | XLSX | 3 sheets | sheets Summary, Monthly, Notes. Summary: merged A1:D1 "FY2026 Budget", rows "Cloud hosting" (120000/130000), "Salaries", "Marketing", SUM formulas (D4 = 250000, B7 = 1010000 etc. after calc); Monthly: 14 columns (Category, Jan..Dec, Total), header + 60 rows, row 2 "Cloud hosting", format `#,##0.00`, bold grey header, freeze panes B2, landscape, print title row 1; Notes: "Budget notes", "Assumptions: 3% inflation", "Prepared by Finance"; 4 pages (LO) |
| office/sales.csv | CSV | UTF-8 | header id,date,customer,product,qty,unit_price,total,comment; 25 data rows, 8 columns; customers "Café Müller", "Straße GmbH", "Élodie & Fils"; comments with quoted commas "Note, with comma #3" (every 3rd row); ISO dates from 2026-01-10 |
| office/deck.pptx | PPTX | 8 slides, 16:9 (13.333 x 7.5 in) | 1 "Q4 Product Review", 2 "Agenda" (notes "NOTES-MARKER-2"), 3 "Two columns", 4 "Table" (R0C0..R2C2), 5 "Bar chart" (series Revenue Q1..Q4 = 10,14,12,18), 6 "Image" (logo.png), 7 "Hidden slide" with "HIDDEN-SLIDE-MARKER" and `show="0"` (hidden), 8 "Thank you" (notes "NOTES-MARKER-8"). LO PDF export = 7 pages, hidden slide absent (verified: 0 occurrences of the marker) |
| office/memo.fodt | Flat ODT | internal memo | meta title "Q4 Memo", creator "Jane Doe"; H1 "Purpose", H1 "Decisions" with H2 "Budget" and "Hiring"; 3x3 table R1C1..R3C3; bold + italic spans; footnote "FOOTNOTE-MARKER text."; header "Northwind internal"; footer "Page <n>"; final text "MEMO-END-MARKER"; 1 page |
| office/memo.odt, memo.doc, memo.rtf, memo-lo.docx, memo.html | converted by LO | same content as memo.fodt | memo.odt meta title "Q4 Memo" |
| office/memo.pdf | PDF (input) | LO export of memo.fodt | 1 page, title "Q4 Memo", text includes "Northwind internal", "MEMO-END-MARKER", "Page 1" |
| office/figures.fods | Flat ODS | 2 sheets | "Revenue": header Month/Amount, dates 2026-01-31 (1000) and 2026-02-28 (2500.5), "REVENUE-SHEET-MARKER" with `=SUM(B2:B3)` cached 3500.5; "Costs": "COSTS-SHEET-MARKER" 700, Total `=SUM(B2:B2)` 700. LO PDF = 2 pages (one per sheet) |
| office/figures.ods, figures.xls, figures-lo.xlsx | converted by LO | same | 2 sheets Revenue, Costs |
| office/pitch.fodp | Flat ODP | 4 slides (28 x 15.75 cm) | titles PITCH-1..PITCH-4, bullets "Bullet A<n>", "Bullet B<n>"; notes on slide 2 "PITCH-NOTES-MARKER-2". LO PDF = 4 pages |
| office/pitch.odp, pitch.ppt, pitch-lo.pptx | converted by LO | same | 4 slides (pitch.ppt is 466 KB) |
| office/form.fodt, form.odt | Flat ODT / ODT | document with real LO form controls | text "FORM-DOC-MARKER"; form "Form1"; controls: text field `fullName`, checkbox `agree` (label "I agree", value "yes"), dropdown list box `country` with options France / Germany / Spain, text area `comments`; 1 page. NOTE: produced by importing an HTML form with LO (HTML (StarWriter) filter) and exporting ODT/FODT, because hand-written `<office:forms>` XML was silently dropped by LO; so form.fodt is LO-generated, not hand-written |
| office/readme.txt | TXT | 60 lines, UTF-8 | lines "Line 01:" .. "Line 60:", each ~270 chars with "Café résumé naïve déjà vu —"; 3 pages (LO) |
| office/letter.rtf | RTF | letter | "Northwind Traders" letterhead (bold), "November 3, 2026", bold "bold" / italic "italic", 2 bullets "First item", "Second item", signature "Jane Doe"; 1 page (LO) |
| office/i18n.docx | DOCX | h1 "Multilingual document" + same 11 paragraphs as web/i18n | markers AR: HE: (bidi paragraphs) ZH: JA: KO: TH: HI: EL: RU: EMOJI: MATH:; 1 page (LO) |
| office/tracked.docx | DOCX | tracked changes | one paragraph "Start of sentence. [ins]INSERTED-TEXT[/ins][del]DELETED-TEXT[/del] End of sentence." `w:ins` author "Jane Doe" 2026-01-15, `w:del` author "John Smith" 2026-01-16; LO renders both texts (1 page) |
| office/images/photo.jpg | JPEG | gradient + circles + big "PHOTO" | 1600x1200 RGB, quality 85, 223,029 bytes |
| office/images/diagram.png | PNG | RGBA with alpha rects/ellipse/polygon, text "DIAGRAM" | 1200x900 RGBA, 12,849 bytes |
| office/images/icon.gif | GIF | palette image | 256x256 mode P, 5,430 bytes |
| office/images/scan.tif | TIFF | grayscale, LZW | 2000x2800 mode L, 60,048 bytes, text "SCAN" |
| office/images/pic.bmp | BMP | text "BMP" | 320x240 RGB, 230,454 bytes |
| office/images/pic.webp | WebP | text "WEBP" | 800x600 RGB, 6,178 bytes |
| office/images/logo.svg | SVG | rect, circle, text | 300x200 viewBox, text "SVG-MARKER", 272 bytes |
| office/page.htm | HTML | h1 "HTM-MARKER", h2 "Section", 2x2 table | 1 page (LO) |
| office/data.xml | XML | catalog | contains "XML-MARKER" (item 1) and "Second"; 1 page (LO) |

## pdf/ (PDF routes)

| file | format | what it contains | assertable facts |
|---|---|---|---|
| pdf/report.pdf | PDF | "Quarterly Report", 3 Letter pages (612x792), uncompressed streams | Title "Quarterly Report", Author "Finance"; Arial TTF embedded (ArialMT subset, only font); each page has body lines "Body line k of page n" and the label "REPORT-PAGE-n" (n=1,2,3), Arial 36 pt, baseline origin x=72, y=400 from bottom-left on every page. Redaction box for the label (bottom-left origin): x=72, y=392.4, width=294.0 (to x=366.05), height=33.8 (to y=426.2); pdftotext bbox (top-left origin) yMin 365.79, yMax 399.58. A safe redaction rect: x=70, y=388, w=300, h=42 |
| pdf/annex.pdf | PDF | 2 pages Letter | "ANNEX-PAGE-1", "ANNEX-PAGE-2" at x=72,y=400 (Helvetica-Bold 36) |
| pdf/letterhead.pdf | PDF | 1 page Letter | blue band 612x90 at top, white text "NORTHWIND LETTERHEAD" |
| pdf/transparent.pdf | PDF | 1 page | three overlapping 50% alpha circles (R,G,B), text "TRANSPARENT-MARKER" |
| pdf/colors.pdf | PDF | 1 page | RGB red/green/blue rects, one CMYK (cyan) rect, one 50% gray rect, text "COLORS-MARKER" |
| pdf/image-heavy.pdf | PDF | 3 pages Letter | each page one full-width photo.jpg (1600x1200 JPEG, embedded once) + "IMAGE-HEAVY-PAGE-n"; 281,662 bytes |
| pdf/encrypted.pdf | PDF | report.pdf, AES-256 | user password `secret`, owner `owner`, 3 pages |
| pdf/encrypted-rc4.pdf | PDF | report.pdf, RC4 128 | user `secret`, owner `owner`, 3 pages; needed `qpdf --allow-weak-crypto` |
| pdf/cert.p12 | PKCS#12 | from testdata/pem cert + key (OpenSSL 4.1 default algorithms) | password `secret` |
| pdf/black.png | PNG | 1x1 black | 1x1, 69 bytes |
| pdf/stamp.jpg | JPEG | red "CONFIDENTIAL" on white | 600x200 |
| pdf/attachment.csv | CSV | 5 lines | `id,name,amount` + alpha/beta/gamma/delta rows |
| pdf/invoice-cii.xml | XML | hand-written CII invoice (no existing CII file in the repo; embed_1.xml in testdata is a dummy `<xml>`) | declares `urn:cen.eu:en16931:2017` -> Factur-X level `EN 16931`; invoice INV-2026-0042, type code 380, seller Northwind Traders, buyer Contoso Ltd, EUR, net 3850.00, VAT 770.00, grand total 4620.00. Passes litho's EN 16931 Schematron: `remediate` with `profile=Factur-X` answers `Gotenberg-Remediation-Residual: 0` |

## Left out (use the existing `test/integration/testdata/` file)

`office/protected.docx` (`protected_page_1.docx`, password `foo`), `office/slideshow.ppsx`, `office/scrolled.xlsx` (`singlepagesheets-scrolled.xlsx`), `pdf/fields.pdf`, `pdf/multi.pdf`, `pdf/xfa.pdf`, `pdf/with-attachment.pdf` (`page_1_with_attachment.pdf`), `pdf/bookmarked.pdf` (`page_1_with_bookmarks.pdf`), `pdf/image-heavy-testdata.pdf` (`image-heavy.pdf`), `pdf/watermark.png`, `pdf/cert.pem` and `pdf/key.pem` (`pem/`). `office/memo.epub` is dropped: LibreOffice has no EPUB import.

## Measured on the current image (differs from the text above)

- `office/readme.txt` converts to 4 pages, not 3.
- `office/tracked.docx` renders the inserted text only: the deleted text is absent from the PDF.
- Cells clip long text in the PDF: "Cloud host" for "Cloud hosting", "REVENUE-SH" and "COSTS-SHEE" for the sheet markers, "Budget not" for "Budget notes".
- A header repeated on each page counts as one image use per page (`contract.docx`: 10 uses of the logo).

## Not produced / caveats

- Nothing listed was skipped. Deviations: form.fodt/form.odt are LO-generated from an HTML form (see above) because hand-written `office:forms` XML was dropped by LO on import.
- Chromium page counts (invoice 2, handbook 4+) are expected values, not measured; no Chromium render was run.
- No Factur-X CII file existed in the repo when the corpus was made, so pdf/invoice-cii.xml is hand-written; `testdata/invoice-{minimum,basic,en16931,xrechnung}.xml` arrived later for the per-level scenarios of `pdf_remediate.feature` and `libreoffice_convert.feature`.
- Extra files not in the brief: none in corpus/ except those listed (build scratch is outside corpus).
