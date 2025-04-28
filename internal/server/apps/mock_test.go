package apps

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockApp(t *testing.T) {
	// Create a mock app with preset ID
	mockApp := NewMockApp("test-app")

	// Verify ID is set correctly
	assert.Equal(t, "test-app", mockApp.ID())

	// Test HandleHTTP behavior
	ctx := context.Background()
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	params := map[string]any{"key": "value"}

	// Set expectation for HandleHTTP to be called with specific arguments
	// and return a specific error
	expectedError := errors.New("test error")
	mockApp.On("HandleHTTP", ctx, w, r, params).Return(expectedError).Once()

	// Call the method
	result := mockApp.HandleHTTP(ctx, w, r, params)

	// Assert expectations
	assert.Equal(t, expectedError, result)
	mockApp.AssertExpectations(t)

	// Test with custom behavior
	customMock := &MockApp{}
	customMock.On("ID").Return("custom-id")
	customMock.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Extract arguments
			writer := args.Get(1).(http.ResponseWriter)
			// Set custom response
			writer.WriteHeader(http.StatusOK)
			_, err = writer.Write([]byte("OK"))
			require.NoError(t, err)
		}).
		Return(nil)

	// Verify ID
	assert.Equal(t, "custom-id", customMock.ID())

	// Test HandleHTTP custom behavior
	newRecorder := httptest.NewRecorder()
	err = customMock.HandleHTTP(ctx, newRecorder, r, nil)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, newRecorder.Code)
	assert.Equal(t, "OK", newRecorder.Body.String())
}
