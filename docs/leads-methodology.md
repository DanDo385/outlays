# Leads methodology library

## Addendum gate

A2 — published, citable procurement red-flag methodologies.

Priority:

This blocks builder task S11, the private leads scaffold.

## EVR status

### Execute

Reviewed citable public methodologies and repo schema for implementable procurement red flags:

- U.S. GAO purchase-card audit guide.
- U.S. DOJ Antitrust Division / Procurement Collusion Strike Force red flags.
- OECD bid-rigging guidance and detection-list summaries.
- Palm Beach County Office of Inspector General split-purchase advisory citing GAO and IACRC.
- International Anti-Corruption Resource Center procurement fraud red flags.
- Academic cartel-screen literature using public procurement data.
- Outlays schema fields in `core/migrations/00001_schema.sql` and assignment model.

### Verify

Verified source snippets used:

- GAO data-mining methods include split transactions, unusual patterns, and vendor analysis.
- DOJ MAPS framework includes Market, Applications, Patterns, and Suspicious behavior red flags.
- DOJ antitrust primer states bid-rigging patterns include identical bids, high bids vs estimates, qualified bidders failing to bid, same company repeatedly winning, identical handwriting/errors, and taking turns winning.
- OECD guidance says red flags include unusual bidding and pricing patterns, cover bidding, bid suppression, bid rotation, and market allocation, and cautions that indicators are not proof.
- PBC OIG split-purchase advisory lists sequential purchase orders/invoices and similar procurements from the same supplier just under limits.
- IACRC lists split purchases, collusive bidding, unjustified sole source, duplicate invoices, and change-order abuse as procurement red flags.
- Academic screens use number of bids, single bidding, price-to-estimate ratios, Benford-style digit tests, subcontracting, consortia, buyer/market counts, ownership/co-bidding network structure, and winner concentration.

### Report

This file defines the initial private lead rules library. It is intentionally conservative and excludes rules requiring assertions of intent.

## Rule discipline

A lead is not an accusation.

A lead says:

```text
These public records match a published red-flag pattern and may deserve review.
```

A lead must not say:

```text
fraud
corruption
bid rigging occurred
the vendor colluded
the agency intended to evade thresholds
```

unless an official adjudication or enforcement action is itself the cited source.

## Implementation assumptions

The current Outlays schema supports:

- `fiscal_fact.amount`
- `fiscal_fact.occurred_on`
- `fiscal_fact.description`
- `fiscal_fact.payee_entity`
- `fiscal_fact.payer_entity`
- `fiscal_fact.derivation_query`
- `fiscal_fact.raw_sha256`
- `entity.canonical_name`
- `entity.uei`
- `entity.ein`
- `entity_alias.name_raw`
- `classification_assignment`
- source categories such as department and acquisition type
- lead rows with `rule_id`

Some methods require future fields not yet present in every source, such as losing bidder identities, bid counts, estimates, invoice IDs, purchase-order IDs, unit quantities, or contract thresholds. Those are marked `future-source`.

## Minimum lead output shape

Each rule should output:

```text
rule_id
rule_version
jurisdiction
fiscal_year
lead_title
lead_summary
severity: info | low | medium | high
basis_citation
source_fact_ids
source_raw_sha256_values
required_fields_present
method_limitations
safe_public_wording
review_status: draft
```

## Method library

### L001 — vendor concentration inside buyer/year

Citation:

- DOJ Red Flags of Collusion, MAPS framework: `Patterns` include vendors taking turns winning awards, one vendor consistently winning, or specific vendors always winning similar amounts of work. URL: https://www.justice.gov/atr/red-flags-collusion
- IACRC red flags include continuous favorable treatment and high-volume purchases from a particular vendor. URL: https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags

Implementable rule:

For each `(jurisdiction, fiscal_year, buyer_dimension, source_category)` group, compute vendor spend share:

```text
vendor_share = sum(amount for vendor) / sum(amount for group)
```

Emit draft lead when all are true:

```text
group_total >= configured_min_group_total
vendor_share >= configured_share_threshold
vendor_fact_count >= configured_min_fact_count
group_vendor_count >= 2
```

