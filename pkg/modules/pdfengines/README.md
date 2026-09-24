# Adding PDF Engine Features

Each new PDF engine capability (bookmarks, watermark, stamp, embed, etc.) requires a matching Makefile entry. The Makefile variables control which engines are passed to Gotenberg at `make run` and `make test-integration` time via `compose.yaml`. Skip this step and `make run` falls back to the default defined in `pdfengines.go`, which may not include the new engine.

Every `--pdfengines-*-engines` flag registered in `pdfengines.go` needs two additions:

1. A variable in the Makefile's variable block (around line 60-70):

   ```makefile
   PDFENGINES_<FEATURE>_ENGINES=<default engines>
   ```

2. A flag in `compose.yaml`'s command args:

   ```yaml
   - "--pdfengines-<feature>-engines=${PDFENGINES_<FEATURE>_ENGINES}"
   ```

The default value must match the `fs.StringSlice(...)` call for that flag in `pdfengines.go`.

## Example: Rotate

Rotate was added with two engines (`pdfcpu` and `pdftk`):

**Makefile** (variable block):

```makefile
PDFENGINES_ROTATE_ENGINES=pdfcpu,pdftk
```

**compose.yaml** (command args):

```yaml
- "--pdfengines-rotate-engines=${PDFENGINES_ROTATE_ENGINES}"
```
