package db

import "testing"

// NewTestDB is testDB for the external db_test package: a fresh, migrated
// schema in TEST_DATABASE_URL, or a skip.
func NewTestDB(t *testing.T) *DB { return testDB(t) }
