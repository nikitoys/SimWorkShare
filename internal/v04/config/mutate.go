package config

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// SetNumericPath updates a numeric configuration field using the same JSON
// path convention accepted by analysis.sensitivity_parameters. Scenario array
// entries are addressed by scenario name, not by an unstable array index.
func SetNumericPath(cfg *Config, path string, number float64) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return fmt.Errorf("value for %q must be finite", path)
	}
	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if segment == "" {
			return fmt.Errorf("invalid empty path segment in %q", path)
		}
	}
	if err := setNumericValue(reflect.ValueOf(cfg).Elem(), segments, number, path); err != nil {
		return err
	}
	return nil
}

func setNumericValue(value reflect.Value, segments []string, number float64, fullPath string) error {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	if len(segments) == 0 {
		switch value.Kind() {
		case reflect.Float32, reflect.Float64:
			value.SetFloat(number)
			return nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if math.Trunc(number) != number {
				return fmt.Errorf("value for integer path %q must be integral", fullPath)
			}
			value.SetInt(int64(number))
			return nil
		default:
			return fmt.Errorf("path %q must resolve to a numeric field", fullPath)
		}
	}
	segment := segments[0]
	switch value.Kind() {
	case reflect.Struct:
		typ := value.Type()
		for index := 0; index < typ.NumField(); index++ {
			if strings.Split(typ.Field(index).Tag.Get("json"), ",")[0] == segment {
				return setNumericValue(value.Field(index), segments[1:], number, fullPath)
			}
		}
		return fmt.Errorf("path %q does not resolve: unknown field %q", fullPath, segment)
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("path %q traverses an unsupported map", fullPath)
		}
		key := reflect.ValueOf(segment)
		entry := value.MapIndex(key)
		if !entry.IsValid() {
			return fmt.Errorf("path %q does not resolve: unknown name %q", fullPath, segment)
		}
		copyValue := reflect.New(value.Type().Elem()).Elem()
		copyValue.Set(entry)
		if err := setNumericValue(copyValue, segments[1:], number, fullPath); err != nil {
			return err
		}
		value.SetMapIndex(key, copyValue)
		return nil
	case reflect.Slice:
		if value.Type().Elem() != reflect.TypeOf(OrganizationalScenario{}) {
			return fmt.Errorf("path %q cannot traverse an array by name", fullPath)
		}
		for index := 0; index < value.Len(); index++ {
			if value.Index(index).FieldByName("Name").String() == segment {
				return setNumericValue(value.Index(index), segments[1:], number, fullPath)
			}
		}
		return fmt.Errorf("path %q does not resolve: unknown scenario %q", fullPath, segment)
	default:
		return fmt.Errorf("path %q continues beyond scalar field at %q", fullPath, segment)
	}
}
