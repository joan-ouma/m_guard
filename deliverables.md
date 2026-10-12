# Deliverables Checklist: MotorGuarrd Fleet Management Portal

**Sources covered:** SRS v1.0.0 (base) + SRS Addendum v1.1 (trip history, multi-tenancy, device health, alert taxonomy)
**Purpose:** A build-tracking checklist that maps every requirement, schema object, and endpoint in both documents to a concrete deliverable. Nothing here introduces new scope — it only inventories what the two SRS documents already committed to, organized by delivery phase.

---

## 0. How to Use This Document

Each item below traces back to a specific FR-code, table, or endpoint in the source SRS documents. Check items off as they are implemented, tested, and reviewed. Items are grouped by phase in dependency order — later phases assume earlier ones are functioning, since (for example) alerts depend on device health, and device health depends on telemetry ingestion existing at all.

---

## Phase 1 — Authentication & Session Foundation
*Source: SRS v1.0.0 §4.1, §4.2*

- [ ] `POST /v1/sessions` — login, returns access + refresh token (FR-AUTH-01, FR-AUTH-03)
- [ ] `DELETE /v1/sessions/current` — logout, revokes session server-side (FR-AUTH-04/05)
- [ ] `PUT /v1/sessions/current` (or documented equivalent) — refresh access token via refresh cookie
- [ ] Generic 401 on failed login, no email/password disclosure (FR-AUTH-02)
- [ ] Rate limiting: 5 attempts/minute/IP on `/v1/sessions` and `/v1/password-resets` (§6.2)
- [ ] Argon2id password hashing (FR-AUTH-02, FR-PWD-04)
- [ ] `POST /v1/password-resets` — request reset, always returns 202 regardless of email existence (FR-PWD-01/03)
- [ ] Cryptographic reset token: 32-byte random, SHA-256 hash stored, 30-minute expiry (FR-PWD-02)
- [ ] `PATCH /v1/password-resets/{token}` — consume token, update password, revoke all sessions, send confirmation email (FR-PWD-04)
- [ ] `users` table + `password_reset_tokens` table (§5.1)
- [ ] TLS 1.3 enforced on all traffic (§6.2)

**Definition of done:** A user can register (via provisioning, see Phase 6), log in, log out, and recover a forgotten password end-to-end, with no credential enumeration possible.

---

## Phase 2 — Multi-Tenancy & Roles
*Source: Addendum v1.1 §2 (FR-ORG)*

> **Sequencing note:** This phase is placed early because every later phase (bikes, geofences, alerts) is scoped by organization/role in the addendum's access model. Building bikes against the old flat `user_id` first and retrofitting tenancy later means redoing every query's authorization check.

- [ ] `organizations` table (id, name, plan_tier, breadcrumb_retention_days)
- [ ] `memberships` table (organization_id, user_id, role, invited_by, accepted_at)
- [ ] `bike_access_grants` table (membership_id, bike_id)
- [ ] Migration: `bikes.organization_id` added, `bikes.user_id` made nullable, backfill script creates one default `individual` organization per existing user (§2.3 migration note)
- [ ] Role model implemented: `owner`, `admin`, `dispatcher`, `viewer` with permissions exactly as specified in §2.2 table
- [ ] Access resolution rule enforced on **every** bike-scoped query: owner/admin/dispatcher see all org bikes; viewer sees only bikes with a matching `bike_access_grants` row (§2.3 access resolution rule) — this **replaces** the flat tenant-isolation rule in base SRS §6.2
- [ ] `GET /v1/organizations/current`
- [ ] `GET /v1/organizations/current/members`
- [ ] `POST /v1/organizations/current/members` — invite by email + role (+ optional `bike_ids` for scoped viewers)
- [ ] `PATCH /v1/organizations/current/members/{membership_id}` — change role
- [ ] `DELETE /v1/organizations/current/members/{membership_id}` — remove member
- [ ] `POST /v1/bikes/{bike_id}/access-grants` — grant viewer access to specific bike
- [ ] `DELETE /v1/bikes/{bike_id}/access-grants/{grant_id}` — revoke grant
- [ ] `membership_audit_log` — immutable log of role/grant changes (actor, target, prior role, new role, timestamp) (FR-ORG-03)
- [ ] Settings view: "Team" tab — invite members, edit roles, manage per-bike grants; visible only to `owner`/`admin` (FR-ORG-04)

**Definition of done:** An organization owner can invite a dispatcher (full fleet visibility) and a viewer scoped to one bike; the viewer's dropdown and API calls are correctly restricted; all role changes are auditable.

