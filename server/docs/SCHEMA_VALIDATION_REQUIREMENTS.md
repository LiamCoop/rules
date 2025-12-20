# Schema Validation Requirements

This document specifies the functional and security requirements for schema validation in the multi-tenant rules engine. Each requirement is verifiable and uses specific language to enable test derivation.

**Requirement Language:**
- **SHALL** = Mandatory requirement
- **SHOULD** = Recommended but not mandatory
- **MAY** = Optional

---

## 1. Schema Structure Requirements

### REQ-SCHEMA-001: Schema Definition Structure
A schema definition **SHALL** be a nested map structure where:
- Top level keys are object names (e.g., "User", "Transaction")
- Values are maps of field names to type names
- Both levels **SHALL** be non-empty maps

**Verification:** Schema with empty object definitions or empty top-level map returns validation error.

### REQ-SCHEMA-002: Non-Empty Schema
A schema **SHALL** contain at least one object definition.

**Verification:** Submitting an empty schema `{}` returns validation error.

### REQ-SCHEMA-003: Non-Empty Object Definitions
Each object in a schema **SHALL** contain at least one field definition.

**Verification:** Schema with object containing empty field map `{"User": {}}` returns validation error.

### REQ-SCHEMA-004: Maximum Schema Size
A schema **SHALL** contain no more than 100 objects.

**Verification:** Submitting schema with 101 objects returns validation error.

### REQ-SCHEMA-005: Maximum Fields Per Object
Each object **SHALL** contain no more than 200 fields.

**Verification:** Object with 201 fields returns validation error.

---

## 2. Type Validation Requirements

### REQ-TYPE-001: Valid CEL Type Names
Field type definitions **SHALL** only use the following CEL type names:
- `int`
- `int64`
- `float64`
- `string`
- `bool`
- `bytes`
- `timestamp`
- `duration`

**Verification:** Schema with unsupported type (e.g., "varchar", "date", "decimal") returns validation error.

### REQ-TYPE-002: Case-Sensitive Type Names
Type names **SHALL** be case-sensitive and exactly match the allowed types.

**Verification:** Schema with "String", "INT", "Bool" returns validation error.

### REQ-TYPE-003: No Custom Types
Schemas **SHALL NOT** allow custom or user-defined types.

**Verification:** Schema referencing custom type (e.g., "CustomObject") returns validation error.

### REQ-TYPE-004: Type Name Format
Type names **SHALL** be non-empty strings with no leading or trailing whitespace.

**Verification:** Schema with type `" int "` or `""` returns validation error.

---

## 3. Identifier Validation Requirements

### REQ-IDENT-001: Valid Identifier Format
Object names and field names **SHALL** match the regular expression: `^[a-zA-Z_][a-zA-Z0-9_]*$`

**Verification:** Names must start with letter or underscore, followed by letters, digits, or underscores.

### REQ-IDENT-002: Reserved Keyword Prohibition
Object names and field names **SHALL NOT** use CEL reserved keywords:
- `true`, `false`, `null`
- `in`, `as`, `break`, `const`, `continue`, `else`, `for`, `function`, `if`, `import`, `let`, `loop`, `package`, `namespace`, `return`, `var`, `void`, `while`

**Verification:** Schema with object or field named "true", "if", "return" returns validation error.

### REQ-IDENT-003: Identifier Length Limits
Object names and field names **SHALL** be between 1 and 100 characters in length.

**Verification:** Empty name or name with 101 characters returns validation error.

### REQ-IDENT-004: No Leading Digits
Object names and field names **SHALL NOT** start with a digit.

**Verification:** Names "123User" or "9Field" return validation error.

### REQ-IDENT-005: No Special Characters
Object names and field names **SHALL NOT** contain special characters except underscore.

**Verification:** Names with "-", ".", " ", "@", "$" return validation error.

### REQ-IDENT-006: Case Sensitivity
Object names and field names **SHALL** be case-sensitive.

**Verification:** "User" and "user" are treated as different objects/fields.

### REQ-IDENT-007: No Duplicate Field Names
Within a single object, field names **SHALL** be unique (case-sensitive).

**Verification:** Object with duplicate field names (e.g., {"Age": "int", "Age": "string"}) returns validation error.

---

## 4. Security Requirements

