package calculation

import (
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculation_MCPToolOption_Registers(t *testing.T) {
	app := New(&Config{ID: "calc"})
	opt := app.MCPToolOption(app.MCPToolName())
	require.NotNil(t, opt)

	h, err := mcpio.NewHandler(opt, mcpio.WithName("test"))
	require.NoError(t, err)
	require.NotNil(t, h)
}

func TestCalculation_ToolFunc(t *testing.T) {
	app := New(&Config{ID: "calc"})

	tests := []struct {
		name       string
		input      Request
		wantResult float64
		wantErr    string
	}{
		{name: "addition", input: Request{Left: 6, Right: 2, Operator: "+"}, wantResult: 8},
		{name: "subtraction", input: Request{Left: 6, Right: 2, Operator: "-"}, wantResult: 4},
		{name: "multiplication", input: Request{Left: 6, Right: 2, Operator: "*"}, wantResult: 12},
		{name: "division", input: Request{Left: 6, Right: 2, Operator: "/"}, wantResult: 3},
		{name: "missing operator", input: Request{Left: 6, Right: 2}, wantErr: "operator is required"},
		{name: "invalid operator", input: Request{Left: 6, Right: 2, Operator: "%"}, wantErr: "operator must be one of +, -, *, /"},
		{name: "divide by zero", input: Request{Left: 6, Right: 0, Operator: "/"}, wantErr: "division by zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := app.calculateToolFunc(t.Context(), nil, tt.input)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.InEpsilon(t, tt.wantResult, out.Result, 0.000001)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