---

## Phase 3 — Fleet & Telemetry Core
*Source: SRS v1.0.0 §4.3, §5.1 (bikes table)*

- [ ] `bikes` table with all base fields (registration_number, label, make, model, imei, status, ignition, speed_kmh, heading_deg, last_location, current_address, sub_county, county, last_telemetry_at) (§5.1)
- [ ] `GET /v1/bikes` — sorted `created_at DESC`, `meta.default_selected_bike_id` populated (FR-DASH-01)
- [ ] `POST /v1/bikes` — register new motorcycle (FR-NAV-02)
- [ ] `GET /v1/bikes/{bike_id}` — single bike + latest telemetry + `active_geofence_ids`
- [ ] `PATCH /v1/bikes/{bike_id}` — edit metadata (FR-NAV-02)
- [ ] `DELETE /v1/bikes/{bike_id}` — remove from fleet (FR-NAV-02)
- [ ] `GET /v1/bikes/{bike_id}/telemetry` — latest & historical telemetry points
- [ ] Status classification logic: `RUNNING` if `ignition_on == true OR speed_kmh > 2.0`; else `STOPPED` (FR-DASH-03) — **note:** base SRS also states a `> 3.0 km/h` variant elsewhere; engineering must confirm and lock a single threshold before implementation
- [ ] Reverse geocoding pipeline populating `current_address`, `sub_county`, `county` on every telemetry update (FR-DASH-04)
- [ ] Fleet Switcher dropdown: selecting a bike fetches latest telemetry + geofences, pans map (FR-DASH-02)
- [ ] `idx_bikes_user_created_desc` and `idx_bikes_last_location` (GIST) indexes
- [ ] Reverse-geocoded lookups meet p95 < 120ms for `GET /v1/bikes` (§6.1)
- [ ] Bike provisioning/pairing flow — **flagged as missing from source SRS**, needed to actually bind a physical IMEI to an organization (see §7 "Open Items Not Yet Specified" below)

**Definition of done:** A logged-in user sees their fleet list defaulting to the most recently added bike, with live status/address/county/sub-county/coordinates rendering correctly on selection.

---

## Phase 4 — Interactive Map & Dashboard UI
*Source: SRS v1.0.0 §3.2, §3.3, §7 (reference frontend)*

- [ ] Left sidenav: `w-64 bg-slate-900`, four items (Home, Bikes, Geofence→**Alerts**, Settings) — see Phase 7 rename
- [ ] Bottom profile dock: avatar, name, email, logout icon button
- [ ] Top control bar: fleet selector dropdown (left), primary action button (right)
- [ ] Telemetry summary ribbon: 5 columns per base SRS (Status, Address, County, Sub-County, Lat/Long) — **becomes 6 columns in Phase 5** with Device Health
- [ ] Full-bleed interactive map viewport rendering marker + heading vector + geofence overlays
- [ ] Color palette and Tailwind tokens applied exactly per §3.2 table (zero gradients, flat fills, `border-stone-200` hairlines)
- [ ] Typography: Inter (UI) + JetBrains Mono (coordinates, IMEI, timestamps)
- [ ] Responsive/collapsible sidenav for mobile breakpoints (implied by §3.3, not detailed — flag for design review)

**Definition of done:** The dashboard visually matches the reference HTML/Tailwind prototype, including flat color system and monospace data fields.

---

## Phase 5 — Device Health & Offline Detection
*Source: Addendum v1.1 §3 (FR-HEALTH)*

- [ ] Firmware telemetry payload extended with `battery_voltage_mv`, `battery_percent`, `signal_quality_csq`, `gnss_fix_type` (§3.2) — **requires firmware change**, coordinate with hardware/embedded team
- [ ] `bikes` table extended: `battery_percent`, `signal_quality_csq`, `gnss_fix_type`, `health_status`, `last_heartbeat_at`
- [ ] `device_health_events` table
- [ ] Background health-check worker running every 60 seconds, classifying each bike as `healthy` / `degraded` / `offline` per thresholds in §3.3
- [ ] Silence Grace Period logic: 60-minute offline threshold when parked + no motion, vs. 20-minute default (§3.3) — depends on MPU-6500 motion data being available (Phase 6 dependency)
- [ ] `GET /v1/bikes/{bike_id}/health`
- [ ] `GET /v1/bikes/{bike_id}/health/events`
- [ ] Telemetry Ribbon gains 6th column: Device Health (battery icon, signal bars, status badge) (FR-HEALTH-01)
- [ ] Health-status change logged immediately on threshold crossing, independent of packet arrival (FR-HEALTH-02) — requires the timer-based worker, not just packet-triggered checks
- [ ] Bikes view: sort/filter fleet list by `health_status` (FR-HEALTH-03)