### REQ-SEC-001: CEL Cost Limit
When creating CEL environments from schemas, a cost limit **SHALL** be applied to prevent runaway expressions.

**Verification:** CEL environment created from schema has `CostLimit` set to 1,000,000.

### REQ-SEC-002: No Dangerous Macros
CEL environments **SHALL** have macros cleared to prevent potentially dangerous operations.

**Verification:** CEL environment created from schema has `ClearMacros()` applied.

### REQ-SEC-003: Input Sanitization
Schema validation **SHALL** occur before any database operations.

**Verification:** Invalid schema is rejected before INSERT/UPDATE queries execute.

### REQ-SEC-004: Validation Error Messages
Validation errors **SHALL** include specific information about what failed but **SHALL NOT** expose internal system details.

**Verification:** Error messages describe validation failure without stack traces or internal paths.

---

## 5. API Integration Requirements

### REQ-API-001: Create Schema Validation
The POST `/api/v1/tenants/:tenantId/schema` endpoint **SHALL** validate the schema before storing.

**Verification:** Invalid schema submitted to create endpoint returns 400 Bad Request.

### REQ-API-002: Update Schema Validation
The PUT `/api/v1/tenants/:tenantId/schema` endpoint **SHALL** validate the new schema before updating.

**Verification:** Invalid schema submitted to update endpoint returns 400 Bad Request.

### REQ-API-003: Validation Error Response Format
Schema validation errors **SHALL** return HTTP 400 with JSON body containing:
- `error`: Human-readable error message
- `field`: Specific field or object that failed validation (if applicable)

**Verification:** Response body matches expected error format.

### REQ-API-004: Atomic Validation
Schema validation **SHALL** be atomic - either all checks pass or the entire schema is rejected.

**Verification:** Schema with multiple errors is rejected entirely, no partial acceptance.

---

## 6. Database Persistence Requirements

### REQ-DB-001: Store Valid Schemas Only
Only schemas that pass all validation checks **SHALL** be persisted to the database.

**Verification:** Database contains no invalid schemas after validation is implemented.

### REQ-DB-002: Schema Version Immutability
Once a schema version is stored, its definition **SHALL** be immutable.

**Verification:** Attempting to modify existing schema version fails.

### REQ-DB-003: Active Schema Validation
When marking a schema as active, it **SHALL** be revalidated.

**Verification:** Cannot activate an invalid schema (defense in depth).

---

## 7. Backward Compatibility Requirements

### REQ-COMPAT-001: Existing Schema Migration
Existing schemas in the database **MAY** be invalid according to new validation rules.

**Verification:** System continues to operate with existing schemas; validation applies only to new/updated schemas.

### REQ-COMPAT-002: Validation Flag
A database flag **MAY** indicate whether a schema was validated with current rules.

**Verification:** `schemas` table has optional `validated` boolean column.

---

## 8. Error Handling Requirements

### REQ-ERROR-001: Descriptive Validation Errors
Validation errors **SHALL** specify:
- Which object or field failed validation
- What rule was violated
- Example of valid format (when applicable)

**Verification:** Error message contains object/field name and specific rule violation.

### REQ-ERROR-002: Multiple Error Reporting
Validation **SHOULD** collect all errors and report them together rather than failing on first error.

**Verification:** Schema with 3 validation errors returns all 3 in error response.

### REQ-ERROR-003: No Panics
Schema validation **SHALL NOT** panic on any input, including malformed or malicious inputs.

**Verification:** Fuzz testing with random inputs does not cause panics.

---

## 9. Performance Requirements

### REQ-PERF-001: Validation Time Limit
Schema validation **SHOULD** complete within 100ms for schemas with up to 100 objects.

**Verification:** Benchmark shows validation time < 100ms for typical schemas.

### REQ-PERF-002: No Database Queries During Validation
Schema validation **SHALL** be performed in-memory without database queries.

**Verification:** Validation runs successfully without database connection.

---

## 10. Testing Requirements

### REQ-TEST-001: Unit Tests for Each Validation Rule
Each validation requirement **SHALL** have at least one unit test.

**Verification:** Test coverage includes all validation rules.

### REQ-TEST-002: Integration Tests
Schema validation **SHALL** be tested through API endpoints.

**Verification:** Integration tests submit invalid schemas to API and verify rejection.

