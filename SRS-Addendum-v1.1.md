# SRS Addendum v1.1: MotoGuard Fleet Management Portal

**Applies to:** SRS Document v1.0.0 (MotoGuard Fleet Management Portal)
**Status:** Approved for Engineering
**Purpose:** This addendum closes four gaps identified in the v1.0.0 review: absence of trip history/playback, absence of a multi-tenant role hierarchy, absence of device health monitoring, and an alert taxonomy limited to geofence breaches. Each section below extends the original SRS's functional requirements, database schema, and REST API without contradicting it. Where a v1.0.0 table or endpoint is modified rather than purely extended, that change is called out explicitly.

---

## 1. Historical Trip Playback (FR-TRIP)

### 1.1 Rationale
The v1.0.0 SRS persists only the *latest* telemetry state per motorcycle plus a breach log. There is no way to answer "where was this bike between 2 PM and 6 PM yesterday," which is a baseline expectation set by Traccar, Samsara, and Verizon Connect. This section introduces a `trips` and `breadcrumbs` model and the endpoints to query them.

### 1.2 Trip Segmentation Logic
A **trip** is a contiguous span of motion bounded by ignition state transitions:

- **Trip Start:** `ignition` transitions from `false` to `true`, OR `speed_kmh` exceeds `2.0` after being at rest for more than 3 consecutive telemetry packets.
- **Trip End:** `ignition` transitions from `true` to `false` AND `speed_kmh` remains at or below `2.0` for a debounce window of **90 seconds** (prevents a trip from being fragmented by a momentary ignition blip at a traffic light or fuel stop).
- **Minimum Trip Duration:** Trips lasting less than **60 seconds** or covering less than **50 meters** of displacement are discarded as GPS/ignition noise and are not persisted as trip records, though their raw breadcrumbs remain in the telemetry history for audit purposes.
- **Idle Merge Rule:** If a new trip starts within **5 minutes** of the previous trip's end at a location within **100 meters** of that trip's end point, the two trips are merged into one (handles brief stops such as a red light misclassified as ignition-off on older firmware).

Trip segmentation is computed **server-side, asynchronously**, by a background worker that consumes the raw telemetry stream — it is never computed on the ESP32 firmware, which only reports raw ignition/speed/coordinate state per FR-DASH-03 of the base SRS.

### 1.3 Database Schema Additions

```sql
CREATE TABLE trips (
    id VARCHAR(32) PRIMARY KEY,
    bike_id VARCHAR(32) NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    start_location GEOGRAPHY(Point, 4326) NOT NULL,
    end_location GEOGRAPHY(Point, 4326),
    start_address TEXT,
    end_address TEXT,
    distance_meters NUMERIC(10,2) NOT NULL DEFAULT 0,
    max_speed_kmh NUMERIC(5,2) NOT NULL DEFAULT 0,
    avg_speed_kmh NUMERIC(5,2) NOT NULL DEFAULT 0,
    duration_seconds INTEGER,
    status VARCHAR(16) NOT NULL DEFAULT 'in_progress'
        CHECK (status IN ('in_progress', 'completed', 'discarded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trips_bike_started ON trips (bike_id, started_at DESC);

CREATE TABLE breadcrumbs (
    id BIGSERIAL PRIMARY KEY,
    bike_id VARCHAR(32) NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    trip_id VARCHAR(32) REFERENCES trips(id) ON DELETE SET NULL,
    location GEOGRAPHY(Point, 4326) NOT NULL,
    speed_kmh NUMERIC(5,2) NOT NULL DEFAULT 0,
    heading_deg SMALLINT NOT NULL DEFAULT 0,
    ignition BOOLEAN NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_breadcrumbs_bike_time ON breadcrumbs (bike_id, recorded_at DESC);
CREATE INDEX idx_breadcrumbs_trip ON breadcrumbs (trip_id);
```

> **Retention note:** `breadcrumbs` grows unbounded on a low-duty-cycle device (roughly 288 rows/bike/day at 5-minute intervals). A partitioning strategy (monthly `PARTITION BY RANGE (recorded_at)`) is recommended at the infrastructure layer once fleets exceed roughly 500 vehicles; this is an implementation note, not a schema requirement.

