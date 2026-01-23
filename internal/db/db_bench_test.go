package db

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/chris/shy/pkg/models"
)

// BenchmarkInsertCommand measures single command insertion performance
func BenchmarkInsertCommand(b *testing.B) {
	sizes := []struct {
		name string
		db   string
	}{
		{"medium", "history-medium.db"},
		{"large", "history-large.db"},
		{"xlarge", "history-xlarge.db"},
	}

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", size.db)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found (run scripts/create-test-databases.sh)", size.name)
			continue
		}

		b.Run(size.name, func(b *testing.B) {
			database, err := NewDatabase(dbPath)
			if err != nil {
				b.Fatalf("failed to open database: %v", err)
			}
			defer database.Close()

			// Create sample commands for insertion
			commands := make([]*models.Command, b.N)
			for i := 0; i < b.N; i++ {
				sourceApp := "zsh"
				sourcePid := int64(99999)
				sourceActive := true
				cmd := &models.Command{
					CommandText:  fmt.Sprintf("echo 'benchmark test %d'", i),
					WorkingDir:   "/home/user/test",
					ExitStatus:   0,
					Timestamp:    1704470400 + int64(i),
					SourceApp:    &sourceApp,
					SourcePid:    &sourcePid,
					SourceActive: &sourceActive,
				}
				commands[i] = cmd
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := database.InsertCommand(commands[i])
				if err != nil {
					b.Fatalf("failed to insert command: %v", err)
				}
			}
		})
	}
}

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

	for _, size := range sizes {
		dbPath := filepath.Join("../../testdata/perf", size.db)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		database, err := NewDatabase(dbPath)
		if err != nil {
			b.Fatalf("failed to open database: %v", err)
		}
		defer database.Close()

		for _, pid := range []int64{12347, 12345} {
			for _, prefix := range []string{"g", "gi", "git", "git "} {
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
		dbPath := filepath.Join("../../testdata/perf", size.db)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		for _, limit := range limits {
			b.Run(fmt.Sprintf("%s/limit-%d", size.name, limit), func(b *testing.B) {
				database, err := NewDatabase(dbPath)
				if err != nil {
					b.Fatalf("failed to open database: %v", err)
				}
				defer database.Close()

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
		dbPath := filepath.Join("../../testdata/perf", size.db)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		for _, limit := range limits {
			b.Run(fmt.Sprintf("%s/limit-%d", size.name, limit), func(b *testing.B) {
				database, err := NewDatabase(dbPath)
				if err != nil {
					b.Fatalf("failed to open database: %v", err)
				}
				defer database.Close()

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

// BenchmarkGetCommandsForFzf measures fzf data source performance
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
		dbPath := filepath.Join("../../testdata/perf", size.db)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		b.Run(size.name, func(b *testing.B) {
			database, err := NewDatabase(dbPath)
			if err != nil {
				b.Fatalf("failed to open database: %v", err)
			}
			defer database.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := database.GetCommandsForFzf(func(id int64, cmdText string) error { return nil })
				if err != nil {
					b.Fatalf("failed to get commands: %v", err)
				}
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
		dbPath := filepath.Join("../../testdata/perf", size.db)
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			b.Logf("Skipping %s: database not found", size.name)
			continue
		}

		database, err := NewDatabase(dbPath)
		if err != nil {
			b.Fatalf("failed to open database: %v", err)
		}
		defer database.Close()

		for _, r := range ranges {
			b.Run(fmt.Sprintf("%s/%s", size.name, r.name), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := database.GetCommandsByRange(r.first, r.last)
					if err != nil {
						b.Fatalf("failed to get commands: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkConcurrentInserts simulates multiple shell sessions inserting concurrently
func BenchmarkConcurrentInserts(b *testing.B) {
	// Use a temporary database for this test
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent.db")

	database, err := NewDatabase(dbPath)
	if err != nil {
		b.Fatalf("failed to create database: %v", err)
	}
	database.Close()

	concurrencyLevels := []int{1, 2, 4}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				// Each goroutine opens its own connection
				db, err := NewDatabase(dbPath)
				if err != nil {
					b.Fatalf("failed to open database: %v", err)
				}
				defer db.Close()

				i := 0
				for pb.Next() {
					sourceApp := "zsh"
					sourcePid := int64(rand.Intn(5) + 10000)
					sourceActive := true
					cmd := &models.Command{
						CommandText:  fmt.Sprintf("concurrent test %d", i),
						WorkingDir:   "/home/user/test",
						ExitStatus:   0,
						Timestamp:    1704470400 + int64(i),
						SourceApp:    &sourceApp,
						SourcePid:    &sourcePid,
						SourceActive: &sourceActive,
					}
					_, err := db.InsertCommand(cmd)
					if err != nil {
						b.Fatalf("failed to insert: %v", err)
					}
					i++
				}
			})
		})
	}
}