Suggested default for first private CA rule:

```text
buyer_dimension = us_ca_department
source_category = us_ca_acquisition_type
configured_share_threshold = 0.50
configured_min_fact_count = 3
configured_min_group_total = 100000
```

Required fields:

- amount
- fiscal_year
- payee entity or vendor alias
- buyer/department classification
- source category classification if grouping below department
- fact IDs and provenance

Safe wording:

```text
One vendor accounts for a high share of this department/category/year in the records loaded by Outlays.
```

Limitations:

High concentration can be normal for specialized goods, emergencies, statewide contracts, framework agreements, utilities, or sole-source markets.

Status:

`ready-now` for CA procurement and other award/transaction data with vendor and department/category fields.

### L002 — repeated awards just below a procurement threshold

Citation:

- GAO-04-87G, purchase-card audit guide: data mining includes split transactions, frequent amounts just under a threshold, and multiple transactions with same vendor on the same day by the same cardholder totaling over a threshold. URL: https://www.gao.gov/assets/gao-04-87g.pdf
- Palm Beach County OIG Tips and Trends #2025-0002: split-purchase indicators include two or more similar procurements from the same supplier just under competitive-bidding or review limits, sequential purchase orders or invoices under limits, and similar purchases within a brief time period. URL: https://pbc.gov/oig/docs/advisories/Tips_and_Trends-2025-0002-Splitting_Purchases.pdf
- IACRC lists split purchases as dividing a large procurement into smaller orders to bypass thresholds, with sequential purchase orders just under the review limit as a red flag. URL: https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags

Implementable rule:

For each `(buyer, vendor, item/category)` and rolling time window, find awards where:

```text
amount >= threshold * lower_band
amount < threshold
count >= configured_min_count
sum(amount) >= threshold
```

Suggested default:

```text
lower_band = 0.85
window_days = 30
configured_min_count = 2
```

Required fields:

- amount
- occurred_on or award date
- vendor
- buyer/department/cardholder if available
- item/category/acquisition type or normalized description
- applicable procurement threshold table

Safe wording:

```text
Multiple similar purchases appear near but below a configured procurement threshold within a short period.
```

Limitations:

Requires accurate threshold tables by jurisdiction, year, category, and purchase method. Without threshold tables, this rule must remain disabled or informational only.

Status:

`future-source` until threshold config and item/date quality are available.

### L003 — sequential same-vendor same-day or short-window purchases

Citation:

- GAO-04-87G: split transactions include multiple transactions with the same vendor on the same day by the same cardholder totaling above the micropurchase threshold. URL: https://www.gao.gov/assets/gao-04-87g.pdf
- PBC OIG split-purchase advisory: sequential purchase orders or invoices under review/approval limits are split-purchase red flags. URL: https://pbc.gov/oig/docs/advisories/Tips_and_Trends-2025-0002-Splitting_Purchases.pdf

Implementable rule:

For each `(buyer, vendor, date)` or `(buyer, vendor, rolling_window_days)`, emit a draft lead when:

```text
count(facts) >= configured_min_count
sum(amount) >= configured_min_total
all individual amounts < configured_review_threshold, if threshold available
```

Required fields:

- vendor
- buyer/department
- date
- amount
- PO/invoice ID if available
- threshold if using threshold condition

Safe wording:

```text
Several same-vendor purchases occurred close together and total more than the review threshold used for this rule.
```

Limitations:

Can be legitimate staged delivery, separate projects, monthly services, or data-entry batch timing.

Status:

`ready-now-info` for sources with dates and vendor; `ready-with-threshold` for stronger threshold lead.

### L004 — single-bid or low-competition award rate

Citation:

- OECD bid-rigging guidance says bid rigging can involve bid suppression and that red flags include unusual bidding patterns over time. URL: https://www.oecd.org/en/publications/oecd-guidelines-for-fighting-bid-rigging-in-public-procurement-2025-update_cbe05a56-en.html
- DOJ antitrust primer lists qualified bidders suddenly failing to bid and declining participation as collusion indicators. URL: https://www.justice.gov/atr/preventing-and-detecting-bid-rigging-price-fixing-and-market-allocation-post-disaster-rebuilding
- Fazekas et al. cartel-screen research uses single bidding and number of bids as elementary indicators. URL: https://www.govtransparency.eu/wp-content/uploads/2026/01/Public_procurement_cartels_A-large-sample-testing-of-screens-using-machine-learning.pdf

