package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"calculator/internal/calculator"
)

// maxBodyBytes caps request bodies; a valid request is well under 200 bytes.
const maxBodyBytes = 1 << 10

type handlers struct {
	reg *calculator.Registry
}

// calculateRequest uses pointers so that a missing or null field can be told
// apart from a zero value.
type calculateRequest struct {
	Operation *string    `json:"operation"`
	Operands  []*float64 `json:"operands"`
}

type calculateResponse struct {
	Operation string    `json:"operation"`
	Operands  []float64 `json:"operands"`
	Result    float64   `json:"result"`
}

type operationsResponse struct {
	Operations []calculator.Operation `json:"operations"`
}

func (h *handlers) calculate(w http.ResponseWriter, r *http.Request) {
	name, operands, err := decodeCalculateRequest(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := h.reg.Calculate(name, operands)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, calculateResponse{Operation: name, Operands: operands, Result: result})
}

func (h *handlers) operations(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, operationsResponse{Operations: h.reg.Operations()})
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// decodeCalculateRequest checks, in order: the Content-Type, the JSON body,
// and that every required field is present and non-null.
func decodeCalculateRequest(w http.ResponseWriter, r *http.Request) (string, []float64, error) {
	if !isJSONContentType(r) {
		return "", nil, &APIError{
			Status:  http.StatusUnsupportedMediaType,
			Code:    CodeUnsupportedMediaType,
			Message: "Content-Type must be application/json",
		}
	}

	var req calculateRequest
	if err := decodeStrictJSON(w, r, &req); err != nil {
		return "", nil, err
	}

	if req.Operation == nil || *req.Operation == "" {
		return "", nil, badRequest(CodeUnsupportedOperation, "operation is required")
	}
	operands, err := requireOperands(req.Operands)
	if err != nil {
		return "", nil, err
	}
	return *req.Operation, operands, nil
}

func isJSONContentType(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

// decodeStrictJSON reads exactly one JSON object of at most maxBodyBytes,
// rejecting unknown fields and anything after the object.
func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return invalidJSON("body must contain a single JSON object")
	}
	return nil
}

// requireOperands rejects a missing operands array or a null element, which
// would otherwise silently become 0.
func requireOperands(raw []*float64) ([]float64, error) {
	if raw == nil {
		return nil, invalidOperands("operands is required")
	}
	operands := make([]float64, len(raw))
	for i, v := range raw {
		if v == nil {
			return nil, invalidOperands(fmt.Sprintf("operands[%d] must be a number", i))
		}
		operands[i] = *v
	}
	return operands, nil
}

// decodeError turns encoding/json failures into client-facing errors without
// leaking Go type names.
func decodeError(err error) *APIError {
	var maxBytes *http.MaxBytesError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &maxBytes):
		return invalidJSON(fmt.Sprintf("request body must not exceed %d bytes", maxBodyBytes))
	case errors.As(err, &typeErr) && strings.HasPrefix(typeErr.Field, "operands"):
		return invalidOperands("operands must be an array of finite numbers")
	case errors.As(err, &typeErr):
		return invalidJSON(fmt.Sprintf("field %q has the wrong type", typeErr.Field))
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		return invalidJSON(strings.TrimPrefix(err.Error(), "json: "))
	case errors.Is(err, io.EOF):
		return invalidJSON("request body is empty")
	default:
		return invalidJSON("request body is not valid JSON")
	}
}
