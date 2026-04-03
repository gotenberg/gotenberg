# Adding PDF Engine Features

Each new PDF engine capability (e.g., bookmarks, watermark, stamp, embed) requires a matching Makefile entry. The Makefile variables control which engines are passed to Gotenberg at `make run` and `make test-integration` time (via `compose.yaml`). If you skip this step, the flag still works when set manually, but `make run` falls back to the default defined in `pdfengines.go`, which may not include the new engine.

Every `--pdfengines-*-engines` flag registered in `pkg/modules/pdfengines/pdfengines.go` must have a corresponding variable and flag in the Makefile:

1. **Add a variable** in the Makefile's variable block (around line 60 to 70):
   ```makefile
   PDFENGINES_<FEATURE>_ENGINES=<default engines>
   ```
2. **Add the flag** in `compose.yaml`'s command args:
   ```yaml
   - "--pdfengines-<feature>-engines=${PDFENGINES_<FEATURE>_ENGINES}"
   ```

The default value must match the `fs.StringSlice(...)` call for that flag in `pdfengines.go`.

## Example: Rotate

The rotate feature was added with two engines (`pdfcpu` and `pdftk`). Here is what the additions look like:

**Makefile** (variable block):

```makefile
PDFENGINES_ROTATE_ENGINES=pdfcpu,pdftk
```

**compose.yaml** (command args):

```yaml
- "--pdfengines-rotate-engines=${PDFENGINES_ROTATE_ENGINES}"
```