**Definition of done:** A bike that stops transmitting is correctly classified as degraded then offline on the expected timeline, and a parked bike in stealth sleep is *not* falsely flagged offline within the grace window.

---

## Phase 6 — Trip History & Playback
*Source: Addendum v1.1 §1 (FR-TRIP)*

- [ ] `trips` table (started_at, ended_at, start/end location + address, distance, max/avg speed, duration, status)
- [ ] `breadcrumbs` table (bike_id, trip_id, location, speed_kmh, heading_deg, ignition, recorded_at)
- [ ] `idx_trips_bike_started`, `idx_breadcrumbs_bike_time`, `idx_breadcrumbs_trip` indexes
- [ ] Async trip-segmentation worker implementing exactly the rules in §1.2:
  - [ ] Trip start: ignition false→true, OR speed > 2.0 km/h after 3+ at-rest packets
  - [ ] Trip end: ignition true→false AND speed ≤ 2.0 km/h for 90-second debounce
  - [ ] Discard trips < 60 seconds or < 50 meters displacement
  - [ ] Idle merge: trips restarting within 5 minutes and 100 meters of prior trip end are merged
  - [ ] Segmentation completes within 60 seconds of trip-end detection (FR-TRIP-01)
- [ ] `GET /v1/bikes/{bike_id}/trips` (date-range filterable)
- [ ] `GET /v1/bikes/{bike_id}/trips/{trip_id}`
- [ ] `GET /v1/bikes/{bike_id}/trips/{trip_id}/breadcrumbs` — powers map polyline replay
- [ ] `GET /v1/bikes/{bike_id}/breadcrumbs` — raw window query independent of trip boundaries
- [ ] Bikes view: "History" tab with date-range picker, trip list, expandable map polyline replay with scrub/playback slider (FR-TRIP-02)
- [ ] Retention: 180 days minimum before cold-storage archival, configurable per organization via `organizations.breadcrumb_retention_days` (FR-TRIP-03)
- [ ] Infra note: monthly partitioning of `breadcrumbs` by `recorded_at` once fleets exceed ~500 vehicles (implementation note, not a hard requirement — track separately in infra backlog)

**Definition of done:** A user can pick a date/time range for any bike and see a replayable route on the map with speed and timestamp data at each point.

---

## Phase 7 — Expanded Alert Taxonomy
*Source: SRS v1.0.0 §4.4 (base geofence) + Addendum v1.1 §4 (FR-ALERT)*

### 7a. Geofence base functionality (carried forward from v1.0.0, unchanged)
- [ ] `geofences` table (name, description, trigger_on, geometry_type, center_point/radius or boundary_polygon, notify_email, is_active)
- [ ] `bike_geofences` join table (last_known_inside, last_breach_notified_at)
- [ ] `geofence_breaches` table — **retained as-is**, now also produces a summary row in `alerts` (Addendum §4.3 relationship note)
- [ ] `POST /v1/geofences` — supports both Circle and Polygon (GeoJSON) geometry (FR-GEO-01/02)
- [ ] `GET /v1/geofences`, `GET /v1/geofences/{id}`, `PATCH /v1/geofences/{id}`, `DELETE /v1/geofences/{id}`
- [ ] Spatial evaluation on every telemetry packet: compare P(t-1) vs P(t) against active geofences (FR-GEO-02) — must complete within 50ms (§6.1)
- [ ] Breach detection + cooldown/debounce (default 15 min) (FR-GEO-03/04)
- [ ] `GET /v1/geofence-breaches` (filterable by `bike_id`)
- [ ] "Create Geofence" button + drawing toolbar/modal (circle center+radius, or polygon vertices) (FR-GEO-01)

### 7b. Generalized alert model (new in Addendum v1.1)
- [ ] `alert_rules` table (bike_id, alert_type, is_active, threshold_value, cooldown_minutes, channels[])
- [ ] `alerts` table (bike_id, alert_rule_id, alert_type, severity, message, location, address, metadata, acknowledged_by/at, dispatched_channels[])
- [ ] `idx_alerts_bike_time`, `idx_alerts_unacknowledged` indexes
- [ ] All six alert types implemented and evaluated per packet / per health-check cycle (FR-ALERT-01):
  - [ ] `geofence_breach` (High) — reuses 7a logic
  - [ ] `device_offline` (High) — depends on Phase 5
  - [ ] `low_battery` (Medium, default threshold 20%) — depends on Phase 5 battery field
  - [ ] `unauthorized_motion` (High) — MPU-6500 interrupt while ignition off, no active maintenance-mode override
  - [ ] `speeding` (Medium, default 100 km/h for 30+ continuous seconds)
  - [ ] `weak_signal` (Low, CSQ < 5 or unknown for 15+ minutes while otherwise online)