### REQ-TEST-003: Boundary Testing
Tests **SHALL** cover boundary conditions (max length, min length, edge cases).

**Verification:** Tests exist for 0, 1, max-1, max, max+1 values.

### REQ-TEST-004: Malicious Input Testing
Tests **SHALL** include attempts to bypass validation with malicious input.

**Verification:** Tests for SQL injection patterns, script injection, unicode exploits.

---

## Requirement Traceability Matrix

| Requirement ID | Category | Priority | Verification Method |
|---------------|----------|----------|---------------------|
| REQ-SCHEMA-001 | Structure | MUST | Unit test |
| REQ-SCHEMA-002 | Structure | MUST | Unit test |
| REQ-SCHEMA-003 | Structure | MUST | Unit test |
| REQ-SCHEMA-004 | Structure | MUST | Unit test |
| REQ-SCHEMA-005 | Structure | MUST | Unit test |
| REQ-TYPE-001 | Type Validation | MUST | Unit test |
| REQ-TYPE-002 | Type Validation | MUST | Unit test |
| REQ-TYPE-003 | Type Validation | MUST | Unit test |
| REQ-TYPE-004 | Type Validation | MUST | Unit test |
| REQ-IDENT-001 | Identifier | MUST | Unit test |
| REQ-IDENT-002 | Identifier | MUST | Unit test |
| REQ-IDENT-003 | Identifier | MUST | Unit test |
| REQ-IDENT-004 | Identifier | MUST | Unit test |
| REQ-IDENT-005 | Identifier | MUST | Unit test |
| REQ-IDENT-006 | Identifier | MUST | Unit test |
| REQ-IDENT-007 | Identifier | MUST | Unit test |
| REQ-SEC-001 | Security | MUST | Environment inspection |
| REQ-SEC-002 | Security | MUST | Environment inspection |
| REQ-SEC-003 | Security | MUST | Integration test |
| REQ-SEC-004 | Security | MUST | Error message test |
| REQ-API-001 | API | MUST | Integration test |
| REQ-API-002 | API | MUST | Integration test |
| REQ-API-003 | API | MUST | Integration test |
| REQ-API-004 | API | MUST | Integration test |
| REQ-DB-001 | Database | MUST | Integration test |
| REQ-DB-002 | Database | MUST | Integration test |
| REQ-DB-003 | Database | MUST | Integration test |
| REQ-COMPAT-001 | Compatibility | MAY | Manual verification |
| REQ-COMPAT-002 | Compatibility | MAY | Schema inspection |
| REQ-ERROR-001 | Error Handling | MUST | Unit test |
| REQ-ERROR-002 | Error Handling | SHOULD | Unit test |
| REQ-ERROR-003 | Error Handling | MUST | Fuzz test |
| REQ-PERF-001 | Performance | SHOULD | Benchmark test |
| REQ-PERF-002 | Performance | MUST | Unit test |
| REQ-TEST-001 | Testing | MUST | Coverage report |
| REQ-TEST-002 | Testing | MUST | Test suite review |
| REQ-TEST-003 | Testing | MUST | Test suite review |
| REQ-TEST-004 | Testing | MUST | Security test suite |

---

## Summary Statistics

- **Total Requirements:** 38
- **MUST Requirements:** 33
- **SHOULD Requirements:** 3
- **MAY Requirements:** 2
- **Categories:** 10

Each requirement is designed to be independently verifiable and will map directly to one or more test cases.

---

## Example Valid Schemas

```json
{
  "User": {
    "Age": "int",
    "Name": "string",
    "IsActive": "bool"
  },
  "Transaction": {
    "Amount": "float64",
    "Timestamp": "timestamp",
    "Description": "string"
  }
}
```

## Example Invalid Schemas

### Invalid Type Name
```json
{
  "User": {
    "Age": "integer"  // Invalid: should be "int" or "int64"
  }
}
```

### Invalid Identifier
```json
{
  "User-Profile": {  // Invalid: contains hyphen
    "Age": "int"
  }
}
```

### Reserved Keyword
```json
{
  "User": {
    "true": "bool"  // Invalid: "true" is reserved keyword
  }
}
```

### Empty Object
```json
{
  "User": {}  // Invalid: object must have at least one field
}
```
