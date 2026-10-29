package pdf

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	litho "github.com/gotenberg/litho/pkg/pdf"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// clientErrors lists the litho errors the request causes: a document litho
// cannot read or edit as asked, an option out of range, or credentials that do
// not fit. Every other litho error is a server error.
var clientErrors = []error{
	litho.ErrInvalidPassword,
	litho.ErrPasswordRejected,
	litho.ErrEncryptedNotAuthenticated,
	litho.ErrUnsupportedEncryption,
	litho.ErrNotAPDF,
	litho.ErrDamagedXRef,
	litho.ErrDamagedStream,
	litho.ErrStreamTooLarge,
	litho.ErrImageTooLarge,
	litho.ErrPageContentTooLarge,
	litho.ErrInvalidPageNumber,
	litho.ErrInvalidPageRange,
	litho.ErrInvalidRotation,
	litho.ErrEmptyPageList,
	litho.ErrEmptyText,
	litho.ErrEmptyData,
	litho.ErrUnsupportedMimeType,
	litho.ErrImageNotDecodable,
	litho.ErrInvalidFont,
	litho.ErrInvalidPageSize,
	litho.ErrSourceIndexOutOfRange,
	litho.ErrSignatureFieldUnavailable,
	litho.ErrSignatureFieldLocked,
	litho.ErrSeedValueUnmet,
	litho.ErrSignerKeyUnsupported,
	litho.ErrSignerKeyMismatch,
	litho.ErrSignerCertificateUsage,
	litho.ErrSignerKeyEncrypted,
	litho.ErrSignerPEM,
	litho.ErrSignerPKCS12,
	litho.ErrSignerPKCS12Password,
	litho.ErrSignFullRewriteIncompatible,
	litho.ErrCertificateRevoked,
	litho.ErrRedactionRefused,
	litho.ErrRedactionIncomplete,
	litho.ErrInvalidRedactOptions,
	litho.ErrPageDeletionIncomplete,
	litho.ErrAutoTagAlreadyTagged,
	litho.ErrAutoTagMarkedContent,
	litho.ErrAutoTagUnreadableContent,
	litho.ErrAutoTagNothingToTag,
	litho.ErrRemediationBrokePDF,
	litho.ErrRemediationUnverified,
	litho.ErrFontNotSupplied,
	litho.ErrFontProgramInvalid,
	litho.ErrFontEmbeddingRestricted,
	litho.ErrFontKind,
	litho.ErrFontCoverage,
	litho.ErrFontWidths,
	litho.ErrFontRejected,
	litho.ErrFontUnwritable,
	litho.ErrFontRolledBack,
	litho.ErrFontMissing,
	litho.ErrRequiresOCR,
	litho.ErrEditAfterRemediation,
	litho.ErrPDFAForbidsEncryption,
	litho.ErrPDFA1ForbidsObjectStreams,
	litho.ErrPDFUAForbidsBarringAccessibility,
	litho.ErrEmptyInvoiceXML,
	litho.ErrInvalidFacturXLevel,
	litho.ErrInvalidFacturXProfile,
	litho.ErrNoOutputIntent,
	litho.ErrColorConversionRefused,
	litho.ErrOutputIntentConflict,
	litho.ErrTransparencyFlatteningRefused,
	litho.ErrAttachmentTreeIncomplete,
	litho.ErrNameHoldsNull,
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

	for _, clientErr := range clientErrors {
		if errors.Is(err, clientErr) {
			message := strings.ReplaceAll(se.err.Error(), dirPath+"/", "")

			return api.WrapError(err, api.NewSentinelHttpError(
				http.StatusBadRequest,
				fmt.Sprintf("Stage '%s' failed on '%s': %s", se.op, se.filename, message),
			))
		}
	}

	return err
}
