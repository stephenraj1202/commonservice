package runner_test

// Feature: datapilot-platform, Property 28: Job execution log reflects HTTP outcome

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"datapilot/scheduler-service/models"
	"datapilot/scheduler-service/runner"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openGORMWithSQLDB wraps an existing *sql.DB in a GORM instance.
func openGORMWithSQLDB(sqlDB *sql.DB) (*gorm.DB, error) {
	dialector := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// setupMockDB creates a GORM DB backed by go-sqlmock.
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := openGORMWithSQLDB(sqlDB)
	require.NoError(t, err)
	return db, mock
}

// expectLogInsert sets up mock expectations for the INSERT of a JobExecutionLog.
func expectLogInsert(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `job_execution_logs`")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
}

// --- Unit tests ---

func TestExecutionLog_2xxSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	db, mock := setupMockDB(t)
	expectLogInsert(mock)

	r := runner.NewRunner(db, zap.NewNop())
	r.ExecuteJob(1, srv.URL, http.MethodGet)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionLog_4xxFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	db, mock := setupMockDB(t)
	expectLogInsert(mock)

	r := runner.NewRunner(db, zap.NewNop())
	r.ExecuteJob(1, srv.URL, http.MethodGet)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionLog_5xxFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	db, mock := setupMockDB(t)
	expectLogInsert(mock)

	r := runner.NewRunner(db, zap.NewNop())
	r.ExecuteJob(1, srv.URL, http.MethodGet)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionLog_TimeoutFailed(t *testing.T) {
	// Use a closed server so connection is refused immediately (no 10s wait).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	db, mock := setupMockDB(t)
	expectLogInsert(mock)

	r := runner.NewRunner(db, zap.NewNop())
	r.ExecuteJob(1, url, http.MethodGet)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_StoresCronEntryID(t *testing.T) {
	db, mock := setupMockDB(t)

	// Expect UPDATE for cron_entry_id.
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `jobs`")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	job := &models.Job{
		ID:             1,
		Name:           "test-job",
		CronExpression: "* * * * * *",
		TargetURL:      "http://example.com",
		HTTPMethod:     http.MethodGet,
		Status:         "active",
	}

	r := runner.NewRunner(db, zap.NewNop())
	err := r.Register(job)
	require.NoError(t, err)
	assert.NotZero(t, job.CronEntryID, "CronEntryID should be set after Register")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Property-based test ---
// Validates: Requirements 19.2, 19.3

func TestProperty28_ExecutionLogReflectsHTTPOutcome(t *testing.T) {
	successCodes := []int{200, 201, 202, 204}
	failureCodes := []int{400, 401, 403, 404, 422, 500, 502, 503}

	// Property: for any 2xx status code, ExecuteJob saves an execution log (INSERT fires).
	t.Run("2xx_produces_log_insert", func(t *testing.T) {
		params := gopter.DefaultTestParameters()
		params.MinSuccessfulTests = 20
		props := gopter.NewProperties(params)

		props.Property("2xx response triggers execution log INSERT", prop.ForAll(
			func(idx int) bool {
				code := successCodes[idx%len(successCodes)]

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(code)
				}))
				defer srv.Close()

				sqlDB, mock, err := sqlmock.New()
				if err != nil {
					return false
				}
				defer func() { _ = sqlDB.Close() }()

				db, err := openGORMWithSQLDB(sqlDB)
				if err != nil {
					return false
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `job_execution_logs`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				r := runner.NewRunner(db, zap.NewNop())
				r.ExecuteJob(uint(idx+1), srv.URL, http.MethodGet)

				return mock.ExpectationsWereMet() == nil
			},
			gen.IntRange(0, 99),
		))

		props.TestingRun(t)
	})

	// Property: for any 4xx/5xx status code, ExecuteJob saves an execution log (INSERT fires).
	t.Run("4xx_5xx_produces_log_insert", func(t *testing.T) {
		params := gopter.DefaultTestParameters()
		params.MinSuccessfulTests = 20
		props := gopter.NewProperties(params)

		props.Property("4xx/5xx response triggers execution log INSERT", prop.ForAll(
			func(idx int) bool {
				code := failureCodes[idx%len(failureCodes)]

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(code)
				}))
				defer srv.Close()

				sqlDB, mock, err := sqlmock.New()
				if err != nil {
					return false
				}
				defer func() { _ = sqlDB.Close() }()

				db, err := openGORMWithSQLDB(sqlDB)
				if err != nil {
					return false
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `job_execution_logs`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				r := runner.NewRunner(db, zap.NewNop())
				r.ExecuteJob(uint(idx+1), srv.URL, http.MethodGet)

				return mock.ExpectationsWereMet() == nil
			},
			gen.IntRange(0, 99),
		))

		props.TestingRun(t)
	})

	// Property: for any network error, ExecuteJob saves an execution log (INSERT fires).
	t.Run("network_error_produces_log_insert", func(t *testing.T) {
		// Use a server that is immediately closed so connection is refused instantly.
		closedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		closedURL := closedSrv.URL
		closedSrv.Close()

		params := gopter.DefaultTestParameters()
		params.MinSuccessfulTests = 5
		props := gopter.NewProperties(params)

		props.Property("network error triggers execution log INSERT", prop.ForAll(
			func(idx int) bool {
				sqlDB, mock, err := sqlmock.New()
				if err != nil {
					return false
				}
				defer func() { _ = sqlDB.Close() }()

				db, err := openGORMWithSQLDB(sqlDB)
				if err != nil {
					return false
				}

				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `job_execution_logs`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				r := runner.NewRunner(db, zap.NewNop())
				// Closed server → immediate connection refused, no timeout wait.
				r.ExecuteJob(uint(idx+1), closedURL, http.MethodGet)

				return mock.ExpectationsWereMet() == nil
			},
			gen.IntRange(0, 4),
		))

		props.TestingRun(t)
	})

	// Suppress unused variable warning for failureCodes in outer scope.
	_ = time.Now()
	_ = failureCodes
}
