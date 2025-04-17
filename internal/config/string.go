package config

// String returns a pretty-printed tree representation of the config
func (c *Config) String() string {
	return ConfigTree(c)
}