Implementable rule:

For each `(buyer, category, fiscal_year)` compute:

```text
single_bid_rate = count(awards where bid_count = 1) / count(awards with bid_count present)
```

Emit draft lead when:

```text
awards_with_bid_count >= configured_min_awards
single_bid_rate >= configured_rate_threshold
```

Suggested default:

```text
configured_min_awards = 20
configured_rate_threshold = 0.50
```

Required fields:

- bid_count or bidder_count
- award ID
- buyer/department
- category/market
- fiscal year
- amount for prioritization

Safe wording:

```text
This buyer/category has a high share of awards with only one recorded bid in the available records.
```

Limitations:

Not implementable on CA Purchase Order Data unless bid counts are added from another source. Single-bid awards can be normal in specialized or emergency markets.

Status:

`future-source`.

### L005 — repeated winner / bid-rotation screen

Citation:

- DOJ Red Flags of Collusion says patterns include rotation and vendors taking turns winning awards. URL: https://www.justice.gov/atr/red-flags-collusion
- OECD guidance describes bid rotation as companies taking turns being the winning bidder. URL: https://www.oecd.org/en/publications/oecd-guidelines-for-fighting-bid-rigging-in-public-procurement-2025-update_cbe05a56-en.html

Implementable rule:

For each `(buyer, category, market_area)` over sequential procurements, compute winner sequence metrics:

```text
winner_repeat_rate
winner_transition_matrix
number_of_unique_winners
share_of_awards_by_top_n_winners
```

Emit draft lead when:

```text
award_count >= configured_min_sequence
unique_winners between 2 and configured_max_winners
winner_sequence shows statistically unusual alternation or repeated allocation compared with historical baseline
```

Required fields:

- award date/order
- winner/vendor
- buyer
- category/market
- ideally losing bidders and bid counts

Safe wording:

```text
The sequence of winners in this market is unusually concentrated or patterned compared with the configured baseline.
```

Limitations:

Needs enough sequential competitive-award history. Without losing bidders, this is a weak screen and should be low severity.

Status:

`future-source`, except concentration-only submetrics can run now.

### L006 — winning price just below next-lowest bid or estimate

Citation:

- DOJ antitrust primer lists identical bids, bids significantly higher than estimates, and consistent gaps between winner and others as red flags. URL: https://www.justice.gov/atr/preventing-and-detecting-bid-rigging-price-fixing-and-market-allocation-post-disaster-rebuilding
- IACRC red flags include winning bid just under next-lowest bid and collusive-bidding price patterns. URL: https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags

Implementable rule:

For each competitive tender with bid prices:

```text
winner_margin = (second_lowest_bid - winning_bid) / second_lowest_bid
estimate_ratio = winning_bid / government_estimate
```

Emit draft lead when any configured pattern holds:

```text
winner_margin is repeatedly inside a narrow low-margin band across same buyer/category/vendor set
estimate_ratio materially exceeds historical baseline
identical bid prices occur among multiple bidders
```

Required fields:

- tender ID
- all bid prices
- winning bidder
- government estimate if available
- category/market
- date

Safe wording:

```text
Bid-price patterns in these tenders are unusual relative to the configured screen.
```

Limitations:

Not implementable on award-only data. Requires full tender/bid-tabulation data.

Status:

`future-source`.

### L007 — price-per-unit outlier inside comparable item group

Citation:

- GAO-04-87G treats abusive purchases as authorized goods/services purchased at excessive costs or questionable needs and recommends data mining for unusual patterns. URL: https://www.gao.gov/assets/gao-04-87g.pdf
- DOJ antitrust primer includes bids significantly higher than estimates and sudden price increases without cost justification as indicators. URL: https://www.justice.gov/atr/preventing-and-detecting-bid-rigging-price-fixing-and-market-allocation-post-disaster-rebuilding

Implementable rule:

