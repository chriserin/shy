package db

// NewDatabase creates a new database using the implementation specified by DB_IMPL environment variable
// DB_IMPL=zombiezen uses ZDB (zombiezen.com/go/sqlite)
// DB_IMPL=modernc or unset uses DB (modernc.org/sqlite)
func NewDatabase(dbPath string) (*DB, error) {
	return New(dbPath)
}

// NewDatabaseReadOnly opens an existing database without schema validation.
// Use this for benchmarks and read-only operations on existing databases.
func NewDatabaseReadOnly(dbPath string) (*DB, error) {
	return NewWithOptions(dbPath, Options{SkipSchemaCheck: true})
}
