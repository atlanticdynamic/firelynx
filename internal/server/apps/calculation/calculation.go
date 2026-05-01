package calculation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	operatorAdd      = "+"
	operatorSubtract = "-"
	operatorMultiply = "*"
	operatorDivide   = "/"
)

var (
	errMissingOperator = errors.New("operator is required")
	errInvalidOperator = errors.New("operator must be one of +, -, *, /")
	errDivisionByZero  = errors.New("division by zero")
)

// App is a calculation application that applies an operator to two operands.
type App struct {
	id string
}

// Request defines the typed input parameters for calculation requests.
type Request struct {
	Left     float64 `json:"left"     jsonschema:"Left operand"`
	Right    float64 `json:"right"    jsonschema:"Right operand"`
	Operator string  `json:"operator" jsonschema:"Operator to apply: +, -, *, /"`
}

// Response defines the typed output structure for calculation responses.
type Response struct {
	Result float64 `json:"result"          jsonschema:"Calculation result"`
	Error  string  `json:"error,omitempty"`
}

// New creates a new calculation app from a Config DTO.
func New(cfg *Config) *App {
	return &App{id: cfg.ID}
}

// String returns the unique identifier of the application.
func (a *App) String() string {
	return a.id
}

// HandleHTTP processes HTTP requests by applying an operator to two operands.
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if writeErr := writeCalculationError(w, http.StatusBadRequest, "invalid JSON request"); writeErr != nil {
			return writeErr
		}
		return nil
	}

	result, err := calculate(req)
	if err != nil {
		if writeErr := writeCalculationError(w, http.StatusBadRequest, err.Error()); writeErr != nil {
			return writeErr
		}
		return nil
	}

	if err := json.NewEncoder(w).Encode(Response{Result: result}); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

func calculate(req Request) (float64, error) {
	switch req.Operator {
	case "":
		return 0, errMissingOperator
	case operatorAdd:
		return req.Left + req.Right, nil
	case operatorSubtract:
		return req.Left - req.Right, nil
	case operatorMultiply:
		return req.Left * req.Right, nil
	case operatorDivide:
		if req.Right == 0 {
			return 0, errDivisionByZero
		}
		return req.Left / req.Right, nil
	default:
		return 0, fmt.Errorf("%w: %q", errInvalidOperator, req.Operator)
	}
}

func writeCalculationError(w http.ResponseWriter, status int, message string) error {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{Error: message}); err != nil {
		return fmt.Errorf("failed to encode error response: %w", err)
	}
	return nil
}
