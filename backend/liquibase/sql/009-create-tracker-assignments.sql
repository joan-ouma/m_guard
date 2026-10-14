CREATE TABLE IF NOT EXISTS tracker_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tracker_id UUID NOT NULL REFERENCES trackers(id) ON DELETE CASCADE,
    bike_id UUID NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unassigned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tracker_assignments_tracker_id ON tracker_assignments(tracker_id);
CREATE INDEX IF NOT EXISTS idx_tracker_assignments_bike_id ON tracker_assignments(bike_id);
CREATE INDEX IF NOT EXISTS idx_tracker_assignments_active ON tracker_assignments(tracker_id) WHERE unassigned_at IS NULL;
