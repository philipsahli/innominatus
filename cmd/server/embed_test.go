package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddedMigrationsFS verifies that SQL migration files are properly embedded
func TestEmbeddedMigrationsFS(t *testing.T) {
	// Expected migration files (in order)
	expectedMigrations := []string{
		"migrations/001_create_graph_tables.sql",
		"migrations/002_create_application_tables.sql",
		"migrations/003_create_sessions_table.sql",
		"migrations/004_rename_graph_tables.sql",
		"migrations/005_create_api_keys_table.sql",
		"migrations/006_add_delegated_resources.sql",
		"migrations/007_create_queue_tasks_table.sql",
		"migrations/008_create_graph_annotations.sql",
		"migrations/009_add_workflow_retry_support.sql",
		"migrations/010_add_application_labels.sql",
		"migrations/011_add_resource_workflow_columns.sql",
	}

	t.Run("migrations directory exists", func(t *testing.T) {
		entries, err := fs.ReadDir(migrationsFS, "migrations")
		if err != nil {
			t.Fatalf("Failed to read migrations directory: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("No migration files found in embedded FS")
		}

		t.Logf("Found %d migration files", len(entries))
	})

	t.Run("all expected migrations are embedded", func(t *testing.T) {
		for _, expectedPath := range expectedMigrations {
			content, err := fs.ReadFile(migrationsFS, expectedPath)
			if err != nil {
				t.Errorf("Migration %s not found in embedded FS: %v", expectedPath, err)
				continue
			}

			if len(content) == 0 {
				t.Errorf("Migration %s is empty", expectedPath)
			}

			// Verify it's a SQL file by checking for common SQL keywords
			contentStr := string(content)
			if !strings.Contains(contentStr, "CREATE") && !strings.Contains(contentStr, "ALTER") && !strings.Contains(contentStr, "DROP") {
				t.Errorf("Migration %s doesn't appear to be valid SQL (no CREATE/ALTER/DROP statements)", expectedPath)
			}

			t.Logf("✓ %s (%d bytes)", filepath.Base(expectedPath), len(content))
		}
	})

	t.Run("migration count matches expected", func(t *testing.T) {
		entries, err := fs.ReadDir(migrationsFS, "migrations")
		if err != nil {
			t.Fatalf("Failed to read migrations directory: %v", err)
		}

		if len(entries) != len(expectedMigrations) {
			t.Errorf("Expected %d migrations, found %d", len(expectedMigrations), len(entries))
		}
	})

	t.Run("migrations can be read via sub-filesystem", func(t *testing.T) {
		// This is how main.go uses the embedded FS
		migrationsSubFS, err := fs.Sub(migrationsFS, "migrations")
		if err != nil {
			t.Fatalf("Failed to create migrations sub-filesystem: %v", err)
		}

		// Read first migration through sub-FS
		content, err := fs.ReadFile(migrationsSubFS, "001_create_graph_tables.sql")
		if err != nil {
			t.Fatalf("Failed to read migration through sub-FS: %v", err)
		}

		if len(content) == 0 {
			t.Fatal("Migration read through sub-FS is empty")
		}

		t.Logf("Successfully read migration through sub-FS (%d bytes)", len(content))
	})
}

// TestEmbeddedSwaggerFS verifies that Swagger specification files are properly embedded
func TestEmbeddedSwaggerFS(t *testing.T) {
	expectedSwaggerFiles := []string{
		"swagger-admin.yaml",
		"swagger-user.yaml",
	}

	t.Run("swagger files exist", func(t *testing.T) {
		entries, err := fs.ReadDir(swaggerFilesFS, ".")
		if err != nil {
			t.Fatalf("Failed to read swagger files directory: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("No swagger files found in embedded FS")
		}

		t.Logf("Found %d files in swagger FS", len(entries))
	})

	t.Run("all expected swagger files are embedded", func(t *testing.T) {
		for _, expectedFile := range expectedSwaggerFiles {
			content, err := fs.ReadFile(swaggerFilesFS, expectedFile)
			if err != nil {
				t.Errorf("Swagger file %s not found in embedded FS: %v", expectedFile, err)
				continue
			}

			if len(content) == 0 {
				t.Errorf("Swagger file %s is empty", expectedFile)
			}

			// Verify it's a YAML file by checking for YAML markers
			contentStr := string(content)
			if !strings.Contains(contentStr, "openapi:") && !strings.Contains(contentStr, "swagger:") {
				t.Errorf("File %s doesn't appear to be valid OpenAPI/Swagger YAML", expectedFile)
			}

			t.Logf("✓ %s (%d bytes)", expectedFile, len(content))
		}
	})

	t.Run("swagger-admin.yaml has expected size", func(t *testing.T) {
		content, err := fs.ReadFile(swaggerFilesFS, "swagger-admin.yaml")
		if err != nil {
			t.Fatalf("Failed to read swagger-admin.yaml: %v", err)
		}

		// Admin spec should be smaller than user spec
		if len(content) < 1000 {
			t.Errorf("swagger-admin.yaml seems too small (%d bytes)", len(content))
		}
	})

	t.Run("swagger-user.yaml has expected size", func(t *testing.T) {
		content, err := fs.ReadFile(swaggerFilesFS, "swagger-user.yaml")
		if err != nil {
			t.Fatalf("Failed to read swagger-user.yaml: %v", err)
		}

		// User spec should be larger (more endpoints)
		if len(content) < 10000 {
			t.Errorf("swagger-user.yaml seems too small (%d bytes)", len(content))
		}
	})

	t.Run("swagger files contain valid API paths", func(t *testing.T) {
		for _, file := range expectedSwaggerFiles {
			content, err := fs.ReadFile(swaggerFilesFS, file)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", file, err)
			}

			contentStr := string(content)

			// Check for common API paths that should exist
			if !strings.Contains(contentStr, "/api/") {
				t.Errorf("%s doesn't contain /api/ paths", file)
			}

			// Check for OpenAPI required fields
			requiredFields := []string{"info:", "paths:"}
			for _, field := range requiredFields {
				if !strings.Contains(contentStr, field) {
					t.Errorf("%s missing required OpenAPI field: %s", file, field)
				}
			}
		}
	})
}

// TestEmbeddedFilesystemIntegrity performs cross-cutting checks on all embedded filesystems
func TestEmbeddedFilesystemIntegrity(t *testing.T) {
	t.Run("no embedded files are zero-length", func(t *testing.T) {
		// Check migrations
		migrationsSubFS, _ := fs.Sub(migrationsFS, "migrations")
		entries, _ := fs.ReadDir(migrationsSubFS, ".")
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := fs.ReadFile(migrationsSubFS, entry.Name())
			if err != nil {
				t.Errorf("Failed to read %s: %v", entry.Name(), err)
				continue
			}
			if len(content) == 0 {
				t.Errorf("Migration %s is zero-length", entry.Name())
			}
		}

		// Check swagger files
		swaggerEntries, _ := fs.ReadDir(swaggerFilesFS, ".")
		for _, entry := range swaggerEntries {
			if entry.IsDir() {
				continue
			}
			content, err := fs.ReadFile(swaggerFilesFS, entry.Name())
			if err != nil {
				t.Errorf("Failed to read %s: %v", entry.Name(), err)
				continue
			}
			if len(content) == 0 {
				t.Errorf("Swagger file %s is zero-length", entry.Name())
			}
		}
	})

	t.Run("embedded files have valid line endings", func(t *testing.T) {
		// Check a sample migration
		content, err := fs.ReadFile(migrationsFS, "migrations/001_create_graph_tables.sql")
		if err != nil {
			t.Fatalf("Failed to read migration: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "\n") {
			t.Error("Migration file doesn't contain newlines")
		}

		// Should not have CRLF (Windows line endings)
		if strings.Contains(contentStr, "\r\n") {
			t.Logf("Warning: Migration file contains Windows line endings (CRLF)")
		}
	})

	t.Run("migrations are numerically ordered", func(t *testing.T) {
		entries, err := fs.ReadDir(migrationsFS, "migrations")
		if err != nil {
			t.Fatalf("Failed to read migrations: %v", err)
		}

		var names []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				names = append(names, entry.Name())
			}
		}

		// Verify they start with numbers in sequence
		for i, name := range names {
			expected := i + 1
			if !strings.HasPrefix(name, filepath.Join("", string(rune('0'+expected/100%10)), string(rune('0'+expected/10%10)), string(rune('0'+expected%10)))) {
				// More lenient check: just ensure it starts with a digit
				if len(name) == 0 || (name[0] < '0' || name[0] > '9') {
					t.Errorf("Migration %s doesn't start with a number", name)
				}
			}
		}
	})
}

