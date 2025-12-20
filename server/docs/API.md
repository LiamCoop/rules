# Multi-Tenant Rules Engine API Documentation

**Version:** 1.0
**Base URL:** `http://localhost:8080`

This document provides comprehensive API documentation for UI developers building applications on top of the multi-tenant rules engine.

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Common Concepts](#common-concepts)
4. [Data Types](#data-types)
5. [API Endpoints](#api-endpoints)
   - [Health Check](#health-check)
   - [Tenant Management](#tenant-management)
   - [Schema Management](#schema-management)
   - [Rule Management](#rule-management)
   - [Rule Evaluation](#rule-evaluation)
6. [Error Handling](#error-handling)
7. [Examples](#examples)
8. [Validation Rules](#validation-rules)

---

## Overview

The Multi-Tenant Rules Engine is a RESTful API that allows you to:
- Manage multiple tenants with isolated data
- Define custom schemas per tenant (dynamic data structures)
- Create and manage CEL (Common Expression Language) rules
- Evaluate facts against rules in real-time

### Key Features
- **Multi-Tenancy**: Each tenant has isolated schemas and rules
- **Dynamic Schemas**: Define schemas at runtime without code changes
- **Zero-Downtime Updates**: Schema updates don't require service restarts
- **CEL Rules**: Powerful, safe expression language for business logic
- **Type Safety**: Runtime type checking ensures data integrity

---

## Authentication

**Current Status:** Not implemented (planned for production)

Future versions will support:
- API Key authentication
- JWT tokens
- Rate limiting per tenant

---

## Common Concepts

### Tenants
A tenant represents an isolated customer or organization. Each tenant has:
- Unique ID (UUID)
- Name
- Isolated schemas and rules

### Schemas
A schema defines the structure of data objects for a tenant. Schemas consist of:
- **Objects**: Top-level data containers (e.g., "User", "Transaction")
- **Fields**: Named properties within objects (e.g., "Age", "Amount")
- **Types**: Data types for each field (see [Data Types](#data-types))

### Rules
Rules are CEL expressions that evaluate to true or false. Rules can:
- Reference schema objects (e.g., `User.Age >= 18`)
- Use operators: `&&`, `||`, `!`, `==`, `!=`, `<`, `>`, `<=`, `>=`
- Access nested fields (e.g., `User.Profile.Email`)

### Facts
Facts are the actual data you evaluate against rules. Facts must conform to the tenant's schema.

---

## Data Types

The following data types are supported in schemas (case-sensitive):

| Type | Description | Example |
|------|-------------|---------|
| `int` | Integer number | `42` |
| `int64` | 64-bit integer | `9223372036854775807` |
| `float64` | Floating-point number | `3.14159` |
| `string` | Text string | `"hello"` |
| `bool` | Boolean value | `true` or `false` |
| `bytes` | Binary data | `b"binary data"` |
| `timestamp` | RFC3339 timestamp | `"2024-01-15T10:30:00Z"` |
| `duration` | Time duration | `"1h30m"` |

**Important:** Type names are case-sensitive. Use lowercase exactly as shown.

---

## API Endpoints

### Health Check

#### GET /health

Check if the service is running.

**Request:** None

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

---

### Tenant Management

#### List Tenants

**GET** `/api/v1/tenants`

Get a list of all tenants.

**Request:** None

**Response:** `200 OK`
```json
{
  "tenants": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "Acme Corp",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### Create Tenant

**POST** `/api/v1/tenants`

Create a new tenant.

**Request Body:**
```json
{
  "name": "Acme Corp"
}
```

**Response:** `201 Created`
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Acme Corp",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Validation:**
- `name` is required
- `name` must be non-empty string

---

### Schema Management

#### Create Schema

**POST** `/api/v1/tenants/{tenantId}/schema`

Create a schema for a tenant. Can only be called once per tenant.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier

**Request Body:**
```json
{
  "definition": {
    "User": {
      "Age": "int",
      "Name": "string",
      "Email": "string",
      "IsActive": "bool",
      "CreatedAt": "timestamp"
    },
    "Transaction": {
      "Amount": "float64",
      "Currency": "string",
      "ProcessedAt": "timestamp"
    }
  }
}
```

**Response:** `201 Created`
```json
{
  "version": 1,
  "status": "active",
  "definition": {
    "User": { ... },
    "Transaction": { ... }
  }
}
```

**Errors:**
- `400 Bad Request`: Invalid schema (see [Validation Rules](#validation-rules))
- `404 Not Found`: Tenant not found
- `409 Conflict`: Schema already exists (use PUT to update)

#### Update Schema

**PUT** `/api/v1/tenants/{tenantId}/schema`

Update a tenant's schema. Zero-downtime operation.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier

**Request Body:** Same as Create Schema

**Response:** `200 OK`
```json
{
  "version": 2,
  "status": "active",
  "definition": { ... }
}
```

**Notes:**
- Schema version increments automatically
- Previous schema version is deactivated
- All rules are recompiled with new schema
- Rules that no longer compile are marked inactive

#### Get Schema

**GET** `/api/v1/tenants/{tenantId}/schema`

Get the active schema for a tenant.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier

**Response:** `200 OK`
```json
{
  "version": 2,
  "definition": {
    "User": {
      "Age": "int",
      "Name": "string"
    }
  },
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Errors:**
- `404 Not Found`: Tenant or schema not found

---

### Rule Management

#### Create Rule

**POST** `/api/v1/tenants/{tenantId}/rules`

Create a new rule for a tenant.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier

**Request Body:**
```json
{
  "name": "Adult User Check",
  "expression": "User.Age >= 18"
}
```

**Response:** `201 Created`
```json
{
  "id": "rule-123e4567-e89b-12d3-a456-426614174000",
  "name": "Adult User Check",
  "expression": "User.Age >= 18",
  "active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Validation:**
- `name` is required
- `expression` is required
- Expression must be valid CEL
- Expression must reference valid schema objects/fields
- Expression is compiled and cached

**Errors:**
- `400 Bad Request`: Invalid expression or compilation error
- `404 Not Found`: Tenant not found

#### List Rules

**GET** `/api/v1/tenants/{tenantId}/rules`

Get all rules for a tenant.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier

**Response:** `200 OK`
```json
{
  "rules": [
    {
      "id": "rule-123",
      "name": "Adult User Check",
      "expression": "User.Age >= 18",
      "active": true,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### Get Rule

**GET** `/api/v1/tenants/{tenantId}/rules/{ruleId}`

Get a specific rule by ID.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier
- `ruleId` (string): Rule identifier

**Response:** `200 OK`
```json
{
  "id": "rule-123",
  "name": "Adult User Check",
  "expression": "User.Age >= 18",
  "active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Errors:**
- `404 Not Found`: Rule not found

#### Update Rule

**PUT** `/api/v1/tenants/{tenantId}/rules/{ruleId}`

Update an existing rule.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier
- `ruleId` (string): Rule identifier

**Request Body:**
```json
{
  "name": "Senior User Check",
  "expression": "User.Age >= 65",
  "active": true
}
```

**Response:** `200 OK`
```json
{
  "id": "rule-123",
  "name": "Senior User Check",
  "expression": "User.Age >= 65",
  "active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T12:00:00Z"
}
```

**Notes:**
- Rule is recompiled when expression changes
- `updated_at` timestamp is updated

#### Delete Rule

**DELETE** `/api/v1/tenants/{tenantId}/rules/{ruleId}`

Delete a rule.

**Path Parameters:**
- `tenantId` (UUID): Tenant identifier
- `ruleId` (string): Rule identifier

**Response:** `204 No Content`

**Errors:**
- `404 Not Found`: Rule not found

---

### Rule Evaluation

#### Evaluate Rules

**POST** `/api/v1/evaluate`

Evaluate facts against rules.

**Request Body:**
```json
{
  "tenantId": "123e4567-e89b-12d3-a456-426614174000",
  "facts": {
    "User": {
      "Age": 25,
      "Name": "John Doe",
      "Email": "john@example.com",
      "IsActive": true,
      "CreatedAt": "2024-01-15T10:30:00Z"
    },
    "Transaction": {
      "Amount": 1500.50,
      "Currency": "USD",
      "ProcessedAt": "2024-01-15T12:00:00Z"
    }
  },
  "rules": ["rule-123", "rule-456"]
}
```

**Request Fields:**
- `tenantId` (required): Tenant identifier
- `facts` (required): Data to evaluate, must match tenant's schema
- `rules` (optional): Array of rule IDs to evaluate. If omitted, evaluates all active rules

**Response:** `200 OK`
```json
{
  "results": [
    {
      "RuleID": "rule-123",
      "RuleName": "Adult User Check",
      "Matched": true,
      "Error": null,
      "Trace": {
        "User.Age": 25,
        "User.Age >= 18": true
      }
    },
    {
      "RuleID": "rule-456",
      "RuleName": "Large Transaction",
      "Matched": true,
      "Error": null,
      "Trace": {
        "Transaction.Amount": 1500.5,
        "Transaction.Amount > 1000": true
      }
    }
  ],
  "evaluationTime": "2.3ms"
}
```

**Response Fields:**
- `results`: Array of evaluation results
  - `RuleID`: Rule identifier
  - `RuleName`: Human-readable rule name
  - `Matched`: Boolean indicating if rule matched (true/false)
  - `Error`: Error message if evaluation failed, null otherwise
  - `Trace`: Evaluation trace showing intermediate values (useful for debugging)
- `evaluationTime`: Total time to evaluate all rules

**Errors:**
- `400 Bad Request`: Invalid facts or missing required fields
- `404 Not Found`: Tenant not found
- `500 Internal Server Error`: Evaluation error

---

## Error Handling

All error responses follow this format:

```json
{
  "error": "Human-readable error message"
}
```

### HTTP Status Codes

| Code | Meaning | When Used |
|------|---------|-----------|
| `200 OK` | Success | GET requests |
| `201 Created` | Resource created | POST requests |
| `204 No Content` | Success, no body | DELETE requests |
| `400 Bad Request` | Invalid input | Validation failures |
| `404 Not Found` | Resource not found | Missing tenant/rule/schema |
| `409 Conflict` | Resource already exists | Duplicate schema creation |
| `500 Internal Server Error` | Server error | Unexpected failures |

---

## Examples

### Complete Workflow: Create Tenant, Schema, Rules, and Evaluate

```bash
# 1. Create a tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "Acme Corp"}'

# Response: {"id": "tenant-123", "name": "Acme Corp", ...}

# 2. Create a schema
curl -X POST http://localhost:8080/api/v1/tenants/tenant-123/schema \
  -H "Content-Type: application/json" \
  -d '{
    "definition": {
      "User": {
        "Age": "int",
        "Country": "string"
      },
      "Transaction": {
        "Amount": "float64"
      }
    }
  }'

# 3. Create rules
curl -X POST http://localhost:8080/api/v1/tenants/tenant-123/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Adult Canadian High Value",
    "expression": "User.Age >= 18 && User.Country == \"CA\" && Transaction.Amount > 1000"
  }'

# Response: {"id": "rule-456", ...}

# 4. Evaluate facts
curl -X POST http://localhost:8080/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-123",
    "facts": {
      "User": {
        "Age": 25,
        "Country": "CA"
      },
      "Transaction": {
        "Amount": 1500.0
      }
    }
  }'

# Response: {"results": [{"RuleID": "rule-456", "Matched": true, ...}], ...}
```

### Update Schema Example

```bash
# Add a new field to User object
curl -X PUT http://localhost:8080/api/v1/tenants/tenant-123/schema \
  -H "Content-Type: application/json" \
  -d '{
    "definition": {
      "User": {
        "Age": "int",
        "Country": "string",
        "Email": "string"
      },
      "Transaction": {
        "Amount": "float64"
      }
    }
  }'

# Zero downtime - rules are automatically recompiled
```

---

## Validation Rules

### Schema Validation

When creating or updating schemas, the following rules apply:

#### Structure Rules
- Schema must contain at least 1 object
- Schema can contain maximum 100 objects
- Each object must contain at least 1 field
- Each object can contain maximum 200 fields

#### Object and Field Names
- Must match pattern: `^[a-zA-Z_][a-zA-Z0-9_]*$`
  - Start with letter or underscore
  - Followed by letters, digits, or underscores
- Length: 1-100 characters
- Case-sensitive
- Cannot use reserved keywords: `true`, `false`, `null`, `if`, `else`, `for`, `while`, `return`, `var`, `let`, `const`, `function`, `in`, `as`, `break`, `continue`, `import`, `package`, `namespace`, `loop`, `void`

**Valid Examples:**
- `User`, `user`, `_private`, `User123`, `user_name`, `CamelCase`, `snake_case`

**Invalid Examples:**
- `123User` (starts with digit)
- `User-Name` (contains hyphen)
- `User Name` (contains space)
- `if` (reserved keyword)

#### Type Names
- Must be one of: `int`, `int64`, `float64`, `string`, `bool`, `bytes`, `timestamp`, `duration`
- Case-sensitive (must be lowercase)
- No leading/trailing whitespace

**Valid:** `"Age": "int"`
**Invalid:** `"Age": "INT"`, `"Age": "Integer"`, `"Age": " int "`

### Rule Expression Validation

When creating or updating rules:
- Expression must be valid CEL syntax
- Expression must compile successfully
- Expression must reference valid schema objects and fields
- Expression should evaluate to boolean (non-boolean treated as false)

**Valid Examples:**
```cel
User.Age >= 18
User.Country == "CA" && Transaction.Amount > 1000
User.Email.contains("@example.com")
```

**Invalid Examples:**
```cel
NonExistentObject.Field > 10  // Object not in schema
User.InvalidField == "test"    // Field not in schema
User.Age >= "18"               // Type mismatch (int vs string)
```

---

## Rate Limiting

**Status:** Planned for future release

Rate limits will be enforced per tenant:
- 100 requests/minute for evaluation
- 10 requests/minute for schema updates
- 50 requests/minute for rule management

---

## Versioning

**Current Version:** 1.0

API versioning is included in the URL path (`/api/v1/...`). Future breaking changes will increment the version number.

---

## Support

For questions or issues:
- GitHub Issues: [your-repo-url]
- Email: support@example.com

---

## Changelog

### Version 1.0 (2024-01-15)
- Initial release
- Multi-tenant support
- Dynamic schemas
- CEL rule engine
- Real-time evaluation
- Schema validation
