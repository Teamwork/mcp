package helpers

import (
	"encoding"
	"fmt"
	"reflect"
	"slices"
	"time"

	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// ParamGroup applies a series of functions to a map of parameters.
func ParamGroup(params map[string]any, funcs ...ParamFunc) error {
	for _, fn := range funcs {
		if err := fn(params); err != nil {
			return fmt.Errorf("error binding parameter: %w", err)
		}
	}
	return nil
}

// ParamFunc defines a function type that takes a map of parameters and
// returns an error. This is used to define functions that can retrieve
// parameters from a map, converting them to a specific type and applying
// middleware functions if necessary.
type ParamFunc func(map[string]any) error

// ParamMiddleware defines a function type that takes a pointer to a specific
// type and returns a boolean indicating whether to continue processing and an
// error if any issue occurs. This is used to apply middleware functions to
// parameters before they are set to the target.
type ParamMiddleware[T any] func(*T) (bool, error)

// RequiredParam retrieves a required parameter from a map, converting it to the
// specified type. It returns an error if the key is not found or if the type
// conversion fails. If the target is nil, it returns an error. It also allows
// for middleware functions to be applied to the value before setting it to the
// target. Each middleware function should return a boolean indicating whether
// to continue processing and an error if any issue occurs.
func RequiredParam[T any](target *T, key string, middlewares ...ParamMiddleware[T]) ParamFunc {
	return func(params map[string]any) error {
		return param(params, target, key, false, middlewares...)
	}
}

// OptionalParam retrieves an optional parameter from a map, converting it to
// the specified type. It returns an error if the type conversion fails. If the
// target is nil, it returns an error. It also allows for middleware functions
// to be applied to the value before setting it to the target. Each middleware
// function should return a boolean indicating whether to continue processing
// and an error if any issue occurs.
func OptionalParam[T any](target *T, key string, middlewares ...ParamMiddleware[T]) ParamFunc {
	return func(params map[string]any) error {
		return param(params, target, key, true, middlewares...)
	}
}

// OptionalPointerParam retrieves an optional parameter from a map and sets
// it to a pointer target. It converts the value to the specified type and
// applies middleware functions to the value before setting it. If the target
// is nil, it returns an error. The middleware functions should return a
// boolean indicating whether to continue processing and an error if any issue
// occurs. If the parameter is not found, it does not set the target pointer.
func OptionalPointerParam[T any](target **T, key string, middlewares ...ParamMiddleware[T]) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		var temp T
		var set bool
		middlewares = append(middlewares, func(*T) (bool, error) { set = true; return true, nil })
		if err := param(params, &temp, key, true, middlewares...); err != nil {
			return err
		}
		if set {
			*target = &temp
		}
		return nil
	}
}

func param[T any](
	params map[string]any,
	target *T,
	key string,
	optional bool,
	middlewares ...ParamMiddleware[T],
) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}
	value, ok := params[key]
	if !ok {
		if optional {
			return nil
		}
		return fmt.Errorf("parameter %s is required", key)
	}
	v, ok := value.(T)
	if !ok {
		// attempt to convert from string if the type is only an alias of string and
		// the value is a string
		var fallback bool
		if vStr, okStr := value.(string); okStr {
			vv := reflect.ValueOf(value)
			if vv.Kind() == reflect.Pointer {
				vv = vv.Elem()
			}
			if vv.Kind() == reflect.String {
				v = reflect.ValueOf(vStr).Convert(reflect.TypeOf(v)).Interface().(T)
				fallback = true
			}
		}
		if !fallback {
			return fmt.Errorf("invalid type for %s: expected %T, got %T", key, *target, value)
		}
	}
	for _, middleware := range middlewares {
		var err error
		if ok, err = middleware(&v); err != nil || !ok {
			return err
		}
	}
	*target = v
	return nil
}

// RequiredNumericParam retrieves a required numeric parameter from a map,
// converting it to the target numeric type. It returns an error if the key is
// not found or if the type conversion fails. If the target is nil, it returns
// an error.
func RequiredNumericParam[T int8 | int16 | int32 | int64 |
	uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	projects.LegacyNumber](
	target *T,
	key string,
	middlewares ...ParamMiddleware[T],
) ParamFunc {
	return func(params map[string]any) error {
		return numericParam(params, target, key, false, middlewares...)
	}
}