- [ ] Per-rule independent cooldown windows (FR-ALERT-02)
- [ ] `GET /v1/alerts` (filterable: bike_id, alert_type, severity, acknowledged)
- [ ] `PATCH /v1/alerts/{alert_id}` — acknowledge
- [ ] `GET /v1/bikes/{bike_id}/alert-rules`
- [ ] `PATCH /v1/bikes/{bike_id}/alert-rules/{rule_id}` — edit threshold/active/channels
- [ ] Sidenav rename: **Geofence → Alerts**, unified filterable list of all six types, geofence config nested as one tab (FR-ALERT-03)
- [ ] "Maintenance mode" / "authorized handling" toggle on Bikes view to suppress `unauthorized_motion` false alarms during servicing (FR-ALERT-04)
- [ ] Default channel policy: High severity → email + SMS; Medium/Low → email only; user-configurable per rule (FR-ALERT-05)
- [ ] Email dispatch SLA: handed to SMTP relay within 5 seconds of confirmed event (§6.1, extends to all alert types not just geofence)
- [ ] **Explicitly out of scope, tracked separately:** SMS provider integration (Africa's Talking / Twilio), delivery receipts, per-message cost tracking (Addendum §4.5 note)

**Definition of done:** All six alert types fire correctly with appropriate severity, default channel, and cooldown; the Alerts view shows a unified feed; geofence creation/editing still works exactly as before under its own tab.

---

## Phase 8 — Non-Functional Requirements (cross-cutting, apply throughout)
*Source: SRS v1.0.0 §6*

- [ ] `GET /v1/bikes` responds within 120ms p95 (§6.1) — validate after Phase 5/6 additions don't regress this
- [ ] Geofence spatial evaluation completes within 50ms of ingestion (§6.1)
- [ ] Email dispatch within 5 seconds of confirmed trigger (§6.1) — now applies to all 6 alert types (Phase 7)
- [ ] TLS 1.3 on all traffic (§6.2)
- [ ] Token-bucket rate limiting (5 req/min/IP) on auth endpoints, implemented in Redis (§6.2)
- [ ] Tenant isolation enforced via the Phase 2 access-resolution rule on every handler (§6.2, superseded/extended by Addendum §2.3)
- [ ] Standard RFC 7807 error envelope on all 4xx/5xx responses (§5.1)

---

## 7. Open Items Not Yet Specified in Either SRS
*(Flagged during review — not yet committed scope, but required for a working product. Track as follow-up SRS sections before building.)*

- [ ] **Device provisioning/pairing flow** — how a new IMEI is actually bound to an organization and a `bikes` row (QR code scan, manual IMEI entry, factory pre-registration). Neither v1.0.0 nor v1.1 defines this.
- [ ] **SMS provider integration** — explicitly deferred in Addendum §4.5.
- [ ] **Remote immobilization command queue** — referenced in the original hardware/firmware design (relay control, server command polling) but never given endpoints or a schema in either SRS version. If immobilization is still in scope, it needs its own FR section (`POST /v1/bikes/{bike_id}/commands`, a `commands` table with ack/status tracking).
- [ ] **Data export** (CSV/PDF trip or breach reports) — common compliance need, not specified.
- [ ] **Billing/plan enforcement** — `organizations.plan_tier` exists as a column (Addendum §2.3) but no endpoint or logic enforces plan limits (e.g., bike count caps per tier).
- [ ] **Map clustering** for fleets exceeding a few dozen visible bikes at once — flagged as a UI scaling concern, not addressed in either document's frontend spec.

---

## Cross-Reference Index

| FR Code Prefix | Source Document | Section |
|---|---|---|
| FR-AUTH | v1.0.0 | §4.1 |
| FR-PWD | v1.0.0 | §4.2 |
| FR-DASH | v1.0.0 | §4.3 |
| FR-GEO | v1.0.0 | §4.4 |
| FR-NAV | v1.0.0 | §4.5 |
| FR-TRIP | Addendum v1.1 | §1 |
| FR-ORG | Addendum v1.1 | §2 |
| FR-HEALTH | Addendum v1.1 | §3 |
| FR-ALERT | Addendum v1.1 | §4 |
