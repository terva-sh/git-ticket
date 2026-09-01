package ticket

import (
	"errors"
	"fmt"
)

// Stable error codes a caller may switch on, per plan section 10. The check
// codes in section 11 overlap these only where the condition is the same.
const (
	CodeStoreNotFound     = "store_not_found"
	CodeStoreExists       = "store_exists"
	CodeTicketNotFound    = "ticket_not_found"
	CodeAmbiguousID       = "ambiguous_id"
	CodeStaleRevision     = "stale_revision"
	CodeInvalidTransition = "invalid_transition"
	CodeInvalidField      = "invalid_field"
	CodeDependencyMissing = "dependency_missing"
	CodeDependencyCycle   = "dependency_cycle"
	CodeClaimConflict     = "claim_conflict"
	CodeParseError        = "parse_error"
	CodeMergeConflict     = "merge_conflict"
	CodeSchemaUnsupported = "schema_unsupported"
	CodeLockTimeout       = "lock_timeout"
	CodeValidationFailed  = "validation_failed"
)

// Check-only codes, per plan section 11.
const (
	CodeDuplicateID                  = "duplicate_id"
	CodeFilenameIDMismatch           = "filename_id_mismatch"
	CodeUnknownField                 = "unknown_field"
	CodeParentMissing                = "parent_missing"
	CodeParentCycle                  = "parent_cycle"
	CodeInvalidStatus                = "invalid_status"
	CodeInvalidType                  = "invalid_type"
	CodeInvalidPriority              = "invalid_priority"
	CodeArchiveLocationMismatch      = "archive_location_mismatch"
	CodeDependencyArchivedIncomplete = "dependency_archived_incomplete"
	CodeClaimExpired                 = "claim_expired"
	CodeReferencePathUnresolved      = "reference_path_unresolved"
	CodeLabelUnknown                 = "label_unknown"
	CodeInProgressUnclaimed          = "in_progress_unclaimed"
)

// Error is a coded failure. The code is the stable part; the message is for a
// person and may change.
type Error struct {
	Code    string
	Message string
	// Ticket is the ID when the failure could be attributed to one, and empty
	// when the file did not parse far enough to know it.
	Ticket string
	// Field is the frontmatter field at fault, empty when the failure is about
	// the file as a whole.
	Field string
	// Details carries the code-specific values a caller needs, such as the
	// expected and actual revision for stale_revision.
	Details map[string]string
	// Err is the underlying failure, when there is one worth unwrapping.
	Err error
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func codedError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// CodeOf returns the stable code of err, or the empty string when err is not a
// coded error.
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
