# API Documentation Guide for UI Developers

Welcome! This guide explains how to use the API documentation for the Multi-Tenant Rules Engine.

---

## Quick Start

### 1. Start the Server

```bash
go run ./cmd/server
```

The server starts on `http://localhost:8080`

### 2. Access Interactive Documentation

Open your browser and navigate to:

**Swagger UI:** `http://localhost:8080/swagger/index.html`

This provides:
- Interactive API documentation
- Ability to test endpoints directly from the browser
- Request/response examples
- Schema definitions

### 3. Download OpenAPI Spec

The OpenAPI 3.0 specification is available at:

**JSON:** `http://localhost:8080/swagger/doc.json`
**YAML:** Available at `cmd/server/docs/swagger.yaml`

You can import this into tools like:
- Postman
- Insomnia
- OpenAPI Generator (for generating client SDKs)

---

## Documentation Files

We provide two types of documentation:

### 1. **API.md** - Human-Friendly Guide

**Location:** `/API.md`

This is a comprehensive markdown document with:
- Complete endpoint reference
- Request/response examples
- Validation rules
- curl examples
- Common workflows
- Error handling guide

**Best for:**
- Learning the API
- Quick reference
- Understanding validation rules
- Copy-paste curl commands

### 2. **Swagger/OpenAPI** - Interactive & Machine-Readable

**Access:** `http://localhost:8080/swagger/index.html` (when server is running)

This provides:
- Interactive testing
- Try-it-out functionality
- Auto-generated from code
- Always up-to-date with implementation
- Machine-readable spec for code generation

**Best for:**
- Testing endpoints interactively
- Generating client code
- Integration with API tools
- Ensuring documentation matches code

---

## Using Swagger UI

### Testing an Endpoint

1. Navigate to `http://localhost:8080/swagger/index.html`
2. Find the endpoint you want to test (e.g., `/api/v1/tenants`)
3. Click on the endpoint to expand it
4. Click "Try it out"
5. Fill in the request parameters/body
6. Click "Execute"
7. View the response below

### Example: Create a Tenant

1. Go to **POST /api/v1/tenants**
2. Click "Try it out"
3. Edit the request body:
   ```json
   {
     "name": "My Test Tenant"
   }
   ```
4. Click "Execute"
5. See the 201 response with the created tenant

### Example: Evaluate Rules

1. First create a tenant and schema (as above)
2. Create a rule via **POST /api/v1/tenants/{tenantId}/rules**
3. Go to **POST /api/v1/evaluate**
4. Fill in the request body with your tenant ID and facts
5. Execute and see which rules matched!

---

## Generating Client Code

You can generate client libraries in various languages using the OpenAPI spec:

### Using OpenAPI Generator

```bash
# Install openapi-generator
npm install @openapitools/openapi-generator-cli -g

# Generate TypeScript client
openapi-generator-cli generate \
  -i http://localhost:8080/swagger/doc.json \
  -g typescript-axios \
  -o ./generated-client

# Generate Python client
openapi-generator-cli generate \
  -i http://localhost:8080/swagger/doc.json \
  -g python \
  -o ./python-client

# Generate Go client
openapi-generator-cli generate \
  -i http://localhost:8080/swagger/doc.json \
  -g go \
  -o ./go-client
```

### Supported Languages

OpenAPI Generator supports 50+ languages including:
- TypeScript/JavaScript
- Python
- Java
- C#
- Go
- Ruby
- PHP
- Kotlin
- Swift
- And many more...

---

## Available Endpoints (Quick Reference)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/tenants` | List all tenants |
| POST | `/api/v1/tenants` | Create a tenant |
| POST | `/api/v1/tenants/{id}/schema` | Create schema |
| PUT | `/api/v1/tenants/{id}/schema` | Update schema |
| GET | `/api/v1/tenants/{id}/schema` | Get schema |
| POST | `/api/v1/tenants/{id}/rules` | Create rule |
| GET | `/api/v1/tenants/{id}/rules` | List rules |
| GET | `/api/v1/tenants/{id}/rules/{ruleId}` | Get rule |
| PUT | `/api/v1/tenants/{id}/rules/{ruleId}` | Update rule |
| DELETE | `/api/v1/tenants/{id}/rules/{ruleId}` | Delete rule |
| POST | `/api/v1/evaluate` | Evaluate rules |

---

## Supported Data Types

When creating schemas, use these exact type names (case-sensitive):

| Type | Example Value | Description |
|------|---------------|-------------|
| `int` | `42` | Integer |
| `int64` | `9223372036854775807` | 64-bit integer |
| `float64` | `3.14159` | Float |
| `string` | `"hello"` | String |
| `bool` | `true` | Boolean |
| `bytes` | Binary data | Byte array |
| `timestamp` | `"2024-01-15T10:30:00Z"` | RFC3339 timestamp |
| `duration` | `"1h30m"` | Duration |

---

## Common Workflows

### 1. Complete Setup: Tenant → Schema → Rules → Evaluate

```bash
# 1. Create tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "Acme Corp"}'
# Response: {"id": "tenant-123", ...}

# 2. Create schema
curl -X POST http://localhost:8080/api/v1/tenants/tenant-123/schema \
  -H "Content-Type: application/json" \
  -d '{
    "definition": {
      "User": {"Age": "int", "Country": "string"},
      "Transaction": {"Amount": "float64"}
    }
  }'

# 3. Create rule
curl -X POST http://localhost:8080/api/v1/tenants/tenant-123/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Adult High Value",
    "expression": "User.Age >= 18 && Transaction.Amount > 1000"
  }'

# 4. Evaluate
curl -X POST http://localhost:8080/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-123",
    "facts": {
      "User": {"Age": 25, "Country": "US"},
      "Transaction": {"Amount": 1500.0}
    }
  }'
```

