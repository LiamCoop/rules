package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/liamcoop/rules/multitenantengine"
	"github.com/liamcoop/rules/rules"
	_ "github.com/lib/pq"
)

type Server struct {
	db            *sql.DB
	engineManager *multitenantengine.MultiTenantEngineManager
	router        *chi.Mux
}

func NewServer(databaseURL string) (*Server, error) {
	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create engine manager
	engineManager := multitenantengine.NewMultiTenantEngineManager(db)

	// Load all tenants
	log.Println("Loading tenants from database...")
	if err := engineManager.LoadAllTenants(); err != nil {
		return nil, fmt.Errorf("failed to load tenants: %w", err)
	}

	tenants := engineManager.ListTenants()
	log.Printf("Loaded %d tenants: %v", len(tenants), tenants)

	s := &Server{
		db:            db,
		engineManager: engineManager,
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check
	r.Get("/api/v1/health", s.handleHealth)

	// Evaluation
	r.Post("/api/v1/evaluate", s.handleEvaluate)

	// Tenant management
	r.Route("/api/v1/tenants", func(r chi.Router) {
		r.Get("/", s.handleListTenants)
		r.Post("/", s.handleCreateTenant)

		r.Route("/{tenantId}", func(r chi.Router) {
			// Schema management
			r.Post("/schema", s.handleUpdateSchema)
			r.Get("/schema", s.handleGetSchema)

			// Rule management
			r.Post("/rules", s.handleCreateRule)
			r.Get("/rules", s.handleListRules)
			r.Get("/rules/{ruleId}", s.handleGetRule)
			r.Put("/rules/{ruleId}", s.handleUpdateRule)
			r.Delete("/rules/{ruleId}", s.handleDeleteRule)
		})
	})

	s.router = r
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":        "healthy",
		"tenantsLoaded": len(s.engineManager.ListTenants()),
	})
}

// Evaluation handler
func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string         `json:"tenantId"`
		Facts    map[string]any `json:"facts"`
		RuleIDs  []string       `json:"rules,omitempty"` // optional
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.TenantID == "" {
		respondError(w, http.StatusBadRequest, "tenantId is required", nil)
		return
	}

	if req.Facts == nil {
		respondError(w, http.StatusBadRequest, "facts are required", nil)
		return
	}

	// Get tenant's engine
	engine, err := s.engineManager.GetEngine(req.TenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "tenant not found", err)
		return
	}

	startTime := time.Now()

	// Evaluate rules
	var results []*rules.EvaluationResult
	if len(req.RuleIDs) > 0 {
		// Evaluate specific rules
		results = make([]*rules.EvaluationResult, 0, len(req.RuleIDs))
		for _, ruleID := range req.RuleIDs {
			result, err := engine.Evaluate(ruleID, req.Facts)
			if err != nil {
				// Continue on error (might be rule not found)
				log.Printf("Error evaluating rule %s: %v", ruleID, err)
				continue
			}
			results = append(results, result)
		}
	} else {
		// Evaluate all active rules
		results, err = engine.EvaluateAll(req.Facts)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "evaluation failed", err)
			return
		}
	}

	evaluationTime := time.Since(startTime)

	// Format response
	response := map[string]any{
		"results":        results,
		"evaluationTime": evaluationTime.String(),
	}

	respondJSON(w, http.StatusOK, response)
}

// List tenants handler
func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, name, created_at, updated_at FROM tenants ORDER BY created_at DESC")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list tenants", err)
		return
	}
	defer rows.Close()

	type tenant struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	tenants := []tenant{}
	for rows.Next() {
		var t tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan tenant", err)
			return
		}
		tenants = append(tenants, t)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tenants": tenants,
	})
}

// Create tenant handler
func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required", nil)
		return
	}

	var tenantID string
	err := s.db.QueryRow(`
		INSERT INTO tenants (name, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		RETURNING id
	`, req.Name).Scan(&tenantID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create tenant", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"id":   tenantID,
		"name": req.Name,
	})
}

