// Package calculator implements the arithmetic core. It is pure: no I/O, no
// HTTP, no global mutable state. Every operation is registered once in a
// Registry and executed through Calculate, which enforces arity and rejects
// non-finite results centrally.
package calculator

import (
	"errors"
	"fmt"
	"math"
)

// Sentinel errors. Callers classify failures with errors.Is.
var (
	ErrUnknownOperation = errors.New("unsupported operation")
	ErrArity            = errors.New("wrong number of operands")
	ErrInvalidOperand   = errors.New("operand must be a finite number")
	ErrDivisionByZero   = errors.New("cannot divide by zero")
	ErrDomain           = errors.New("result is undefined for the given operands")
	ErrNonFinite        = errors.New("result is too large to represent")
)

// Func computes a result from operands whose count has already been checked.
type Func func(operands []float64) (float64, error)

// Operation describes one supported calculation.
type Operation struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Arity  int    `json:"arity"`
	Label  string `json:"label"`
	Fn     Func   `json:"-"`
}

// Registry holds the supported operations, keyed by name, in display order.
type Registry struct {
	byName map[string]Operation
	order  []string
}

// NewRegistry builds a registry from the given operations. It panics on a
// duplicate name or a non-positive arity, because both are programmer errors
// caught at startup.
func NewRegistry(ops ...Operation) *Registry {
	r := &Registry{byName: make(map[string]Operation, len(ops))}
	for _, op := range ops {
		if _, dup := r.byName[op.Name]; dup {
			panic(fmt.Sprintf("calculator: duplicate operation %q", op.Name))
		}
		if op.Arity < 1 || op.Fn == nil {
			panic(fmt.Sprintf("calculator: invalid operation %q", op.Name))
		}
		r.byName[op.Name] = op
		r.order = append(r.order, op.Name)
	}
	return r
}

// Operations returns the registered operations in registration order.
func (r *Registry) Operations() []Operation {
	out := make([]Operation, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.byName[name])
	}
	return out
}

// Calculate runs the named operation. It returns ErrUnknownOperation,
// ErrArity or ErrInvalidOperand for bad input, the operation's own error for
// undefined math, and ErrNonFinite if the result overflows or is NaN.
func (r *Registry) Calculate(name string, operands []float64) (float64, error) {
	op, ok := r.byName[name]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownOperation, name)
	}
	if len(operands) != op.Arity {
		return 0, fmt.Errorf("%w: %s takes %d, got %d", ErrArity, name, op.Arity, len(operands))
	}
	for _, v := range operands {
		if !isFinite(v) {
			return 0, ErrInvalidOperand
		}
	}
	result, err := op.Fn(operands)
	if err != nil {
		return 0, err
	}
	if !isFinite(result) {
		return 0, ErrNonFinite
	}
	return result, nil
}

func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// domainError carries an operation-specific message while still matching
// ErrDomain with errors.Is.
type domainError string

func (e domainError) Error() string        { return string(e) }
func (e domainError) Is(target error) bool { return target == ErrDomain }

// Default returns the registry with all seven supported operations.
func Default() *Registry {
	return NewRegistry(
		Operation{Name: "add", Symbol: "+", Arity: 2, Label: "Add", Fn: add},
		Operation{Name: "subtract", Symbol: "−", Arity: 2, Label: "Subtract", Fn: subtract},
		Operation{Name: "multiply", Symbol: "×", Arity: 2, Label: "Multiply", Fn: multiply},
		Operation{Name: "divide", Symbol: "÷", Arity: 2, Label: "Divide", Fn: divide},
		Operation{Name: "power", Symbol: "^", Arity: 2, Label: "Power", Fn: power},
		Operation{Name: "sqrt", Symbol: "√", Arity: 1, Label: "Square root", Fn: sqrt},
		Operation{Name: "percent", Symbol: "%", Arity: 2, Label: "Percent (b% of a)", Fn: percent},
	)
}

func add(o []float64) (float64, error)      { return o[0] + o[1], nil }
func subtract(o []float64) (float64, error) { return o[0] - o[1], nil }
func multiply(o []float64) (float64, error) { return o[0] * o[1], nil }

func divide(o []float64) (float64, error) {
	if o[1] == 0 {
		return 0, ErrDivisionByZero
	}
	return o[0] / o[1], nil
}

// power returns a^b. A negative base with a fractional exponent has no real
// result, and 0 raised to a negative power is a division by zero.
func power(o []float64) (float64, error) {
	base, exp := o[0], o[1]
	if base < 0 && exp != math.Trunc(exp) {
		return 0, domainError("cannot raise a negative number to a fractional power")
	}
	if base == 0 && exp < 0 {
		return 0, ErrDivisionByZero
	}
	return math.Pow(base, exp), nil
}

func sqrt(o []float64) (float64, error) {
	if o[0] < 0 {
		return 0, domainError("cannot take the square root of a negative number")
	}
	return math.Sqrt(o[0]), nil
}

// percent returns b% of a, i.e. a × b / 100.
func percent(o []float64) (float64, error) { return o[0] * o[1] / 100, nil }
