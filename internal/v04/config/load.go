package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// LoadFile reads and strictly validates a v0.4 configuration file.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	cfg, err := loadBytes(data)
	if err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", path, err)
	}
	return cfg, nil
}

// Load reads one JSON document from r and strictly validates it. Unknown
// fields and duplicate keys are rejected at every nesting level, and a second
// JSON value after the configuration is never accepted.
func Load(r io.Reader) (Config, error) {
	if r == nil {
		return Config{}, fieldError("config", "reader must not be nil")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	return loadBytes(data)
}

func loadBytes(data []byte) (Config, error) {
	root, err := parseJSONDocument(data)
	if err != nil {
		return Config{}, err
	}
	if err := validateJSONShape(root, reflect.TypeOf(Config{}), "", false); err != nil {
		return Config{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, err
	}
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fieldError("config", "multiple JSON values are not allowed")
	}
	return fieldError("config", "invalid trailing JSON: %v", err)
}

type jsonKind uint8

const (
	kindNull jsonKind = iota
	kindObject
	kindArray
	kindString
	kindNumber
	kindBoolean
)

type jsonNode struct {
	kind   jsonKind
	object map[string]*jsonNode
	array  []*jsonNode
	value  any
}

func parseJSONDocument(data []byte) (*jsonNode, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	root, err := consumeJSONValue(decoder, "config")
	if err != nil {
		return nil, err
	}
	token, err := decoder.Token()
	if errors.Is(err, io.EOF) {
		return root, nil
	}
	if err != nil {
		return nil, fieldError("config", "invalid trailing JSON: %v", err)
	}
	return nil, fieldError("config", "multiple JSON values are not allowed (unexpected %v)", token)
}

func consumeJSONValue(decoder *json.Decoder, path string) (*jsonNode, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, fieldError(path, "invalid JSON value: %v", err)
	}
	if token == nil {
		return &jsonNode{kind: kindNull}, nil
	}

	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			object := make(map[string]*jsonNode)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, fieldError(path, "invalid object key: %v", err)
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fieldError(path, "object key must be a string")
				}
				childPath := joinJSONPath(pathForChildren(path), key)
				if _, exists := object[key]; exists {
					return nil, fieldError(childPath, "duplicate field")
				}
				child, err := consumeJSONValue(decoder, childPath)
				if err != nil {
					return nil, err
				}
				object[key] = child
			}
			closing, err := decoder.Token()
			if err != nil {
				return nil, fieldError(path, "invalid JSON object: %v", err)
			}
			if closing != json.Delim('}') {
				return nil, fieldError(path, "invalid JSON object closing delimiter")
			}
			return &jsonNode{kind: kindObject, object: object}, nil
		case '[':
			var array []*jsonNode
			for index := 0; decoder.More(); index++ {
				childPath := fmt.Sprintf("%s[%d]", pathForChildren(path), index)
				child, err := consumeJSONValue(decoder, childPath)
				if err != nil {
					return nil, err
				}
				array = append(array, child)
			}
			closing, err := decoder.Token()
			if err != nil {
				return nil, fieldError(path, "invalid JSON array: %v", err)
			}
			if closing != json.Delim(']') {
				return nil, fieldError(path, "invalid JSON array closing delimiter")
			}
			return &jsonNode{kind: kindArray, array: array}, nil
		default:
			return nil, fieldError(path, "unexpected JSON delimiter %q", value)
		}
	case string:
		return &jsonNode{kind: kindString, value: value}, nil
	case json.Number:
		parsed, parseErr := strconv.ParseFloat(value.String(), 64)
		if parseErr != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return nil, fieldError(path, "must be a finite JSON number")
		}
		return &jsonNode{kind: kindNumber, value: value}, nil
	case bool:
		return &jsonNode{kind: kindBoolean, value: value}, nil
	default:
		return nil, fieldError(path, "unsupported JSON token %T", token)
	}
}

func validateJSONShape(node *jsonNode, typ reflect.Type, path string, nullable bool) error {
	if node.kind == kindNull {
		if nullable {
			return nil
		}
		return fieldError(displayPath(path), "must not be null")
	}
	if typ.Kind() == reflect.Pointer {
		return validateJSONShape(node, typ.Elem(), path, nullable)
	}

	switch typ.Kind() {
	case reflect.Struct:
		if node.kind != kindObject {
			return typeError(path, "object", node.kind)
		}
		fields := make(map[string]reflect.StructField, typ.NumField())
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			fields[name] = field
		}
		keys := make([]string, 0, len(node.object))
		for key := range node.object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if _, ok := fields[key]; !ok {
				return fieldError(joinJSONPath(path, key), "unknown field")
			}
		}
		fieldNames := make([]string, 0, len(fields))
		for name := range fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)
		for _, name := range fieldNames {
			field := fields[name]
			child, ok := node.object[name]
			childPath := joinJSONPath(path, name)
			if !ok {
				return fieldError(childPath, "is required")
			}
			if err := validateJSONShape(child, field.Type, childPath, field.Tag.Get("nullable") == "true"); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if node.kind != kindObject {
			return typeError(path, "object", node.kind)
		}
		if typ.Key().Kind() != reflect.String {
			return fieldError(displayPath(path), "internal configuration type has a non-string map key")
		}
		keys := make([]string, 0, len(node.object))
		for key := range node.object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := validateJSONShape(node.object[key], typ.Elem(), joinJSONPath(path, key), false); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		if node.kind != kindArray {
			return typeError(path, "array", node.kind)
		}
		for index, child := range node.array {
			childPath := fmt.Sprintf("%s[%d]", displayPath(path), index)
			if err := validateJSONShape(child, typ.Elem(), childPath, false); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		if node.kind != kindString {
			return typeError(path, "string", node.kind)
		}
		return nil
	case reflect.Bool:
		if node.kind != kindBoolean {
			return typeError(path, "boolean", node.kind)
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if node.kind != kindNumber {
			return typeError(path, "integer", node.kind)
		}
		number := node.value.(json.Number)
		value, err := strconv.ParseInt(number.String(), 10, typ.Bits())
		if err != nil {
			return fieldError(displayPath(path), "must be an integer in the supported range")
		}
		_ = value
		return nil
	case reflect.Float32, reflect.Float64:
		if node.kind != kindNumber {
			return typeError(path, "number", node.kind)
		}
		return nil
	default:
		return fieldError(displayPath(path), "unsupported configuration field type %s", typ)
	}
}

func typeError(path, expected string, actual jsonKind) error {
	return fieldError(displayPath(path), "must be a JSON %s, got %s", expected, actual)
}

func (kind jsonKind) String() string {
	switch kind {
	case kindNull:
		return "null"
	case kindObject:
		return "object"
	case kindArray:
		return "array"
	case kindString:
		return "string"
	case kindNumber:
		return "number"
	case kindBoolean:
		return "boolean"
	default:
		return "unknown value"
	}
}

func pathForChildren(path string) string {
	if path == "config" {
		return ""
	}
	return path
}

func displayPath(path string) string {
	if path == "" {
		return "config"
	}
	return path
}

func joinJSONPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func fieldError(path, format string, args ...any) error {
	return fmt.Errorf("%s: %s", displayPath(path), fmt.Sprintf(format, args...))
}
