package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkLikeRecentWithFilters measures prefix search with pwd/session filters
func BenchmarkLikeRecentWithFilters(b *testing.B) {
	sizes := []struct {
		name string
		db   string
	}{
		{"medium", "history-medium.db"},
		{"large", "history-large.db"},
		{"xlarge", "history-xlarge.db"},
	}

	fmt.Println("DBTYPE", DbType())

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", fmt.Sprintf("norm-%s-%s", DbType(), size.db))
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		database := OpenDB(b, dbPath)
		defer database.Close()

		for _, pid := range []int64{12347, 12345} {
			for _, prefix := range []string{"g", "git "} {
				b.Run(fmt.Sprintf("%s/prefix-%s-%d", size.name, prefix, pid), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err := database.LikeRecent(LikeRecentOptions{
							Prefix:     prefix,
							WorkingDir: "/home/user/projects/shy",
							SourceApp:  "zsh",
							SourcePid:  pid,
						})
						if err != nil {
							b.Fatalf("failed to search: %v", err)
						}
					}
				})
			}
		}
	}
}

func OpenDB(b *testing.B, dbPath string) DatabaseInterface {
	database, err := NewDatabaseReadOnly(dbPath)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	return database
}

// BenchmarkGetRecentCommandsWithSession measures last-command with session filter
func BenchmarkGetRecentCommandsWithSession(b *testing.B) {
	sizes := []struct {
		name string
		db   string
	}{
		{"medium", "history-medium.db"},
		{"large", "history-large.db"},
		{"xlarge", "history-xlarge.db"},
	}

	limits := []int{1, 20, 50}

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", fmt.Sprintf("norm-%s-%s", DbType(), size.db))
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		database := OpenDB(b, dbPath)
		defer database.Close()
		for _, limit := range limits {
			b.Run(fmt.Sprintf("%s/limit-%d", size.name, limit), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := database.GetRecentCommandsWithoutConsecutiveDuplicates(limit, "zsh", 12345, "/home/user/projects/shy")
					if err != nil {
						b.Fatalf("failed to get commands: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkListCommands measures list command performance
func BenchmarkListCommands(b *testing.B) {
	sizes := []struct {
		name string
		db   string
	}{
		{"medium", "history-medium.db"},
		{"large", "history-large.db"},
		{"xlarge", "history-xlarge.db"},
	}

	limits := []int{20, 100, 500}

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", fmt.Sprintf("norm-%s-%s", DbType(), size.db))
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		database := OpenDB(b, dbPath)
		defer database.Close()

		for _, limit := range limits {
			b.Run(fmt.Sprintf("%s/limit-%d", size.name, limit), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := database.ListCommands(limit, "", 0)
					if err != nil {
						b.Fatalf("failed to list commands: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkGetCommandsForFzf measures fzf data source performance with different deduplication strategies
func BenchmarkGetCommandsForFzf(b *testing.B) {
	sizes := []struct {
		name string
		db   string
	}{
		{"medium", "history-medium.db"},
		{"large", "history-large.db"},
		{"xlarge", "history-xlarge.db"},
	}

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", fmt.Sprintf("norm-%s-%s", DbType(), size.db))
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				database := OpenDB(b, dbPath)
				err := database.GetCommandsForFzf(func(id int64, cmdText string) error { return nil })
				if err != nil {
					b.Fatalf("failed to get commands: %v", err)
				}
				database.Close()
			}
		})
	}
}

// BenchmarkGetCommandsByRange measures fc command performance
func BenchmarkGetCommandsByRange(b *testing.B) {
	sizes := []struct {
		name string
		db   string
	}{
		{"medium", "history-medium.db"},
		{"large", "history-large.db"},
		{"xlarge", "history-xlarge.db"},
	}

	ranges := []struct {
		name  string
		first int64
		last  int64
	}{
		{"last-10", -10, -1},
		{"last-50", -50, -1},
		{"range-100", 1, 100},
	}

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", fmt.Sprintf("norm-%s-%s", DbType(), size.db))
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		for _, r := range ranges {
			b.Run(fmt.Sprintf("%s/%s", size.name, r.name), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					database := OpenDB(b, dbPath)
					_, err := database.GetCommandsByRange(r.first, r.last)
					if err != nil {
						b.Fatalf("failed to get commands: %v", err)
					}
					database.Close()
				}
			})
		}
	}
}
