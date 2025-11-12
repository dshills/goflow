# Execution History Query Tests Summary

## Overview

Created comprehensive unit tests for execution history queries in `/Users/dshills/Development/projects/goflow/tests/unit/execution/history_test.go`. These tests follow TDD principles and **will initially fail** until the corresponding `List()` method is implemented in the SQLite repository.

## Test Coverage

### 1. Pagination Tests (`TestHistoryListWithPagination`)

**Purpose**: Verify execution listing supports proper pagination for large result sets.

**Test Cases**:
- First page retrieval (10 items, offset 0)
- Second page retrieval (10 items, offset 10)
- Partial page (remaining 5 items, offset 20)
- Offset beyond total count (should return empty)
- Large limit returning all results
- Zero limit behavior (default handling)

**Key Assertions**:
- Correct result count per page
- Accurate total count metadata
- Results ordered by `started_at DESC` (most recent first)
- Correct first and last execution IDs on each page

**Performance Target**: < 50ms for paginated queries on 1000+ executions

---

### 2. Status Filtering Tests (`TestHistoryFilterByStatus`)

**Purpose**: Test filtering executions by their current status.

**Test Cases**:
- Filter completed executions (10 expected)
- Filter failed executions (5 expected)
- Filter running executions (3 expected)
- Filter cancelled executions (2 expected)
- Filter pending executions (5 expected)

**Data Setup**: Creates 25 executions with known status distribution

**Key Assertions**:
- Correct count for each status
- All returned executions have the filtered status
- No cross-contamination between status filters

**Performance Target**: < 100ms with indexed status column

---

### 3. Workflow Filtering Tests (`TestHistoryFilterByWorkflowID`)

**Purpose**: Test filtering executions by workflow identifier.

**Test Cases**:
- Filter by workflow-auth (7 executions)
- Filter by workflow-payment (12 executions)
- Filter by workflow-notification (5 executions)
- Filter by non-existent workflow (0 results)

**Data Setup**: Creates executions across 3 different workflows

**Key Assertions**:
- Correct count per workflow
- All results have matching workflow ID
- Empty result handling for non-existent workflows

**Performance Target**: < 100ms with indexed workflow_id column

---

### 4. Date Range Filtering Tests (`TestHistoryFilterByDateRange`)

**Purpose**: Test filtering executions by start time ranges.

**Test Cases**:
- Last 7 days filter
- Custom range (days 10-20)
- No time filters (all results)
- Before first execution (empty results)
- After last execution (empty results)
- Only start time (from date to present)
- Only end time (from beginning to date)

**Data Setup**: Creates 30 executions spread across 30 days

**Key Assertions**:
- Correct count in each date range
- Results within specified time boundaries
- Proper handling of nil time filters
- DESC ordering maintained within date ranges

**Performance Target**: < 100ms with indexed started_at column

---

### 5. Workflow Name Search Tests (`TestHistorySearchByWorkflowName`)

**Purpose**: Test case-insensitive substring search on workflow names.

**Test Cases**:
- Search "user" (finds user-authentication, user-registration)
- Search "payment" (finds payment-processing, payment-refund)
- Search "notification" (finds email-notification, sms-notification)
- Search exact match "email-notification"
- Search non-existent term (empty results)

**Data Setup**: Creates executions with semantically grouped workflow names

**Key Assertions**:
- All matching workflows returned
- Minimum expected count verified
- Case-insensitive matching works

**Implementation Note**: Requires SQL `LIKE` with `%` wildcards or full-text search

---

### 6. Execution Detail Tests (`TestHistoryExecutionDetail`)

**Purpose**: Test loading complete execution details including node executions.

**Data Setup**:
- Creates execution with 5 node executions
- Each node has inputs and outputs
- Execution completed successfully with return value

**Key Assertions**:
- All execution fields loaded correctly
- All 5 node executions present
- Node execution details intact (inputs, outputs, timestamps)
- Return value properly deserialized

**Performance Target**: < 50ms including all related data

---

### 7. Concurrent Query Tests (`TestHistoryConcurrentQueries`)

**Purpose**: Verify thread-safety and concurrent access handling.

**Test Setup**:
- 10 concurrent goroutines
- 20 queries per goroutine (200 total)
- Mix of List, Load, and filter queries

**Key Assertions**:
- No race conditions
- No errors from concurrent access
- All queries return valid results

**Implementation Note**: SQLite connection pool configured for single connection (optimal for SQLite)

---

### 8. Performance Tests (`TestHistoryQueryPerformance`)

**Purpose**: Benchmark query performance with realistic dataset sizes.

**Data Setup**: 1000 executions with varied data

**Scenarios Tested**:
- List first page (100 items) - max 50ms
- Deep pagination (offset 900) - max 100ms
- Status filtering - max 100ms
- Workflow filtering - max 100ms
- Single execution load with nodes - max 50ms
- Date range filtering - max 100ms

