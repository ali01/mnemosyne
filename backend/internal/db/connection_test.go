package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB_InvalidConnection(t *testing.T) {
	cfg := Config{
		Host:     "invalid-host",
		Port:     5432,
		User:     "test",
		Password: "test",
		DBName:   "test",
		SSLMode:  "disable",
	}

	db, err := NewDB(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestDB_Transaction_Success(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO test").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	var executed bool
	err = db.Transaction(func(tx *sqlx.Tx) error {
		executed = true
		_, execErr := tx.Exec("INSERT INTO test VALUES (?)", 1)
		return execErr
	})

	assert.NoError(t, err)
	assert.True(t, executed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Transaction_Rollback(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO test").WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	testErr := errors.New("test error")
	err = db.Transaction(func(tx *sqlx.Tx) error {
		_, execErr := tx.Exec("INSERT INTO test VALUES (?)", 1)
		if execErr != nil {
			return testErr
		}
		return nil
	})

	assert.Equal(t, testErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Transaction_Panic(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin()
	mock.ExpectRollback()

	assert.Panics(t, func() {
		_ = db.Transaction(func(_ *sqlx.Tx) error {
			panic("test panic")
		})
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Transaction_BeginError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err = db.Transaction(func(_ *sqlx.Tx) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ExecuteSchema(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	schemaSQL := `
	CREATE TABLE IF NOT EXISTS test (
		id INT PRIMARY KEY,
		name VARCHAR(255)
	);
	`

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = db.ExecuteSchema(schemaSQL)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_ExecuteSchema_Error(t *testing.T) {
	// Test schema execution failure
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	schemaSQL := `CREATE TABLE invalid_syntax`

	mock.ExpectExec("CREATE TABLE invalid_syntax").
		WillReturnError(errors.New("syntax error"))

	err = db.ExecuteSchema(schemaSQL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute schema")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_GetNodeCount(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	rows := sqlmock.NewRows([]string{"count"}).AddRow(42)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM nodes").
		WillReturnRows(rows)

	count, err := db.GetNodeCount()
	assert.NoError(t, err)
	assert.Equal(t, 42, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_GetNodeCount_QueryError(t *testing.T) {
	// Test query failure
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM nodes").
		WillReturnError(sql.ErrNoRows)

	count, err := db.GetNodeCount()
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_GetNodeCount_ScanError(t *testing.T) {
	// Test scanning failure with invalid data type
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	rows := sqlmock.NewRows([]string{"count"}).AddRow("not-a-number")
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM nodes").
		WillReturnRows(rows)

	count, err := db.GetNodeCount()
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_GetEdgeCount(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	rows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM edges").
		WillReturnRows(rows)

	count, err := db.GetEdgeCount()
	assert.NoError(t, err)
	assert.Equal(t, 100, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_GetEdgeCount_Error(t *testing.T) {
	// Test edge count query failure
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM edges").
		WillReturnError(errors.New("table does not exist"))

	count, err := db.GetEdgeCount()
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Close(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	// Expect the close call
	mock.ExpectClose()

	err = db.Close()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Transaction_RollbackError(t *testing.T) {
	// Test when both transaction and rollback fail
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO test").WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback().WillReturnError(errors.New("rollback failed"))

	transactionErr := errors.New("transaction error")
	err = db.Transaction(func(tx *sqlx.Tx) error {
		_, execErr := tx.Exec("INSERT INTO test VALUES (?)", 1)
		if execErr != nil {
			return transactionErr
		}
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction failed")
	assert.Contains(t, err.Error(), "rollback failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Transaction_CommitError(t *testing.T) {
	// Test when commit fails
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO test").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err = db.Transaction(func(tx *sqlx.Tx) error {
		_, execErr := tx.Exec("INSERT INTO test VALUES (?)", 1)
		return execErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDB_Transaction_NestedError(t *testing.T) {
	// Test transaction with multiple operations where second fails
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{DB: sqlx.NewDb(mockDB, "postgres")}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO logs").WillReturnError(errors.New("constraint violation"))
	mock.ExpectRollback()

	err = db.Transaction(func(tx *sqlx.Tx) error {
		// First operation succeeds
		_, execErr := tx.Exec("INSERT INTO users VALUES (?)", "user1")
		if execErr != nil {
			return execErr
		}

		// Second operation fails
		_, execErr = tx.Exec("INSERT INTO logs VALUES (?)", "log1")
		return execErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "constraint violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}