### 1.4 New API Endpoints

| Method | URI Path | Description | Status Codes |
|---|---|---|---|
| GET | `/v1/bikes/{bike_id}/trips` | List completed trips for a bike within a date range | 200, 401, 404 |
| GET | `/v1/bikes/{bike_id}/trips/{trip_id}` | Retrieve a single trip's summary | 200, 401, 404 |
| GET | `/v1/bikes/{bike_id}/trips/{trip_id}/breadcrumbs` | Retrieve the full breadcrumb path for route replay | 200, 401, 404 |
| GET | `/v1/bikes/{bike_id}/breadcrumbs` | Retrieve raw breadcrumbs across a time window, independent of trip boundaries | 200, 400, 401, 404 |

**`GET /v1/bikes/{bike_id}/trips?from=2026-09-25T14:00:00Z&to=2026-09-25T18:00:00Z`**

Response (200 OK):
```json
{
  "data": [
    {
      "id": "trp_01J9A1B2C3D4E5F6G7H8J9K0",
      "started_at": "2026-09-25T14:12:03Z",
      "ended_at": "2026-09-25T14:47:19Z",
      "duration_seconds": 2116,
      "distance_meters": 8420.5,
      "max_speed_kmh": 62.0,
      "avg_speed_kmh": 34.2,
      "start_address": "Magadi Road, Ongata Rongai",
      "end_address": "Waiyaki Way, Westlands",
      "status": "completed"
    }
  ],
  "meta": { "total_count": 1, "range_from": "2026-09-25T14:00:00Z", "range_to": "2026-09-25T18:00:00Z" }
}
```

**`GET /v1/bikes/{bike_id}/trips/{trip_id}/breadcrumbs`** — used to draw the playback polyline on the map.

Response (200 OK):
```json
{
  "data": [
    { "lat": -1.396420, "lng": 36.756810, "speed_kmh": 0.0, "heading_deg": 0, "recorded_at": "2026-09-25T14:12:03Z" },
    { "lat": -1.389110, "lng": 36.761230, "speed_kmh": 41.5, "heading_deg": 68, "recorded_at": "2026-09-25T14:17:03Z" },
    { "lat": -1.371980, "lng": 36.778450, "speed_kmh": 55.0, "heading_deg": 72, "recorded_at": "2026-09-25T14:22:03Z" }
  ],
  "meta": { "point_count": 3, "trip_id": "trp_01J9A1B2C3D4E5F6G7H8J9K0" }
}
```

### 1.5 New Functional Requirements
- **FR-TRIP-01:** The system shall segment raw telemetry into discrete trips per Section 1.2's rules, computed asynchronously within 60 seconds of trip-end detection.
- **FR-TRIP-02:** The **Bikes** view (FR-NAV-02) shall gain a "History" tab per motorcycle, presenting a date-range picker and a scrollable trip list, each trip expandable into a map polyline replay with a scrub/playback slider.
- **FR-TRIP-03:** Trip and breadcrumb data shall be retained for a minimum of **180 days** before archival to cold storage, configurable per organization (see Section 2).

---

## 2. Multi-Tenancy & Role Hierarchy (FR-ORG)

### 2.1 Rationale
The v1.0.0 schema binds every `bike` directly to a single `user_id`. This works only for an individual owner. It cannot express a fleet coordinator who needs to grant a mechanic read-only access to one bike, nor can it support selling the platform to more than one paying customer without data leaking across tenants. This section introduces **Organizations**, **Memberships**, and **Bike-level access grants**.

### 2.2 Role Model

| Role | Scope | Permissions |
|---|---|---|
| `owner` | Organization | Full control: billing, member management, all bikes, all geofences, org deletion |
| `admin` | Organization | Manage all bikes/geofences/members except billing and org deletion |
| `dispatcher` | Organization | View all bikes, create/edit geofences, acknowledge alerts; cannot manage members or billing |
| `viewer` | Organization or single bike | Read-only access to assigned bikes' telemetry, trips, and alerts |

A `viewer` role may be scoped to **one or more specific bikes** rather than the whole organization — this satisfies the "grant a mechanic access to one bike" case explicitly.

### 2.3 Database Schema Additions (and one modification)

