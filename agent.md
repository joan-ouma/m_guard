# Agents: MotorGuarrd Fleet Management Portal

**Source:** SRS v1.0.0 + Addendum v1.1
**Purpose:** This document defines the autonomous background services ("agents") the SRS documents imply but don't fully specify as standalone units. Each agent runs independently of the request/response API layer, consumes and produces the schema objects defined in the SRS, and has a clear trigger, responsibility boundary, and failure mode. This is not new scope — every agent below exists because a functional requirement in the SRS says something "shall" happen asynchronously, on a timer, or as a side effect of ingestion, without dictating what process does it.

---

## Agent Index

| Agent | Trigger | Primary Table(s) Written | Source Requirement |
|---|---|---|---|
| Telemetry Ingestion Agent | Inbound device packet | `bikes`, `breadcrumbs` | v1.0.0 §2.1, Addendum §1.2 |
| Reverse Geocoding Agent | New coordinate on ingestion | `bikes` (address fields) | FR-DASH-04 |
| Geofence Evaluation Agent | New coordinate on ingestion | `geofence_breaches`, `alerts`, `bike_geofences` | FR-GEO-02/03 |
| Trip Segmentation Agent | Ignition/speed state change | `trips`, `breadcrumbs.trip_id` | FR-TRIP-01 |
| Device Health Monitor Agent | 60-second timer | `bikes` (health fields), `device_health_events`, `alerts` | FR-HEALTH-02 |
| Alert Evaluation Agent | New telemetry packet or health-check tick | `alerts` | FR-ALERT-01 |
| Notification Dispatch Agent | New unacknowledged `alerts` row | `alerts.dispatched_channels` | FR-GEO-04, FR-ALERT-05 |
| Retention & Archival Agent | Daily timer | `breadcrumbs`, `trips` (cold storage) | FR-TRIP-03 |

---

## 1. Telemetry Ingestion Agent

**Responsibility:** Single entry point for all data arriving from ESP32/SIM868 hardware. Validates, normalizes, and writes the raw packet before any downstream agent acts on it.

**Trigger:** HTTPS POST from a device (authenticated via `X-Device-Key` HMAC signature per base SRS §5.1).

**Inputs (per packet):**
- Core fields: `bike_id` (resolved from IMEI), `latitude`, `longitude`, `speed_kmh`, `heading_deg`, `ignition` (base SRS)
- Health fields added in Addendum §3.2: `battery_voltage_mv`, `battery_percent`, `signal_quality_csq`, `gnss_fix_type`

