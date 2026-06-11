-- Rule: ca_vendor_concentration_department_category_v1
-- Method: L001 — vendor concentration inside buyer/year
-- Basis: docs/leads-methodology.md#l001--vendor-concentration-inside-buyeryear
--
-- A neutral concentration screen: for each (department, acquisition type) group in one
-- jurisdiction-year, find vendors whose share of recorded spending crosses the library's
-- suggested v1 thresholds. Facts plus statistical context only — the SQL computes shares,
-- never conclusions. Thresholds are versioned constants of this rule (changing them is a
-- new rule version), mirrored in the sidecar metadata for transparency.
--
-- Params: $1 jurisdiction, $2 fiscal_year.
WITH fy AS (
  SELECT f.fact_id, f.amount, f.payee_entity, f.raw_sha256
  FROM fiscal_fact f
  WHERE f.jurisdiction = $1 AND f.fiscal_year = $2 AND f.flow = 'spending'
    AND f.grain IN ('transaction', 'award')
    AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)
),
dept AS (
  SELECT DISTINCT ON (a.fact_id) a.fact_id, a.code
  FROM classification_assignment a JOIN fy ON fy.fact_id = a.fact_id
  WHERE a.scheme_id = 'us_ca_department'
  ORDER BY a.fact_id, a.version DESC, a.code
),
acq AS (
  SELECT DISTINCT ON (a.fact_id) a.fact_id, a.code
  FROM classification_assignment a JOIN fy ON fy.fact_id = a.fact_id
  WHERE a.scheme_id = 'us_ca_acquisition_type'
  ORDER BY a.fact_id, a.version DESC, a.code
),
joined AS (
  SELECT fy.fact_id, fy.amount, fy.payee_entity, fy.raw_sha256,
         dept.code AS department, acq.code AS acquisition_type
  FROM fy
  JOIN dept ON dept.fact_id = fy.fact_id
  JOIN acq ON acq.fact_id = fy.fact_id
),
groups AS (
  SELECT department, acquisition_type,
         sum(amount) AS group_amount,
         count(DISTINCT payee_entity) FILTER (WHERE payee_entity IS NOT NULL) AS group_vendor_count
  FROM joined GROUP BY 1, 2
),
vendors AS (
  SELECT department, acquisition_type, payee_entity,
         sum(amount) AS vendor_amount,
         count(*) AS vendor_fact_count,
         array_agg(fact_id::text ORDER BY fact_id) AS fact_ids,
         array_agg(DISTINCT raw_sha256) FILTER (WHERE raw_sha256 IS NOT NULL) AS raw_sha256s
  FROM joined WHERE payee_entity IS NOT NULL GROUP BY 1, 2, 3
)
SELECT
  v.department,
  v.acquisition_type,
  v.payee_entity::text AS payee_entity,
  e.canonical_name AS vendor_name,
  v.vendor_amount::numeric(24,4)::text AS vendor_amount,
  v.vendor_fact_count,
  g.group_amount::numeric(24,4)::text AS group_amount,
  g.group_vendor_count,
  round(v.vendor_amount / NULLIF(g.group_amount, 0), 4)::text AS share,
  v.fact_ids,
  v.raw_sha256s
FROM vendors v
JOIN groups g USING (department, acquisition_type)
JOIN entity e ON e.entity_id = v.payee_entity
WHERE g.group_amount >= 100000          -- configured_min_group_total (L001 suggested default)
  AND v.vendor_fact_count >= 3          -- configured_min_fact_count
  AND g.group_vendor_count >= 2         -- group must have competition to measure share against
  AND v.vendor_amount / NULLIF(g.group_amount, 0) >= 0.50  -- configured_share_threshold
ORDER BY share DESC, v.department, v.acquisition_type, v.payee_entity::text