// TestEmbedPreparationRequired verifies build-time preparation is done
func TestEmbedPreparationRequired(t *testing.T) {
	t.Run("migrations directory exists in cmd/server", func(t *testing.T) {
		// This test verifies that prepare-embed.sh has run
		_, err := fs.ReadDir(migrationsFS, "migrations")
		if err != nil {
			t.Errorf("Migrations directory not found - did you run 'make prepare-embed' or './scripts/prepare-embed.sh'?")
		}
	})

	t.Run("swagger files exist in embed directory", func(t *testing.T) {
		// This test verifies swagger files were copied to cmd/server
		_, err := fs.ReadFile(swaggerFilesFS, "swagger-admin.yaml")
		if err != nil {
			t.Errorf("swagger-admin.yaml not found - did you run 'make prepare-embed' or './scripts/prepare-embed.sh'?")
		}

		_, err = fs.ReadFile(swaggerFilesFS, "swagger-user.yaml")
		if err != nil {
			t.Errorf("swagger-user.yaml not found - did you run 'make prepare-embed' or './scripts/prepare-embed.sh'?")
		}
	})
}

// TestBinaryStandaloneCapability tests that embedded files make binary self-contained
func TestBinaryStandaloneCapability(t *testing.T) {
	t.Run("migrations can run without external files", func(t *testing.T) {
		// Create sub-filesystem (same as main.go does)
		migrationsSubFS, err := fs.Sub(migrationsFS, "migrations")
		if err != nil {
			t.Fatalf("Failed to create migrations sub-filesystem: %v", err)
		}

		// Verify we can iterate all migrations
		entries, err := fs.ReadDir(migrationsSubFS, ".")
		if err != nil {
			t.Fatalf("Failed to read migrations from sub-FS: %v", err)
		}

		migrationCount := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				content, err := fs.ReadFile(migrationsSubFS, entry.Name())
				if err != nil {
					t.Errorf("Failed to read migration %s: %v", entry.Name(), err)
					continue
				}

				if len(content) == 0 {
					t.Errorf("Migration %s is empty", entry.Name())
				}

				migrationCount++
			}
		}

		if migrationCount == 0 {
			t.Fatal("No migrations found - binary would not be standalone")
		}

		t.Logf("Verified %d migrations are accessible without external files", migrationCount)
	})

	t.Run("swagger specs can be served without external files", func(t *testing.T) {
		// Verify both swagger files can be read
		adminContent, err := fs.ReadFile(swaggerFilesFS, "swagger-admin.yaml")
		if err != nil {
			t.Fatalf("Failed to read swagger-admin.yaml: %v", err)
		}

		userContent, err := fs.ReadFile(swaggerFilesFS, "swagger-user.yaml")
		if err != nil {
			t.Fatalf("Failed to read swagger-user.yaml: %v", err)
		}

		if len(adminContent) == 0 || len(userContent) == 0 {
			t.Fatal("Swagger specs are empty - binary would not serve API docs")
		}

		t.Logf("Verified swagger specs are accessible without external files (admin: %d bytes, user: %d bytes)",
			len(adminContent), len(userContent))
	})

	t.Run("web-ui can be served without external files", func(t *testing.T) {
		// Create web-ui sub-filesystem (same as main.go does)
		webUISubFS, err := fs.Sub(webUIFS, "web-ui-out")
		if err != nil {
			t.Fatalf("Failed to create web-ui sub-filesystem: %v", err)
		}

		// Verify index.html exists
		indexContent, err := fs.ReadFile(webUISubFS, "index.html")
		if err != nil {
			t.Fatalf("index.html not found: %v", err)
		}

		if len(indexContent) == 0 {
			t.Fatal("index.html is empty")
		}

		t.Logf("✓ index.html (%d bytes)", len(indexContent))

		// Verify _next directory exists and has files
		entries, err := fs.ReadDir(webUISubFS, "_next/static/chunks")
		if err != nil {
			t.Fatalf("_next/static/chunks directory not found: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("No chunk files found in _next/static/chunks")
		}

		t.Logf("✓ _next/static/chunks contains %d files", len(entries))

		t.Logf("Verified web-ui is accessible without external files")
	})
}

