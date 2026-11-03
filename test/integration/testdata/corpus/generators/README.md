# Corpus generators

These scripts are the provenance of the corpus fixtures one directory up. Run them only to regenerate a fixture; the generated files are checked in.

Run them in this order, with `C` set to a scratch copy of the corpus root and `B` to a scratch directory. The font paths are macOS paths (`/System/Library/Fonts/Supplemental/Arial*.ttf`); change them on another system.

```bash
export C=/tmp/corpus B=/tmp/corpus-build
mkdir -p $C/web $C/office/images $C/pdf $B
uv run --python 3.12 --with pillow python web.py
uv run --python 3.12 --with pillow --with reportlab python img_pdf.py
uv run --python 3.12 --with pillow --with python-docx --with openpyxl --with python-pptx python office1.py
```

`web.py` writes `i18n.json`, which `office1.py` reads; a copy is kept here. `conv.sh <dir>` converts the flat OpenDocument sources (`memo.fodt`, `figures.fods`, `pitch.fodp`, `form.fodt`) with the LibreOffice of a Gotenberg image into the `.odt`, `.doc`, `.rtf`, `-lo.docx`, `.html`, `.ods`, `.xls`, `-lo.xlsx`, `.odp`, `.ppt` and `-lo.pptx` variants.

Not scripted, made once by hand: the flat OpenDocument sources, `pdf/encrypted.pdf` and `pdf/encrypted-rc4.pdf` (`qpdf` on `report.pdf`, user password `secret`, owner password `owner`), `pdf/cert.p12` (`openssl pkcs12` from `testdata/pem`, password `secret`) and `pdf/invoice-cii.xml`.
