package summary

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFormatTable(t *testing.T) {
	// Get actual home directory for tests
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name        string
		summaries   []ContextSummary
		date        string
		isYesterday bool
		wantStrings []string // Strings that should appear in output
	}{
		{
			name: "multiple contexts sorted by duration",
			summaries: []ContextSummary{
				{
					WorkingDir:   homeDir + "/shy",
					GitBranch:    strPtr("main"),
					CommandCount: 2,
					FirstTime:    parseTimeLocal("2026-01-14 08:00:00"),
					LastTime:     parseTimeLocal("2026-01-14 17:00:00"),
				},
				{
					WorkingDir:   homeDir + "/webapp",
					GitBranch:    strPtr("dev"),
					CommandCount: 2,
					FirstTime:    parseTimeLocal("2026-01-14 10:00:00"),
					LastTime:     parseTimeLocal("2026-01-14 11:30:00"),
				},
			},
			date:        "2026-01-14",
			isYesterday: true,
			wantStrings: []string{
				"Yesterday's Work Summary - 2026-01-14",
				"Directory",
				"Branch",
				"Commands",
				"Time Span",
				"Duration",
				"~/shy",
				"main",
				"08:00 - 17:00",
				"9h",
				"~/webapp",
				"dev",
				"10:00 - 11:30",
				"1h 30m",
				"Total: 4 commands across 2 contexts",
			},
		},
		{
			name: "non-git directory shows dash",
			summaries: []ContextSummary{
				{
					WorkingDir:   homeDir + "/dotfiles",
					GitBranch:    nil,
					CommandCount: 2,
					FirstTime:    parseTimeLocal("2026-01-14 14:10:15"),
					LastTime:     parseTimeLocal("2026-01-14 14:35:22"),
				},
			},
			date:        "2026-01-14",
			isYesterday: true,
			wantStrings: []string{
				"~/dotfiles",
				"-",
				"14:10 - 14:35",
				"25m",
			},
		},
		{
			name:        "empty state",
			summaries:   []ContextSummary{},
			date:        "2026-01-13",
			isYesterday: false,
			wantStrings: []string{
				"Work Summary - 2026-01-13",
				"No commands found for this date.",
			},
		},
		{
			name: "single context",
			summaries: []ContextSummary{
				{
					WorkingDir:   homeDir + "/shy",
					GitBranch:    strPtr("main"),
					CommandCount: 3,
					FirstTime:    parseTimeLocal("2026-01-14 09:15:00"),
					LastTime:     parseTimeLocal("2026-01-14 17:32:00"),
				},
			},
			date:        "2026-01-14",
			isYesterday: true,
			wantStrings: []string{
				"Yesterday's Work Summary - 2026-01-14",
				"~/shy",
				"main",
				"3",
				"09:15 - 17:32",
				"8h 17m",
				"Total: 3 commands across 1 context",
			},
		},
		{
			name: "today not yesterday",
			summaries: []ContextSummary{
				{
					WorkingDir:   homeDir + "/shy",
					GitBranch:    strPtr("main"),
					CommandCount: 1,
					FirstTime:    parseTimeLocal("2026-01-15 10:00:00"),
					LastTime:     parseTimeLocal("2026-01-15 10:00:00"),
				},
			},
			date:        "2026-01-15",
			isYesterday: false,
			wantStrings: []string{
				"Work Summary - 2026-01-15",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatTable(tt.summaries, tt.date, tt.isYesterday)

			for _, want := range tt.wantStrings {
				if !strings.Contains(output, want) {
					t.Errorf("FormatTable() output missing expected string %q\nOutput:\n%s", want, output)
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration int64 // seconds
		want     string
	}{
		{"zero duration", 0, "0s"},
		{"seconds only", 45, "45s"},
		{"minutes only", 45 * 60, "45m"},
		{"one minute", 60, "1m"},
		{"hours and minutes", 8*3600 + 12*60, "8h 12m"},
		{"hours only", 10 * 3600, "10h"},
		{"one hour", 3600, "1h"},
		{"large duration", 10*3600 + 30*60, "10h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := ContextSummary{
				FirstTime: 0,
				LastTime:  tt.duration,
			}
			got := summary.FormatDuration()
			if got != tt.want {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTimeSpan(t *testing.T) {
	tests := []struct {
		name      string
		firstTime string
		lastTime  string
		want      string
	}{
		{
			name:      "morning to evening",
			firstTime: "2026-01-14 08:23:00",
			lastTime:  "2026-01-14 16:35:00",
			want:      "08:23 - 16:35",
		},
		{
			name:      "same time",
			firstTime: "2026-01-14 10:00:00",
			lastTime:  "2026-01-14 10:00:00",
			want:      "10:00 - 10:00",
		},
		{
			name:      "late night",
			firstTime: "2026-01-14 23:45:00",
			lastTime:  "2026-01-14 23:59:00",
			want:      "23:45 - 23:59",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := ContextSummary{
				FirstTime: parseTimeLocal(tt.firstTime),
				LastTime:  parseTimeLocal(tt.lastTime),
			}
			got := summary.FormatTimeSpan()
			if got != tt.want {
				t.Errorf("FormatTimeSpan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBranchDisplay(t *testing.T) {
	tests := []struct {
		name   string
		branch *string
		want   string
	}{
		{"git branch", strPtr("main"), "main"},
		{"feature branch", strPtr("feature/auth"), "feature/auth"},
		{"nil branch", nil, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := ContextSummary{
				GitBranch: tt.branch,
			}
			got := summary.BranchDisplay()
			if got != tt.want {
				t.Errorf("BranchDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeDirectory(t *testing.T) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"home directory", homeDir, "~"},
		{"subdirectory of home", homeDir + "/projects/shy", "~/projects/shy"},
		{"not home directory", "/usr/local/bin", "/usr/local/bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDirectory(tt.path)
			if got != tt.want {
				t.Errorf("normalizeDirectory(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSortSummaries(t *testing.T) {
	tests := []struct {
		name      string
		summaries []ContextSummary
		wantOrder []string // Expected order of WorkingDir
	}{
		{
			name: "sort by duration descending",
			summaries: []ContextSummary{
				{WorkingDir: "/short", FirstTime: 0, LastTime: 300, CommandCount: 2},   // 5m
				{WorkingDir: "/long", FirstTime: 0, LastTime: 32400, CommandCount: 2},  // 9h
				{WorkingDir: "/medium", FirstTime: 0, LastTime: 5400, CommandCount: 2}, // 1h 30m
			},
			wantOrder: []string{"/long", "/medium", "/short"},
		},
		{
			name: "sort by command count when duration equal",
			summaries: []ContextSummary{
				{WorkingDir: "/few", FirstTime: 0, LastTime: 3600, CommandCount: 2},
				{WorkingDir: "/many", FirstTime: 0, LastTime: 3600, CommandCount: 4},
			},
			wantOrder: []string{"/many", "/few"},
		},
		{
			name: "sort by directory when both equal",
			summaries: []ContextSummary{
				{WorkingDir: "/zebra", FirstTime: 0, LastTime: 3600, CommandCount: 2},
				{WorkingDir: "/apple", FirstTime: 0, LastTime: 3600, CommandCount: 2},
			},
			wantOrder: []string{"/apple", "/zebra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summaries := make([]ContextSummary, len(tt.summaries))
			copy(summaries, tt.summaries)

			sortSummaries(summaries)

			for i, want := range tt.wantOrder {
				if summaries[i].WorkingDir != want {
					t.Errorf("sortSummaries() position %d = %v, want %v", i, summaries[i].WorkingDir, want)
				}
			}
		})
	}
}

func TestTableAlignment(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	summaries := []ContextSummary{
		{
			WorkingDir:   homeDir + "/a",
			GitBranch:    strPtr("main"),
			CommandCount: 2,
			FirstTime:    parseTimeLocal("2026-01-14 08:00:00"),
			LastTime:     parseTimeLocal("2026-01-14 09:00:00"),
		},
		{
			WorkingDir:   homeDir + "/very/long/path/name",
			GitBranch:    strPtr("dev"),
			CommandCount: 2,
			FirstTime:    parseTimeLocal("2026-01-14 10:00:00"),
			LastTime:     parseTimeLocal("2026-01-14 11:00:00"),
		},
	}

	output := FormatTable(summaries, "2026-01-14", true)
	lines := strings.Split(output, "\n")

	// Find the header line and data lines
	var headerLine string
	var dataLines []string
	for _, line := range lines {
		if strings.Contains(line, "Directory") && strings.Contains(line, "Branch") {
			headerLine = line
		} else if strings.Contains(line, "~/") {
			dataLines = append(dataLines, line)
		}
	}

	if headerLine == "" {
		t.Fatal("Could not find header line in output")
	}

	// Check that Branch column aligns
	branchPos := strings.Index(headerLine, "Branch")
	if branchPos == -1 {
		t.Fatal("Could not find Branch column in header")
	}

	for _, line := range dataLines {
		// Skip lines that are too short
		if len(line) < branchPos+6 {
			continue
		}

		// Check that there's content near the Branch column position
		branchArea := line[branchPos : branchPos+6]
		if !strings.Contains(branchArea, "main") && !strings.Contains(branchArea, "dev") {
			t.Errorf("Branch column not aligned properly in line: %q", line)
		}
	}
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func parseTime(timeStr string) int64 {
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(err)
	}
	return t.Unix()
}

func parseTimeLocal(timeStr string) int64 {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local)
	if err != nil {
		panic(err)
	}
	return t.Unix()
}
