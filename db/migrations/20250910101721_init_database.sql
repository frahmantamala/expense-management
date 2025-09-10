-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  department VARCHAR(255),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE permissions (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE user_permissions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  granted_by BIGINT REFERENCES users(id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  UNIQUE (user_id, permission_id)
);

CREATE TABLE expense_categories (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  description TEXT,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE expenses (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  amount_idr BIGINT NOT NULL,
  description TEXT NOT NULL,
  category VARCHAR(255),
  receipt_url VARCHAR(1024),
  receipt_filename VARCHAR(512),
  expense_status VARCHAR(50) NOT NULL DEFAULT 'pending_approval',
  expense_date DATE NOT NULL,
  submitted_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  processed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- optional FK from expenses.category -> expense_categories.name
ALTER TABLE expenses
  ADD CONSTRAINT fk_expense_category
  FOREIGN KEY (category)
  REFERENCES expense_categories(name);

CREATE TABLE payments (
  id BIGSERIAL PRIMARY KEY,
  expense_id BIGINT NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
  amount_idr BIGINT NOT NULL,
  external_id VARCHAR(255) NOT NULL UNIQUE,
  payment_api_id VARCHAR(255),
  payment_status VARCHAR(50) NOT NULL DEFAULT 'processing',
  payment_provider_response JSONB,
  error_message TEXT,
  retry_count INTEGER DEFAULT 0,
  initiated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  completed_at TIMESTAMP WITH TIME ZONE,
  failed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX idx_user_status ON expenses (user_id, expense_status);
CREATE INDEX idx_status_amount ON expenses (expense_status, amount_idr);
CREATE INDEX idx_submitted_date ON expenses (submitted_at);
CREATE INDEX idx_expense_payment_status ON payments (expense_id, payment_status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payments;
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS fk_expense_category;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS expense_categories;
DROP TABLE IF EXISTS user_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
