# Adding PDF Engine Features

When adding a new PDF engine capability (e.g., bookmarks, watermark, stamp, embed), you must update the Makefile to include the corresponding engine list variable and flag. Every `--pdfengines-*-engines` flag registered in `pkg/modules/pdfengines/pdfengines.go` must have a matching entry in the Makefile:

1. **Add a variable** in the Makefile's variable block (around line 60-70):
   ```makefile
   PDFENGINES_<FEATURE>_ENGINES=<default engines>
   ```
2. **Add the flag** in the Makefile's command args block (around line 140-155):
   ```makefile
   --pdfengines-<feature>-engines=$(PDFENGINES_<FEATURE>_ENGINES) \
   ```

The default value should match what is defined in `pdfengines.go`'s `fs.StringSlice(...)` call for that flag.
