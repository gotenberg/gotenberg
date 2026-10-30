package pdf

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	litho "github.com/gotenberg/litho/pkg/pdf"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

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
	if litho.IsInvalidInput(err) {
		message := strings.ReplaceAll(se.err.Error(), dirPath+"/", "")

		return api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Stage '%s' failed on '%s': %s", se.op, se.filename, message),
		))
	}

	return err
}
