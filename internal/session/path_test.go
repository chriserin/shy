package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeDatabasePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSuffix  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "empty path should error",
			input:      "",
			wantErr:    true,
			errContains: "cannot be empty",
		},
		{
			name:       "auto-append .db extension when missing",
			input:      "/tmp/myproject",
			wantSuffix: "/tmp/myproject.db",
			wantErr:    false,
		},
		{
			name:       "preserve .db extension when present",
			input:      "/tmp/myproject.db",
			wantSuffix: "/tmp/myproject.db",
			wantErr:    false,
		},
		{
			name:       "preserve other extensions",
			input:      "/tmp/myproject.sqlite",
			wantSuffix: "/tmp/myproject.sqlite",
			wantErr:    false,
		},
		{
			name:       "relative path converted to absolute",
			input:      "local_history",
			wantSuffix: "local_history.db",
			wantErr:    false,
		},
		{
			name:       "relative path with .db",
			input:      "./local_history.db",
			wantSuffix: "local_history.db",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDatabasePath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeDatabasePath() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NormalizeDatabasePath() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NormalizeDatabasePath() unexpected error = %v", err)
				return
			}

			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("NormalizeDatabasePath() = %v, want suffix %v", got, tt.wantSuffix)
			}

			// Result should be absolute path
			if !filepath.IsAbs(got) {
				t.Errorf("NormalizeDatabasePath() = %v, want absolute path", got)
			}
		})
	}
}

func TestNormalizeDatabasePath_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		name       string
		input      string
		wantPrefix string
	}{
		{
			name:       "tilde alone",
			input:      "~/test.db",
			wantPrefix: filepath.Join(home, "test.db"),
		},
		{
			name:       "tilde with subdirectory",
			input:      "~/projects/history.db",
			wantPrefix: filepath.Join(home, "projects", "history.db"),
		},
		{
			name:       "tilde without slash",
			input:      "~test",
			wantPrefix: filepath.Join(home, "test.db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDatabasePath(tt.input)
			if err != nil {
				t.Errorf("NormalizeDatabasePath() unexpected error = %v", err)
				return
			}

			if got != tt.wantPrefix {
				t.Errorf("NormalizeDatabasePath() = %v, want %v", got, tt.wantPrefix)
			}
		})
	}
}

func TestValidateDatabasePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid path in existing directory",
			setup: func() string {
				return filepath.Join(tmpDir, "test.db")
			},
			wantErr: false,
		},
		{
			name: "existing file should be writable",
			setup: func() string {
				path := filepath.Join(tmpDir, "existing.db")
				os.WriteFile(path, []byte("test"), 0644)
				return path
			},
			wantErr: false,
		},
		{
			name: "path is a directory should error",
			setup: func() string {
				return tmpDir
			},
			wantErr:     true,
			errContains: "directory",
		},
		{
			name: "parent directory doesn't exist but can be created",
			setup: func() string {
				return filepath.Join(tmpDir, "new_dir", "test.db")
			},
			wantErr: false,
		},
		{
			name: "read-only file should error",
			setup: func() string {
				path := filepath.Join(tmpDir, "readonly.db")
				os.WriteFile(path, []byte("test"), 0444)
				return path
			},
			wantErr:     true,
			errContains: "cannot write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			err := ValidateDatabasePath(path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateDatabasePath() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateDatabasePath() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateDatabasePath() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateDatabasePath_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	err := ValidateDatabasePath("/root/restricted.db")
	if err == nil {
		t.Error("ValidateDatabasePath() expected error for restricted path but got none")
		return
	}

	if !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "cannot") {
		t.Errorf("ValidateDatabasePath() error = %v, want permission denied error", err)
	}
}
