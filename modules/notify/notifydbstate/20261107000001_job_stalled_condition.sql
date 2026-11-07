-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE notify.notification_condition ADD VALUE IF NOT EXISTS 'on_job_stalled';

-- +goose Down
DELETE FROM notify.notification_rules WHERE condition = 'on_job_stalled';
