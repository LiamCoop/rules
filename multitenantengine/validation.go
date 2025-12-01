package multitenantengine

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateSchema validates a schema definition according to all validation requirements
// Returns an error if validation fails, nil if schema is valid
func ValidateSchema(schema Schema) error {
	// REQ-SCHEMA-002: Schema SHALL contain at least one object
	if len(schema) == 0 {
		return fmt.Errorf("schema cannot be empty, must contain at least one object definition")
	}

	// REQ-SCHEMA-004: Maximum 100 objects
	if len(schema) > 100 {
		return fmt.Errorf("schema contains %d objects, maximum allowed is 100", len(schema))
	}

	// Validate each object
	for objectName, fields := range schema {
		// Validate object name
		if err := validateIdentifier(objectName); err != nil {
			return fmt.Errorf("invalid object name %q: %w", objectName, err)
		}

		// REQ-SCHEMA-003: Objects SHALL contain at least one field
		if len(fields) == 0 {
			return fmt.Errorf("object %q must contain at least one field", objectName)
		}

		// REQ-SCHEMA-005: Maximum 200 fields per object
		if len(fields) > 200 {
			return fmt.Errorf("object %q contains %d fields, maximum allowed is 200", objectName, len(fields))
		}

		// Validate each field
		for fieldName, typeName := range fields {
			// Validate field name
			if err := validateIdentifier(fieldName); err != nil {
				return fmt.Errorf("invalid field name %q in object %q: %w", fieldName, objectName, err)
			}

			// REQ-TYPE-004: Type names cannot be empty
			if typeName == "" {
				return fmt.Errorf("field %q in object %q has empty type name", fieldName, objectName)
			}

			// REQ-TYPE-004: No leading/trailing whitespace
			if strings.TrimSpace(typeName) != typeName {
				return fmt.Errorf("field %q in object %q has type with leading/trailing whitespace: %q", fieldName, objectName, typeName)
			}

			// REQ-TYPE-001: Only valid CEL types allowed
			if !isValidCELType(typeName) {
				return fmt.Errorf("field %q in object %q has invalid type %q (must be one of: int, int64, float64, string, bool, bytes, timestamp, duration)", fieldName, objectName, typeName)
			}
		}
	}

	return nil
}

// validateIdentifier validates an object or field name according to identifier requirements
// REQ-IDENT-001: Must match pattern ^[a-zA-Z_][a-zA-Z0-9_]*$
// REQ-IDENT-002: Cannot be a reserved keyword
// REQ-IDENT-003: Must be 1-100 characters
func validateIdentifier(name string) error {
	// REQ-IDENT-003: Length limits (1-100 characters)
	if len(name) == 0 {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("identifier length %d exceeds maximum of 100 characters", len(name))
	}

	// REQ-IDENT-001: Valid identifier format
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("must match pattern ^[a-zA-Z_][a-zA-Z0-9_]*$ (start with letter or underscore, followed by letters, digits, or underscores)")
	}

	// REQ-IDENT-002: Reserved keyword prohibition
	if isReservedKeyword(name) {
		return fmt.Errorf("cannot use reserved keyword %q as identifier", name)
	}

	return nil
}

// isValidCELType checks if a type name is a valid CEL type
// REQ-TYPE-001: Only specific CEL types are allowed
// REQ-TYPE-002: Type names are case-sensitive
func isValidCELType(typeName string) bool {
	validTypes := map[string]bool{
		"int":       true,
		"int64":     true,
		"float64":   true,
		"string":    true,
		"bool":      true,
		"bytes":     true,
		"timestamp": true,
		"duration":  true,
	}

	return validTypes[typeName]
}

// isReservedKeyword checks if a name is a CEL reserved keyword
// REQ-IDENT-002: List of reserved keywords that cannot be used as identifiers
func isReservedKeyword(name string) bool {
	reservedKeywords := map[string]bool{
		// Boolean and null literals
		"true":  true,
		"false": true,
		"null":  true,
		// Control flow
		"if":       true,
		"else":     true,
		"for":      true,
		"while":    true,
		"break":    true,
		"continue": true,
		"return":   true,
		// Declarations
		"var":      true,
		"let":      true,
		"const":    true,
		"function": true,
		// Other keywords
		"in":        true,
		"as":        true,
		"import":    true,
		"package":   true,
		"namespace": true,
		"loop":      true,
		"void":      true,
	}

	return reservedKeywords[name]
}