For each `(item_code or normalized_description, unit, fiscal_year)` compute robust unit price statistics:

```text
unit_price = amount / quantity
median_unit_price
mad = median(abs(unit_price - median_unit_price))
robust_z = 0.6745 * (unit_price - median_unit_price) / mad
```

Emit draft lead when:

```text
abs(robust_z) >= configured_z_threshold
comparison_group_count >= configured_min_group_count
amount >= configured_min_amount
```

Suggested default:

```text
configured_z_threshold = 5
configured_min_group_count = 30
```

Required fields:

- amount
- quantity
- unit of measure
- item code or normalized description
- vendor
- buyer
- date/fiscal year

Safe wording:

```text
This purchase has a unit price far from comparable public-record purchases in the same item group.
```

Limitations:

Requires real units and comparable item normalization. Do not run on vague descriptions or mixed bundles.

Status:

`future-source` for most current Outlays data.

### L008 — duplicate payment / duplicate invoice candidate

Citation:

- IACRC lists false, inflated, or duplicate invoices with red flags including multiple invoices for the same amount, invoice number, or purchase order number. URL: https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags
- GAO-04-87G recommends data mining and forensic follow-up for high-risk transaction patterns. URL: https://www.gao.gov/assets/gao-04-87g.pdf

Implementable rule:

Emit draft lead for duplicate candidate when two or more facts share a configured duplicate key:

```text
same vendor
same buyer
same amount
same invoice_id or same purchase_order_id if available
close date window if exact ID absent
```

Required fields:

- vendor
- buyer
- amount
- invoice ID or purchase-order ID preferred
- date
- raw source reference

Safe wording:

```text
These records share duplicate-like payment attributes and may require source review.
```

Limitations:

Without invoice IDs, false positives are common for recurring payments, installments, or batch entries.

Status:

`future-source`, unless source has reliable invoice/PO identifiers.

### L009 — change-order growth after award

Citation:

- IACRC lists change-order abuse, including low initial bids followed by inflated profits through subsequent unjustified change orders, as a red flag. URL: https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags

Implementable rule:

For each contract with original award amount and modifications:

```text
change_order_ratio = sum(change_order_amounts) / original_award_amount
change_order_count = count(change_orders)
```

Emit draft lead when:

```text
change_order_ratio >= configured_ratio_threshold
change_order_count >= configured_min_count
elapsed_days_from_award_to_first_change <= configured_early_days, optional
```

Suggested default:

```text
configured_ratio_threshold = 0.25
configured_min_count = 2
```

Required fields:

- contract ID
- original award amount
- modification/change-order amounts
- dates
- vendor
- buyer
- description/category

Safe wording:

```text
This contract's recorded change orders materially increased the award amount relative to the original award.
```

Limitations:

Change orders can be legitimate scope changes, emergencies, litigation settlements, inflation, or data corrections.

Status:

`future-source`.

### L010 — no-competition / sole-source share

Citation:

- IACRC lists unjustified sole-source awards and previously competitive procurements becoming non-competitive as red flags. URL: https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags
- DOJ antitrust primer emphasizes expanding the bidder pool and notes collusion is more likely with few competitors. URL: https://www.justice.gov/atr/preventing-and-detecting-bid-rigging-price-fixing-and-market-allocation-post-disaster-rebuilding

Implementable rule:

For each `(buyer, category, fiscal_year)` compute:

```text
sole_source_share = sum(amount where competition_type in configured_noncompetitive_values) / sum(amount)
sole_source_count_share = count(noncompetitive awards) / count(awards)
```

Emit draft lead when:

```text
awards_count >= configured_min_awards
sole_source_share >= configured_share_threshold
```

Required fields:

- competition type or procurement method
- amount
- vendor
- buyer
- category
- date/fiscal year

Safe wording:

```text
A high share of this buyer/category's recorded spending used noncompetitive procurement methods.
```

Limitations:

Requires source-coded competition method. Sole-source can be justified by law, emergency, compatibility, monopoly supply, or set-aside policy.

Status:

`future-source`.

### L011 — declining bidder participation over time

Citation:

