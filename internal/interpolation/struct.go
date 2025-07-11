package interpolation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// InterpolateStruct applies environment variable interpolation to string fields in a struct
// based on the env_interpolation struct tag. Only fields tagged with env_interpolation:"yes"
// will be interpolated. This function modifies the struct in place.
func InterpolateStruct(v any) error {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct or pointer to struct, got %T", v)
	}

	typ := val.Type()
	var errs []error

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

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
			original := field.String()
			if original != "" {
				interpolated, err := ExpandEnvVarsWithDefaults(original)
				if err != nil {
					errs = append(errs, fmt.Errorf("field %s: %w", fieldType.Name, err))
				} else {
					field.SetString(interpolated)
				}
			}

		case reflect.Map:
			// Special handling for map[string]string fields (like headers)
			if field.Type().Key().Kind() == reflect.String &&
				field.Type().Elem().Kind() == reflect.String {
				if !field.IsNil() {
					// Create a new map to avoid modifying while iterating
					newMap := reflect.MakeMap(field.Type())
					for _, key := range field.MapKeys() {
						value := field.MapIndex(key)
						interpolated, err := ExpandEnvVarsWithDefaults(value.String())
						if err != nil {
							errs = append(
								errs,
								fmt.Errorf("field %s[%s]: %w", fieldType.Name, key.String(), err),
							)
						} else {
							newMap.SetMapIndex(key, reflect.ValueOf(interpolated))
						}
					}
					field.Set(newMap)
				}
			}

		case reflect.Struct:
			// Recursively process nested structs
			if err := InterpolateStruct(field.Addr().Interface()); err != nil {
				errs = append(errs, fmt.Errorf("field %s: %w", fieldType.Name, err))
			}

		case reflect.Ptr:
			if field.Type().Elem().Kind() == reflect.Struct && !field.IsNil() {
				if err := InterpolateStruct(field.Interface()); err != nil {
					errs = append(errs, fmt.Errorf("field %s: %w", fieldType.Name, err))
				}
			}

		case reflect.Slice:
			// Process slices of strings or structs
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if elem.Kind() == reflect.String {
					// Handle slice of strings
					original := elem.String()
					if original != "" {
						interpolated, err := ExpandEnvVarsWithDefaults(original)
						if err != nil {
							errs = append(
								errs,
								fmt.Errorf("field %s[%d]: %w", fieldType.Name, j, err),
							)
						} else {
							elem.SetString(interpolated)
						}
					}
				} else if elem.Kind() == reflect.Struct {
					if err := InterpolateStruct(elem.Addr().Interface()); err != nil {
						errs = append(errs, fmt.Errorf("field %s[%d]: %w", fieldType.Name, j, err))
					}
				} else if elem.Kind() == reflect.Ptr && elem.Type().Elem().Kind() == reflect.Struct && !elem.IsNil() {
					if err := InterpolateStruct(elem.Interface()); err != nil {
						errs = append(errs, fmt.Errorf("field %s[%d]: %w", fieldType.Name, j, err))
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}
