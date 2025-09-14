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
- **Event-driven payment initiation**: Payments triggered by ExpenseApprovedEvent
- **Async processing**: Payment API calls happen in background
- **Automatic status updates**: PaymentCompletedEvent updates expense status to "completed"
- **External payment gateway integration** with retry capability
- **Payment status tracking** (pending → processing → success/failed)
- **Failed payments can be retried** by authorized users 

### Permission System
- **User**: Submit and view own expenses
- **Approver**: Approve/reject expenses, retry failed payments
- **Admin**: Full system access

## API Examples
API doc available in Swagger:
**http://localhost:8080/swagger/index.html**

### Key API Features
- **Pagination with total count**: All list endpoints include `total_data` for frontend pagination
- **Advanced filtering**: Search, category, status, and sorting options
- **Event-driven responses**: Operations return immediately while processing continues async

## Architecture
This app following approach domain driven where each module isolate in his own domain

**Why Event Bus?**
- **Decoupling**: Services communicate through events rather than direct calls, avoid tight coupling
- **Scalability**: Async processing prevents blocking operations from affecting user experience
- **Reliability**: Event-driven approach handles failures gracefully with potential for retry mechanisms

**Event Flow**:
```
1. Expense Created/Approved → ExpenseApprovedEvent
2. Payment Service processes event async
3. Payment Completed → PaymentCompletedEvent  
4. Expense Service updates status to "completed"
```

**Benefits**:
- HTTP requests return immediately
- Payment processing happens in background
- Failed payments don't affect expense creation/approval
- Natural separation of payment and expense concerns

### Directory Structure
```
├── cmd/                    # Application commands
├── internal/
│   ├── auth/              # Authentication & authorization
│   ├── expense/           # Core expense business logic
│   ├── payment/           # Payment processing
│   ├── user/              # User management
│   ├── core/
│   │   ├── events/        # Event bus & event definitions
│   │   └── datamodel/     # Shared data models
│   └── transport/         # HTTP layer & middleware
├── db/migrations/         # Database migrations
└── api/                   # OpenAPI specification
```

### Event System Components

**Event Bus** (`internal/core/events/bus.go`):
- In-memory event publisher/subscriber
- Goroutine-based async event processing
- Error handling and logging for event failures

**Event Types** (`internal/core/events/payment_events.go`):
- `ExpenseApprovedEvent`: Triggered when expense needs payment processing
- `PaymentCompletedEvent`: Triggered when payment gateway confirms success

**Event Handlers**:
- Expense Service: Handles payment completion to update expense status
- Payment Service: Handles expense approval to initiate payments

### Key Design Decisions

**Event-Driven Payment Processing**: 
- Prevents blocking HTTP requests during payment API calls
- Enables payment retries without affecting expense approval workflow
- Separates payment concerns from expense management

**Layered Architecture**: Handler => Service => Repository pattern for clear separation of concerns and testability.

**Permission Middleware**: Middleware and service-level permission checks with RBAC.

**Async Background Processing**: payments run asynchronously



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
