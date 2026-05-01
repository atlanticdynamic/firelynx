package calculation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculation_HandleHTTP(t *testing.T) {
	app := New(&Config{ID: "calc"})

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantResult float64
		wantError  string
	}{
		{name: "addition", body: `{"left":6,"right":2,"operator":"+"}`, wantStatus: http.StatusOK, wantResult: 8},
		{name: "subtraction", body: `{"left":6,"right":2,"operator":"-"}`, wantStatus: http.StatusOK, wantResult: 4},
		{name: "multiplication", body: `{"left":6,"right":2,"operator":"*"}`, wantStatus: http.StatusOK, wantResult: 12},
		{name: "division", body: `{"left":6,"right":2,"operator":"/"}`, wantStatus: http.StatusOK, wantResult: 3},
		{name: "missing operator", body: `{"left":6,"right":2}`, wantStatus: http.StatusBadRequest, wantError: "operator is required"},
		{name: "invalid operator", body: `{"left":6,"right":2,"operator":"%"}`, wantStatus: http.StatusBadRequest, wantError: "operator must be one of +, -, *, /"},
		{name: "divide by zero", body: `{"left":6,"right":0,"operator":"/"}`, wantStatus: http.StatusBadRequest, wantError: "division by zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/calc", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			err := app.HandleHTTP(t.Context(), rr, req)

			res := rr.Result()
			defer func() {
				require.NoError(t, res.Body.Close())
			}()
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

			var got Response
			require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
			if tt.wantError == "" {
				require.NoError(t, err)
				assert.InEpsilon(t, tt.wantResult, got.Result, 0.000001)
				assert.Empty(t, got.Error)
			} else {
				require.Error(t, err)
				assert.Contains(t, got.Error, tt.wantError)
			}
		})
	}
}

func TestCalculation_HandleHTTP_MethodNotAllowed(t *testing.T) {
	app := New(&Config{ID: "calc"})
	req := httptest.NewRequest(http.MethodGet, "/calc", nil)
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)

	require.Error(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Result().StatusCode)
}

func TestCalculation_String(t *testing.T) {
	assert.Equal(t, "calc", New(&Config{ID: "calc"}).String())
}

func TestCalculation_HandleHTTP_InvalidJSON(t *testing.T) {
	app := New(&Config{ID: "calc"})
	req := httptest.NewRequest(http.MethodPost, "/calc", bytes.NewBufferString("{not json"))
	rr := httptest.NewRecorder()

	err := app.HandleHTTP(t.Context(), rr, req)
	require.Error(t, err)

	res := rr.Result()
	defer func() {
		require.NoError(t, res.Body.Close())
	}()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var got Response
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Contains(t, got.Error, "invalid JSON request")
}
