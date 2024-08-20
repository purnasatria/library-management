package env

import (
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// Get retrieves the value of the environment variable named by the key.
// If the variable is not present, it returns the defaultValue.
func Get(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetInt retrieves the value of the environment variable named by the key as an integer.
// If the variable is not present or cannot be parsed as an integer, it returns the defaultValue.
func GetInt(key string, defaultValue int) int {
	valueStr := Get(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	log.Warn().Str("key", key).Msg("Failed to parse env var as int, using default")
	return defaultValue
}

// GetDuration retrieves the value of the environment variable named by the key as a duration.
// If the variable is not present or cannot be parsed as a duration, it returns the defaultValue.
func GetDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := Get(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	log.Warn().Str("key", key).Msg("Failed to parse env var as duration, using default")
	return defaultValue
}

// GetBool retrieves the value of the environment variable named by the key as a boolean.
// If the variable is not present or cannot be parsed as a boolean, it returns the defaultValue.
func GetBool(key string, defaultValue bool) bool {
	valueStr := Get(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	log.Warn().Str("key", key).Msg("Failed to parse env var as bool, using default")
	return defaultValue
}
