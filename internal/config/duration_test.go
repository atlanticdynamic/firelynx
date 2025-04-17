package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_String(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{"Zero", 0, "0s"},
		{"Seconds", Duration(5 * time.Second), "5s"},
		{"Minutes", Duration(10 * time.Minute), "10m0s"},
		{"Hours", Duration(2 * time.Hour), "2h0m0s"},
		{"Milliseconds", Duration(500 * time.Millisecond), "500ms"},
		{"Mixed", Duration(1*time.Hour + 30*time.Minute + 45*time.Second), "1h30m45s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.duration.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDuration_Milliseconds(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected int64
	}{
		{"Zero", 0, 0},
		{"Seconds", Duration(5 * time.Second), 5000},
		{"Minutes", Duration(2 * time.Minute), 120000},
		{"Milliseconds", Duration(500 * time.Millisecond), 500},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.duration.Milliseconds()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDuration_TimeUnits(t *testing.T) {
	duration := Duration(90 * time.Minute)

	assert.Equal(t, 5400.0, duration.Seconds())
	assert.Equal(t, 90.0, duration.Minutes())
	assert.Equal(t, 1.5, duration.Hours())
}

func TestFromDuration(t *testing.T) {
	timeDuration := 5 * time.Minute
	result := FromDuration(timeDuration)

	assert.Equal(t, Duration(timeDuration), result)
	assert.Equal(t, "5m0s", result.String())
}

func TestDuration_AsDuration(t *testing.T) {
	configDuration := Duration(30 * time.Second)
	result := configDuration.AsDuration()

	assert.Equal(t, time.Duration(configDuration), result)
	assert.Equal(t, 30*time.Second, result)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   Duration
		shouldFail bool
	}{
		{"Seconds", "30s", Duration(30 * time.Second), false},
		{"Minutes", "5m", Duration(5 * time.Minute), false},
		{"Hours", "2h", Duration(2 * time.Hour), false},
		{"Mixed", "1h30m", Duration(90 * time.Minute), false},
		{"Milliseconds", "500ms", Duration(500 * time.Millisecond), false},
		{"Invalid", "not a duration", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseDuration(tc.input)

			if tc.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
