package calculator

import (
	"errors"
	"math"
	"testing"
)

func TestCalculate(t *testing.T) {
	r := Default()
	tests := []struct {
		name     string
		op       string
		operands []float64
		want     float64
		wantErr  error
	}{
		{"add", "add", []float64{2, 3}, 5, nil},
		{"add negatives", "add", []float64{-2.5, -3.5}, -6, nil},
		{"add float keeps IEEE-754 semantics", "add", []float64{0.1, 0.2}, 0.30000000000000004, nil},
		{"subtract", "subtract", []float64{10, 4}, 6, nil},
		{"subtract to negative", "subtract", []float64{4, 10}, -6, nil},
		{"multiply", "multiply", []float64{6, 7}, 42, nil},
		{"multiply by zero", "multiply", []float64{1e300, 0}, 0, nil},
		{"multiply overflow", "multiply", []float64{1e308, 10}, 0, ErrNonFinite},
		{"divide", "divide", []float64{10, 4}, 2.5, nil},
		{"divide negative", "divide", []float64{-9, 3}, -3, nil},
		{"divide by zero", "divide", []float64{1, 0}, 0, ErrDivisionByZero},
		{"zero divided by zero", "divide", []float64{0, 0}, 0, ErrDivisionByZero},
		{"divide by negative zero", "divide", []float64{1, math.Copysign(0, -1)}, 0, ErrDivisionByZero},
		{"divide tiny", "divide", []float64{1, 1e-320}, 0, ErrNonFinite},
		{"power", "power", []float64{2, 10}, 1024, nil},
		{"power negative exponent", "power", []float64{2, -2}, 0.25, nil},
		{"power fractional", "power", []float64{9, 0.5}, 3, nil},
		{"power negative base integer exp", "power", []float64{-2, 3}, -8, nil},
		{"power negative base fractional exp", "power", []float64{-8, 1.0 / 3}, 0, ErrDomain},
		{"power zero to negative", "power", []float64{0, -1}, 0, ErrDivisionByZero},
		{"power zero to zero", "power", []float64{0, 0}, 1, nil},
		{"power overflow", "power", []float64{10, 400}, 0, ErrNonFinite},
		{"sqrt", "sqrt", []float64{16}, 4, nil},
		{"sqrt zero", "sqrt", []float64{0}, 0, nil},
		{"sqrt irrational", "sqrt", []float64{2}, math.Sqrt2, nil},
		{"sqrt negative", "sqrt", []float64{-1}, 0, ErrDomain},
		{"percent", "percent", []float64{200, 15}, 30, nil},
		{"percent negative", "percent", []float64{50, -10}, -5, nil},
		{"unknown op", "modulo", []float64{1, 2}, 0, ErrUnknownOperation},
		{"too few operands", "add", []float64{1}, 0, ErrArity},
		{"too many operands", "sqrt", []float64{1, 2}, 0, ErrArity},
		{"no operands", "add", nil, 0, ErrArity},
		{"NaN operand", "add", []float64{math.NaN(), 1}, 0, ErrInvalidOperand},
		{"Inf operand", "sqrt", []float64{math.Inf(1)}, 0, ErrInvalidOperand},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.Calculate(tc.op, tc.operands)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOperationsOrderAndMetadata(t *testing.T) {
	ops := Default().Operations()
	want := []string{"add", "subtract", "multiply", "divide", "power", "sqrt", "percent"}
	if len(ops) != len(want) {
		t.Fatalf("got %d operations, want %d", len(ops), len(want))
	}
	for i, op := range ops {
		if op.Name != want[i] {
			t.Errorf("ops[%d] = %q, want %q", i, op.Name, want[i])
		}
		if op.Symbol == "" || op.Label == "" {
			t.Errorf("op %q is missing symbol or label", op.Name)
		}
	}
}

func TestNewRegistryPanics(t *testing.T) {
	fn := func([]float64) (float64, error) { return 0, nil }
	cases := map[string][]Operation{
		"duplicate":  {{Name: "x", Arity: 1, Fn: fn}, {Name: "x", Arity: 1, Fn: fn}},
		"zero arity": {{Name: "x", Arity: 0, Fn: fn}},
		"nil fn":     {{Name: "x", Arity: 1}},
	}
	for name, ops := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			NewRegistry(ops...)
		})
	}
}

func TestDomainErrorMessages(t *testing.T) {
	r := Default()
	_, err := r.Calculate("sqrt", []float64{-1})
	if err == nil || err.Error() != "cannot take the square root of a negative number" {
		t.Fatalf("sqrt err = %v", err)
	}
	_, err = r.Calculate("power", []float64{-8, 0.5})
	if err == nil || !errors.Is(err, ErrDomain) || err.Error() != "cannot raise a negative number to a fractional power" {
		t.Fatalf("power err = %v", err)
	}
}

func TestOperationErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	r := NewRegistry(Operation{Name: "fail", Arity: 1, Fn: func([]float64) (float64, error) { return 0, boom }})
	if _, err := r.Calculate("fail", []float64{1}); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
