# Release QA corpus

Realistic documents with known contents, and the facts the scenarios assert about them. This suite turns the black-box QA of Gotenberg 9.0.0 into a regression suite that runs against the full image. It is separate from the quick per-route scenarios of [`test/integration/`](../integration/README.md), and shares their step library and runner.

## Why

The fixtures of `test/integration/testdata/` are minimal: a page with a marker, a form with one field. They prove that a route answers. They say nothing about what a 10-page contract, a 3-sheet budget, a deck with a hidden slide, a multilingual page or an encrypted report comes out as.

The 9.0.0 release QA converted such documents, about 80 files, through every route and option, and compared the output with what each document contains. It found defects the minimal fixtures could not: a CSV converted to one cell per line, a Factur-X invoice embedded without a check, a truncated PDF that `metadata` accepted, element backgrounds that the docs described wrong. Keeping the documents here, with their facts written down, means the next change to Gotenberg, lok, litho, Chromium or LibreOffice runs against the same documents and the same facts.

The rule: a file that exists to exercise one field belongs in `test/integration/testdata/`; a document that stands for a real user file, with its facts in the manifest, belongs in `test/corpus/testdata/`.

Each fact is known by construction or measured once. A generated document knows its page breaks, markers, metadata and sizes because the script wrote them. A fact the engine decides, such as the page count of a text file, was measured on the image and recorded as measured, not as hoped. [`MANIFEST.md`](MANIFEST.md) lists every file with its format, what it contains and what a scenario can assert, then where the image differs from the document's intent.

## What is in it

Under `testdata/`:

- `web/`, for the Chromium routes: sites with their assets, CSS page sizes and breaks, a header and a footer, a delayed render, a console error, a Markdown bundle, a long landing page, multilingual text, a wide table, and the invoice site as a ZIP archive.
- `office/`, for the LibreOffice route: the same memo, figures and pitch across fifteen formats, a contract, a budget, a deck with a hidden slide and notes, a CSV, tracked changes, a form, plain text, a letter, and images in seven formats.
- `pdf/`, for the PDF routes: a report with text at known coordinates for redaction and stamps, an annex to merge, a letterhead, transparency, CMYK colors, an image-heavy file, AES and RC4 encrypted copies, a CII invoice, a certificate, a stamp image and an attachment.
- `generators/`: the scripts that wrote the generated files, and the commands that made the hand-made ones. See its [README](testdata/generators/README.md).

## How the scenarios use it

`features/` holds one file per engine: `chromium.feature`, `libreoffice.feature` and `pdf.feature`, tagged `@chromium`, `@libreoffice` and `@pdf`. A scenario sends one document, or one list of stages, and asserts the manifest's facts on the answer: the page count, the page size in points, the text on a given page, the metadata, the outline, the number of images, the embedded files, PDF/A conformance, the Factur-X XMP. Two scenario outlines cover the format variants of the memo, the figures and the pitch, and the image inputs.

```bash
make build
make test-corpus
make test-corpus TAGS=chromium
make test-corpus TAGS=libreoffice
make test-corpus TAGS=pdf KEEP_FAILED=/tmp/failed
```

A scenario asserts what the image produces, and the manifest says when that differs from the document's intent: `readme.txt` paginates to 4 pages, a spreadsheet cell clips its text to `REVENUE-SH`, the deleted text of a tracked change is absent. When a fix changes such an outcome, update the manifest and the scenario together.

## When it fails in CI

CI runs this suite on every push and pull request, on the full image, after the integration scenarios. A failure after a dependency bump, such as Chromium, LibreOffice, litho, lok or a font, means a real document comes out differently. Rerun the tag with `KEEP_FAILED` and look at the output. A regression gets fixed. An acceptable change gets its new fact in the manifest and in the scenario, in the same pull request, with a note in the manifest saying what moved and why. Don't loosen an assertion to pass: change the fact, or fix the engine.

## How to add a fixture

1. Generate the document with a script in `generators/`, so its facts are known by construction, and check in the script with the file. A file made once by hand, such as an encrypted copy or a certificate, goes in the "Not scripted" list of `generators/README.md` with the command that made it.
2. Put a unique marker in every region a scenario reads (`REPORT-PAGE-2`, `COSTS-SHEET-MARKER`), not prose: text extraction keeps a marker intact across fonts, scripts and line breaks.
3. Record the file in `MANIFEST.md`: format, contents, and the facts a scenario can assert. Measure a page count on the image rather than expecting it, and say so.
4. Write the scenario in the engine's file under `features/`, asserting those facts, before any Go change. Reuse the steps listed in [`test/integration/README.md`](../integration/README.md), and add one there only when no existing step reads the fact.
5. Keep the corpus small. It weighs 2 MB: share an asset between files rather than embedding a new one, and prefer an existing `test/integration/testdata/` file over a copy. The manifest's "Left out" list names the ones the corpus reuses.
6. Run `make test-corpus TAGS=<engine>` and `make lint-prettier`.

`.prettierignore` excludes the fixtures: the manifest records their byte sizes, and the HTML and CSS are what the generators wrote. Don't reformat them.

## What it is not

The per-route feature files of `test/integration/` stay the place for one field or one stage at a time, every error message, and every edge of the API. The corpus asserts outcomes on realistic documents end to end. It is not a benchmark either: it runs against the same image as the integration scenarios, and its timings mean nothing.
