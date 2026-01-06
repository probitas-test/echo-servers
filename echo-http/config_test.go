package main

import (
	"os"
	"reflect"
	"testing"
)

func TestParseScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single scope",
			input:    "openid",
			expected: []string{"openid"},
		},
		{
			name:     "multiple scopes",
			input:    "openid,profile,email",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "scopes with spaces",
			input:    " openid , profile , email ",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single scope with spaces",
			input:    "  openid  ",
			expected: []string{"openid"},
		},
		{
			name:     "multiple scopes with extra commas",
			input:    "openid,,profile,,email",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "trailing comma",
			input:    "openid,profile,",
			expected: []string{"openid", "profile"},
		},
		{
			name:     "leading comma",
			input:    ",openid,profile",
			expected: []string{"openid", "profile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseScopes(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseScopes(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		setEnv       bool
		defaultValue bool
		expected     bool
	}{
		{
			name:         "env not set returns default false",
			key:          "TEST_BOOL_UNSET",
			setEnv:       false,
			defaultValue: false,
			expected:     false,
		},
		{
			name:         "env not set returns default true",
			key:          "TEST_BOOL_UNSET",
			setEnv:       false,
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "env set to true",
			key:          "TEST_BOOL_TRUE",
			envValue:     "true",
			setEnv:       true,
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "env set to 1",
			key:          "TEST_BOOL_ONE",
			envValue:     "1",
			setEnv:       true,
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "env set to false",
			key:          "TEST_BOOL_FALSE",
			envValue:     "false",
			setEnv:       true,
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "env set to 0",
			key:          "TEST_BOOL_ZERO",
			envValue:     "0",
			setEnv:       true,
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "env set to invalid value returns false",
			key:          "TEST_BOOL_INVALID",
			envValue:     "invalid",
			setEnv:       true,
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "env set to empty string returns default",
			key:          "TEST_BOOL_EMPTY",
			envValue:     "",
			setEnv:       true,
			defaultValue: true,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				_ = os.Setenv(tt.key, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.key) }()
			}

			result := getBoolEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getBoolEnv(%q, %v) = %v, want %v (env=%q)",
					tt.key, tt.defaultValue, result, tt.expected, tt.envValue)
			}
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		setEnv       bool
		defaultValue int
		expected     int
	}{
		{
			name:         "env not set returns default",
			key:          "TEST_INT_UNSET",
			setEnv:       false,
			defaultValue: 42,
			expected:     42,
		},
		{
			name:         "env set to valid integer",
			key:          "TEST_INT_VALID",
			envValue:     "100",
			setEnv:       true,
			defaultValue: 42,
			expected:     100,
		},
		{
			name:         "env set to zero",
			key:          "TEST_INT_ZERO",
			envValue:     "0",
			setEnv:       true,
			defaultValue: 42,
			expected:     0,
		},
		{
			name:         "env set to negative integer",
			key:          "TEST_INT_NEGATIVE",
			envValue:     "-10",
			setEnv:       true,
			defaultValue: 42,
			expected:     -10,
		},
		{
			name:         "env set to invalid value returns default",
			key:          "TEST_INT_INVALID",
			envValue:     "not-a-number",
			setEnv:       true,
			defaultValue: 42,
			expected:     42,
		},
		{
			name:         "env set to empty string returns default",
			key:          "TEST_INT_EMPTY",
			envValue:     "",
			setEnv:       true,
			defaultValue: 42,
			expected:     42,
		},
		{
			name:         "env set to float returns default",
			key:          "TEST_INT_FLOAT",
			envValue:     "3.14",
			setEnv:       true,
			defaultValue: 42,
			expected:     42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				_ = os.Setenv(tt.key, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.key) }()
			}

			result := getIntEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getIntEnv(%q, %d) = %d, want %d (env=%q)",
					tt.key, tt.defaultValue, result, tt.expected, tt.envValue)
			}
		})
	}
}
