package pdf

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
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

// lithoStep matches the prefixes litho's pipeline puts on an edit's error, as
// in "pipeline stage 1 (write): applying mutation 0: page number out of
// bounds": they locate the failure among litho's own steps, which mean nothing
// to a client.
var lithoStep = regexp.MustCompile(`^applying (?:staged )?mutation \d+(?: before [a-z ]+)?$`)

// clientMessage returns the message of err without litho's step prefixes,
// [litho.PipelineStageError]'s and those of [lithoStep]. The full chain stays
// in the logs.
func clientMessage(err error) string {
	var inputErr *stageInputError
	if errors.As(err, &inputErr) {
		return inputErr.message
	}

	var stageErr *litho.PipelineStageError
	if errors.As(err, &stageErr) && stageErr.Err != nil {
		err = stageErr.Err
	}

	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			break
		}

		prefix, ok := strings.CutSuffix(err.Error(), ": "+inner.Error())
		if !ok || !lithoStep.MatchString(prefix) {
			break
		}

		err = inner
	}

	return err.Error()
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

	// litho classifies its own errors, so a new litho error never reaches the
	// client as a 500 by omission.
	if isInvalidInput(err) {
		message := strings.ReplaceAll(clientMessage(se.err), dirPath+"/", "")

		return api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Stage '%s' failed on '%s': %s", se.op, se.filename, message),
		))
	}

	return err
}
