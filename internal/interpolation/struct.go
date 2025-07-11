package interpolation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// InterpolateStruct applies environment variable interpolation to fields tagged with `env_interpolation:"yes"`.
// This function modifies the provided struct in place. It handles string fields, string maps, and string slices.
// Interface types will return an error - each concrete type should call this function on itself.
func InterpolateStruct(v any) error {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)

	// Fail fast if passed an interface type
	if val.Kind() == reflect.Interface {
		return fmt.Errorf(
			"InterpolateStruct cannot handle interface types, call from concrete type instead",
		)
	}

	// Handle pointer to struct
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Must be a struct at this point
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct or pointer to struct, got %T", v)
	}

	typ := val.Type()
	var errs []error

	// Process each field in the struct
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

		// Handle different field types
		switch field.Kind() {
		case reflect.String:
			original := field.String()
			if original == "" {
				continue
			}

			interpolated, err := ExpandEnvVarsWithDefaults(original)
			if err != nil {
				errs = append(errs, fmt.Errorf("field %s: %w", fieldType.Name, err))
				continue
			}
			field.SetString(interpolated)

		case reflect.Map:
			// Handle map[string]string fields
			if field.Type().Key().Kind() != reflect.String ||
				field.Type().Elem().Kind() != reflect.String ||
				field.IsNil() {
				continue
			}

			for _, key := range field.MapKeys() {
				value := field.MapIndex(key)
				interpolated, err := ExpandEnvVarsWithDefaults(value.String())
				if err != nil {
					errs = append(
						errs,
						fmt.Errorf("field %s[%s]: %w", fieldType.Name, key.String(), err),
					)
					continue
				}
				field.SetMapIndex(key, reflect.ValueOf(interpolated))
			}

		case reflect.Slice:
			elemType := field.Type().Elem()

			switch elemType.Kind() {
			case reflect.String:
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					original := elem.String()
					if original == "" {
						continue
					}

					interpolated, err := ExpandEnvVarsWithDefaults(original)
					if err != nil {
						errs = append(errs, fmt.Errorf("field %s[%d]: %w", fieldType.Name, j, err))
						continue
					}
					elem.SetString(interpolated)
				}

			case reflect.Struct:
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					if err := InterpolateStruct(elem.Addr().Interface()); err != nil {
						errs = append(errs, fmt.Errorf("field %s[%d]: %w", fieldType.Name, j, err))
					}
				}

			case reflect.Ptr:
				if elemType.Elem().Kind() == reflect.Struct {
					for j := 0; j < field.Len(); j++ {
						elem := field.Index(j)
						if elem.IsNil() {
							continue
						}
						if err := InterpolateStruct(elem.Interface()); err != nil {
							errs = append(
								errs,
								fmt.Errorf("field %s[%d]: %w", fieldType.Name, j, err),
							)
						}
					}
				}
			}

		case reflect.Struct:
			// Handle nested struct fields
			if err := InterpolateStruct(field.Addr().Interface()); err != nil {
				errs = append(errs, fmt.Errorf("field %s: %w", fieldType.Name, err))
			}

		case reflect.Ptr:
			// Handle *SomeStruct fields
			if field.Type().Elem().Kind() == reflect.Struct && !field.IsNil() {
				if err := InterpolateStruct(field.Interface()); err != nil {
					errs = append(errs, fmt.Errorf("field %s: %w", fieldType.Name, err))
				}
			}
		}
	}

	return errors.Join(errs...)
}