- DOJ Red Flags of Collusion notes declining participation and fewer vendors submitting proposals compared with the past. URL: https://www.justice.gov/atr/red-flags-collusion
- OECD detection guidance says patterns over time are better indicators than one-off suspicions and red flags include unusual bidding patterns. URL: https://www.oecd.org/en/publications/oecd-guidelines-for-fighting-bid-rigging-in-public-procurement-2025-update_cbe05a56-en.html

Implementable rule:

For each `(buyer, category)` time series:

```text
median_bid_count_prior_period
median_bid_count_current_period
participation_drop = current / prior
```

Emit draft lead when:

```text
prior_tender_count >= configured_min_prior
current_tender_count >= configured_min_current
participation_drop <= configured_drop_ratio
```

Suggested default:

```text
configured_drop_ratio = 0.60
```

Required fields:

- tender IDs
- bid counts
- dates
- buyer
- category

Safe wording:

```text
Bidder participation in this buyer/category appears lower than its prior public-record baseline.
```

Limitations:

Market consolidation, emergencies, changed specifications, framework contracts, and reporting changes can produce the same pattern.

Status:

`future-source`.

### L012 — Benford/digit-pattern screen for bid prices

Citation:

- Fazekas et al. public procurement cartel-screen research uses Benford’s-law compliance by market/year as a collusion-risk indicator. URL: https://www.govtransparency.eu/wp-content/uploads/2026/01/Public_procurement_cartels_A-large-sample-testing-of-screens-using-machine-learning.pdf

Implementable rule:

For each `(market/category, fiscal_year)` with enough bid prices, test first-digit or second-digit distribution against Benford expected distribution:

```text
chi_square_statistic
p_value
sample_size
```

Emit draft lead when:

```text
sample_size >= configured_min_sample
p_value <= configured_p_threshold
```

Suggested default:

```text
configured_min_sample = 100
configured_p_threshold = 0.01
```

Required fields:

- individual bid prices, not just awards
- market/category
- fiscal year
- buyer

Safe wording:

```text
Bid-price digit patterns in this market differ from the configured benchmark and may deserve statistical review.
```

Limitations:

Digit tests are weak alone and sensitive to price controls, round pricing, catalog prices, data truncation, and small samples. Never publish as standalone high-severity lead.

Status:

`future-source`.

### L013 — ownership/co-bidding network risk screen

Citation:

- Collusion risk in corporate networks, PMC/NIH: ownership density and network centrality were associated with single bidding, missing bidders, and winning probability in Swedish procurement markets. URL: https://pmc.ncbi.nlm.nih.gov/articles/PMC10850496

Implementable rule:

For each market/year with ownership and bid participation data:

```text
ownership_link_density among bidders
co_bidding_graph_centrality
winner_centrality_rank
single_bid_rate by ownership-density bucket
```

Emit draft lead when:

```text
market has high ownership density or high winner centrality
and competition outcomes show high single-bid/missing-bid/concentration indicators
```

Required fields:

- bidder identities
- owner/beneficial-owner graph
- tender participation
- winner
- bid counts
- market/category

Safe wording:

```text
This procurement market combines concentrated ownership/co-bidding network features with low-competition outcome indicators.
```

Limitations:

Requires ownership data. Ownership links are not wrongdoing; they are a structural risk factor.

Status:

`future-source`.

## First S11 rule recommendation

Use `L001 — vendor concentration inside buyer/year` as the first end-to-end private rule.

Why:

- It matches S11's suggested example.
- It is implementable from current CA procurement facts.
- It requires no claim about intent.
- It can cite DOJ/IACRC pattern guidance while remaining a neutral concentration screen.
- It can link directly to source facts and provenance.

Recommended S11 rule ID:

```text
ca_vendor_concentration_department_category_v1
```

Draft SQL sketch:

```sql
WITH group_totals AS (
  SELECT
    f.jurisdiction,
    f.fiscal_year,
    dept.code AS department,
    acq.code AS acquisition_type,
    f.payee_entity,
    SUM(f.amount) AS vendor_amount,
    COUNT(*) AS vendor_fact_count,
    SUM(SUM(f.amount)) OVER (
      PARTITION BY f.jurisdiction, f.fiscal_year, dept.code, acq.code
    ) AS group_amount,
    COUNT(DISTINCT f.payee_entity) OVER (
      PARTITION BY f.jurisdiction, f.fiscal_year, dept.code, acq.code
    ) AS group_vendor_count
  FROM fiscal_fact f
  JOIN classification_assignment dept
    ON dept.fact_id = f.fact_id
   AND dept.scheme_id = 'us_ca_department'
  JOIN classification_assignment acq
    ON acq.fact_id = f.fact_id
   AND acq.scheme_id = 'us_ca_acquisition_type'
  WHERE f.jurisdiction = 'us-ca'
    AND f.flow = 'spending'
    AND f.grain IN ('award','transaction')
  GROUP BY f.jurisdiction, f.fiscal_year, dept.code, acq.code, f.payee_entity
)
SELECT *
FROM group_totals
WHERE group_amount >= 100000
  AND vendor_fact_count >= 3
  AND group_vendor_count >= 2
  AND vendor_amount / NULLIF(group_amount, 0) >= 0.50;
```

Implementation note:

Window `COUNT(DISTINCT ...)` support varies by database. If Postgres rejects it in a window expression, compute group vendor count in a separate CTE. Do not let SQL cleverness block the builder.

## Severity policy

Initial severity mapping:

| Condition | Severity |
|---|---:|
| Single weak screen, no threshold/control total | info |
| One published red-flag screen with enough sample size and provenance | low |
| Multiple independent screens on same buyer/vendor/category | medium |
| Official audit/enforcement source plus Outlays matching records | high |

No automated lead should be `high` from Outlays data alone.

## Publication policy

All generated leads start private/draft.

Public release requires:

- reviewer handle,
- source fact links,
- method citation,
- limitations text,
- neutral wording,
- no accusation language,
- correction pathway.

## Sources

- GAO, `Audit Guide: Auditing and Investigating Internal Control of Government Purchase Card Programs`, GAO-04-87G, November 2003. https://www.gao.gov/assets/gao-04-87g.pdf
- U.S. DOJ Antitrust Division, `Red Flags of Collusion`. https://www.justice.gov/atr/red-flags-collusion
- U.S. DOJ Antitrust Division, `Preventing and Detecting Bid Rigging, Price Fixing, and Market Allocation in Post-Disaster Rebuilding Projects`. https://www.justice.gov/atr/preventing-and-detecting-bid-rigging-price-fixing-and-market-allocation-post-disaster-rebuilding
- U.S. DOJ Antitrust Division, `Procurement Collusion Strike Force`. https://www.justice.gov/atr/procurement-collusion-strike-force
- OECD, `OECD Guidelines for Fighting Bid Rigging in Public Procurement (2025 Update)`. https://www.oecd.org/en/publications/oecd-guidelines-for-fighting-bid-rigging-in-public-procurement-2025-update_cbe05a56-en.html
- Palm Beach County Office of Inspector General, `Tips and Trends #2025-0002: Splitting Purchases`, April 2025. https://pbc.gov/oig/docs/advisories/Tips_and_Trends-2025-0002-Splitting_Purchases.pdf
- International Anti-Corruption Resource Center, `The Most Common Procurement Fraud Schemes and their Primary Red Flags`. https://iacrc.org/fraud-and-corruption/the-most-common-procurement-fraud-schemes-and-their-primary-red-flags
- Fazekas, M., Tóth, B., Wachs, J., and Abdou, A., `Public procurement cartels: A large-sample testing of screens using machine learning`, International Journal of Industrial Organization, 104, 103228. https://www.govtransparency.eu/wp-content/uploads/2026/01/Public_procurement_cartels_A-large-sample-testing-of-screens-using-machine-learning.pdf
- `Collusion risk in corporate networks`, PMC/NIH. https://pmc.ncbi.nlm.nih.gov/articles/PMC10850496

## A2 closeout

A2 status: closed.

Builder unblock:

Implement `ca_vendor_concentration_department_category_v1` first. It is the safest S11 rule because it is implementable on current CA procurement data and does not require bidder counts, unit prices, estimates, invoice IDs, or intent claims.
