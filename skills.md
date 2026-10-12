# Skills: MotoGuard Fleet Management Portal

**Source:** SRS v1.0.0 + Addendum v1.1 + `agent.md`
**Purpose:** This document inventories the discrete technical skills required to build and operate every requirement in the SRS and its addendum. Each skill entry states what it's for, which FR-codes or agents depend on it, and the concrete competency involved — so that engineering staffing, AI-agent tool selection, or contractor scoping can be matched against real requirements rather than guessed at generically.

---

## Skill Index by Category

| Category | Skills |
|---|---|
| Backend & API | REST API Design, Authentication & Session Security, Background Job/Queue Processing |
| Spatial & Geospatial | PostGIS Spatial Queries, Reverse Geocoding Integration, Geofence Geometry (Circle/Polygon) |
| Data Modeling | Relational Schema Design, Multi-Tenant Data Isolation, Time-Series/High-Volume Data Retention |
| Frontend | Tailwind CSS Flat Design Systems, Interactive Map Rendering (Leaflet/MapLibre), State-Driven Dashboard UI |
| Embedded/Hardware Integration | Telemetry Payload Design, Device Health Signal Interpretation |
| Notifications | Transactional Email Dispatch, SMS Gateway Integration (deferred) |
| Operations | Rate Limiting & Abuse Prevention, Observability for Async Agents |

---

## 1. REST API Design

**What it's for:** Every endpoint across both SRS documents follows a consistent resource-oriented convention (plural nouns, no verbs in URIs, standard status codes, RFC 7807 error envelope). This isn't incidental — the base SRS explicitly calls out "no RPC-style verbs" (e.g., `/sessions` not `/login`), and every addendum endpoint follows the same pattern.

**Depends on this skill:**
- All `FR-AUTH`, `FR-PWD`, `FR-DASH`, `FR-GEO` endpoints (v1.0.0 §5)
- All `FR-TRIP`, `FR-ORG`, `FR-HEALTH`, `FR-ALERT` endpoints (Addendum §1–4)

**Concrete competency:**
- Designing nested resource paths correctly (`/v1/bikes/{bike_id}/trips/{trip_id}/breadcrumbs` — four levels deep, still noun-only)
- Choosing the right HTTP method semantics (e.g., why alert acknowledgment is `PATCH /v1/alerts/{alert_id}` and not a `POST /v1/alerts/{alert_id}/acknowledge` action route)
- Consistent pagination/filtering query parameter conventions (`?sort=-created_at`, `?from=...&to=...`, `?acknowledged=false&severity=high`)
- RFC 7807 Problem Details error format applied uniformly

---

## 2. Authentication & Session Security

**What it's for:** Stateless JWT auth, refresh token rotation, password reset without account enumeration, and session revocation on password change.

**Depends on this skill:**
- FR-AUTH-01 through FR-AUTH-05
- FR-PWD-01 through FR-PWD-04
- §6.2 rate limiting on auth endpoints