```sql
CREATE TABLE organizations (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    plan_tier VARCHAR(24) NOT NULL DEFAULT 'individual'
        CHECK (plan_tier IN ('individual', 'fleet', 'enterprise')),
    breadcrumb_retention_days INTEGER NOT NULL DEFAULT 180,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE memberships (
    id VARCHAR(32) PRIMARY KEY,
    organization_id VARCHAR(32) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(16) NOT NULL CHECK (role IN ('owner', 'admin', 'dispatcher', 'viewer')),
    invited_by VARCHAR(32) REFERENCES users(id),
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, user_id)
);

CREATE TABLE bike_access_grants (
    id VARCHAR(32) PRIMARY KEY,
    membership_id VARCHAR(32) NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    bike_id VARCHAR(32) NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (membership_id, bike_id)
);

-- Modification to v1.0.0 bikes table: ownership moves from user_id to organization_id
ALTER TABLE bikes ADD COLUMN organization_id VARCHAR(32) REFERENCES organizations(id);
ALTER TABLE bikes ALTER COLUMN user_id DROP NOT NULL;
-- user_id is retained temporarily for backward migration only; organization_id becomes
-- the authoritative ownership column. A data migration must backfill organization_id
-- by creating one default 'individual' organization per existing user_id before this
-- column is made NOT NULL and user_id is dropped in a follow-up migration.
```

> **Access resolution rule:** A user can access a bike if either (a) their membership role in the bike's organization is `owner`, `admin`, or `dispatcher`, or (b) their membership role is `viewer` **and** a matching row exists in `bike_access_grants`. This single rule replaces the flat `user_id` scoping described in the base SRS's Section 6.2 (Tenant Isolation) and must be enforced on every bike-scoped query.

### 2.4 New / Modified API Endpoints

| Method | URI Path | Description | Status Codes |
|---|---|---|---|
| GET | `/v1/organizations/current` | Retrieve the authenticated user's active organization | 200, 401 |
| GET | `/v1/organizations/current/members` | List all members and their roles | 200, 401, 403 |
| POST | `/v1/organizations/current/members` | Invite a user by email with a specified role | 201, 400, 401, 403, 409 |
| PATCH | `/v1/organizations/current/members/{membership_id}` | Change a member's role | 200, 400, 401, 403, 404 |
| DELETE | `/v1/organizations/current/members/{membership_id}` | Remove a member from the organization | 204, 401, 403, 404 |
| POST | `/v1/bikes/{bike_id}/access-grants` | Grant a `viewer`-role member access to a specific bike | 201, 400, 401, 403, 404 |
| DELETE | `/v1/bikes/{bike_id}/access-grants/{grant_id}` | Revoke a bike-specific access grant | 204, 401, 403, 404 |

**`POST /v1/organizations/current/members`**
```json
{
  "email": "mechanic@MotoGuard.co.ke",
  "role": "viewer",
  "bike_ids": ["bike_01J8Z9X2A4B6C8D0E2F4G6H8"]
}
```
Response (201 Created):
```json
{
  "data": {
    "membership_id": "mem_01J9B2C3D4E5F6G7H8J9K0L1",
    "email": "mechanic@MotoGuard.co.ke",
    "role": "viewer",
    "status": "invited",
    "scoped_bikes": ["bike_01J8Z9X2A4B6C8D0E2F4G6H8"]
  }
}
```

### 2.5 New Functional Requirements
- **FR-ORG-01:** Every bike shall belong to exactly one organization; no bike may exist without an owning organization.
- **FR-ORG-02:** A `viewer` member scoped to specific bikes shall see only those bikes in the fleet dropdown (FR-DASH-02) and shall receive 403 Forbidden on any endpoint referencing an out-of-scope `bike_id`.
- **FR-ORG-03:** Role changes and access-grant changes shall be written to an immutable audit log (`membership_audit_log`, structurally identical in intent to `geofence_breaches`) capturing actor, target, prior role, new role, and timestamp.
- **FR-ORG-04:** The **Settings** view (FR-NAV-04) shall gain a "Team" tab exposing member invitation, role editing, and per-bike access grant management, visible only to `owner` and `admin` roles.