**Methodology**:
- Warm-up run to eliminate cold start
- 10 measurement runs per scenario
- Average duration calculated
- Skipped in short test mode (`go test -short`)

**Success Criteria**: Average duration under specified thresholds

---

### 9. Combined Filter Tests (`TestHistoryCombinedFilters`)

**Purpose**: Test combining multiple filters in a single query.

**Test Cases**:
- Workflow + Status
- Workflow + Status + Date Range
- Status + Pagination

**Key Assertions**:
- Filters applied in AND logic
- Correct intersection of filter conditions
- Expected result counts

**Implementation Note**: Requires proper SQL WHERE clause construction

---

### 10. Edge Case Tests

#### Empty Results (`TestHistoryEmptyResults`)
- Empty database queries
- Non-existent workflow filtering
- Status filters with no matches

#### Invalid Filters (`TestHistoryInvalidFilters`)
- Negative limit (should error)
- Negative offset (should error)
- End time before start time (should error)

---

## Query Patterns Covered

### 1. **Basic Pagination Pattern**
```go
result, err := repo.List(storage.ListOptions{
    Limit:  10,
    Offset: 20,
})
```

**SQL Pattern**:
```sql
SELECT ... FROM executions
ORDER BY started_at DESC
LIMIT 10 OFFSET 20
```

### 2. **Single Filter Pattern**
```go
status := execution.StatusCompleted
result, err := repo.List(storage.ListOptions{
    Status: &status,
})
```

**SQL Pattern**:
```sql
SELECT ... FROM executions
WHERE status = ?
ORDER BY started_at DESC
```

### 3. **Combined Filter Pattern**
```go
result, err := repo.List(storage.ListOptions{
    WorkflowID:    &workflowID,
    Status:        &status,
    StartedAfter:  &startTime,
    Limit:         20,
})
```

**SQL Pattern**:
```sql
SELECT ... FROM executions
WHERE workflow_id = ?
  AND status = ?
  AND started_at >= ?
ORDER BY started_at DESC
LIMIT 20
```

### 4. **Search Pattern**
```go
searchTerm := "user"
result, err := repo.List(storage.ListOptions{
    WorkflowNameSearch: &searchTerm,
})
```

**SQL Pattern**:
```sql
SELECT ... FROM executions
WHERE workflow_id LIKE '%user%'
ORDER BY started_at DESC
```

### 5. **Count + Paginate Pattern**
```go
result, err := repo.List(storage.ListOptions{
    Limit:  50,
    Offset: 0,
})
// result.TotalCount contains total matching records
// result.Executions contains current page
```

**SQL Pattern**:
```sql
-- Count query
SELECT COUNT(*) FROM executions WHERE ...

-- Data query
SELECT ... FROM executions WHERE ...
ORDER BY started_at DESC
LIMIT 50 OFFSET 0
```

---

## Performance Considerations Identified

### 1. **Index Strategy**

**Required Indexes**:
```sql
CREATE INDEX idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_started_at ON executions(started_at DESC);
CREATE INDEX idx_executions_workflow_status ON executions(workflow_id, status);
```

**Why**:
- Most queries filter by workflow_id, status, or started_at
- Composite index on (workflow_id, status) optimizes common combined queries
- DESC index on started_at matches query order

### 2. **Query Optimization Strategies**

**Two-Query Approach for Pagination**:
1. Count query (without node executions) for `TotalCount`
2. Data query (with pagination) for actual results

**Benefit**: Avoids loading unnecessary data for count calculation

**N+1 Prevention**:
- List queries do NOT load node executions (noted in current implementation)
- Only `Load(id)` loads full details including node executions
- Prevents performance degradation on large result sets

### 3. **Connection Pool Configuration**

**Current Setting**:
```go
db.SetMaxOpenConns(1)  // SQLite single connection
db.SetMaxIdleConns(1)
```

**Why**: SQLite uses file-based locking; single connection avoids lock contention

**Concurrent Query Handling**: Connection pooling queues concurrent requests

### 4. **Query Result Streaming**

**Current Approach**: Load all results into slice
```go
executions := make([]*execution.Execution, 0)
for rows.Next() {
    executions = append(executions, &exec)
}
```

**Future Optimization**: Consider iterator pattern for very large result sets
```go
type ExecutionIterator interface {
    Next() (*execution.Execution, error)
    Close() error
}
```

### 5. **Memory Management**

**Pre-allocation Strategy**:
```go
// If LIMIT is known
executions := make([]*execution.Execution, 0, limit)
```

**Benefit**: Reduces slice reallocation during append

### 6. **Query Plan Analysis**

**SQLite EXPLAIN Support**:
```sql
EXPLAIN QUERY PLAN
SELECT ... FROM executions
WHERE workflow_id = ? AND status = ?
```

**Use for**: Verifying index usage and identifying slow queries

### 7. **Caching Considerations**

**Cache Candidates**:
- Total count for frequently-accessed workflows
- First page of recent executions (most common query)

