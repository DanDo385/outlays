// Package api is the read-only public HTTP surface (ARCHITECTURE.md Section 5), served with
// chi over the store's read queries. No endpoint ever returns an unsourced number, and
// /v1/leads returns only published leads.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/djmagro/outlays/core/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

var fiscalYearRe = regexp.MustCompile(`^\d{4}(-\d{2})?$`)

// Server holds dependencies for the API handlers.
type Server struct {
	Pool *pgxpool.Pool
}

// Router builds the chi router with all v1 routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)

	r.Get("/v1/healthz", s.healthz)
	r.Get("/v1/jurisdictions", s.jurisdictions)
	r.Get("/v1/{jur}/years", s.years)
	r.Get("/v1/{jur}/{year}/view", s.view)
	r.Get("/v1/{jur}/{year}/coverage", s.coverage)
	r.Get("/v1/entities", s.entities)
	r.Get("/v1/entities/{id}/flows", s.entityFlows)
	r.Get("/v1/facts", s.facts)
	r.Get("/v1/fact/{id}/provenance", s.provenance)
	r.Get("/v1/compare", s.compare)
	r.Get("/v1/leads", s.leads)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func validFlow(flow string) bool { return flow == "spending" || flow == "revenue" }

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if err := s.Pool.Ping(r.Context()); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "db unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) jurisdictions(w http.ResponseWriter, r *http.Request) {
	js, err := store.Jurisdictions(r.Context(), s.Pool)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jurisdictions": js})
}

func (s *Server) years(w http.ResponseWriter, r *http.Request) {
	jur := chi.URLParam(r, "jur")
	ys, err := store.Years(r.Context(), s.Pool, jur)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jurisdiction": jur, "years": ys})
}

func (s *Server) view(w http.ResponseWriter, r *http.Request) {
	jur := chi.URLParam(r, "jur")
	year := chi.URLParam(r, "year")
	if !fiscalYearRe.MatchString(year) {
		writeErr(w, http.StatusBadRequest, "invalid fiscal year")
		return
	}
	q := r.URL.Query()
	scheme := q.Get("scheme")
	if scheme == "" {
		writeErr(w, http.StatusBadRequest, "scheme is required")
		return
	}
	flow := q.Get("flow")
	if flow == "" {
		flow = "spending"
	}
	if !validFlow(flow) {
		writeErr(w, http.StatusBadRequest, "invalid flow")
		return
	}

	ctx := r.Context()
	var (
		v   *store.View
		err error
	)
	if scheme == "payee" {
		v, err = store.ViewByPayee(ctx, s.Pool, jur, year, flow)
	} else {
		exists, sErr := store.SchemeExists(ctx, s.Pool, scheme)
		if sErr != nil {
			writeErr(w, http.StatusInternalServerError, sErr.Error())
			return
		}
		if !exists {
			writeErr(w, http.StatusBadRequest, "unknown scheme: "+scheme)
			return
		}
		v, err = store.ViewByScheme(ctx, s.Pool, jur, year, flow, scheme)
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) coverage(w http.ResponseWriter, r *http.Request) {
	jur := chi.URLParam(r, "jur")
	year := chi.URLParam(r, "year")
	if !fiscalYearRe.MatchString(year) {
		writeErr(w, http.StatusBadRequest, "invalid fiscal year")
		return
	}
	c, err := store.CoverageFor(r.Context(), s.Pool, jur, year)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) entities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		writeErr(w, http.StatusBadRequest, "q is required")
		return
	}
	hits, err := store.SearchEntities(r.Context(), s.Pool, q, clampLimit(r, 50, 200))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "entities": hits})
}

func (s *Server) entityFlows(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	year := r.URL.Query().Get("year")
	if !fiscalYearRe.MatchString(year) {
		writeErr(w, http.StatusBadRequest, "year is required and must match the fiscal-year pattern")
		return
	}
	ef, err := store.EntityFlowsByDepartment(r.Context(), s.Pool, id, year)
	if err != nil {
		writeErr(w, http.StatusNotFound, "entity not found or query failed")
		return
	}
	writeJSON(w, http.StatusOK, ef)
}

func (s *Server) facts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	year := q.Get("year")
	if year != "" && !fiscalYearRe.MatchString(year) {
		writeErr(w, http.StatusBadRequest, "invalid fiscal year")
		return
	}
	flow := q.Get("flow")
	if flow != "" && !validFlow(flow) {
		writeErr(w, http.StatusBadRequest, "invalid flow")
		return
	}
	limit := clampLimit(r, 100, 1000)
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	rows, err := store.ListFacts(r.Context(), s.Pool, q.Get("jurisdiction"), year, flow, q.Get("payee"), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"limit": limit, "offset": offset, "facts": rows})
}

func (s *Server) provenance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := store.FactProvenance(r.Context(), s.Pool, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeErr(w, http.StatusNotFound, "fact not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) compare(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scheme, code := q.Get("scheme"), q.Get("code")
	jurs := splitCSV(q.Get("jurisdictions"))
	if scheme == "" || code == "" || len(jurs) == 0 {
		writeErr(w, http.StatusBadRequest, "scheme, code, and jurisdictions are required")
		return
	}
	rows, err := store.Compare(r.Context(), s.Pool, scheme, code, jurs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scheme": scheme, "code": code, "results": rows})
}

func (s *Server) leads(w http.ResponseWriter, r *http.Request) {
	// Published-only, always — other statuses are never reachable here (Hard Rule 6).
	if status := r.URL.Query().Get("status"); status != "" && status != "published" {
		writeErr(w, http.StatusBadRequest, "only status=published is available")
		return
	}
	leads, err := store.PublishedLeads(r.Context(), s.Pool)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "published", "leads": leads})
}

func clampLimit(r *http.Request, def, max int) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Ping is a convenience for main to verify DB connectivity at startup.
func Ping(ctx context.Context, pool *pgxpool.Pool) error { return pool.Ping(ctx) }
