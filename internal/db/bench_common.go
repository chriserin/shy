package db

import (
	"github.com/chris/shy/internal/summary"
	"github.com/chris/shy/pkg/models"
)

// DatabaseInterface defines the common interface for both DB and ZDB
type DatabaseInterface interface {
	Close() error
	Path() string
	InsertCommand(cmd *models.Command) (int64, error)
	GetCommand(id int64) (*models.Command, error)
	CountCommands() (int, error)
	GetCommandsByDateRange(startTime, endTime int64, sourceApp *string) ([]models.Command, error)
	TableExists() (bool, error)
	GetTableSchema() ([]map[string]interface{}, error)
	ListCommands(limit int, sourceApp string, sourcePid int64) ([]models.Command, error)
	ListCommandsInRange(startTime, endTime int64, limit int, sourceApp string, sourcePid int64) ([]models.Command, error)
	GetRecentCommandsWithoutConsecutiveDuplicates(offset int, sourceApp string, sourcePid int64, workingDir string) (*models.Command, error)
	GetMostRecentEventID() (int64, error)
	GetCommandsByRange(first, last int64) ([]models.Command, error)
	GetCommandsByRangeWithPattern(first, last int64, pattern string) ([]models.Command, error)
	FindMostRecentMatching(prefix string) (int64, error)
	FindMostRecentMatchingBefore(prefix string, beforeID int64) (int64, error)
	GetCommandsByRangeInternal(first, last, sessionPid int64) ([]models.Command, error)
	GetCommandsByRangeWithPatternInternal(first, last, sessionPid int64, pattern string) ([]models.Command, error)
	CloseSession(sessionPid int64) (int64, error)
	LikeRecent(opts LikeRecentOptions) ([]string, error)
	LikeRecentAfter(opts LikeRecentAfterOptions) ([]string, error)
	GetCommandsForFzf(fn func(id int64, cmdText string) error) error
	GetCommandWithContext(id int64, contextSize int) ([]models.Command, *models.Command, []models.Command, error)
	GetContextSummary(startTime, endTime int64) ([]summary.ContextSummary, error)
}

// NewDatabase creates a new database using the implementation specified by DB_IMPL environment variable
// DB_IMPL=zombiezen uses ZDB (zombiezen.com/go/sqlite)
// DB_IMPL=modernc or unset uses DB (modernc.org/sqlite)
func NewDatabase(dbPath string) (DatabaseInterface, error) {
	return New(dbPath)
}

// NewDatabaseReadOnly opens an existing database in read-only mode (skips table creation).
// Use this for benchmarks and read-only operations on existing databases.
func NewDatabaseReadOnly(dbPath string) (DatabaseInterface, error) {
	return NewWithOptions(dbPath, Options{ReadOnly: true})
}
