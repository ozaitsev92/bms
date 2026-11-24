CREATE INDEX IF NOT EXISTS idx_apartment_building_id ON apartment(building_id);

ALTER TABLE apartment
    DROP CONSTRAINT IF EXISTS chk_apartment_sq_meters_positive,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE building
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;