// OptionalNumericParam retrieves an optional numeric parameter from a map,
// converting it to the target numeric type. It returns an error if the type
// conversion fails. If the target is nil, it returns an error.
func OptionalNumericParam[T int8 | int16 | int32 | int64 |
	uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	projects.LegacyNumber](
	target *T,
	key string,
	middlewares ...ParamMiddleware[T],
) ParamFunc {
	return func(params map[string]any) error {
		return numericParam(params, target, key, true, middlewares...)
	}
}

// OptionalNumericPointerParam retrieves an optional numeric parameter from a
// map and sets it to a pointer target. It converts the value to the specified
// numeric type and applies middleware functions to the value before setting it.
// If the target is nil, it returns an error.
func OptionalNumericPointerParam[T int8 | int16 | int32 | int64 |
	uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	projects.LegacyNumber](
	target **T,
	key string,
	middlewares ...ParamMiddleware[T],
) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		var temp T
		var set bool
		middlewares = append(middlewares, func(*T) (bool, error) { set = true; return true, nil })
		if err := numericParam(params, &temp, key, true, middlewares...); err != nil {
			return err
		}
		if set {
			*target = &temp
		}
		return nil
	}
}

func numericParam[T int8 | int16 | int32 | int64 |
	uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	projects.LegacyNumber](
	params map[string]any,
	target *T,
	key string,
	optional bool,
	middlewares ...ParamMiddleware[T],
) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}
	value, ok := params[key]
	if !ok {
		if optional {
			return nil
		}
		return fmt.Errorf("parameter %s is required", key)
	}
	v, ok := value.(float64)
	if !ok {
		return fmt.Errorf("invalid type for %s: expected %T, got %T", key, *target, value)
	}
	vType := T(v)
	for _, middleware := range middlewares {
		var err error
		if ok, err = middleware(&vType); err != nil || !ok {
			return err
		}
	}
	*target = vType
	return nil
}

// RequiredTimeParam retrieves a required time parameter from a map, converting
// it to a time.Time type. It returns an error if the key is not found or if the
// type conversion fails. If the target is nil, it returns an error.
func RequiredTimeParam(
	target *time.Time,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return timeParam(params, target, key, false, middlewares...)
	}
}

// OptionalTimeParam retrieves an optional time parameter from a map, converting
// it to a time.Time type. It returns an error if the type conversion fails. If
// the target is nil, it returns an error.
func OptionalTimeParam(
	target *time.Time,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return timeParam(params, target, key, true, middlewares...)
	}
}

// OptionalTimePointerParam retrieves an optional time parameter from a map and
// sets it to a pointer target. It converts the value to a time.Time type and
// applies middleware functions to the value before setting it. If the target is
// nil, it returns an error.
func OptionalTimePointerParam(
	target **time.Time,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		var temp time.Time
		var set bool
		middlewares = append(middlewares, func(*string) (bool, error) { set = true; return true, nil })
		if err := timeParam(params, &temp, key, true, middlewares...); err != nil {
			return err
		}
		if set {
			*target = &temp
		}
		return nil
	}
}

func timeParam(
	params map[string]any,
	target *time.Time,
	key string,
	optional bool,
	middlewares ...ParamMiddleware[string],
) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}
	value, ok := params[key]
	if !ok {
		if optional {
			return nil
		}
		return fmt.Errorf("parameter %s is required", key)
	}
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for %s: expected string, got %T", key, value)
	}
	for _, middleware := range middlewares {
		var err error
		if ok, err = middleware(&v); err != nil || !ok {
			return err
		}
	}
	var err error
	*target, err = time.Parse(time.RFC3339, v)
	if err != nil {
		return fmt.Errorf("invalid time format for %s: %w", key, err)
	}
	return nil
}

// RequiredTimeOnlyParam retrieves a required time parameter from a map,
// converting it to a twapi.Time type. It returns an error if the key is not
// found or if the type conversion fails. If the target is nil, it returns an
// error.
func RequiredTimeOnlyParam(
	target *twapi.Time,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return timeOnlyParam(params, target, key, false, middlewares...)
	}
}

// OptionalTimeOnlyParam retrieves an optional time parameter from a map,
// converting it to a twapi.Time type. It returns an error if the type
// conversion fails. If the target is nil, it returns an error.
func OptionalTimeOnlyParam(
	target *twapi.Time,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return timeOnlyParam(params, target, key, true, middlewares...)
	}
}

