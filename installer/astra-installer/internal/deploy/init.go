package deploy

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

var (
	//go:embed initdata/001_schema.sql
	schemaSQL string

	//go:embed initdata/002_schema_enhancements.sql
	schemaEnhancementsSQL string
)

func writeInitSQL(composeDir string) error {
	initDir := filepath.Join(composeDir, "init")
	if err := os.MkdirAll(initDir, 0755); err != nil {
		return fmt.Errorf("create init dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(initDir, "001_schema.sql"), []byte(schemaSQL), 0644); err != nil {
		return fmt.Errorf("write 001_schema.sql: %w", err)
	}
	if err := os.WriteFile(filepath.Join(initDir, "002_schema_enhancements.sql"), []byte(schemaEnhancementsSQL), 0644); err != nil {
		return fmt.Errorf("write 002_schema_enhancements.sql: %w", err)
	}

	return nil
}
