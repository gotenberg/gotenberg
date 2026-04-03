# Bruno API Collection

A [Bruno](https://www.usebruno.com/) collection in `.bruno/` mirrors every Gotenberg route. Update the collection when adding or updating a route.

## Structure

```
.bruno/
├── bruno.json                     # Collection config
├── collection.bru                 # Collection-level defaults (Gotenberg-Trace header)
├── environments/
│   ├── Local.bru                  # baseUrl: http://localhost:3000
│   └── Demo.bru                   # baseUrl: https://demo.gotenberg.dev
├── Health & Info/                  # GET routes
├── Chromium/Convert/               # POST routes grouped by module
├── Chromium/Screenshot/
├── LibreOffice/
└── PDF Engines/<Feature>/          # One folder per feature (Merge, Split, Rotate, …)
```

## `.bru` File Format

```bru
meta {
  name: <Human-readable name>
  type: http
  seq: <order within folder>
}

post {
  url: {{baseUrl}}/forms/<path>
  body: multipartForm
  auth: none
}

body:multipart-form {
  files: @file(../../test/integration/testdata/<file>)
  <mandatoryField>: <value>
  ~<optionalField>: <value>
}

headers {
  ~Gotenberg-Output-Filename: <name>
  ~Gotenberg-Webhook-Url: http://localhost:8080/webhook
  ~Gotenberg-Webhook-Error-Url: http://localhost:8080/webhook/error
  ~Gotenberg-Webhook-Method: POST
  ~Gotenberg-Webhook-Error-Method: POST
  ~Gotenberg-Webhook-Extra-Http-Headers: {"X-Custom":"value"}
}
```

## Conventions

- **Mandatory fields** have no prefix; **optional fields** use the `~` prefix (disabled by default in Bruno).
- **File references** use relative paths to `test/integration/testdata/`.
- **Webhook and output filename headers** appear on every POST route as optional (`~`).
- **One `.bru` file per request.** For routes with read/write variants (e.g., bookmarks, metadata), create separate files in the same folder.

## Checklist When Adding/Updating a Route

1. Create or update the `.bru` file in the matching folder under `.bruno/`.
2. Include all form fields from the route handler. Check `FormData*` calls in the route function.
3. For file upload fields (`files`, `watermark`, `stamp`, `embeds`), use `@file(...)` with a suitable test file.
4. Verify the URL path matches the route's `Path` field exactly.
5. For new module folders, keep the naming consistent (e.g., `PDF Engines/Rotate/`).