// OptionalTimeOnlyPointerParam retrieves an optional time parameter from a map
// and sets it to a pointer target. It converts the value to a twapi.Time
// type and applies middleware functions to the value before setting it. If the
// target is nil, it returns an error.
func OptionalTimeOnlyPointerParam(
	target **twapi.Time,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		var temp twapi.Time
		var set bool
		middlewares = append(middlewares, func(*string) (bool, error) { set = true; return true, nil })
		if err := timeOnlyParam(params, &temp, key, true, middlewares...); err != nil {
			return err
		}
		if set {
			*target = &temp
		}
		return nil
	}
}

func timeOnlyParam(
	params map[string]any,
	target *twapi.Time,
	key string,
	optional bool,
	middlewares ...ParamMiddleware[string],
) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}
	value, ok := params[key]
	if !ok {
		if optional {
			return nil
		}
		return fmt.Errorf("parameter %s is required", key)
	}
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for %s: expected string, got %T", key, value)
	}
	for _, middleware := range middlewares {
		var err error
		if ok, err = middleware(&v); err != nil || !ok {
			return err
		}
	}
	t, err := time.Parse("15:04:05", v)
	if err != nil {
		return fmt.Errorf("invalid time-only format for %s: %w", key, err)
	}
	*target = twapi.Time(t)
	return nil
}

// RequiredDateParam retrieves a required date parameter from a map, converting
// it to a twapi.Date type. It returns an error if the key is not found or if
// the type conversion fails. The date format is expected to be "YYYY-MM-DD". If
// the target is nil, it returns an error.
func RequiredDateParam(
	target *twapi.Date,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return dateParam(params, target, key, false, middlewares...)
	}
}

// OptionalDateParam retrieves an optional date parameter from a map, converting
// it to a twapi.Date type. It returns an error if the type conversion fails.
// The date format is expected to be "YYYY-MM-DD". If the target is nil, it
// returns an error. If the key is not found, it does not set the target.
func OptionalDateParam(
	target *twapi.Date,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return dateParam(params, target, key, true, middlewares...)
	}
}

// OptionalDatePointerParam retrieves an optional date parameter from a map and
// sets it to a pointer target. It converts the value to a twapi.Date type
// and applies middleware functions to the value before setting it. The date
// format is expected to be "YYYY-MM-DD". If the target is nil, it returns an
// error.
func OptionalDatePointerParam(
	target **twapi.Date,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		var temp twapi.Date
		var set bool
		middlewares = append(middlewares, func(*string) (bool, error) { set = true; return true, nil })
		if err := dateParam(params, &temp, key, true, middlewares...); err != nil {
			return err
		}
		if set {
			*target = &temp
		}
		return nil
	}
}

func dateParam(
	params map[string]any,
	target *twapi.Date,
	key string,
	optional bool,
	middlewares ...ParamMiddleware[string],
) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}
	value, ok := params[key]
	if !ok {
		if optional {
			return nil
		}
		return fmt.Errorf("parameter %s is required", key)
	}
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for %s: expected string, got %T", key, value)
	}
	for _, middleware := range middlewares {
		var err error
		if ok, err = middleware(&v); err != nil || !ok {
			return err
		}
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return fmt.Errorf("invalid date format for %s: %w", key, err)
	}
	*target = twapi.Date(t)
	return nil
}

// RequiredLegacyDateParam retrieves a required legacy date parameter from a
// map, converting it to a projects.LegacyDate type. It returns an error if the
// key is not found or if the type conversion fails. The date format is expected
// to be "YYYYMMDD". If the target is nil, it returns an error.
func RequiredLegacyDateParam(
	target *projects.LegacyDate,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return legacyDateParam(params, target, key, false, middlewares...)
	}
}

// OptionalLegacyDateParam retrieves an optional legacy date parameter from a
// map, converting it to a projects.LegacyDate type. It returns an error if the
// type conversion fails. The date format is expected to be "YYYYMMDD". If the
// target is nil, it returns an error. If the key is not found, it does not set
// the target.
func OptionalLegacyDateParam(
	target *projects.LegacyDate,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		return legacyDateParam(params, target, key, true, middlewares...)
	}
}

