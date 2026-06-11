// Package leads is the private leads scaffold (task S11, D30). Rules are versioned SQL
// files with sidecar metadata, embedded in the binary; the runner executes one rule over
// one jurisdiction-year and writes draft leads — facts plus statistical context, never
// conclusions (Hard Rule 6). Every status change after that is a human act: an append-only
// lead_event row carrying a mandatory reviewer handle. Nothing reaches the public endpoint
// unless the latest event says 'published'.
package leads

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed rules/*.sql rules/*.json
var rulesFS embed.FS

// leadNS namespaces deterministic lead ids (UUIDv5), so re-running a rule over unchanged
// evidence is an ON CONFLICT no-op (same idempotency discipline as D21).
var leadNS = uuid.MustParse("3a0c0b6e-1d2f-5a4b-8c6d-7e8f9a0b1c23")

// MethodologyDoc is the research deliverable every rule citation must anchor to.
const MethodologyDoc = "docs/leads-methodology.md"

// bannedWording enforces the library's rule discipline mechanically: generated lead text
// must never assert wrongdoing or intent (Hard Rule 6).
var bannedWording = regexp.MustCompile(`(?i)\b(fraud|fraudulent|corrupt|corruption|collusion|colluded|collude|bid[- ]?rigging|rigged|kickback|bribe|bribery|evade|evasion|intended to|intentional)\b`)

// Meta is a rule's sidecar metadata (rules/<ruleId>.json).
type Meta struct {
	RuleID            string          `json:"ruleId"`
	RuleVersion       int             `json:"ruleVersion"`
	MethodID          string          `json:"methodId"`
	Title             string          `json:"title"`
	Citation          []string        `json:"citation"`
	Severity          string          `json:"severity"`
	RequiredFields    []string        `json:"requiredFields"`
	Params            json.RawMessage `json:"params"`
	SafePublicWording string          `json:"safePublicWording"`
	Limitations       string          `json:"limitations"`
}

// Rule is a loaded, validated rule: metadata plus its versioned SQL.
type Rule struct {
	Meta Meta
	SQL  string
}

// Rules lists the embedded rule ids.
func Rules() ([]string, error) {
	entries, err := rulesFS.ReadDir("rules")
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// LoadRule loads and validates one embedded rule. Validation enforces Hard Rule 5 (a rule
// needs an id and a citation) and the methodology library's discipline: the first citation
// must anchor into the library itself, severity can never be 'high' for an automated rule,
// and the safety texts must be present.
func LoadRule(ruleID string) (*Rule, error) {
	metaRaw, err := rulesFS.ReadFile("rules/" + ruleID + ".json")
	if err != nil {
		return nil, fmt.Errorf("unknown rule %q: %w", ruleID, err)
	}
	sqlRaw, err := rulesFS.ReadFile("rules/" + ruleID + ".sql")
	if err != nil {
		return nil, fmt.Errorf("rule %q has no SQL: %w", ruleID, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(metaRaw)))
	dec.DisallowUnknownFields()
	var m Meta
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("rule %q metadata: %w", ruleID, err)
	}
	if m.RuleID != ruleID {
		return nil, fmt.Errorf("rule %q metadata declares ruleId %q", ruleID, m.RuleID)
	}
	if m.RuleVersion < 1 {
		return nil, fmt.Errorf("rule %q: ruleVersion must be >= 1", ruleID)
	}
	if len(m.Citation) == 0 || !strings.HasPrefix(m.Citation[0], MethodologyDoc+"#") {
		return nil, fmt.Errorf("rule %q: first citation must anchor into %s", ruleID, MethodologyDoc)
	}
	if m.Severity != "info" && m.Severity != "low" && m.Severity != "medium" {
		return nil, fmt.Errorf("rule %q: severity %q not allowed for an automated rule (no automated lead is 'high')", ruleID, m.Severity)
	}
	if strings.TrimSpace(m.SafePublicWording) == "" || strings.TrimSpace(m.Limitations) == "" {
		return nil, fmt.Errorf("rule %q: safePublicWording and limitations are required", ruleID)
	}
	if len(m.RequiredFields) == 0 {
		return nil, fmt.Errorf("rule %q: requiredFields is required", ruleID)
	}
	return &Rule{Meta: m, SQL: string(sqlRaw)}, nil
}

// Draft is one generated lead before insertion.
type Draft struct {
	LeadID  string
	FactIDs []string
	Score   string // the vendor share, a ratio (not money)
	Body    map[string]any
}

// RunResult summarizes a runner execution.
type RunResult struct {
	RuleID       string `json:"ruleId"`
	Jurisdiction string `json:"jurisdiction"`
	FiscalYear   string `json:"fiscalYear"`
	Matches      int    `json:"matches"`
	Inserted     int    `json:"inserted"`
	AlreadyKnown int    `json:"alreadyKnown"`
}

// Run executes a rule over one jurisdiction-year and writes draft leads. Lead ids are
// deterministic over (rule id+version, subject, evidence fact set), so unchanged evidence
// re-inserts nothing and changed evidence yields a new draft for review.
func Run(ctx context.Context, pool *pgxpool.Pool, r *Rule, jur, year string) (*RunResult, error) {
	rows, err := pool.Query(ctx, r.SQL, jur, year)
	if err != nil {
		return nil, fmt.Errorf("rule %s: %w", r.Meta.RuleID, err)
	}
	defer rows.Close()

	drafts := []Draft{}
	for rows.Next() {
		var (
			department, acquisitionType, payeeEntity, vendorName string
			vendorAmount, groupAmount, share                     string
			vendorFactCount, groupVendorCount                    int64
			factIDs, rawShas                                     []string
		)
		if err := rows.Scan(&department, &acquisitionType, &payeeEntity, &vendorName,
			&vendorAmount, &vendorFactCount, &groupAmount, &groupVendorCount, &share,
			&factIDs, &rawShas); err != nil {
			return nil, fmt.Errorf("rule %s scan: %w", r.Meta.RuleID, err)
		}

		title := fmt.Sprintf("Vendor concentration screen: %s — %s / %s (%s %s)",
			vendorName, department, acquisitionType, jur, year)
		summary := fmt.Sprintf(
			"%s accounts for %s (%s of %s USD) of recorded %s spending coded to %s in %s fiscal year %s, "+
				"across %d line-item facts; the group has %d distinct vendors. %s",
			vendorName, sharePercent(share), vendorAmount, groupAmount,
			acquisitionType, department, jur, year,
			vendorFactCount, groupVendorCount, r.Meta.SafePublicWording)
		if bannedWording.MatchString(title) || bannedWording.MatchString(summary) {
			return nil, fmt.Errorf("rule %s: generated wording violates the rule discipline (banned term)", r.Meta.RuleID)
		}

		d := Draft{
			LeadID:  leadID(r.Meta, jur, year, []string{department, acquisitionType, payeeEntity}, factIDs),
			FactIDs: factIDs,
			Score:   share,
			Body: map[string]any{
				"ruleId":                r.Meta.RuleID,
				"ruleVersion":           r.Meta.RuleVersion,
				"methodId":              r.Meta.MethodID,
				"jurisdiction":          jur,
				"fiscalYear":            year,
				"leadTitle":             title,
				"leadSummary":           summary,
				"severity":              r.Meta.Severity,
				"basisCitation":         r.Meta.Citation,
				"sourceRawSha256Values": rawShas,
				"requiredFieldsPresent": r.Meta.RequiredFields,
				"methodLimitations":     r.Meta.Limitations,
				"safePublicWording":     r.Meta.SafePublicWording,
				"params":                json.RawMessage(r.Meta.Params),
				"subject": map[string]any{
					"department":       department,
					"acquisitionType":  acquisitionType,
					"payeeEntity":      payeeEntity,
					"vendorName":       vendorName,
					"vendorAmount":     vendorAmount,
					"vendorFactCount":  vendorFactCount,
					"groupAmount":      groupAmount,
					"groupVendorCount": groupVendorCount,
					"share":            share,
				},
			},
		}
		drafts = append(drafts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	res := &RunResult{RuleID: r.Meta.RuleID, Jurisdiction: jur, FiscalYear: year, Matches: len(drafts)}
	if len(drafts) == 0 {
		return res, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for _, d := range drafts {
		body, err := json.Marshal(d.Body)
		if err != nil {
			return nil, err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO lead (lead_id, rule_id, fact_ids, score, generated_query, status, body)
			VALUES ($1, $2, $3::uuid[], $4::numeric, $5, 'draft', $6)
			ON CONFLICT (lead_id) DO NOTHING`,
			d.LeadID, r.Meta.RuleID, d.FactIDs, d.Score, r.SQL, body)
		if err != nil {
			return nil, fmt.Errorf("insert lead: %w", err)
		}
		if tag.RowsAffected() == 1 {
			res.Inserted++
		} else {
			res.AlreadyKnown++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

// leadID derives the deterministic lead id from rule identity, subject, and the exact
// evidence fact set (sorted), so a lead names one reviewable claim about one body of
// evidence.
func leadID(m Meta, jur, year string, subject, factIDs []string) string {
	sorted := append([]string(nil), factIDs...)
	sort.Strings(sorted)
	key := fmt.Sprintf("%s|v%d|%s|%s|%s|%s",
		m.RuleID, m.RuleVersion, jur, year, strings.Join(subject, "\x00"), strings.Join(sorted, ","))
	return uuid.NewSHA1(leadNS, []byte(key)).String()
}

// sharePercent renders a 4dp ratio string like "0.8558" as "85.58%" using string math
// only (the ratio is statistical context, but float formatting is still avoided).
func sharePercent(share string) string {
	intPart, frac, ok := strings.Cut(share, ".")
	if !ok {
		frac = ""
	}
	frac = (frac + "0000")[:4]
	whole := strings.TrimLeft(intPart, "-") + frac[:2]
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	sign := ""
	if strings.HasPrefix(share, "-") {
		sign = "-"
	}
	return sign + whole + "." + frac[2:] + "%"
}

// Status workflow ----------------------------------------------------------------------

// statusCTE resolves each lead's current status: the latest event, else the initial row
// status (always 'draft' from the runner).
const statusCTE = `
	SELECT l.lead_id,
	       COALESCE(ev.status, l.status) AS status,
	       ev.reviewer, ev.note, ev.created_at
	FROM lead l
	LEFT JOIN LATERAL (
		SELECT e.status, e.reviewer, e.note, e.created_at
		FROM lead_event e WHERE e.lead_id = l.lead_id
		ORDER BY e.created_at DESC, e.event_id DESC LIMIT 1
	) ev ON true`

// Summary is one row of the review CLI's list view.
type Summary struct {
	LeadID   string  `json:"leadId"`
	RuleID   string  `json:"ruleId"`
	Status   string  `json:"status"`
	Score    *string `json:"score"`
	Title    string  `json:"title"`
	Reviewer *string `json:"reviewer"`
}

// List returns leads (optionally filtered by current status), newest first. Private:
// serves the review CLI only, never the public API.
func List(ctx context.Context, pool *pgxpool.Pool, status string) ([]Summary, error) {
	q := `
	WITH cur AS (` + statusCTE + `)
	SELECT l.lead_id::text, l.rule_id, cur.status, l.score::text,
	       COALESCE(l.body->>'leadTitle', ''), cur.reviewer
	FROM lead l JOIN cur ON cur.lead_id = l.lead_id`
	args := []any{}
	if status != "" {
		q += ` WHERE cur.status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY l.inserted_at DESC, l.lead_id`
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.LeadID, &s.RuleID, &s.Status, &s.Score, &s.Title, &s.Reviewer); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Event is one review action in a lead's history.
type Event struct {
	Status    string  `json:"status"`
	Reviewer  string  `json:"reviewer"`
	Note      *string `json:"note"`
	CreatedAt string  `json:"createdAt"`
}

// Detail is the review CLI's inspect view: the full lead document plus its event history.
type Detail struct {
	LeadID         string          `json:"leadId"`
	RuleID         string          `json:"ruleId"`
	Status         string          `json:"status"`
	Score          *string         `json:"score"`
	FactIDs        []string        `json:"factIds"`
	Body           json.RawMessage `json:"body"`
	GeneratedQuery string          `json:"generatedQuery"`
	Events         []Event         `json:"events"`
}

// Inspect returns one lead with its full body and event history.
func Inspect(ctx context.Context, pool *pgxpool.Pool, leadID string) (*Detail, error) {
	var d Detail
	err := pool.QueryRow(ctx, `
		WITH cur AS (`+statusCTE+`)
		SELECT l.lead_id::text, l.rule_id, cur.status, l.score::text,
		       l.fact_ids::text[], l.body, l.generated_query
		FROM lead l JOIN cur ON cur.lead_id = l.lead_id
		WHERE l.lead_id = $1`, leadID,
	).Scan(&d.LeadID, &d.RuleID, &d.Status, &d.Score, &d.FactIDs, &d.Body, &d.GeneratedQuery)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT status, reviewer, note, created_at::text FROM lead_event
		WHERE lead_id = $1 ORDER BY created_at, event_id`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Status, &e.Reviewer, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		d.Events = append(d.Events, e)
	}
	return &d, rows.Err()
}

// SetStatus appends a review event (the only way a lead's status ever changes). The
// reviewer handle is mandatory (Hard Rule 5/6). Publishing additionally requires the lead
// document to satisfy the library's publication policy: citation anchored in the
// methodology doc, limitations, safe wording, and fact links all present.
func SetStatus(ctx context.Context, pool *pgxpool.Pool, leadID, status, reviewer, note string) error {
	switch status {
	case "reviewed", "published", "dismissed":
	default:
		return fmt.Errorf("status must be reviewed, published, or dismissed (drafts are machine-made)")
	}
	if strings.TrimSpace(reviewer) == "" {
		return fmt.Errorf("a reviewer handle is required")
	}

	d, err := Inspect(ctx, pool, leadID)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("lead %s not found", leadID)
	}
	if status == "published" {
		var body map[string]any
		if err := json.Unmarshal(d.Body, &body); err != nil {
			return fmt.Errorf("lead body unreadable: %w", err)
		}
		citations, _ := body["basisCitation"].([]any)
		anchored := false
		for _, c := range citations {
			if s, ok := c.(string); ok && strings.HasPrefix(s, MethodologyDoc+"#") {
				anchored = true
			}
		}
		switch {
		case !anchored:
			return fmt.Errorf("publication blocked: no citation anchored in %s", MethodologyDoc)
		case body["methodLimitations"] == "" || body["methodLimitations"] == nil:
			return fmt.Errorf("publication blocked: limitations text missing")
		case body["safePublicWording"] == "" || body["safePublicWording"] == nil:
			return fmt.Errorf("publication blocked: safe public wording missing")
		case len(d.FactIDs) == 0:
			return fmt.Errorf("publication blocked: no fact links")
		}
		title, _ := body["leadTitle"].(string)
		summary, _ := body["leadSummary"].(string)
		if bannedWording.MatchString(title) || bannedWording.MatchString(summary) {
			return fmt.Errorf("publication blocked: wording violates the rule discipline")
		}
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO lead_event (lead_id, status, reviewer, note)
		VALUES ($1, $2, $3, NULLIF($4, ''))`, leadID, status, reviewer, note)
	if err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	return nil
}
