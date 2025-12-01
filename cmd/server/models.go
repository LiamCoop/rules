package main

import (
	"time"

	"github.com/liamcoop/rules/multitenantengine"
)

// API Request and Response Models with Swagger annotations

// CreateTenantRequest represents the request body for creating a tenant
type CreateTenantRequest struct {
	Name string `json:"name" example:"Acme Corp" binding:"required"`
} // @name CreateTenantRequest

// TenantResponse represents a tenant in API responses
type TenantResponse struct {
	ID        string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" example:"Acme Corp"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
} // @name TenantResponse

// TenantsListResponse represents the response for listing tenants
type TenantsListResponse struct {
	Tenants []TenantResponse `json:"tenants"`
} // @name TenantsListResponse

// CreateSchemaRequest represents the request body for creating a schema
type CreateSchemaRequest struct {
	Definition multitenantengine.Schema `json:"definition" binding:"required"`
} // @name CreateSchemaRequest

// SchemaResponse represents a schema in API responses
type SchemaResponse struct {
	Version    int                       `json:"version" example:"1"`
	Status     string                    `json:"status" example:"active"`
	Definition multitenantengine.Schema  `json:"definition"`
	CreatedAt  *time.Time                `json:"created_at,omitempty" example:"2024-01-15T10:30:00Z"`
} // @name SchemaResponse

// CreateRuleRequest represents the request body for creating a rule
type CreateRuleRequest struct {
	Name       string `json:"name" example:"Adult User Check" binding:"required"`
	Expression string `json:"expression" example:"User.Age >= 18" binding:"required"`
} // @name CreateRuleRequest

// UpdateRuleRequest represents the request body for updating a rule
type UpdateRuleRequest struct {
	Name       string `json:"name" example:"Adult User Check"`
	Expression string `json:"expression" example:"User.Age >= 18"`
	Active     *bool  `json:"active,omitempty" example:"true"`
} // @name UpdateRuleRequest

// RuleResponse represents a rule in API responses
type RuleResponse struct {
	ID         string    `json:"id" example:"rule-123e4567-e89b-12d3-a456-426614174000"`
	Name       string    `json:"name" example:"Adult User Check"`
	Expression string    `json:"expression" example:"User.Age >= 18"`
	Active     bool      `json:"active" example:"true"`
	CreatedAt  time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt  time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
} // @name RuleResponse

// RulesListResponse represents the response for listing rules
type RulesListResponse struct {
	Rules []RuleResponse `json:"rules"`
} // @name RulesListResponse

// EvaluateRequest represents the request body for evaluating rules
type EvaluateRequest struct {
	TenantID string                 `json:"tenantId" example:"123e4567-e89b-12d3-a456-426614174000" binding:"required"`
	Facts    map[string]interface{} `json:"facts" binding:"required"`
	Rules    []string               `json:"rules,omitempty" example:"rule-123,rule-456"`
} // @name EvaluateRequest

// EvaluationResultResponse represents a single rule evaluation result
type EvaluationResultResponse struct {
	RuleID   string                 `json:"RuleID" example:"rule-123"`
	RuleName string                 `json:"RuleName" example:"Adult User Check"`
	Matched  bool                   `json:"Matched" example:"true"`
	Error    *string                `json:"Error,omitempty"`
	Trace    map[string]interface{} `json:"Trace,omitempty"`
} // @name EvaluationResultResponse

// EvaluateResponse represents the response for rule evaluation
type EvaluateResponse struct {
	Results        []EvaluationResultResponse `json:"results"`
	EvaluationTime string                     `json:"evaluationTime" example:"2.3ms"`
} // @name EvaluateResponse

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"validation failed: schema cannot be empty"`
} // @name ErrorResponse

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
} // @name HealthResponse

// Example schema definition for Swagger documentation
type ExampleSchema struct {
	User struct {
		Age       string `json:"Age" example:"int"`
		Name      string `json:"Name" example:"string"`
		Email     string `json:"Email" example:"string"`
		IsActive  string `json:"IsActive" example:"bool"`
		CreatedAt string `json:"CreatedAt" example:"timestamp"`
	} `json:"User"`
	Transaction struct {
		Amount      string `json:"Amount" example:"float64"`
		Currency    string `json:"Currency" example:"string"`
		ProcessedAt string `json:"ProcessedAt" example:"timestamp"`
	} `json:"Transaction"`
} // @name ExampleSchema

// Example facts for Swagger documentation
type ExampleFacts struct {
	User struct {
		Age       int    `json:"Age" example:"25"`
		Name      string `json:"Name" example:"John Doe"`
		Email     string `json:"Email" example:"john@example.com"`
		IsActive  bool   `json:"IsActive" example:"true"`
		CreatedAt string `json:"CreatedAt" example:"2024-01-15T10:30:00Z"`
	} `json:"User"`
	Transaction struct {
		Amount      float64 `json:"Amount" example:"1500.50"`
		Currency    string  `json:"Currency" example:"USD"`
		ProcessedAt string  `json:"ProcessedAt" example:"2024-01-15T12:00:00Z"`
	} `json:"Transaction"`
} // @name ExampleFacts