### 2. Update Schema (Zero Downtime)

```bash
# Add new fields to existing schema
curl -X PUT http://localhost:8080/api/v1/tenants/tenant-123/schema \
  -H "Content-Type: application/json" \
  -d '{
    "definition": {
      "User": {
        "Age": "int",
        "Country": "string",
        "Email": "string"  // New field
      },
      "Transaction": {"Amount": "float64"}
    }
  }'
# Rules are automatically recompiled
```

---

## Validation Rules

### Schema Validation

- **Minimum:** 1 object
- **Maximum:** 100 objects per schema
- **Fields:** 1-200 fields per object
- **Object/Field Names:**
  - Pattern: `^[a-zA-Z_][a-zA-Z0-9_]*$`
  - Length: 1-100 characters
  - No reserved keywords: `true`, `false`, `null`, `if`, `else`, `for`, etc.

### Common Validation Errors

**❌ Invalid:**
```json
{
  "123User": {"field": "int"}  // Starts with digit
}
```

**✅ Valid:**
```json
{
  "User123": {"field": "int"}
}
```

**❌ Invalid:**
```json
{
  "User": {"Age": "Integer"}  // Wrong type name
}
```

**✅ Valid:**
```json
{
  "User": {"Age": "int"}  // Correct type name (lowercase)
}
```

---

## Error Handling

All errors return JSON with this format:

```json
{
  "error": "Human-readable error message"
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success (GET) |
| 201 | Created (POST) |
| 204 | Success, no content (DELETE) |
| 400 | Bad Request (validation error) |
| 404 | Not Found |
| 409 | Conflict (duplicate) |
| 500 | Server Error |

---

## Development Workflow

### Adding New Endpoints

If you're extending the API:

1. **Add Handler Function** in `cmd/server/main.go`
2. **Add Swagger Annotations** above the handler:
   ```go
   // handleNewEndpoint godoc
   // @Summary Short summary
   // @Description Detailed description
   // @Tags tag-name
   // @Accept json
   // @Produce json
   // @Param param-name body RequestType true "Description"
   // @Success 200 {object} ResponseType
   // @Failure 400 {object} ErrorResponse
   // @Router /api/v1/path [method]
   func (s *Server) handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
     // ...
   }
   ```
3. **Regenerate Docs:**
   ```bash
   cd cmd/server
   ~/go/bin/swag init -g main.go -o ./docs --parseDependency --parseInternal
   ```
4. **Rebuild Server:**
   ```bash
   go build ./cmd/server
   ```

### Updating Existing Endpoints

1. Update handler and/or annotations
2. Regenerate docs (step 3 above)
3. Rebuild server (step 4 above)
4. Documentation automatically updates in Swagger UI

---

## Tips for UI Developers

### 1. Start with Swagger UI
- Use it to understand the API
- Test endpoints before writing code
- See real request/response examples

### 2. Generate a Client Library
- Don't write HTTP calls manually
- Use OpenAPI Generator for type-safe clients
- Saves time and reduces errors

### 3. Check API.md for Details
- Validation rules
- Data type constraints
- Common workflows
- Error handling strategies

### 4. Test Edge Cases
- Empty schemas (should fail)
- Invalid types (should fail)
- Reserved keywords (should fail)
- Maximum limits (100 objects, 200 fields)

### 5. Handle Errors Gracefully
- All endpoints return JSON error format
- Check HTTP status codes
- Display user-friendly error messages

---

## Example UI Flow

### Tenant Management Page

```typescript
// 1. Fetch tenants
const tenants = await fetch('/api/v1/tenants').then(r => r.json());

// 2. Create new tenant
const newTenant = await fetch('/api/v1/tenants', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name: 'New Tenant' })
}).then(r => r.json());
```

### Schema Builder Page

```typescript
// 1. Get current schema
const schema = await fetch(`/api/v1/tenants/${tenantId}/schema`)
  .then(r => r.json());

// 2. Update schema
const updated = await fetch(`/api/v1/tenants/${tenantId}/schema`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    definition: {
      User: { Age: 'int', Email: 'string' },
      Transaction: { Amount: 'float64' }
    }
  })
}).then(r => r.json());
```

### Rule Evaluation Page

```typescript
// Evaluate facts against rules
const results = await fetch('/api/v1/evaluate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    tenantId: 'tenant-123',
    facts: {
      User: { Age: 25, Email: 'user@example.com' },
      Transaction: { Amount: 1500.0 }
    }
  })
}).then(r => r.json());

// Show which rules matched
results.results.forEach(result => {
  if (result.Matched) {
    console.log(`✓ ${result.RuleName} matched`);
  }
});
```

---

## Support & Resources

- **Markdown Docs:** `/API.md` - Comprehensive reference guide
- **Swagger UI:** `http://localhost:8080/swagger/index.html` - Interactive docs
- **OpenAPI Spec:** `http://localhost:8080/swagger/doc.json` - Machine-readable
- **Source Code:** Check handler implementations in `cmd/server/main.go`
- **Schema Validation:** See `SCHEMA_VALIDATION_REQUIREMENTS.md` for detailed rules

---

## Next Steps

1. **Start the server:** `go run ./cmd/server`
2. **Open Swagger UI:** http://localhost:8080/swagger/index.html
3. **Try the examples** in the "Try it out" sections
4. **Read API.md** for detailed information
5. **Generate a client** for your preferred language
6. **Build your UI!**

Happy coding! 🚀
