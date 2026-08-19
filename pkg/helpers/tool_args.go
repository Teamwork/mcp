package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolArguments is a map of string keys to arbitrary values.
type ToolArguments map[string]any

// NewToolArguments creates a ToolArguments instance from a CallToolRequest.
func NewToolArguments(request *mcp.CallToolRequest) (ToolArguments, error) {
	var arguments map[string]any
	if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}
	return ToolArguments(arguments), nil
}

// GetString returns a string argument by key, or the default value if not
// found.
func (t ToolArguments) GetString(key string, defaultValue string) string {
	if val, ok := t[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// RequireString returns a string argument by key, or an error if not found or
// not a string.
func (t ToolArguments) RequireString(key string) (string, error) {
	if val, ok := t[key]; ok {
		if str, ok := val.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("argument %q is not a string", key)
	}
	return "", fmt.Errorf("required argument %q not found", key)
}

// GetInt returns an int argument by key, or the default value if not found.
func (t ToolArguments) GetInt(key string, defaultValue int) int {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// RequireInt returns an int argument by key, or an error if not found or not
// convertible to int.
func (t ToolArguments) RequireInt(key string) (int, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i, nil
			}
			return 0, fmt.Errorf("argument %q cannot be converted to int", key)
		default:
			return 0, fmt.Errorf("argument %q is not an int", key)
		}
	}
	return 0, fmt.Errorf("required argument %q not found", key)
}

// GetFloat returns a float64 argument by key, or the default value if not
// found.
func (t ToolArguments) GetFloat(key string, defaultValue float64) float64 {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return defaultValue
}

// RequireFloat returns a float64 argument by key, or an error if not found or
// not convertible to float64.
func (t ToolArguments) RequireFloat(key string) (float64, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f, nil
			}
			return 0, fmt.Errorf("argument %q cannot be converted to float64", key)
		default:
			return 0, fmt.Errorf("argument %q is not a float64", key)
		}
	}
	return 0, fmt.Errorf("required argument %q not found", key)
}

// GetBool returns a bool argument by key, or the default value if not found.
func (t ToolArguments) GetBool(key string, defaultValue bool) bool {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		case int:
			return v != 0
		case float64:
			return v != 0
		}
	}
	return defaultValue
}

// RequireBool returns a bool argument by key, or an error if not found or not
// convertible to bool.
func (t ToolArguments) RequireBool(key string) (bool, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b, nil
			}
			return false, fmt.Errorf("argument %q cannot be converted to bool", key)
		case int:
			return v != 0, nil
		case float64:
			return v != 0, nil
		default:
			return false, fmt.Errorf("argument %q is not a bool", key)
		}
	}
	return false, fmt.Errorf("required argument %q not found", key)
}

// GetStringSlice returns a string slice argument by key, or the default value
// if not found.
func (t ToolArguments) GetStringSlice(key string, defaultValue []string) []string {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []string:
			return v
		case []any:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireStringSlice returns a string slice argument by key, or an error if not
// found or not convertible to string slice.
func (t ToolArguments) RequireStringSlice(key string) ([]string, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []string:
			return v, nil
		case []any:
			result := make([]string, 0, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				} else {
					return nil, fmt.Errorf("item %d in argument %q is not a string", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not a string slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// GetIntSlice returns an int slice argument by key, or the default value if not
// found.
func (t ToolArguments) GetIntSlice(key string, defaultValue []int) []int {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []int:
			return v
		case []any:
			result := make([]int, 0, len(v))
			for _, item := range v {
				switch num := item.(type) {
				case int:
					result = append(result, num)
				case float64:
					result = append(result, int(num))
				case string:
					if i, err := strconv.Atoi(num); err == nil {
						result = append(result, i)
					}
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireIntSlice returns an int slice argument by key, or an error if not
// found or not convertible to int slice.
func (t ToolArguments) RequireIntSlice(key string) ([]int, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []int:
			return v, nil
		case []any:
			result := make([]int, 0, len(v))
			for i, item := range v {
				switch num := item.(type) {
				case int:
					result = append(result, num)
				case float64:
					result = append(result, int(num))
				case string:
					if i, err := strconv.Atoi(num); err == nil {
						result = append(result, i)
					} else {
						return nil, fmt.Errorf("item %d in argument %q cannot be converted to int", i, key)
					}
				default:
					return nil, fmt.Errorf("item %d in argument %q is not an int", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not an int slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// GetFloatSlice returns a float64 slice argument by key, or the default value
// if not found.
func (t ToolArguments) GetFloatSlice(key string, defaultValue []float64) []float64 {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []float64:
			return v
		case []any:
			result := make([]float64, 0, len(v))
			for _, item := range v {
				switch num := item.(type) {
				case float64:
					result = append(result, num)
				case int:
					result = append(result, float64(num))
				case string:
					if f, err := strconv.ParseFloat(num, 64); err == nil {
						result = append(result, f)
					}
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireFloatSlice returns a float64 slice argument by key, or an error if not
// found or not convertible to float64 slice.
func (t ToolArguments) RequireFloatSlice(key string) ([]float64, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []float64:
			return v, nil
		case []any:
			result := make([]float64, 0, len(v))
			for i, item := range v {
				switch num := item.(type) {
				case float64:
					result = append(result, num)
				case int:
					result = append(result, float64(num))
				case string:
					if f, err := strconv.ParseFloat(num, 64); err == nil {
						result = append(result, f)
					} else {
						return nil, fmt.Errorf("item %d in argument %q cannot be converted to float64", i, key)
					}
				default:
					return nil, fmt.Errorf("item %d in argument %q is not a float64", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not a float64 slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// GetBoolSlice returns a bool slice argument by key, or the default value if
// not found.
func (t ToolArguments) GetBoolSlice(key string, defaultValue []bool) []bool {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []bool:
			return v
		case []any:
			result := make([]bool, 0, len(v))
			for _, item := range v {
				switch b := item.(type) {
				case bool:
					result = append(result, b)
				case string:
					if parsed, err := strconv.ParseBool(b); err == nil {
						result = append(result, parsed)
					}
				case int:
					result = append(result, b != 0)
				case float64:
					result = append(result, b != 0)
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireBoolSlice returns a bool slice argument by key, or an error if not
// found or not convertible to bool slice.
func (t ToolArguments) RequireBoolSlice(key string) ([]bool, error) {
	if val, ok := t[key]; ok {
		switch v := val.(type) {
		case []bool:
			return v, nil
		case []any:
			result := make([]bool, 0, len(v))
			for i, item := range v {
				switch b := item.(type) {
				case bool:
					result = append(result, b)
				case string:
					if parsed, err := strconv.ParseBool(b); err == nil {
						result = append(result, parsed)
					} else {
						return nil, fmt.Errorf("item %d in argument %q cannot be converted to bool", i, key)
					}
				case int:
					result = append(result, b != 0)
				case float64:
					result = append(result, b != 0)
				default:
					return nil, fmt.Errorf("item %d in argument %q is not a bool", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not a bool slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}