**Concrete competency:**
- Short-lived JWT (15-min) + rotating refresh token (7-day) design, including where the refresh token is stored (HttpOnly, Secure, SameSite=Strict cookie per v1.0.0's later revision)
- Argon2id password hashing configuration (memory/time cost tuning)
- Cryptographically secure token generation (32-byte random) and hash-before-storage pattern for reset tokens, so a database read alone can't be used to complete a reset
- Building the **enumeration-safe** reset flow: identical `202 Accepted` response regardless of whether the email exists (FR-PWD-03) — a common point where engineers accidentally leak account existence through timing or response differences
- Session invalidation cascade: a password change must revoke *all* existing sessions, not just issue a new token (FR-PWD-04)

---

## 3. Background Job / Queue Processing

**What it's for:** Every agent in `agent.md` except direct request/response API handlers runs as an asynchronous worker. This is the connective tissue skill underlying the entire Phase 5–7 build.

**Depends on this skill:**
- Agent 1 (Ingestion) → Agent 2/3/4/6 event handoffs
- Agent 5 (Device Health Monitor) — fixed 60-second timer independent of request traffic
- Agent 8 (Retention & Archival) — daily timer
- FR-TRIP-01's 60-second segmentation SLA
- FR-GEO-02's 50ms spatial-evaluation SLA (tight enough that this must be an optimized in-process or in-memory-cached path, not a naive queue round-trip)

**Concrete competency:**
- Event-driven architecture: ingestion writes truth first, then emits an event, rather than each downstream concern being inline in the request handler (`agent.md` §"Cross-Agent Ordering Guarantees")
- Designing for graceful degradation — a failing downstream consumer (e.g., SMTP outage) must never block or fail the ingestion write path
- Timer-driven workers distinct from event-driven ones, and knowing when each is appropriate (Agent 5 must be timer-driven specifically because it detects *absence* of events, which an event-driven design structurally cannot do)
- Debounce/cooldown state management across workers (trip-end 90-second debounce, alert cooldowns, geofence breach cooldowns) without race conditions when multiple packets arrive in quick succession

---

## 4. PostGIS Spatial Queries

**What it's for:** Every geofence evaluation, distance calculation, and location-indexed lookup in the system.

**Depends on this skill:**
- FR-GEO-02 (breach detection via `ST_Contains`/`ST_DWithin`)
- Trip distance computation (`distance_meters` in `trips` table, Addendum §1.3)
- GIST indexing on `bikes.last_location`, `geofences.boundary_polygon`, `geofences.center_point`

**Concrete competency:**
- Writing efficient `ST_Contains` (polygon) and `ST_DWithin` (circle/radius) queries against `GEOGRAPHY(Point, 4326)` and `GEOGRAPHY(Polygon, 4326)` columns
- Understanding why `GEOGRAPHY` (not `GEOMETRY`) is the correct type here — it handles great-circle distance correctly on a WGS84 spheroid, which matters for radius-based geofences at the scale of kilometers
- GIST index design and query-plan verification to hit the sub-50ms latency budget (FR-GEO-02 / §6.1) at fleet scale
- Computing cumulative trip distance from a breadcrumb sequence (sum of pairwise great-circle distances) without n+1 query patterns

---

## 5. Reverse Geocoding Integration

**What it's for:** Converting raw lat/long into `current_address`, `sub_county`, `county` — a field set specific to Kenyan administrative hierarchy, not a generic "city/state" model.

**Depends on this skill:**
- FR-DASH-04
- Agent 2 (Reverse Geocoding Agent)
- Trip start/end address resolution (Agent 4, reusing Agent 2's output)

**Concrete competency:**
- Integrating a geocoding provider that resolves Kenya's two-tier administrative structure (County → Sub-County) correctly — this is not the same hierarchy as most default geocoding APIs (which assume state/city), so provider selection and response mapping needs specific validation against Kenyan boundary data
- Designing the stale-data fallback: retaining the last known address on geocoder failure rather than nulling the field (Agent 2 spec)
- Meeting the 120ms p95 dashboard-load budget by deciding what can be cached vs. what must be looked up live (§6.1)

---

## 6. Geofence Geometry (Circle & Polygon)

**What it's for:** The UI and data model must support two distinct geometry input methods.

**Depends on this skill:**
- FR-GEO-01 ("click a center point with a radius" or "draw a polygon")
- `geofences.geometry_type` schema field

**Concrete competency:**
- Client-side drawing tools for both a radius-circle (center-click + drag) and a polygon (vertex-by-vertex click, closing the ring) on a Leaflet/MapLibre map
- Serializing polygon input to valid GeoJSON coordinate rings (first and last point must match, per the RFC 7946 standard referenced in the base SRS's definitions section)
- Validating user-drawn polygons for self-intersection before persisting (a malformed polygon will silently break `ST_Contains` queries)

---

## 7. Relational Schema Design (Multi-Table, Foreign-Key Heavy)

**What it's for:** The combined schema across both SRS documents spans 15 tables with cascading deletes, unique constraints, and check constraints — this is a non-trivial normalization exercise, not a handful of flat tables.

**Depends on this skill:**
- Every `CREATE TABLE` statement across v1.0.0 §5.1 and Addendum §1.3, §2.3, §3.4, §4.3

**Concrete competency:**
- Correct `ON DELETE CASCADE` vs. `ON DELETE SET NULL` choices (e.g., deleting a `geofence` cascades to `geofence_breaches`, but deleting a `trip` should `SET NULL` on `breadcrumbs.trip_id` rather than deleting the raw location history)
- `CHECK` constraint design for enum-like fields (`status`, `role`, `alert_type`, `trigger_on`) as a data-integrity backstop below the application layer
- Composite unique constraints where identity is compound (`UNIQUE (organization_id, user_id)` on `memberships`, `UNIQUE (bike_id, alert_type)` on `alert_rules`)

---

## 8. Multi-Tenant Data Isolation

**What it's for:** The single hardest correctness requirement in the whole system — Addendum §2 exists specifically because the base SRS's flat `user_id` model cannot express organizations, roles, or scoped viewer access.

**Depends on this skill:**
- FR-ORG-01 through FR-ORG-04
- The access-resolution rule in Addendum §2.3, which **must** be enforced on every single bike-scoped query across the entire API surface (not just the new Phase 2 endpoints — Phase 3's fleet endpoints, Phase 6's trip endpoints, and Phase 7's alert endpoints all inherit this requirement)

**Concrete competency:**
- Implementing row-level authorization as a shared, tested library or middleware layer rather than re-implementing the owner/admin/dispatcher/viewer check in every handler — a missed check in one endpoint is a full tenant-isolation breach
- Migration strategy for a live system: adding `organization_id` as nullable, backfilling one organization per existing user, then only *later* making it `NOT NULL` and dropping `user_id` — doing this in the wrong order causes downtime or data loss (Addendum §2.3 migration note)
- Designing the bike-scoped `viewer` grant so a scoped user's fleet dropdown, map, and every API response are filtered identically — inconsistency between "what the UI shows" and "what the API allows" is a common multi-tenancy bug class

---

## 9. Time-Series / High-Volume Data Retention

**What it's for:** `breadcrumbs` grows roughly 288 rows/bike/day even on a low-duty-cycle device — this is a genuinely high-volume table that needs a retention and partitioning strategy from day one, not as an afterthought.

**Depends on this skill:**
- FR-TRIP-03 (180-day minimum retention, configurable per org)
- Agent 8 (Retention & Archival Agent)
- Addendum §1.3's partitioning note at ~500 vehicles

**Concrete competency:**
- Designing `PARTITION BY RANGE (recorded_at)` monthly partitioning for `breadcrumbs` before it's needed, so the migration isn't done under production load pressure later
- Building the archival job to move (not delete) aged rows to cold storage, respecting the addendum's explicit framing of retention as a floor before archival eligibility, not a deletion policy
- Per-organization configurable retention (`organizations.breadcrumb_retention_days`) rather than a single global constant

---

## 10. Tailwind CSS Flat Design Systems

**What it's for:** The frontend has an unusually specific, explicitly-constrained visual language: zero gradients, flat solid fills, 1px structural borders, a fixed color-token table — this is a deliberate anti-pattern choice (avoiding the "looks AI-generated" gradient/glassmorphism aesthetic), and needs to be followed precisely rather than approximated.

**Depends on this skill:**
- v1.0.0 §3.2 (full color palette table), §3.3 (layout architecture)
- The reference HTML/Tailwind implementation in v1.0.0 §7

**Concrete competency:**
- Disciplined use of Tailwind's utility classes against a fixed token table rather than ad hoc hex values — every color in the UI should trace back to the §3.2 table
- Building a sidenav + top-bar + content-ribbon + map layout using flexbox/grid without relying on shadow-heavy or gradient-heavy conventions common in default component libraries
- Monospace-vs-sans-serif discipline: coordinates, IMEI numbers, and timestamps in `font-mono` (JetBrains Mono); everything else in `font-sans` (Inter) — a small detail that's called out explicitly and easy to drift from over time

---

## 11. Interactive Map Rendering (Leaflet / MapLibre)

**What it's for:** Live marker rendering, geofence overlay drawing, and — new in the addendum — route polyline playback with a scrub slider.

**Depends on this skill:**
- FR-DASH's interactive map viewport (v1.0.0)
- FR-GEO-01's geofence drawing toolbar
- FR-TRIP-02's trip playback polyline with scrub/playback slider (new capability, not present in v1.0.0)

**Concrete competency:**
- Rendering a directional marker with heading rotation (`heading_deg`) rather than a static pin
- Drawing and editing both circle and polygon overlays interactively (see Skill 6)
- Building a **timeline scrubber** control that maps a slider position to a breadcrumb index and animates the marker along the polyline — this is new functionality the addendum requires that has no precedent in the v1.0.0 reference implementation, so it can't simply be copied from the existing dashboard code

---

## 12. State-Driven Dashboard UI

**What it's for:** The Telemetry Ribbon, fleet selector, and health/alert badges all need to reflect live backend state consistently across multiple concurrently-open views.

**Depends on this skill:**
- FR-DASH-02/03 (fleet switcher updates ribbon + map together)
- FR-HEALTH-01 (6th ribbon column added by the addendum without breaking the existing 5)
- FR-ALERT-03 (Geofence nav item renamed to Alerts, restructured as a tabbed unified view)

**Concrete competency:**
- Structuring frontend state so that selecting a different bike atomically updates telemetry, map center, geofence overlays, and health badge together (avoiding a flash of stale data from one section while another has updated)
- Extending an existing 5-column grid layout to 6 columns without a full redesign — a practical CSS/layout skill specifically because the addendum modifies an already-shipped component rather than building fresh
- Building the tabbed Alerts view (unified feed + nested Geofence config tab) as a rename-and-restructure of the existing Geofence nav item, preserving all existing geofence CRUD UI inside the new structure

---

## 13. Telemetry Payload Design (Embedded/Firmware Coordination)

**What it's for:** The addendum adds four new fields to the telemetry payload (`battery_voltage_mv`, `battery_percent`, `signal_quality_csq`, `gnss_fix_type`) that must be sourced from the ESP32/SIM868 firmware — this is a hardware-software boundary, not a pure backend task.

**Depends on this skill:**
- Addendum §3.2
- Agent 1 (Ingestion Agent) input contract

**Concrete competency:**
- Reading `AT+CSQ` (signal quality) and battery ADC values from the SIM868/ESP32 firmware side and packaging them into the existing telemetry payload format without breaking backward compatibility with devices running older firmware
- Deriving `battery_percent` from raw `battery_voltage_mv` for a Li-ion cell (a non-linear discharge curve, not a simple linear percentage of voltage range)
- Coordinating a firmware version rollout so the backend can distinguish "device doesn't support health fields" from "device is offline" during the transition period — this is a real operational concern the addendum doesn't address and should be planned for explicitly

---

## 14. Device Health Signal Interpretation

**What it's for:** Turning raw signal/battery/fix-type values into the three-state `healthy`/`degraded`/`offline` classification, including the stealth-mode-aware grace period — this is a domain-specific interpretation skill, not generic monitoring.

**Depends on this skill:**
- FR-HEALTH-02/03
- Agent 5 (Device Health Monitor Agent)

**Concrete competency:**
- Implementing the graduated threshold logic (1.5×/4×/12× the expected interval) correctly, including the parked-and-motionless exception that extends the grace period specifically to avoid false "offline" alerts during intentional stealth sleep cycles — getting this multiplier logic wrong in either direction either spams the user with false offline alerts or delays genuine theft detection
- Correlating MPU-6500 motion-interrupt absence with ignition state to decide which threshold branch applies

---

## 15. Transactional Email Dispatch

**What it's for:** Every alert type's default or fallback channel; the only fully-specified notification channel in either SRS document.

**Depends on this skill:**
- FR-GEO-04 (original geofence breach email)
- FR-ALERT-05 (extended to all six alert types)
- §6.1's 5-second SMTP handoff SLA

**Concrete competency:**
- Templating transactional emails with the specific field sets called out per alert type (registration plate, geofence name, breach type, timestamp in EAT/UTC+3, address/county/sub-county, direct portal link — v1.0.0 §4.4)
- Meeting the 5-second handoff SLA under load — this implies queuing to the SMTP/API provider asynchronously from Agent 7 rather than a synchronous send-and-wait call in the request path

---

## 16. SMS Gateway Integration (Deferred Scope)

**What it's for:** High-severity alerts default to email + SMS (FR-ALERT-05), but the addendum explicitly defers the provider integration itself.

**Depends on this skill:**
- FR-ALERT-05's channel policy (the *decision* to send SMS is in scope; the *mechanism* is not)
- Explicitly flagged in Addendum §4.5 as needing a separate integration SRS

**Concrete competency (for when this is unblocked):**
- Africa's Talking or Twilio API integration for the Kenyan market specifically (Africa's Talking has better local delivery rates/cost for Kenyan numbers)
- GSM 160-character message segmentation for alert text
- Delivery receipt handling and per-message cost tracking — both explicitly named as unspecified in the addendum and needing their own design

**Status:** Not to be built against this document alone — treat as blocked pending a dedicated SMS integration SRS.

---

## 17. Rate Limiting & Abuse Prevention

**What it's for:** Auth and password-reset endpoints need brute-force protection; this is called out specifically enough (5 req/min/IP, token-bucket, Redis-backed) that it's a distinct implementation skill rather than a vague "add security" note.

**Depends on this skill:**
- §6.2 non-functional requirement

**Concrete competency:**
- Token-bucket algorithm implementation backed by Redis, keyed by IP address
- Applying the same primitive consistently across `/v1/sessions` and `/v1/password-resets` rather than building two different rate-limiting mechanisms

---

## 18. Observability for Async Agents

**What it's for:** With eight independent background agents (per `agent.md`), operational visibility into each agent's health, latency, and failure rate is not explicitly in the SRS but is required to actually operate the system described — flagged here as a necessary skill even though neither SRS document names it.

**Depends on this skill:**
- Verifying the 50ms geofence-evaluation SLA and 60-second trip-segmentation SLA are actually being met in production (§6.1, FR-TRIP-01)
- Detecting when Agent 5 (Health Monitor) itself stops running — a monitoring agent that silently fails is worse than no monitoring at all, since it produces false confidence
- Alerting engineering (distinct from alerting the *end user* via Agent 7) when any agent's queue backs up or its cooldown/dedup logic misfires

**Note:** This is the one skill on this list not directly traceable to a numbered FR-code in either SRS — it is inferred from the operational reality of running eight interdependent async agents in production, and should be treated as an infrastructure prerequisite rather than optional polish.