---

## 3. Device Health & Offline Detection (FR-HEALTH)

### 3.1 Rationale
Because the ESP32/SIM868 hardware transmits on a **5-minute duty cycle** (per the hardware design's stealth power strategy), silence from a device is inherently ambiguous: it could mean the bike is parked and the tracker is correctly asleep, or it could mean the tracker has been discovered and destroyed, the battery has died, or the SIM has lost signal. This ambiguity is more consequential here than in a continuously-transmitting commercial fleet tracker, so device health must be modeled explicitly rather than inferred from telemetry alone.

### 3.2 Health Signal Definitions
Every telemetry packet sent by the firmware shall include, in addition to the fields already defined in the base SRS, the following supplementary fields:

| Field | Type | Description |
|---|---|---|
| `battery_voltage_mv` | integer | Raw tracker battery voltage in millivolts (from the Li-ion cell, not the motorcycle battery) |
| `battery_percent` | integer (0–100) | Derived state-of-charge estimate |
| `signal_quality_csq` | integer (0–31, or 99 if unknown) | Raw `AT+CSQ` response from the SIM868 |
| `gnss_fix_type` | enum (`no_fix`, `2d`, `3d`) | GPS fix quality at time of packet |

### 3.3 Offline Classification Logic
A background health-check worker evaluates every bike on a fixed schedule (every 60 seconds) against its **expected reporting interval**:

- **Healthy:** Last packet received within `expected_interval_seconds × 1.5` (default expected interval: 300s, so healthy threshold = 450s).
- **Degraded:** Last packet received between `1.5×` and `4×` the expected interval (i.e., 450s–1200s). Indicates possible signal trouble or a missed window, not yet alarm-worthy.
- **Offline:** No packet received for longer than `4×` the expected interval (>1200s / 20 minutes). Triggers an offline alert per Section 4.
- **Silence Grace Period:** If the bike's last known `ignition` state was `false` (parked) **and** no motion was detected by the MPU-6500 interrupt, the offline threshold is extended to `12×` the expected interval (60 minutes) before alerting, since a correctly-functioning stealth device may intentionally delay reporting while parked. This distinguishes expected stealth-mode silence from a genuinely dead or destroyed tracker.

### 3.4 Database Schema Additions

```sql
-- Extend bikes with the latest known health snapshot for fast dashboard reads
ALTER TABLE bikes ADD COLUMN battery_percent SMALLINT;
ALTER TABLE bikes ADD COLUMN signal_quality_csq SMALLINT;
ALTER TABLE bikes ADD COLUMN gnss_fix_type VARCHAR(8);
ALTER TABLE bikes ADD COLUMN health_status VARCHAR(16) NOT NULL DEFAULT 'healthy'
    CHECK (health_status IN ('healthy', 'degraded', 'offline'));
ALTER TABLE bikes ADD COLUMN last_heartbeat_at TIMESTAMPTZ;

CREATE TABLE device_health_events (
    id VARCHAR(32) PRIMARY KEY,
    bike_id VARCHAR(32) NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    event_type VARCHAR(16) NOT NULL
        CHECK (event_type IN ('went_offline', 'came_online', 'degraded', 'low_battery', 'recovered')),
    battery_percent SMALLINT,
    signal_quality_csq SMALLINT,
    minutes_since_last_packet INTEGER,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_health_bike_time ON device_health_events (bike_id, occurred_at DESC);
```

### 3.5 New API Endpoints

| Method | URI Path | Description | Status Codes |
|---|---|---|---|
| GET | `/v1/bikes/{bike_id}/health` | Retrieve current health snapshot (battery, signal, offline status) | 200, 401, 404 |
| GET | `/v1/bikes/{bike_id}/health/events` | Retrieve historical health state transitions | 200, 401, 404 |

**`GET /v1/bikes/{bike_id}/health`**
```json
{
  "data": {
    "health_status": "degraded",
    "battery_percent": 34,
    "signal_quality_csq": 9,
    "gnss_fix_type": "3d",
    "last_heartbeat_at": "2026-09-26T08:42:00Z",
    "minutes_since_last_packet": 14,
    "expected_interval_seconds": 300,
    "silence_grace_active": false
  }
}
```

### 3.6 New Functional Requirements
- **FR-HEALTH-01:** The Telemetry Ribbon (FR-DASH-03 of the base SRS) shall gain a sixth column, "Device Health," displaying a battery-percent icon, signal-bar icon, and a health-status badge (`Healthy` / `Degraded` / `Offline`) using the same flat badge style as the Running/Stopped indicators.
- **FR-HEALTH-02:** The system shall transition and log a `health_status` change immediately upon crossing a threshold defined in Section 3.3, independent of whether new telemetry has arrived (i.e., the check runs on a timer, not only on packet receipt).
- **FR-HEALTH-03:** The **Bikes** view shall support sorting and filtering the fleet list by `health_status` to allow a coordinator to quickly triage which vehicles need physical inspection.

---

## 4. Expanded Alert Taxonomy (FR-ALERT)

### 4.1 Rationale
The base SRS wires exactly one condition — geofence breach — to exactly one channel — email. The hardware layer already produces motion-interrupt data (MPU-6500), ignition state, battery voltage, and signal quality, none of which currently reach the user. This section defines a unified alert model covering five alert types and two delivery channels, superseding the single-purpose breach-email flow in FR-GEO-04 of the base SRS (that flow is preserved as one case of the general model below, not removed).

### 4.2 Alert Types

| Alert Type | Trigger Condition | Default Severity |
|---|---|---|
| `geofence_breach` | Per FR-GEO-02/03 of base SRS | High |
| `device_offline` | `health_status` transitions to `offline` per Section 3.3 | High |
| `low_battery` | `battery_percent` drops below a configurable threshold (default 20%) | Medium |
| `unauthorized_motion` | MPU-6500 motion interrupt fires while `ignition == false` and no active "armed-off" override is set by the user | High |
| `speeding` | `speed_kmh` exceeds a per-bike configurable limit (default 100 km/h) for longer than 30 continuous seconds | Medium |
| `weak_signal` | `signal_quality_csq` reports below 5 (or 99/unknown) for longer than 15 minutes while the device is otherwise online | Low |

The `unauthorized_motion` type is the direct portal-side surface for the motion-triggered alarm feature designed into the firmware; without it, the hardware's motion sensor produces no user-visible outcome, which was the specific gap identified.

### 4.3 Database Schema Additions

```sql
CREATE TABLE alert_rules (
    id VARCHAR(32) PRIMARY KEY,
    bike_id VARCHAR(32) NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    alert_type VARCHAR(24) NOT NULL
        CHECK (alert_type IN ('geofence_breach', 'device_offline', 'low_battery',
                               'unauthorized_motion', 'speeding', 'weak_signal')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    threshold_value NUMERIC(10,2),
    cooldown_minutes INTEGER NOT NULL DEFAULT 15,
    channels VARCHAR(8)[] NOT NULL DEFAULT ARRAY['email'],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bike_id, alert_type)
);

CREATE TABLE alerts (
    id VARCHAR(32) PRIMARY KEY,
    bike_id VARCHAR(32) NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    alert_rule_id VARCHAR(32) REFERENCES alert_rules(id) ON DELETE SET NULL,
    alert_type VARCHAR(24) NOT NULL,
    severity VARCHAR(8) NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    message TEXT NOT NULL,
    location GEOGRAPHY(Point, 4326),
    address TEXT,
    metadata JSONB,
    acknowledged_by VARCHAR(32) REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ,
    dispatched_channels VARCHAR(8)[] NOT NULL DEFAULT ARRAY[]::VARCHAR(8)[],
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_bike_time ON alerts (bike_id, occurred_at DESC);
CREATE INDEX idx_alerts_unacknowledged ON alerts (bike_id) WHERE acknowledged_at IS NULL;
```

> **Relationship to `geofence_breaches`:** The base SRS's `geofence_breaches` table is retained as the detailed record for that specific alert type (it carries geofence-specific fields the generic `alerts` table does not, such as `breach_type`). Every row inserted into `geofence_breaches` also produces a corresponding summary row in `alerts` with `alert_type = 'geofence_breach'`, so that the Geofence view's audit log and a unified "All Alerts" view can both be served without duplicating logic.

### 4.4 New API Endpoints

| Method | URI Path | Description | Status Codes |
|---|---|---|---|
| GET | `/v1/alerts` | List alerts across the fleet, filterable by `bike_id`, `alert_type`, `severity`, `acknowledged` | 200, 401 |
| PATCH | `/v1/alerts/{alert_id}` | Acknowledge an alert | 200, 401, 404 |
| GET | `/v1/bikes/{bike_id}/alert-rules` | List configured alert rules for a bike | 200, 401, 404 |
| PATCH | `/v1/bikes/{bike_id}/alert-rules/{rule_id}` | Update threshold, active state, or delivery channels for a rule | 200, 400, 401, 404 |

**`GET /v1/alerts?acknowledged=false&severity=high`**
```json
{
  "data": [
    {
      "id": "alt_01J9C3D4E5F6G7H8J9K0L1M2",
      "bike_id": "bike_01J8Z9X2A4B6C8D0E2F4G6H8",
      "alert_type": "unauthorized_motion",
      "severity": "high",
      "message": "Motion detected on KMDG 412X while parked and ignition off.",
      "address": "Magadi Road, Opposite Quickmart",
      "acknowledged_at": null,
      "dispatched_channels": ["email", "sms"],
      "occurred_at": "2026-09-26T02:14:08Z"
    },
    {
      "id": "alt_01J9C4D5E6F7G8H9J0K1L2M3",
      "bike_id": "bike_01J7A1B2C3D4E5F6G7H8J9K0",
      "alert_type": "device_offline",
      "severity": "high",
      "message": "KMDF 889P has not reported for 24 minutes; last known state was moving.",
      "acknowledged_at": null,
      "dispatched_channels": ["email"],
      "occurred_at": "2026-09-26T08:50:00Z"
    }
  ],
  "meta": { "total_count": 2, "unacknowledged_high_severity": 2 }
}
```

**`PATCH /v1/bikes/{bike_id}/alert-rules/{rule_id}`**
```json
{ "threshold_value": 15, "channels": ["email", "sms"] }
```

### 4.5 New Functional Requirements
- **FR-ALERT-01:** The system shall evaluate all six alert types defined in Section 4.2 against every incoming telemetry packet and against the periodic device-health check defined in Section 3.3.
- **FR-ALERT-02:** Each `alert_rules` row shall independently configure its own cooldown window (default 15 minutes, matching FR-GEO-04's precedent) to prevent notification flooding per alert type per bike.
- **FR-ALERT-03:** The **Geofence** nav item (FR-NAV-03 of base SRS) shall be renamed **Alerts** in the sidenav and shall present a unified, filterable list of all six alert types (not geofence breaches alone), with geofence-specific configuration nested as one tab within that view.
- **FR-ALERT-04:** `unauthorized_motion` alerts shall be suppressed when the user has explicitly toggled a bike into a "maintenance mode" or "authorized handling" state from the Bikes view, preventing false alarms during servicing.
- **FR-ALERT-05:** High-severity alerts (`geofence_breach`, `device_offline`, `unauthorized_motion`) shall dispatch via **both** email and SMS by default; medium and low severity default to email only. Channel selection per rule remains user-configurable via `PATCH /v1/bikes/{bike_id}/alert-rules/{rule_id}`.

> **Note on SMS delivery:** This addendum defines the alert *model* and default channel policy but does not specify an SMS provider integration (e.g., Africa's Talking, Twilio). That integration — including provider selection, delivery-receipt handling, and per-message cost tracking — is out of scope for this addendum and should be tracked as a separate integration SRS.

---

## 5. Summary of Sidenav Impact

The base SRS's four-item sidenav (Home, Bikes, Geofence, Settings) is affected as follows:

- **Home** — unchanged structurally; Telemetry Ribbon gains the Device Health column (Section 3.6).
- **Bikes** — gains a History tab (Section 1.5) and health-based sort/filter (Section 3.6).
- **Geofence → renamed Alerts** — becomes the umbrella view for all alert types, with geofence configuration as one tab (Section 4.5).
- **Settings** — gains a Team tab for organization/role management (Section 2.5).

No new top-level nav item is required; all additions nest within the existing four sections.
