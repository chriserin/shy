package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

var generateTestdataCmd = &cobra.Command{
	Use:   "generate-testdata",
	Short: "Generate test databases for performance testing",
	Long:  "Efficiently generate test databases with realistic command history for benchmarking",
	RunE:  runGenerateTestdata,
}

func init() {
	rootCmd.AddCommand(generateTestdataCmd)
}

func runGenerateTestdata(cmd *cobra.Command, args []string) error {
	// Get project root (assuming we're running from project directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	testDataDir := filepath.Join(projectRoot, "testdata", "perf")
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create testdata directory: %w", err)
	}

	fmt.Println("Generating test databases for performance testing...")
	fmt.Println()

	// Generate databases
	databases := []struct {
		name string
		size int
	}{
		{"medium", 10000},
		{"large", 100000},
		// Uncomment for xlarge (takes a few minutes)
		// {"xlarge", 1000000},
	}

	for _, dbConfig := range databases {
		dbPath := filepath.Join(testDataDir, fmt.Sprintf("history-%s.db", dbConfig.name))

		// Remove existing database
		os.Remove(dbPath)

		start := time.Now()
		fmt.Printf("Generating %s database (%d commands)...\n", dbConfig.name, dbConfig.size)

		if err := generateDatabase(dbPath, dbConfig.size); err != nil {
			return fmt.Errorf("failed to generate %s database: %w", dbConfig.name, err)
		}

		elapsed := time.Since(start)

		// Get database size
		info, err := os.Stat(dbPath)
		if err != nil {
			return fmt.Errorf("failed to stat database: %w", err)
		}
		sizeInMB := float64(info.Size()) / 1024 / 1024

		fmt.Printf("✓ Created %s (%.2f MB) in %s\n\n", dbPath, sizeInMB, elapsed)
	}

	fmt.Println("✓ All test databases created in", testDataDir)
	return nil
}

func generateDatabase(dbPath string, size int) error {
	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer database.Close()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Sample data
	commands := []string{
		"ls -la",
		"cd /home/user/projects",
		"git status",
		"git add .",
		"git commit -m 'update'",
		"git push",
		"git pull",
		"git log",
		"git diff",
		"git branch",
		"npm install",
		"npm test",
		"npm run build",
		"npm start",
		"go build",
		"go test ./...",
		"go mod tidy",
		"docker ps",
		"docker-compose up -d",
		"docker-compose down",
		"kubectl get pods",
		"kubectl logs",
		"ssh user@server",
		"vim README.md",
		"cat file.txt",
		"grep -r 'pattern' .",
		"find . -name '*.go'",
		"make build",
		"cargo build",
		"cargo test",
		"python script.py",
		"pytest",
		"curl https://api.example.com",
		"wget https://example.com/file",
		"tar -xzf archive.tar.gz",
		"rsync -av src/ dest/",
		"systemctl status nginx",
		"journalctl -u service",
		"top",
		"htop",
		"df -h",
		"du -sh *",
		"ps aux",
		"netstat -tuln",
		"echo 'hello world'",
		"mkdir -p dir",
		"rm -rf temp",
		"cp file1 file2",
		"mv file1 file2",
		"touch newfile",
	}

	dirs := []string{
		"/home/user/projects/shy",
		"/home/user/projects/webapp",
		"/home/user/projects/api",
		"/home/user/projects/frontend",
		"/home/user/documents",
		"/home/user",
		"/tmp",
	}

	gitRepos := []string{
		"github.com/user/shy",
		"github.com/user/webapp",
		"github.com/user/api",
		"github.com/company/project",
		"",
	}

	gitBranches := []string{
		"main",
		"develop",
		"feature/new-feature",
		"feature/update",
		"bugfix/issue-123",
		"",
	}

	sourceApps := []string{"zsh", "bash"}
	sourcePids := []int64{12345, 67890, 11111, 22222, 33333}

	// Base timestamp (Jan 1, 2024)
	baseTs := int64(1704067200)

	// Progress reporting interval
	progressInterval := 10000

	for i := 0; i < size; i++ {
		// Pick random values
		cmdText := commands[rand.Intn(len(commands))]
		dir := dirs[rand.Intn(len(dirs))]
		repo := gitRepos[rand.Intn(len(gitRepos))]
		branch := gitBranches[rand.Intn(len(gitBranches))]
		app := sourceApps[rand.Intn(len(sourceApps))]
		pid := sourcePids[rand.Intn(len(sourcePids))]
		status := 0
		if rand.Intn(10) == 0 { // 10% failure rate
			status = 1
		}

		// Increment timestamp (1-60 seconds between commands)
		ts := baseTs + int64(i)*(int64(rand.Intn(60))+1)

		// Random duration (100ms to 10s)
		duration := int64(rand.Intn(9900) + 100)

		active := true

		// Build command
		command := &models.Command{
			CommandText:  cmdText,
			WorkingDir:   dir,
			ExitStatus:   status,
			Timestamp:    ts,
			Duration:     &duration,
			SourceApp:    &app,
			SourcePid:    &pid,
			SourceActive: &active,
		}

		if repo != "" {
			command.GitRepo = &repo
		}
		if branch != "" {
			command.GitBranch = &branch
		}

		// Insert command
		if _, err := database.InsertCommand(command); err != nil {
			return fmt.Errorf("failed to insert command %d: %w", i, err)
		}

		// Progress indicator
		if (i+1)%progressInterval == 0 {
			fmt.Printf("  Inserted %d / %d commands...\n", i+1, size)
		}
	}

	return nil
}
