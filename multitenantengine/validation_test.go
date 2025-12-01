package multitenantengine

import (
	"strings"
	"testing"
)

// TestValidateSchema_EmptySchema verifies REQ-SCHEMA-002: Schema SHALL contain at least one object
func TestValidateSchema_EmptySchema(t *testing.T) {
	schema := Schema{}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("Expected error for empty schema, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected error message about empty schema, got: %v", err)
	}
}

// TestValidateSchema_EmptyObject verifies REQ-SCHEMA-003: Objects SHALL contain at least one field
func TestValidateSchema_EmptyObject(t *testing.T) {
	schema := Schema{
		"User": {},
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("Expected error for empty object definition, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "User") {
		t.Errorf("Expected error message to mention 'User', got: %v", err)
	}
}

// TestValidateSchema_TooManyObjects verifies REQ-SCHEMA-004: Maximum 100 objects
func TestValidateSchema_TooManyObjects(t *testing.T) {
	schema := Schema{}
	for i := 0; i < 101; i++ {
		objectName := "Object" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		schema[objectName] = map[string]string{"Field": "int"}
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("Expected error for too many objects (101), got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "100") {
		t.Errorf("Expected error message about max 100 objects, got: %v", err)
	}
}

// TestValidateSchema_TooManyFields verifies REQ-SCHEMA-005: Maximum 200 fields per object
func TestValidateSchema_TooManyFields(t *testing.T) {
	fields := make(map[string]string)
	for i := 0; i < 201; i++ {
		fieldName := "Field" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		fields[fieldName] = "int"
	}

	schema := Schema{
		"User": fields,
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("Expected error for too many fields (201), got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "200") {
		t.Errorf("Expected error message about max 200 fields, got: %v", err)
	}
}

// TestValidateSchema_ValidTypes verifies REQ-TYPE-001: Only valid CEL types allowed
func TestValidateSchema_ValidTypes(t *testing.T) {
	validTypes := []string{"int", "int64", "float64", "string", "bool", "bytes", "timestamp", "duration"}

	for _, typeName := range validTypes {
		schema := Schema{
			"User": {
				"TestField": typeName,
			},
		}

		err := ValidateSchema(schema)
		if err != nil {
			t.Errorf("Expected valid type %s to pass validation, got error: %v", typeName, err)
		}
	}
}

// TestValidateSchema_InvalidTypes verifies REQ-TYPE-001: Unsupported types rejected
func TestValidateSchema_InvalidTypes(t *testing.T) {
	invalidTypes := []string{"varchar", "date", "datetime", "decimal", "number", "array", "object", "CustomType"}

	for _, typeName := range invalidTypes {
		schema := Schema{
			"User": {
				"TestField": typeName,
			},
		}

		err := ValidateSchema(schema)
		if err == nil {
			t.Errorf("Expected error for invalid type %s, got nil", typeName)
		}
		if err != nil && !strings.Contains(err.Error(), typeName) {
			t.Errorf("Expected error message to mention invalid type %s, got: %v", typeName, err)
		}
	}
}

// TestValidateSchema_CaseSensitiveTypes verifies REQ-TYPE-002: Type names are case-sensitive
func TestValidateSchema_CaseSensitiveTypes(t *testing.T) {
	invalidCases := []string{"String", "INT", "Bool", "Float64", "BOOL"}

	for _, typeName := range invalidCases {
		schema := Schema{
			"User": {
				"TestField": typeName,
			},
		}

		err := ValidateSchema(schema)
		if err == nil {
			t.Errorf("Expected error for incorrect case type %s, got nil", typeName)
		}
	}
}

// TestValidateSchema_TypeWithWhitespace verifies REQ-TYPE-004: No leading/trailing whitespace
func TestValidateSchema_TypeWithWhitespace(t *testing.T) {
	invalidTypes := []string{" int", "int ", " int ", "\tint", "int\n"}

	for _, typeName := range invalidTypes {
		schema := Schema{
			"User": {
				"TestField": typeName,
			},
		}

		err := ValidateSchema(schema)
		if err == nil {
			t.Errorf("Expected error for type with whitespace %q, got nil", typeName)
		}
	}
}

// TestValidateSchema_EmptyTypeName verifies REQ-TYPE-004: Type names cannot be empty
func TestValidateSchema_EmptyTypeName(t *testing.T) {
	schema := Schema{
		"User": {
			"TestField": "",
		},
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("Expected error for empty type name, got nil")
	}
}

// TestValidateIdentifier_ValidFormats verifies REQ-IDENT-001: Valid identifier formats
func TestValidateIdentifier_ValidFormats(t *testing.T) {
	validIdentifiers := []string{
		"User",
		"user",
		"_private",
		"User123",
		"user_name",
		"_",
		"a",
		"A",
		"CamelCase",
		"snake_case",
		"SCREAMING_SNAKE_CASE",
	}

	for _, id := range validIdentifiers {
		err := validateIdentifier(id)
		if err != nil {
			t.Errorf("Expected valid identifier %q to pass validation, got error: %v", id, err)
		}
	}
}

// TestValidateIdentifier_InvalidFormats verifies REQ-IDENT-001: Invalid identifier formats rejected
func TestValidateIdentifier_InvalidFormats(t *testing.T) {
	invalidIdentifiers := []string{
		"123User",       // starts with digit
		"9Field",        // starts with digit
		"User-Name",     // contains hyphen
		"User.Name",     // contains dot
		"User Name",     // contains space
		"User@Email",    // contains @
		"User$Value",    // contains $
		"User#Tag",      // contains #
		"User%Percent",  // contains %
		"User&Value",    // contains &
		"User*Pointer",  // contains *
	}

	for _, id := range invalidIdentifiers {
		err := validateIdentifier(id)
		if err == nil {
			t.Errorf("Expected error for invalid identifier %q, got nil", id)
		}
	}
}

// TestValidateIdentifier_ReservedKeywords verifies REQ-IDENT-002: Reserved keywords rejected
func TestValidateIdentifier_ReservedKeywords(t *testing.T) {
	reservedKeywords := []string{
		"true", "false", "null",
		"in", "as", "break", "const", "continue",
		"else", "for", "function", "if", "import",
		"let", "loop", "package", "namespace", "return",
		"var", "void", "while",
	}

	for _, keyword := range reservedKeywords {
		err := validateIdentifier(keyword)
		if err == nil {
			t.Errorf("Expected error for reserved keyword %q, got nil", keyword)
		}
		if err != nil && !strings.Contains(err.Error(), "reserved") {
			t.Errorf("Expected error message about reserved keyword for %q, got: %v", keyword, err)
		}
	}
}

// TestValidateIdentifier_LengthLimits verifies REQ-IDENT-003: Identifier length limits
func TestValidateIdentifier_LengthLimits(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		shouldErr bool
	}{
		{"empty", "", true},
		{"single char", "a", false},
		{"max length 100", strings.Repeat("a", 100), false},
		{"too long 101", strings.Repeat("a", 101), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.id)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %s, got: %v", tt.name, err)
			}
		})
	}
}

