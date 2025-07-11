package interpolation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// InterpolateStruct applies environment variable interpolation to fields tagged with `env_interpolation:"yes"`.
// This function modifies the provided struct in place and builds detailed error context with field paths.
func InterpolateStruct(v any) error {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		return interpolateRecursive(val, "")
	}

	// Handle non-pointer types by checking if they're structs
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct or pointer to struct, got %T", v)
	}

	// For struct values, check if addressable first
	if !val.CanAddr() {
		return fmt.Errorf("cannot interpolate non-addressable struct value, pass a pointer instead")
	}

	// For struct values, get address to pass as pointer
	return interpolateRecursive(val.Addr(), "")
}

// interpolateRecursive handles the actual interpolation work with proper error context building
func interpolateRecursive(val reflect.Value, fieldPath string) error {
	// Dereference pointer to get the actual element
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("expected a struct, got %T", val.Interface())
	}

	typ := elem.Type()
	var errs []error

	// Helper function to build field path for error context
	buildFieldPath := func(fieldName string) string {
		if fieldPath == "" {
			return fieldName
		}
		return fieldPath + "." + fieldName
	}

	// Helper function to handle recursive calls with proper error context
	processRecursively := func(fieldVal reflect.Value, currentPath string) error {
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				return nil
			}
			return interpolateRecursive(fieldVal, currentPath)
		}
		// For struct values, get address to pass as pointer
		return interpolateRecursive(fieldVal.Addr(), currentPath)
	}

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := typ.Field(i)
		currentPath := buildFieldPath(fieldType.Name)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Check for env_interpolation tag
		tag := strings.ToLower(fieldType.Tag.Get("env_interpolation"))
		if tag != "yes" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			if original := field.String(); original != "" {
				interpolated, err := ExpandEnvVarsWithDefaults(original)
				if err != nil {
					errs = append(errs, fmt.Errorf("field %s: %w", currentPath, err))
				} else {
					field.SetString(interpolated)
				}
			}

		case reflect.Map:
			// Handle map[string]string fields
			if field.Type().Key().Kind() == reflect.String &&
				field.Type().Elem().Kind() == reflect.String && !field.IsNil() {
				// Collect keys first to avoid modification during iteration
				keys := field.MapKeys()
				for _, key := range keys {
					value := field.MapIndex(key)
					interpolated, err := ExpandEnvVarsWithDefaults(value.String())
					mapPath := fmt.Sprintf("%s[%s]", currentPath, key.String())

					if err != nil {
						errs = append(errs, fmt.Errorf("field %s: %w", mapPath, err))
					} else if interpolated != value.String() {
						// Only update if value changed
						field.SetMapIndex(key, reflect.ValueOf(interpolated))
					}
				}
			}

		case reflect.Struct:
			if err := processRecursively(field, currentPath); err != nil {
				errs = append(errs, err)
			}

		case reflect.Ptr:
			if field.Type().Elem().Kind() == reflect.Struct && !field.IsNil() {
				if err := processRecursively(field, currentPath); err != nil {
					errs = append(errs, err)
				}
			}

		case reflect.Slice:
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				slicePath := fmt.Sprintf("%s[%d]", currentPath, j)

				if elem.Kind() == reflect.String {
					if original := elem.String(); original != "" {
						interpolated, err := ExpandEnvVarsWithDefaults(original)
						if err != nil {
							errs = append(errs, fmt.Errorf("field %s: %w", slicePath, err))
						} else {
							elem.SetString(interpolated)
						}
					}
				} else if elem.Kind() == reflect.Struct {
					if err := processRecursively(elem, slicePath); err != nil {
						errs = append(errs, err)
					}
				} else if elem.Kind() == reflect.Ptr &&
					elem.Type().Elem().Kind() == reflect.Struct && !elem.IsNil() {
					if err := processRecursively(elem, slicePath); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}
