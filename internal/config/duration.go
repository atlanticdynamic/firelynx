package config

import (
	"time"
)

// Duration wraps time.Duration for configuration purposes
type Duration time.Duration

// String returns the string representation of Duration
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Milliseconds returns the duration as milliseconds
func (d Duration) Milliseconds() int64 {
	return time.Duration(d).Milliseconds()
}

// Seconds returns the duration as seconds
func (d Duration) Seconds() float64 {
	return time.Duration(d).Seconds()
}

// Minutes returns the duration as minutes
func (d Duration) Minutes() float64 {
	return time.Duration(d).Minutes()
}

// Hours returns the duration as hours
func (d Duration) Hours() float64 {
	return time.Duration(d).Hours()
}

// FromDuration creates a config.Duration from a time.Duration
func FromDuration(d time.Duration) Duration {
	return Duration(d)
}

// AsDuration converts a config.Duration to a time.Duration
func (d Duration) AsDuration() time.Duration {
	return time.Duration(d)
}

// ParseDuration parses a duration string and returns a config.Duration
func ParseDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return Duration(d), nil
}
