package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkDBSize defines a database size for benchmarking
type BenchmarkDBSize struct {
	Name string
	File string
}

// BenchmarkDBSizes contains all available database sizes for benchmarking
var BenchmarkDBSizes = []BenchmarkDBSize{
	{"medium", "test-sqlite-history-medium.db"},
	{"large", "test-sqlite-history-large.db"},
	{"xlarge", "test-sqlite-history-xlarge.db"},
	{"xl-2", "test-sqlite-history-xl-2.db"},
	{"xl-3", "test-sqlite-history-xl-3.db"},
	{"xl-4", "test-sqlite-history-xl-4.db"},
	{"xl-5", "test-sqlite-history-xl-5.db"},
}

// BenchmarkLikeRecentWithFilters measures prefix search with pwd/session filters
func BenchmarkLikeRecentWithFilters(b *testing.B) {
	fmt.Println("DBTYPE", DbType())

	for _, size := range BenchmarkDBSizes {
		dbPath := filepath.Join("../../testdata/perf", size.File)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.Name)
			continue
		}

		for _, pid := range []int64{12347, 12345} {
			for _, prefix := range []string{"g", "git "} {
				b.Run(fmt.Sprintf("%s/prefix-%s-%d", size.Name, prefix, pid), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						database := OpenDB(b, dbPath)
						defer database.Close()
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
	offsets := []int{0, 19, 49} // 0-indexed: 0=1st, 19=20th, 49=50th

	for _, size := range BenchmarkDBSizes {
		dbPath := filepath.Join("../../testdata/perf", size.File)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.Name)
			continue
		}

		for _, offset := range offsets {
			b.Run(fmt.Sprintf("%s/offset-%d", size.Name, offset), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					database := OpenDB(b, dbPath)
					defer database.Close()
					_, err := database.GetRecentCommandsWithoutConsecutiveDuplicates(offset, "zsh", 12345, "/home/user/projects/shy")
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
	limits := []int{20, 100, 500}

	for _, size := range BenchmarkDBSizes {
		dbPath := filepath.Join("../../testdata/perf", size.File)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.Name)
			continue
		}

		for _, limit := range limits {
			b.Run(fmt.Sprintf("%s/limit-%d", size.Name, limit), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					database := OpenDB(b, dbPath)
					defer database.Close()
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
	for _, size := range BenchmarkDBSizes {
		dbPath := filepath.Join("../../testdata/perf", size.File)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.Name)
			continue
		}

		b.Run(size.Name, func(b *testing.B) {
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
	ranges := []struct {
		name  string
		first int64
		last  int64
	}{
		{"last-10", -10, -1},
		{"last-50", -50, -1},
		{"range-100", 1, 100},
	}

	for _, size := range BenchmarkDBSizes {
		dbPath := filepath.Join("../../testdata/perf", size.File)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.Name)
			continue
		}

		for _, r := range ranges {
			b.Run(fmt.Sprintf("%s/%s", size.Name, r.name), func(b *testing.B) {
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

// BenchmarkGetAllHistory measures "shy history" / "shy fc -l -1 0" performance (all history)
func BenchmarkGetAllHistory(b *testing.B) {
	for _, size := range BenchmarkDBSizes {
		dbPath := filepath.Join("../../testdata/perf", size.File)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.Name)
			continue
		}

		b.Run(size.Name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				database := OpenDB(b, dbPath)
				// -1 means first event (1), 0 means most recent event
				// This fetches all history like "shy fc -l -1 0" or "shy history"
				mostRecent, err := database.GetMostRecentEventID()
				if err != nil {
					b.Fatalf("failed to get most recent event id: %v", err)
				}

				_, err = database.GetCommandsByRange(1, mostRecent)
				if err != nil {
					b.Fatalf("failed to get commands: %v", err)
				}
				database.Close()
			}
		})
	}
}
