-- +goose Up
CREATE INDEX idx_ads_created_at ON ads (created_at);
CREATE INDEX idx_ads_category_city ON ads (category_id, city_id);

-- +goose Down
DROP INDEX IF EXISTS idx_ads_category_city;
DROP INDEX IF EXISTS idx_ads_created_at;
