package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"calculator/internal/calculator"
)

// Stable, machine-readable error codes. The frontend keys off these; the
// message is for humans and may change.
const (
	CodeInvalidJSON          = "INVALID_JSON"
	CodeUnsupportedOperation = "UNSUPPORTED_OPERATION"
	CodeInvalidOperands      = "INVALID_OPERANDS"
	CodeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	CodeMethodNotAllowed     = "METHOD_NOT_ALLOWED"
	CodeNotFound             = "NOT_FOUND"
	CodeDivisionByZero       = "DIVISION_BY_ZERO"
	CodeDomainError          = "DOMAIN_ERROR"
	CodeNonFiniteResult      = "NON_FINITE_RESULT"
	CodeInternal             = "INTERNAL"
)

// APIError is the body of every non-2xx response.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

type errorEnvelope struct {
	Error *APIError `json:"error"`
}

// Constructors name the HTTP status at every call site.

func badRequest(code, msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: code, Message: msg}
}

func unprocessable(code, msg string) *APIError {
	return &APIError{Status: http.StatusUnprocessableEntity, Code: code, Message: msg}
}

// internalError never includes details: those go to the log, not the client.
func internalError() *APIError {
	return &APIError{Status: http.StatusInternalServerError, Code: CodeInternal, Message: "internal server error"}
}

func invalidJSON(msg string) *APIError     { return badRequest(CodeInvalidJSON, msg) }
func invalidOperands(msg string) *APIError { return badRequest(CodeInvalidOperands, msg) }

// toAPIError is the single place where calculator errors become HTTP
// statuses: 400 for malformed requests, 422 for well-formed requests that
// are mathematically undefined.
func toAPIError(err error) *APIError {
	var apiErr *APIError
	switch {
	case errors.As(err, &apiErr):
		return apiErr
	case errors.Is(err, calculator.ErrUnknownOperation):
		return badRequest(CodeUnsupportedOperation, err.Error())
	case errors.Is(err, calculator.ErrArity), errors.Is(err, calculator.ErrInvalidOperand):
		return invalidOperands(err.Error())
	case errors.Is(err, calculator.ErrDivisionByZero):
		return unprocessable(CodeDivisionByZero, err.Error())
	case errors.Is(err, calculator.ErrDomain):
		return unprocessable(CodeDomainError, err.Error())
	case errors.Is(err, calculator.ErrNonFinite):
		return unprocessable(CodeNonFiniteResult, err.Error())
	default:
		return internalError()
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	apiErr := toAPIError(err)
	writeJSON(w, apiErr.Status, errorEnvelope{Error: apiErr})
}