**Actions:**
1. Verify device signature; reject with `401` on failure (do not write anything).
2. Reject packets with `gnss_fix_type == 'no_fix'` from being treated as a location update, but still record the health fields and heartbeat timestamp (a device can be alive with no fix — that's a health signal, not a location signal).
3. Write a row to `breadcrumbs` unconditionally (every valid packet, regardless of trip state).
4. Update `bikes.last_location`, `speed_kmh`, `heading_deg`, `ignition`, `last_telemetry_at`, `last_heartbeat_at`, and the four health fields in place.
5. Emit an internal event (queue message, not a direct function call) so the Reverse Geocoding, Geofence Evaluation, Trip Segmentation, and Alert Evaluation agents can each react independently without the ingestion path blocking on any of them.

**Failure mode:** If any downstream agent is unavailable, the raw packet must still be persisted to `breadcrumbs`/`bikes`. Ingestion never fails because an alert or geocoding service is down — those are best-effort consumers of the event, not gatekeepers of the write.

**Explicitly not this agent's job:** Trip boundary logic, address resolution, geofence math. It only writes truth and announces it happened.

---

## 2. Reverse Geocoding Agent

**Responsibility:** Convert raw coordinates into `current_address`, `sub_county`, `county` (FR-DASH-04).

**Trigger:** Ingestion event from Agent 1.

**Actions:**
1. Look up the coordinate against the administrative boundary and street index.
2. Write `current_address`, `sub_county`, `county` back onto the `bikes` row.
3. On lookup failure (e.g., third-party geocoder timeout), retain the previous known address rather than nulling it, and log the failure — a stale-but-present address is more useful to a dispatcher than a blank field.

**Rate/latency note:** This agent's output feeds the Telemetry Ribbon directly, which has a documented p95 target of 120ms on `GET /v1/bikes` (base SRS §6.1). If geocoding cannot complete synchronously within that budget, it must run asynchronously and the API must serve the last-known address rather than blocking the request.

---

## 3. Geofence Evaluation Agent

**Responsibility:** Detect boundary crossings and record breaches (FR-GEO-02/03).

**Trigger:** Ingestion event from Agent 1, specifically a coordinate change.

**Actions:**
1. Load all active geofences linked to the bike via `bike_geofences`.
2. For each, compare the previous known point (`P(t-1)`) and current point (`P(t)`) using `ST_Contains` / `ST_DWithin` against `boundary_polygon` or `center_point`+`radius_meters`.
3. On a rule match (`ON_EXIT`, `ON_ENTER`, or `BOTH`) not currently in cooldown:
   - Write a `geofence_breaches` row.
   - Write a corresponding summary row to `alerts` with `alert_type = 'geofence_breach'` (Addendum §4.3 relationship note) — this keeps the Alerts view and the geofence-specific audit log both correct from one write path.
   - Update `bike_geofences.last_known_inside` and `last_breach_notified_at`.
4. Respect per-geofence cooldown (default 15 minutes, FR-GEO-04) — do not re-fire while in cooldown even if the bike oscillates across the boundary.

**Latency requirement:** Must complete within 50ms of ingestion (base SRS §6.1). This is the tightest latency budget of any agent in the system; it should run against an in-memory or cached geofence set per bike rather than re-querying `geofences` from cold storage on every packet.

---

## 4. Trip Segmentation Agent

**Responsibility:** Convert the raw breadcrumb stream into discrete `trips` records (FR-TRIP-01, Addendum §1.2).

**Trigger:** Ignition or speed state change detected in an ingestion event — this agent does not need to run on every packet, only on packets that could plausibly start or end a trip.

**State machine (exact rules from Addendum §1.2):**
- **Start a trip** when `ignition` goes `false → true`, OR `speed_kmh > 2.0` after 3+ consecutive at-rest packets.
- **End a trip** when `ignition` goes `true → false` AND `speed_kmh ≤ 2.0` holds for a continuous 90-second debounce window.
- **Discard** any candidate trip under 60 seconds duration or 50 meters displacement — do not persist to `trips`, but leave the underlying `breadcrumbs` rows untouched.
- **Merge** a new trip into the immediately preceding one if it starts within 5 minutes and 100 meters of that trip's end point.

**Actions on trip close:**
1. Compute `distance_meters` (sum of great-circle distances between consecutive breadcrumbs), `max_speed_kmh`, `avg_speed_kmh`, `duration_seconds`.
2. Resolve `start_address`/`end_address` from the already-geocoded breadcrumb endpoints (reuse Agent 2's output rather than re-geocoding).
3. Backfill `breadcrumbs.trip_id` for every breadcrumb in the segment.
4. Set `trips.status = 'completed'` (or `'discarded'` if the minimum-duration/distance filter fails).

**Timing requirement:** Segmentation must complete within 60 seconds of trip-end detection (FR-TRIP-01) — this agent can run as a short-delay queued job rather than inline with ingestion, since trip history is a lookback feature, not a live-status one.

---

## 5. Device Health Monitor Agent

**Responsibility:** Classify every bike's connectivity state on a fixed cadence, independent of whether new packets arrive (FR-HEALTH-02).

**Trigger:** Fixed 60-second timer (Addendum §3.3) — this is the one agent in the system that is **not** event-driven off ingestion, precisely because its job is to detect the *absence* of events.

**Actions per tick, per bike:**
1. Compute `minutes_since_last_packet` from `last_heartbeat_at`.
2. Apply thresholds from Addendum §3.3:
   - `≤ 450s` → `healthy`
   - `450s–1200s` → `degraded`
   - `> 1200s` → `offline`
   - **Exception:** if the bike's last known `ignition == false` and no MPU-6500 motion interrupt was seen, extend the offline threshold to `1200s × 6 = 7200s`... **correction:** per spec this is stated directly as `12×` the expected interval = 3600s (60 minutes), not derived from the 1200s offline threshold — implement the grace period as `12 × expected_interval_seconds`, independent of the 4× multiplier used for the non-parked case.
3. On any status transition, write a `device_health_events` row (`went_offline`, `came_online`, `degraded`, `recovered`) and update `bikes.health_status`.
4. On transition to `offline`, hand off to the Alert Evaluation Agent to raise a `device_offline` alert — this agent does not write to `alerts` directly, keeping alert-severity/channel logic centralized in Agent 6.

**Why this agent must be timer-driven, not packet-driven:** A dead or destroyed tracker produces *no packets at all*. If this logic only ran inside the ingestion path, a bike whose tracker was physically destroyed would never be evaluated again. The 60-second sweep is what catches silence.

---

## 6. Alert Evaluation Agent

**Responsibility:** Central decision point for all six alert types (FR-ALERT-01). Every other agent that detects a condition worth alerting on hands off to this agent rather than writing to `alerts` directly, so severity, cooldown, and channel-selection logic lives in exactly one place.

**Trigger:** Event handoff from Agent 1 (telemetry-based conditions) or Agent 5 (health-based conditions).

**Alert types and the condition each one watches (Addendum §4.2):**

| Alert Type | Watched Condition | Severity |
|---|---|---|
| `geofence_breach` | Handoff from Agent 3 | High |
| `device_offline` | Handoff from Agent 5 on `offline` transition | High |
| `low_battery` | `battery_percent` < configurable threshold (default 20%) | Medium |
| `unauthorized_motion` | MPU-6500 interrupt fires while `ignition == false` and bike is not in maintenance mode | High |
| `speeding` | `speed_kmh` > configurable limit (default 100) for 30+ continuous seconds | Medium |
| `weak_signal` | `signal_quality_csq` < 5 (or 99) for 15+ minutes while otherwise online | Low |

**Actions:**
1. Look up the bike's `alert_rules` row for the matching `alert_type`; skip entirely if `is_active = false`.
2. Check `cooldown_minutes` against the most recent `alerts` row of the same type for this bike; skip if still in cooldown (FR-ALERT-02).
3. For `unauthorized_motion` specifically: check the bike's maintenance-mode/authorized-handling flag before raising (FR-ALERT-04) — this is a hard suppression, not just a lower severity.
4. On a valid, non-cooled-down match: write an `alerts` row with `severity`, `message`, `location`, `address`, and relevant `metadata` (e.g., the exact CSQ value or speed reading that triggered it).
5. Hand off the new `alerts` row to the Notification Dispatch Agent — this agent does not send email/SMS itself.

**Design constraint:** This agent must not duplicate the spatial math already done by Agent 3 for geofences — it only receives the *result* of that evaluation as an event, then applies the shared severity/cooldown/rule-lookup logic uniformly across all six types.

---

## 7. Notification Dispatch Agent

**Responsibility:** Deliver a raised alert to the correct channel(s) (FR-ALERT-05, extending FR-GEO-04's email-only precedent).

**Trigger:** New row in `alerts` from Agent 6.

**Channel policy (Addendum §4.5, unless overridden by the bike's `alert_rules.channels`):**
- `high` severity → email **and** SMS by default
- `medium` / `low` severity → email only by default

**Actions:**
1. Read `alert_rules.channels` for this bike/alert_type combination; fall back to the severity-based default above if no override is set.
2. Format the notification: for email, an HTML/plain-text transactional message; for SMS, a condensed version (registration plate, alert type, one-line location, timestamp) sized for GSM 160-character segments given the eventual Africa's Talking/Twilio integration.
3. Dispatch to each configured channel and update `alerts.dispatched_channels` with only the channels that actually succeeded (not the ones attempted).
4. **SLA:** hand off to the SMTP relay within 5 seconds of the confirmed trigger (base SRS §6.1, extended to all alert types by the addendum's general model).

**Explicitly out of scope for this agent (per Addendum §4.5 note):** Selecting an SMS provider, handling delivery receipts, and tracking per-message SMS cost. This agent assumes an SMS-sending capability exists behind an interface; the provider integration itself is a separate, not-yet-written SRS.

---

## 8. Retention & Archival Agent

**Responsibility:** Enforce the data-retention policy on high-volume tables (FR-TRIP-03).

**Trigger:** Daily timer.

**Actions:**
1. For each organization, read `organizations.breadcrumb_retention_days` (default 180).
2. Move `breadcrumbs` and `trips` rows older than the retention window to cold storage (exact storage target — S3/Glacier-equivalent — is an infrastructure decision, not specified in either SRS).
3. Apply monthly partitioning maintenance on `breadcrumbs` if the fleet size has crossed the ~500-vehicle threshold noted in Addendum §1.3 as the point where partitioning becomes necessary.

**Note:** This agent enforces a floor (180 days minimum retention before archival is even eligible), not a hard deletion — the addendum specifies retention *before archival*, not data destruction. Actual deletion policy beyond archival is not specified and should not be assumed.

---

## Cross-Agent Ordering Guarantees

Because several agents react to the same ingestion event, the following ordering must hold to avoid inconsistent reads:

1. **Agent 1 (Ingestion)** must complete its write to `bikes`/`breadcrumbs` before any other agent reads that packet's data.
2. **Agent 2 (Geocoding)** should complete before **Agent 4 (Trip Segmentation)** attempts to resolve `start_address`/`end_address` on trip close, so trip segmentation reuses geocoded output rather than triggering a second geocode call.
3. **Agent 3 (Geofence Evaluation)** and **Agent 5 (Health Monitor)** are independent of each other and of Agent 4 — they may run in parallel.
4. **Agent 6 (Alert Evaluation)** must run strictly after Agents 3 and 5 for any given event, since it only reacts to their handoffs and never re-derives geofence or health state itself.
5. **Agent 7 (Notification Dispatch)** must run strictly after Agent 6 writes the `alerts` row.

No agent in this system should perform a synchronous, blocking call to another agent's logic. All handoffs described above are event/queue-based so that a slow or failing downstream agent (e.g., an SMTP outage in Agent 7) cannot back-pressure ingestion (Agent 1), which is the one write path that must never fail.
