-- +goose Up
-- Add payment tracking fields to expenses table
ALTER TABLE expenses ADD COLUMN payment_status VARCHAR(20);
ALTER TABLE expenses ADD COLUMN payment_id VARCHAR(255);
ALTER TABLE expenses ADD COLUMN payment_external_id VARCHAR(255);
ALTER TABLE expenses ADD COLUMN paid_at TIMESTAMP;

-- Add index for payment tracking
CREATE INDEX idx_expenses_payment_status ON expenses(payment_status);
CREATE INDEX idx_expenses_payment_external_id ON expenses(payment_external_id);

-- +goose Down
-- Remove payment tracking fields and indexes
DROP INDEX IF EXISTS idx_expenses_payment_external_id;
DROP INDEX IF EXISTS idx_expenses_payment_status;
ALTER TABLE expenses DROP COLUMN IF EXISTS paid_at;
ALTER TABLE expenses DROP COLUMN IF EXISTS payment_external_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS payment_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS payment_status;
