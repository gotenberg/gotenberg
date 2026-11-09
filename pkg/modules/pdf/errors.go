package pdf

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	litho "github.com/gotenberg/litho/pkg/pdf"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// stageInputError is a stage failure the request causes that litho does not
// classify, such as detaching a file the document does not embed, or one
// whose litho message Gotenberg words for the client. Its message reaches the
// client, and err, when set, the logs.
type stageInputError struct {
	message string
	err     error
}

func (e *stageInputError) Error() string {
	if e.err == nil {
		return e.message
	}

	return fmt.Sprintf("%s: %s", e.message, e.err)
}

func (e *stageInputError) Unwrap() error {
	return e.err
}

// isInvalidInput reports whether the request caused err.
func isInvalidInput(err error) bool {
	var inputErr *stageInputError

	return litho.IsInvalidInput(err) || errors.As(err, &inputErr)
}

// errorType classifies a stage failure for the error.type of its span.
func errorType(err error) string {
	if isInvalidInput(err) {
		return gotenberg.ErrorTypeInvalidInput
	}

	return gotenberg.ClassifyError(err)
}

// editError returns the innermost [litho.EditError] of err, the edit that
// failed, or nil. An edit litho applies for another, such as a signature
// applying the edits queued before it, wraps that one's error.
func editError(err error) *litho.EditError {
	var found *litho.EditError
	for {
		var e *litho.EditError
		if !errors.As(err, &e) || e.Err == nil {
			return found
		}

		found = e
		err = e.Err
	}
}

// clientMessage returns the message of err without the steps litho reports it
// from, such as the pipeline stage or the Writer method that queued the edit,
// which mean nothing to a client. The full chain stays in the logs.
func clientMessage(err error) string {
	var inputErr *stageInputError
	if errors.As(err, &inputErr) {
		return inputErr.message
	}

	if e := editError(err); e != nil {
		return e.Err.Error()
	}

	var stageErr *litho.PipelineStageError
	if errors.As(err, &stageErr) && stageErr.Err != nil {
		return stageErr.Err.Error()
	}

	return err.Error()
}

// lithoFixes words the fix of the litho errors whose message alone leaves the
// client without one.
var lithoFixes = []struct {
	err error
	fix func(message string) string
}{
	{
		err: litho.ErrTextNotEncodable,
		fix: func(string) string {
			return "the standard fonts draw Windows-1252 text only (Latin letters with their accents, the euro sign and the usual punctuation); stamp other text with a stamp-pdf stage and a page Chromium renders, or with nativeWatermarkText on LibreOffice"
		},
	},
	{
		err: litho.ErrNoFontCovers,
		fix: func(message string) string {
			return message + "; the fonts installed in the image draw Latin, Greek, Cyrillic, Chinese, Japanese and Korean text, among others: install a font that has these characters, see https://gotenberg.dev/docs/troubleshooting#install-fonts"
		},
	},
	{
		err: litho.ErrInvalidImage,
		fix: func(message string) string {
			return message + "; send a valid PNG or JPEG image, its extension naming its format (.png, .jpg or .jpeg)"
		},
	},
	{
		err: litho.ErrNoPageLeft,
		fix: func(message string) string {
			return message + "; leave at least one page out of 'pages'"
		},
	},
	{
		err: litho.ErrRenderTooLarge,
		fix: func(message string) string {
			return message + "; draw at a lower 'dpi', or raise --pdf-max-render-pixels (env PDF_MAX_RENDER_PIXELS)"
		},
	},
	{
		err: litho.ErrIncompatibleProfiles,
		fix: func(message string) string {
			return message + "; the pairs are " + validPairs()
		},
	},
}

// handleError turns a stage failure the request caused into a 400 response
// naming the stage and the document. dirPath, the request's working
// directory, is stripped from the message so that disk paths never reach the
// client.
func handleError(err error, dirPath string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, ErrInvalidStages) {
		return api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Invalid form data: form field 'stages' is invalid: %s", strings.TrimPrefix(err.Error(), ErrInvalidStages.Error()+": ")),
		))
	}

	var se *stageError
	if !errors.As(err, &se) {
		return err
	}

	// The timestamp authority URL comes from the client, and the outbound
	// guard may refuse it. Keep the reason, which can reveal the policy, in
	// the logs only.
	if errors.Is(err, litho.ErrTimestampAuthority) || errors.Is(err, litho.ErrValidationData) {
		return api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Stage '%s' failed on '%s': the timestamp authority could not be reached or refused the request", se.op, se.filename),
		))
	}

	// Only a decrypt stage, the first, opens an encrypted PDF: the routes of
	// the other stages take no password. The read routes take a "password"
	// form field.
	if errors.Is(err, litho.ErrInvalidPassword) && se.op != "decrypt" {
		message := fmt.Sprintf(`Stage '%s' failed on '%s': the PDF is encrypted; open it with a "decrypt" stage first, such as [{"op":"decrypt","password":"..."},{"op":"%s"}] on /forms/pdf/pipe`, se.op, se.filename, se.op)
		if _, ok := stageFactories[se.op]; !ok {
			message = fmt.Sprintf("Reading '%s' failed: the PDF is encrypted, and the 'password' form field is missing or wrong", se.filename)
		}

		return api.WrapError(err, api.NewSentinelHttpError(http.StatusBadRequest, message))
	}

	// The decrypt stage is where the password goes: name it as the read
	// routes do.
	if errors.Is(err, litho.ErrInvalidPassword) && se.op == "decrypt" {
		return api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Stage 'decrypt' failed on '%s': the PDF is encrypted, and the 'password' is missing or wrong", se.filename),
		))
	}

	// litho classifies its own errors, so a new litho error never reaches the
	// client as a 500 by omission.
	if isInvalidInput(err) {
		message := strings.ReplaceAll(clientMessage(se.err), dirPath+"/", "")
		for _, f := range lithoFixes {
			if errors.Is(err, f.err) {
				message = f.fix(message)
				break
			}
		}

		return api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Stage '%s' failed on '%s': %s", se.op, se.filename, message),
		))
	}

	return err
}
