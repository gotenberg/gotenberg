<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/38133342-11df3bd8-340f-11e8-9fe4-50baecdceeca.png" alt="Gotenberg's logo" width="250" height="250" />
</p>
<h3 align="center">Gotenberg</h3>
<p align="center">A stateless API for converting Markdown files, HTML files and Office documents to PDF</p>
<p align="center">
    <a href="https://microbadger.com/images/thecodingmachine/gotenberg:2.0.0">
        <img src="https://images.microbadger.com/badges/image/thecodingmachine/gotenberg:2.0.0.svg" alt="MicroBadger layers">
    </a>
    <a href="https://travis-ci.org/thecodingmachine/gotenberg">
        <img src="https://travis-ci.org/thecodingmachine/gotenberg.svg?branch=master" alt="Travis CI">
    </a>
    <a href="https://godoc.org/github.com/thecodingmachine/gotenberg">
        <img src="https://godoc.org/github.com/thecodingmachine/gotenberg?status.svg" alt="GoDoc">
    </a>
    <a href="https://goreportcard.com/report/thecodingmachine/gotenberg">
        <img src="https://goreportcard.com/badge/github.com/thecodingmachine/gotenberg" alt="Go Report Card">
    </a>
    <a href="https://codecov.io/gh/thecodingmachine/gotenberg/branch/master">
        <img src="https://codecov.io/gh/thecodingmachine/gotenberg/branch/master/graph/badge.svg" alt="Codecov">
    </a>
</p>

---

At TheCodingMachine, we build a lot of web applications (intranets, extranets and so on) which require to generate PDF from 
various sources. Each time, we ended up using some well known libraries like **wkhtmltopdf** or **unoconv** and kind of lost time by
reimplementing a solution from a project to another project. Meh.

# Menu

* [Usage](#usage)
* [Security](#security)
* [Scalability](#scalability)
* [Custom implementation](#custom-implementation)
* [Clients](#clients)

## Usage

Let's say you're starting the API using this simple command:

```sh
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:2.0.0
```

The API is now available on your host under `http://127.0.0.1:3000`.

It accepts `POST` requests with a `multipart/form-data` Content-Type. Your form data should provide one or more files to convert.
The default image accepts the following:

* Markdown files
* HTML files
* Office documents (.docx, .doc, .odt, .pptx, .ppt, .odp and so on)
* PDF files

**Heads up:** the API relies on the file extension to determine which library to use for conversion.

There are two use cases:

* If you send one file, it will convert it and return the resulting PDF
* If many files, it will convert them to PDF, merge the resulting PDFs into a single PDF and return it

### Examples:

* One file

```sh
$ curl --request POST \
    --url http://127.0.0.1:3000 \
    --header 'Content-Type: multipart/form-data' \
    --form files=@file.docx \
    > result.pdf
```

* Many files

```sh
$ curl --request POST \
    --url http://127.0.0.1:3000 \
    --header 'Content-Type: multipart/form-data' \
    --form files=@file.md \
    --form files=@file.html \
    --form files=@file.pdf \
    --form files=@file.docx \
    > result.pdf
```

## Security

The API does not provide any authentication mechanisms. Make sure to not put it on a public facing port and your client(s) should always 
controls what is sent to the API.

## Scalability

Some libraries like **unoconv** cannot perform concurrent conversions. That's why the API does only one conversion at a time.
If your API is under heavy load, a request will take time to be processed. 

Fortunately, you may pass through this limitation by scaling the API.

In the following example, I'll demonstrate how to do some vertical scaling (= on the same machine) with Docker Compose, but of course horizontal scaling works too!

```yaml
version: '3'

services:

  # your others services
      
  gotenberg:
    image: thecodingmachine/gotenberg:2.0.0
```

You may now launch your services using:

```bash
docker-compose up --scale gotenberg=your_number_of_instances
```

When requesting the Gotenberg service with your client(s), Docker will automatically redirect a request to a Gotenberg container
according to the round-robin strategy.

## Custom implementation

The API relies on a simple YAML configuration file called `gotenberg.yml`. It allows you to tweak some values and even provides you 
a way to change the commands called for each kind of conversion.

Below the default configuration file:

```yaml
# The port the application will listen to.
port: 3000

logs:
  # Accepted values, in order of severity: DEBUG, INFO, WARN, ERROR, FATAL, PANIC.
  # Messages at and above the selected level will be logged.
  level: "INFO"

  # Accepted values: text, json.
  # When a TTY is not attached, the output will be in the defined format.
  formatter: "text"

# You don't like a library which is used for a conversion? You want to handle a new file type?
# You may provide here your own implementation!
commands:

  # Some libraries like unoconv cannot perform concurrent conversions. That's why the API does only one conversion at a time.
  # If your current implementation uses libraries which are able to perform concurrent conversions, you may
  # change this value to false.
  lock: true

  # Unlike others commands' templates, you have access to FilesPaths instead of FilePath: it gathers all PDF files which should be merged.
  merge:
    template: "pdftk {{ range $filePath := .FilesPaths }} {{ $filePath }} {{ end }} cat output {{ .ResultFilePath }}"
    interpreter: "/bin/sh -c"
    timeout: 30
  
  # You may add more commands (or less, or even none).
  conversions:

      # The command template: you have access to FilePath and ResultFilePath variables.
    - template: "markdown-pdf {{ .FilePath }} -o {{ .ResultFilePath }}"
      # The binary which will call the command.
      interpreter: "/bin/sh -c"
      # Duration in seconds after which the command will be killed if it has not finished.
      timeout: 30
      # Files with the following extensions will be converted by the current command.
      extensions:
        - ".md"

    - template: "xvfb-run -e /dev/stdout wkhtmltopdf {{ .FilePath }} {{ .ResultFilePath }}"
      interpreter: "/bin/sh -c"
      timeout: 30
      extensions:
        - ".html"
        - ".htm"

    - template: "unoconv --format pdf --output \"{{ .ResultFilePath }}\" \"{{ .FilePath }}\""
      interpreter: "/bin/sh -c"
      timeout: 30
      extensions:
        - ".doc"
        - ".docx"
        - ".odt"
        - ".xls"
        - ".xlsx"
        - ".ods"
        - ".ppt"
        - ".pptx"
        - ".odp"
```

We provide binaries for a wide range of OS and architecture in the [releases page](../../releases), 
so feel free to create your own Docker image for your implementation of the Gotenberg API :metal:

## Clients

* https://github.com/thecodingmachine/gotenberg-php-client (PHP client)
* Add your own client by submitting a [pull request](../../pulls)!

---

Would you like to update this documentation ? Feel free to open an [issue](../../issues).