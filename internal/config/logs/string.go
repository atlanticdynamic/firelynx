package logs

import "fmt"

// String returns a string representation of the log configuration
func (lc *Config) String() string {
	return fmt.Sprintf("Log Config: format=%s, level=%s", lc.Format, lc.Level)
}
