package sdk

import "fmt"

// Config provides type-safe access to resource configuration
// Platform provisioners use this interface to read configuration parameters
type Config interface {
	// Get retrieves a configuration value by key
	// Returns nil if key doesn't exist
	Get(key string) interface{}

	// GetString retrieves a string value
	// Returns empty string if key doesn't exist or value is not a string
	GetString(key string) string

	// GetInt retrieves an integer value
	// Returns 0 if key doesn't exist or value is not an integer
	GetInt(key string) int

	// GetBool retrieves a boolean value
	// Returns false if key doesn't exist or value is not a boolean
	GetBool(key string) bool

	// GetFloat retrieves a float64 value
	// Returns 0.0 if key doesn't exist or value is not a number
	GetFloat(key string) float64

	// GetMap retrieves a map value
	// Returns empty map if key doesn't exist or value is not a map
	GetMap(key string) map[string]interface{}

	// GetSlice retrieves a slice value
	// Returns empty slice if key doesn't exist or value is not a slice
	GetSlice(key string) []interface{}

	// Has checks if a key exists in the configuration
	Has(key string) bool

	// Keys returns all configuration keys
	Keys() []string

	// AsMap returns the entire configuration as a map
	AsMap() map[string]interface{}
}

// MapConfig implements Config interface backed by a map
type MapConfig struct {
	data map[string]interface{}
}

// NewMapConfig creates a new MapConfig from a map
func NewMapConfig(data map[string]interface{}) *MapConfig {
	if data == nil {
		data = make(map[string]interface{})
	}
	return &MapConfig{data: data}
}

// Get retrieves a configuration value by key
func (c *MapConfig) Get(key string) interface{} {
	return c.data[key]
}

// GetString retrieves a string value
func (c *MapConfig) GetString(key string) string {
	if v, ok := c.data[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt retrieves an integer value
func (c *MapConfig) GetInt(key string) int {
	if v, ok := c.data[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

// GetBool retrieves a boolean value
func (c *MapConfig) GetBool(key string) bool {
	if v, ok := c.data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetFloat retrieves a float64 value
func (c *MapConfig) GetFloat(key string) float64 {
	if v, ok := c.data[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0.0
}

// GetMap retrieves a map value
func (c *MapConfig) GetMap(key string) map[string]interface{} {
	if v, ok := c.data[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return make(map[string]interface{})
}

// GetSlice retrieves a slice value
func (c *MapConfig) GetSlice(key string) []interface{} {
	if v, ok := c.data[key]; ok {
		if s, ok := v.([]interface{}); ok {
			return s
		}
	}
	return make([]interface{}, 0)
}

// Has checks if a key exists in the configuration
func (c *MapConfig) Has(key string) bool {
	_, ok := c.data[key]
	return ok
}

// Keys returns all configuration keys
func (c *MapConfig) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// AsMap returns the entire configuration as a map
func (c *MapConfig) AsMap() map[string]interface{} {
	// Return a copy to prevent external modifications
	result := make(map[string]interface{}, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// Set sets a configuration value (for testing and internal use)
func (c *MapConfig) Set(key string, value interface{}) {
	c.data[key] = value
}

// String returns a string representation of the configuration
func (c *MapConfig) String() string {
	return fmt.Sprintf("MapConfig{%v}", c.data)
}