**Cache Invalidation**: On any Save() operation

**Trade-off**: Added complexity vs. query performance gain

---

## Implementation Requirements

### 1. **Repository Interface Extension**

Add to `pkg/domain/execution/repository.go`:
```go
// List retrieves executions matching the provided filters.
// Results are ordered by StartedAt descending (most recent first).
// Returns ListResult with executions and pagination metadata.
List(options storage.ListOptions) (*storage.ListResult, error)
```

### 2. **SQLite Implementation**

Add to `pkg/storage/sqlite.go`:
```go
func (r *SQLiteExecutionRepository) List(opts storage.ListOptions) (*storage.ListResult, error) {
    // 1. Validate options (negative limit/offset)
    // 2. Build WHERE clause from filters
    // 3. Build COUNT query for TotalCount
    // 4. Execute COUNT query
    // 5. Build SELECT query with filters + pagination
    // 6. Execute SELECT query
    // 7. Scan results into execution slice
    // 8. Return ListResult
}
```

### 3. **Query Builder Pattern**

**Helper Functions**:
```go
func buildWhereClause(opts ListOptions) (string, []interface{})
func buildOrderClause(opts ListOptions) string
func buildPaginationClause(opts ListOptions) string
```

**Benefit**: Separates SQL construction logic, easier to test

### 4. **Error Handling**

**Validation Errors**:
- Negative limit → `fmt.Errorf("limit must be non-negative")`
- Negative offset → `fmt.Errorf("offset must be non-negative")`
- End before start → `fmt.Errorf("end time must be after start time")`

**Query Errors**:
- Wrap with context: `fmt.Errorf("failed to list executions: %w", err)`

### 5. **Testing Database Schema**

**Ensure Indexes Exist**:
Update `pkg/storage/migrations.go` to include all required indexes

**Verify in Tests**:
```go
// In setupTestRepository
rows, _ := repo.db.Query("SELECT name FROM sqlite_master WHERE type='index'")
// Verify expected indexes present
```

---

## Test Execution Guide

### Run All History Tests
```bash
go test -v ./tests/unit/execution -run TestHistory
```

### Run Specific Test
```bash
go test -v ./tests/unit/execution -run TestHistoryListWithPagination
```

### Run Without Performance Tests
```bash
go test -short -v ./tests/unit/execution -run TestHistory
```

### Run With Race Detection
```bash
go test -race ./tests/unit/execution -run TestHistory
```

### Run With Coverage
```bash
go test -cover -coverprofile=coverage.out ./tests/unit/execution
go tool cover -html=coverage.out
```

---

## Success Criteria

### All Tests Pass When:
1. ✅ `List()` method implemented in SQLiteExecutionRepository
2. ✅ ListOptions and ListResult types defined in storage package
3. ✅ Query builder correctly handles all filter combinations
4. ✅ Pagination math is correct (offset + limit)
5. ✅ Date range filtering uses correct SQL operators
6. ✅ Workflow name search is case-insensitive
7. ✅ TotalCount calculated correctly
8. ✅ Results ordered by started_at DESC
9. ✅ All edge cases handled (empty results, invalid filters)
10. ✅ Performance targets met (<100ms for most queries)

---

## Next Steps

1. **Implement List() Method**: Add to `pkg/storage/sqlite.go`
2. **Add Indexes**: Update database migrations
3. **Extend Repository Interface**: Add List to interface definition
4. **Run Tests**: Verify all tests pass
5. **Performance Tuning**: If targets not met, optimize queries
6. **Integration Testing**: Test with real workflows in integration tests

---

## Related Files

- **Test File**: `/Users/dshills/Development/projects/goflow/tests/unit/execution/history_test.go`
- **Query Types**: `/Users/dshills/Development/projects/goflow/pkg/storage/query_types.go`
- **Repository Interface**: `/Users/dshills/Development/projects/goflow/pkg/domain/execution/repository.go`
- **SQLite Implementation**: `/Users/dshills/Development/projects/goflow/pkg/storage/sqlite.go`
- **Migrations**: `/Users/dshills/Development/projects/goflow/pkg/storage/migrations.go`

---

## Design Decisions

### Why Pointer Fields in ListOptions?
- Distinguishes between "not specified" (nil) and "zero value"
- Example: `Limit: 0` might mean "no limit" vs not specified
- Allows optional filters without complex zero-value handling

### Why TotalCount in ListResult?
- Enables pagination UI (show "Page 1 of 10")
- Client knows if more pages exist
- Standard pattern in REST APIs

### Why DESC Ordering?
- Most recent executions typically most relevant
- Matches common use case (view latest runs)
- Consistent with workflow history UX patterns

### Why Skip Node Executions in List?
- Performance: Reduces data transfer for summary views
- Use Case: List shows overview, Load shows detail
- Follows separation of concerns (list vs detail)

---

**Status**: Tests written (failing as expected - TDD). Ready for implementation phase.