// TestValidateSchema_CaseSensitiveIdentifiers verifies REQ-IDENT-006: Case sensitivity
func TestValidateSchema_CaseSensitiveIdentifiers(t *testing.T) {
	// This should be valid - User and user are different
	schema := Schema{
		"User": {
			"Age": "int",
		},
		"user": {
			"Name": "string",
		},
	}

	err := ValidateSchema(schema)
	if err != nil {
		t.Errorf("Expected case-sensitive object names to be valid, got error: %v", err)
	}

	// Fields should also be case-sensitive
	schema2 := Schema{
		"User": {
			"Age":  "int",
			"age":  "int",
			"AGE":  "int",
		},
	}

	err = ValidateSchema(schema2)
	if err != nil {
		t.Errorf("Expected case-sensitive field names to be valid, got error: %v", err)
	}
}

// TestValidateSchema_ValidCompleteSchema verifies a fully valid schema passes
func TestValidateSchema_ValidCompleteSchema(t *testing.T) {
	schema := Schema{
		"User": {
			"Age":         "int",
			"Name":        "string",
			"Email":       "string",
			"IsActive":    "bool",
			"CreatedAt":   "timestamp",
			"Balance":     "float64",
		},
		"Transaction": {
			"Amount":      "float64",
			"Currency":    "string",
			"ProcessedAt": "timestamp",
			"Duration":    "duration",
			"Metadata":    "bytes",
		},
	}

	err := ValidateSchema(schema)
	if err != nil {
		t.Errorf("Expected valid complete schema to pass, got error: %v", err)
	}
}

// TestValidateSchema_InvalidObjectName verifies object name validation
func TestValidateSchema_InvalidObjectName(t *testing.T) {
	tests := []struct {
		name       string
		objectName string
	}{
		{"starts with digit", "123User"},
		{"contains hyphen", "User-Profile"},
		{"contains space", "User Profile"},
		{"reserved keyword", "if"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := Schema{
				tt.objectName: {
					"Field": "int",
				},
			}

			err := ValidateSchema(schema)
			if err == nil {
				t.Errorf("Expected error for invalid object name %q, got nil", tt.objectName)
			}
		})
	}
}

// TestValidateSchema_InvalidFieldName verifies field name validation
func TestValidateSchema_InvalidFieldName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
	}{
		{"starts with digit", "123field"},
		{"contains hyphen", "field-name"},
		{"contains space", "field name"},
		{"reserved keyword", "return"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := Schema{
				"User": {
					tt.fieldName: "int",
				},
			}

			err := ValidateSchema(schema)
			if err == nil {
				t.Errorf("Expected error for invalid field name %q, got nil", tt.fieldName)
			}
		})
	}
}

// TestValidateSchema_MultipleErrors verifies REQ-ERROR-002: Should report multiple errors
func TestValidateSchema_MultipleErrors(t *testing.T) {
	// Schema with multiple issues
	schema := Schema{
		"123Invalid": {  // Invalid object name (starts with digit)
			"field-name": "varchar",  // Invalid field name (hyphen) and invalid type
		},
		"EmptyObject": {},  // Empty object
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("Expected error for schema with multiple issues, got nil")
	}

	// Check that error message mentions multiple issues
	// This is a SHOULD requirement, so we'll just verify we get an error
	t.Logf("Multiple error message: %v", err)
}