// Update schema handler
func (s *Server) handleUpdateSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req struct {
		Definition multitenantengine.Schema `json:"definition"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Update schema (zero downtime!)
	err := s.engineManager.UpdateTenantSchema(tenantID, req.Definition)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update schema", err)
		return
	}

	// Return success with compiled rule count
	engine, _ := s.engineManager.GetEngine(tenantID)
	store := rules.NewPostgresRuleStore(s.db, tenantID)
	activeRules, _ := store.ListActive()

	respondJSON(w, http.StatusOK, map[string]any{
		"status":          "active",
		"rulesRecompiled": len(activeRules),
	})

	_ = engine // Silence unused warning
}

// Get schema handler
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var schemaJSON []byte
	var version int
	err := s.db.QueryRow(`
		SELECT version, definition
		FROM schemas
		WHERE tenant_id = $1 AND active = true
	`, tenantID).Scan(&version, &schemaJSON)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "schema not found", nil)
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get schema", err)
		return
	}

	var schema multitenantengine.Schema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to parse schema", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"version":    version,
		"definition": schema,
	})
}

// Create rule handler
func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var req struct {
		Name       string `json:"name"`
		Expression string `json:"expression"`
		Active     bool   `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Name == "" || req.Expression == "" {
		respondError(w, http.StatusBadRequest, "name and expression are required", nil)
		return
	}

	// Get engine to add and compile the rule
	engine, err := s.engineManager.GetEngine(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "tenant not found", err)
		return
	}

	// Create rule
	rule := &rules.Rule{
		Name:       req.Name,
		Expression: req.Expression,
		Active:     req.Active,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Generate UUID for rule ID
	rule.ID = generateUUID()

	// Add rule (this validates and compiles it)
	if err := engine.AddRule(rule); err != nil {
		respondError(w, http.StatusBadRequest, "failed to add rule", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"id":         rule.ID,
		"name":       rule.Name,
		"expression": rule.Expression,
		"active":     rule.Active,
	})
}

// List rules handler
func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	rows, err := s.db.Query(`
		SELECT id, name, expression, active, created_at, updated_at
		FROM rules
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list rules", err)
		return
	}
	defer rows.Close()

	rulesList := []*rules.Rule{}
	for rows.Next() {
		var r rules.Rule
		if err := rows.Scan(&r.ID, &r.Name, &r.Expression, &r.Active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan rule", err)
			return
		}
		rulesList = append(rulesList, &r)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"rules": rulesList,
	})
}

// Get rule handler
func (s *Server) handleGetRule(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	ruleID := chi.URLParam(r, "ruleId")

	store := rules.NewPostgresRuleStore(s.db, tenantID)
	rule, err := store.Get(ruleID)
	if err != nil {
		respondError(w, http.StatusNotFound, "rule not found", err)
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

// Update rule handler
func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	ruleID := chi.URLParam(r, "ruleId")

	var req struct {
		Name       string `json:"name"`
		Expression string `json:"expression"`
		Active     bool   `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Get engine
	engine, err := s.engineManager.GetEngine(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "tenant not found", err)
		return
	}

	// Update rule
	rule := &rules.Rule{
		ID:         ruleID,
		Name:       req.Name,
		Expression: req.Expression,
		Active:     req.Active,
		UpdatedAt:  time.Now(),
	}

	if err := engine.UpdateRule(rule); err != nil {
		respondError(w, http.StatusBadRequest, "failed to update rule", err)
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

// Delete rule handler
func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	ruleID := chi.URLParam(r, "ruleId")

	engine, err := s.engineManager.GetEngine(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "tenant not found", err)
		return
	}

	if err := engine.DeleteRule(ruleID); err != nil {
		respondError(w, http.StatusNotFound, "rule not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]string{
		"error": message,
	}
	if err != nil {
		response["details"] = err.Error()
	}
	respondJSON(w, status, response)
}

func generateUUID() string {
	// Simple UUID generation - in production, use github.com/google/uuid
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

func main() {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Create server
	server, err := NewServer(databaseURL)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.db.Close()

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