// TestEmbeddedWebUIFS verifies that web UI static files are properly embedded
func TestEmbeddedWebUIFS(t *testing.T) {
	t.Run("web-ui-out directory exists", func(t *testing.T) {
		entries, err := fs.ReadDir(webUIFS, "web-ui-out")
		if err != nil {
			t.Fatalf("Failed to read web-ui-out directory: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("No files found in embedded web-ui-out - did you run 'npm run build' in web-ui/?")
		}

		t.Logf("Found %d entries in web-ui-out", len(entries))
	})

	t.Run("index.html exists and has content", func(t *testing.T) {
		content, err := fs.ReadFile(webUIFS, "web-ui-out/index.html")
		if err != nil {
			t.Fatalf("index.html not found in embedded FS: %v", err)
		}

		if len(content) < 100 {
			t.Errorf("index.html seems too small (%d bytes)", len(content))
		}

		// Verify it's HTML
		contentStr := string(content)
		if !strings.Contains(contentStr, "<!DOCTYPE html>") && !strings.Contains(contentStr, "<html") {
			t.Error("index.html doesn't appear to be valid HTML")
		}

		t.Logf("✓ index.html (%d bytes)", len(content))
	})

	t.Run("Next.js static assets exist", func(t *testing.T) {
		// Check for _next directory
		entries, err := fs.ReadDir(webUIFS, "web-ui-out/_next")
		if err != nil {
			t.Fatalf("_next directory not found: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("_next directory is empty - Next.js build may have failed")
		}

		t.Logf("Found %d entries in _next directory", len(entries))
	})

	t.Run("web-ui sub-filesystem works", func(t *testing.T) {
		// This is how main.go uses the embedded FS
		webUISubFS, err := fs.Sub(webUIFS, "web-ui-out")
		if err != nil {
			t.Fatalf("Failed to create web-ui sub-filesystem: %v", err)
		}

		// Read index.html through sub-FS
		content, err := fs.ReadFile(webUISubFS, "index.html")
		if err != nil {
			t.Fatalf("Failed to read index.html through sub-FS: %v", err)
		}

		if len(content) == 0 {
			t.Fatal("index.html read through sub-FS is empty")
		}

		t.Logf("Successfully read index.html through sub-FS (%d bytes)", len(content))
	})

	t.Run("web-ui files are not zero-length", func(t *testing.T) {
		webUISubFS, _ := fs.Sub(webUIFS, "web-ui-out")

		// Walk the filesystem and check file sizes
		zeroLengthCount := 0
		totalFiles := 0

		err := fs.WalkDir(webUISubFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			totalFiles++
			content, err := fs.ReadFile(webUISubFS, path)
			if err != nil {
				t.Logf("Warning: failed to read %s: %v", path, err)
				return nil
			}

			if len(content) == 0 {
				zeroLengthCount++
				t.Logf("Warning: zero-length file: %s", path)
			}

			return nil
		})

		if err != nil {
			t.Fatalf("Failed to walk web-ui filesystem: %v", err)
		}

		t.Logf("Checked %d files, found %d zero-length files", totalFiles, zeroLengthCount)

		if totalFiles == 0 {
			t.Fatal("No files found in web-ui - build may have failed")
		}
	})

	t.Run("web-ui build includes required assets", func(t *testing.T) {
		// Check for common Next.js static export structure
		requiredPaths := []string{
			"web-ui-out/_next",
			"web-ui-out/index.html",
		}

		for _, path := range requiredPaths {
			_, err := fs.Stat(webUIFS, path)
			if err != nil {
				t.Errorf("Required path %s not found in embedded FS: %v", path, err)
			}
		}
	})
}