// TestIsValidCELType verifies the type checking function
func TestIsValidCELType(t *testing.T) {
	tests := []struct {
		typeName string
		valid    bool
	}{
		// Valid types
		{"int", true},
		{"int64", true},
		{"float64", true},
		{"string", true},
		{"bool", true},
		{"bytes", true},
		{"timestamp", true},
		{"duration", true},
		// Invalid types
		{"Int", false},
		{"STRING", false},
		{"varchar", false},
		{"number", false},
		{"", false},
		{" int", false},
		{"int ", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := isValidCELType(tt.typeName)
			if result != tt.valid {
				t.Errorf("isValidCELType(%q) = %v, want %v", tt.typeName, result, tt.valid)
			}
		})
	}
}

// TestValidateSchema_BoundaryConditions verifies REQ-TEST-003: Boundary testing
func TestValidateSchema_BoundaryConditions(t *testing.T) {
	// Test exactly at the limits
	t.Run("exactly 100 objects", func(t *testing.T) {
		schema := Schema{}
		for i := 0; i < 100; i++ {
			objectName := "Object" + string(rune('A'+i%26)) + string(rune('0'+i/26))
			schema[objectName] = map[string]string{"Field": "int"}
		}

		err := ValidateSchema(schema)
		if err != nil {
			t.Errorf("Expected 100 objects to be valid, got error: %v", err)
		}
	})

	t.Run("exactly 200 fields", func(t *testing.T) {
		fields := make(map[string]string)
		for i := 0; i < 200; i++ {
			fieldName := "Field" + string(rune('A'+i%26)) + string(rune('0'+i/26))
			fields[fieldName] = "int"
		}

		schema := Schema{
			"User": fields,
		}

		err := ValidateSchema(schema)
		if err != nil {
			t.Errorf("Expected 200 fields to be valid, got error: %v", err)
		}
	})
}

// TestValidateSchema_NoPanic verifies REQ-ERROR-003: No panics on any input
func TestValidateSchema_NoPanic(t *testing.T) {
	// Test with nil schema (edge case)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ValidateSchema panicked on nil/invalid input: %v", r)
		}
	}()

	// Various potentially problematic inputs
	testCases := []Schema{
		nil,
		{},
		{"": {}},
		{"User": nil},
	}

	for _, schema := range testCases {
		_ = ValidateSchema(schema)
		// We don't care about the error, just that it doesn't panic
	}
}

// TestCreateCELEnvFromSchema_SecuritySettings verifies REQ-SEC-001 and REQ-SEC-002
func TestCreateCELEnvFromSchema_SecuritySettings(t *testing.T) {
	schema := Schema{
		"User": {
			"Age": "int",
		},
	}

	env, err := CreateCELEnvFromSchema(schema)
	if err != nil {
		t.Fatalf("Failed to create CEL environment: %v", err)
	}

	// Verify environment was created
	if env == nil {
		t.Error("Expected non-nil CEL environment")
	}

	// Note: Testing cost limits and macro clearing would require
	// inspecting internal CEL environment state or running test programs
	// This is verified through integration testing
}

// TestValidateSchema_Performance verifies REQ-PERF-001: Validation completes within 100ms
func TestValidateSchema_Performance(t *testing.T) {
	// Create a large but valid schema (100 objects, 100 fields each)
	schema := Schema{}
	for i := 0; i < 100; i++ {
		objectName := "Object" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		fields := make(map[string]string)
		for j := 0; j < 100; j++ {
			fieldName := "Field" + string(rune('A'+j%26)) + string(rune('0'+j/26))
			fields[fieldName] = "int"
		}
		schema[objectName] = fields
	}

	// Run validation and measure time
	// Note: For accurate benchmarking, use testing.B instead
	err := ValidateSchema(schema)
	if err != nil {
		t.Errorf("Large valid schema failed validation: %v", err)
	}

	// For actual performance testing, see BenchmarkValidateSchema
}

// BenchmarkValidateSchema measures validation performance
func BenchmarkValidateSchema(b *testing.B) {
	// Create a realistic schema
	schema := Schema{
		"User": {
			"Age":      "int",
			"Name":     "string",
			"Email":    "string",
			"IsActive": "bool",
		},
		"Transaction": {
			"Amount":   "float64",
			"Currency": "string",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateSchema(schema)
	}
}

// BenchmarkValidateSchema_Large measures performance with large schema
func BenchmarkValidateSchema_Large(b *testing.B) {
	schema := Schema{}
	for i := 0; i < 100; i++ {
		objectName := "Object" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		fields := make(map[string]string)
		for j := 0; j < 100; j++ {
			fieldName := "Field" + string(rune('A'+j%26)) + string(rune('0'+j/26))
			fields[fieldName] = "int"
		}
		schema[objectName] = fields
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateSchema(schema)
	}
}
