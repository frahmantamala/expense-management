# Expense Management System

Expense management where are handle some functionality like create expense, approve and reject expense. This Expense also integrate with payment mocking api. 


Setup

```bash
# Clone the repository
git clone https://github.com/frahmantamala/expense-management.git
cd expense-management


# Access the application
# API: http://localhost:8080
# Swagger: http://localhost:8080/swagger/index.html
or
# API: http://localhost:8081
# Swagger: http://localhost:8081/swagger/index.html
```

**Seeding users:**
- Admin: `padil@mail.com` / `password` (can approve/reject)
- User: `fadhil@mail.com` / `password` (can only create)

## Development Setup

### Prerequisites
- Go 1.23+
- PostgreSQL 14+
- Make

### Manual Setup
```bash
# Clone and install dependencies
git clone https://github.com/frahmantamala/expense-management.git
cd expense-management
make deps

# Copy and update configuration
cp config.example.yml config.yml
# Edit config.yml with your database settings

# Database setup
make migrate
make seed

# Start the server
make run
```
### Or Use Docker
```bash
git clone https://github.com/frahmantamala/expense-management.git
cd expense-management

make docker-up
```

The API will be available at `http://localhost:8080` with Swagger docs at `/swagger/index.html`.

## 🧪 Testing
```bash
# Run all tests
make test

# Run specific test suites
make test-auth          # Authentication tests
make test-expense       # Expense module tests
make test-payment       # Payment module tests
make test-postgres      # Repository tests
```

## Business Logic

### Expense Approval
- **Auto-approval**: Expenses under Rp1.000.000 are automatically approved and paid
- **Manual approval**: Expenses > Rp1.000.000 require approval before payment
- **Categories**:(perjalanan, makan, kantor, pemasaran, etc)

### Payment Processing
- Automate payment initiation after approval
- External payment gateway integration with retry capability
- Payment status tracking (pending → success/failed)
- Failed payments can be retried by authorized users

### Permission System
- **User**: Submit and view own expenses
- **Approver**: Approve/reject expenses, retry failed payments
- **Admin**: Full system access

## API Examples
provide in swagger
http://localhost:8080/swagger/index.html

## Architecture
This app following approach domain driven which each module isolate in his own domain.

### Directory Structure
```
├── cmd/                    # Application commands
├── internal/
│   ├── auth/              # Authentication & authorization
│   ├── expense/           # Core expense business logic
│   ├── payment/           # Payment processing
│   ├── user/              # User management
│   └── transport/         # HTTP layer & middleware
├── db/migrations/         # Database migrations
└── api/                   # OpenAPI specification
```

### Key Design Decisions

**Layered Architecture**: Handler => Service => Repository pattern for clear separation of concerns and testability.

**Permission Middleware**: middleware and service-level permission checks.

**Async Payment Processing**: Payment operations run in background goroutines to avoid blocking HTTP requests.

**Configuration-Driven**: All external dependencies (database, payment API, security settings) are externalized to config files.

**OpenAPI-First**: API specification drives code generation and documentation.

## Database Schema

```sql
-- Core tables
users (id, email, password_hash, role, permissions)
expenses (id, user_id, amount_idr, description, category, expense_status)
payments

-- Audit fields on all tables
created_at, updated_at, processed_at, paid_at
```

Payment fields are separate concerns but stored in the expense table for simplicity.

## Development

### Commands
```bash
make build              # Compile binary
make run               # Start development server
make test              # Run tests
make migrate           # Run database migrations
make seed              # Seed test data
make generate.openapi  # Generate API types
```

### Environment Variables
```bash
DB_SOURCE=postgresql://user:pass@localhost:5432/dbname
JWT_SECRET=your-secret-key
BCRYPT_COST=10
```

### Assumptions Made
- Single currency (IDR) business requirement
- Manual approval threshold at 1M IDR is business rule
- Payment gateway provides external_id for tracking
- Users belong to single role no multi-role complexity