// OptionalLegacyDatePointerParam retrieves an optional date parameter from a
// map and sets it to a pointer target. It converts the value to a
// projects.LegacyDate type and applies middleware functions to the value before
// setting it. The date format is expected to be "YYYYMMDD". If the target is
// nil, it returns an error.
func OptionalLegacyDatePointerParam(
	target **projects.LegacyDate,
	key string,
	middlewares ...ParamMiddleware[string],
) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		var temp projects.LegacyDate
		var set bool
		middlewares = append(middlewares, func(*string) (bool, error) { set = true; return true, nil })
		if err := legacyDateParam(params, &temp, key, true, middlewares...); err != nil {
			return err
		}
		if set {
			*target = &temp
		}
		return nil
	}
}

func legacyDateParam(
	params map[string]any,
	target *projects.LegacyDate,
	key string,
	optional bool,
	middlewares ...ParamMiddleware[string],
) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}
	value, ok := params[key]
	if !ok {
		if optional {
			return nil
		}
		return fmt.Errorf("parameter %s is required", key)
	}
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for %s: expected string, got %T", key, value)
	}
	for _, middleware := range middlewares {
		var err error
		if ok, err = middleware(&v); err != nil || !ok {
			return err
		}
	}
	t, err := time.Parse("20060102", v)
	if err != nil {
		return fmt.Errorf("invalid date format for %s: %w", key, err)
	}
	*target = projects.LegacyDate(t)
	return nil
}

// OptionalListParam retrieves an optional list parameter from a map, converting
// each item to the specified type. It returns an error if the key is not found
// or if the type conversion fails. If the target is nil, it returns an error.
func OptionalListParam[T any](target *[]T, key string) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		value, ok := params[key]
		if !ok {
			return nil
		}
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("invalid type for %s: expected []any, got %T", key, value)
		}
		*target = make([]T, 0, len(array))
		for _, item := range array {
			var zero T

			// check if the type implements encoding.TextUnmarshaler
			zeroPointer := reflect.New(reflect.TypeOf(zero))
			if decoder, ok := zeroPointer.Interface().(encoding.TextUnmarshaler); ok {
				var input []byte
				var inputOK bool
				switch item := item.(type) {
				case string:
					input = []byte(item)
					inputOK = true
				case []byte:
					input = item
					inputOK = true
				}
				if inputOK {
					if err := decoder.UnmarshalText(input); err != nil {
						return fmt.Errorf("failed to decode %v: %w", item, err)
					}
					*target = append(*target, zeroPointer.Elem().Interface().(T))
					continue
				}
			}

			v, ok := item.(T)
			if !ok {
				return fmt.Errorf("invalid type in %s: expected %T, got %T", key, zero, item)
			}
			*target = append(*target, v)
		}
		return nil
	}
}

// OptionalNumericListParam retrieves an optional list of numeric parameters
// from a map, converting each item to the specified numeric type. It returns
// an error if the key is not found or if the type conversion fails. If the
// target is nil, it returns an error.
func OptionalNumericListParam[T int8 | int16 | int32 | int64 |
	uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	projects.LegacyNumber](
	target *[]T, key string,
) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		value, ok := params[key]
		if !ok {
			return nil
		}
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("invalid type for %s: expected []any, got %T", key, value)
		}
		*target = make([]T, 0, len(array))
		for _, item := range array {
			v, ok := item.(float64)
			if !ok {
				return fmt.Errorf("invalid type in %s: expected float64, got %T", key, item)
			}
			*target = append(*target, T(v))
		}
		return nil
	}
}

// OptionalCustomNumericListParam retrieves an optional list of numeric
// parameters from a map, converting each item to the specified numeric type
// using a custom type that implements the Add method. It returns an error if
// the key is not found or if the type conversion fails.
func OptionalCustomNumericListParam[T interface{ Add(float64) }](target T, key string) ParamFunc {
	return func(params map[string]any) error {
		value, ok := params[key]
		if !ok {
			return nil
		}
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("invalid type for %s: expected []any, got %T", key, value)
		}
		for _, item := range array {
			v, ok := item.(float64)
			if !ok {
				return fmt.Errorf("invalid type in %s: expected float64, got %T", key, item)
			}
			target.Add(v)
		}
		return nil
	}
}

// RestrictValues restricts the values of a parameter to a predefined set of
// allowed values. It can be used as a middleware function in the Param or
// OptionalParam functions.
func RestrictValues[T comparable](allowedValues ...T) ParamMiddleware[T] {
	return func(value *T) (bool, error) {
		if value == nil {
			return true, nil
		}
		if slices.Contains(allowedValues, *value) {
			return true, nil
		}
		return false, fmt.Errorf("value %v is not allowed, must be one of %v", *value, allowedValues)
	}
}
