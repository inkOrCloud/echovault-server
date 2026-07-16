package config_test

import (
	"os"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/config"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("GRPC_PORT")
	os.Unsetenv("DB_DRIVER")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("STORAGE_TYPE")
	os.Unsetenv("STORAGE_PATH")
	os.Unsetenv("JWT_SECRET")

	// Ensure no config.yaml interferes by running in a temp dir
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCPort != 9090 {
		t.Errorf("GRPCPort = %d, want 9090", cfg.GRPCPort)
	}
	if cfg.DBDriver != "sqlite3" {
		t.Errorf("DBDriver = %q, want sqlite3", cfg.DBDriver)
	}
	if cfg.DBPath != "data/echovault.db" {
		t.Errorf("DBPath = %q, want data/echovault.db", cfg.DBPath)
	}
	if cfg.StorageType != "local" {
		t.Errorf("StorageType = %q, want local", cfg.StorageType)
	}
	if cfg.StoragePath != "data/files" {
		t.Errorf("StoragePath = %q, want data/files", cfg.StoragePath)
	}
	if cfg.JWTSecret != "change-me-in-production" {
		t.Errorf("JWTSecret = %q, want change-me-in-production", cfg.JWTSecret)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	t.Setenv("GRPC_PORT", "8080")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_PATH", "/custom/db.sqlite")
	t.Setenv("STORAGE_TYPE", "s3")
	t.Setenv("STORAGE_PATH", "/custom/storage")
	t.Setenv("JWT_SECRET", "my-custom-secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCPort != 8080 {
		t.Errorf("GRPCPort = %d, want 8080", cfg.GRPCPort)
	}
	if cfg.DBDriver != "postgres" {
		t.Errorf("DBDriver = %q, want postgres", cfg.DBDriver)
	}
	if cfg.DBPath != "/custom/db.sqlite" {
		t.Errorf("DBPath = %q, want /custom/db.sqlite", cfg.DBPath)
	}
	if cfg.StorageType != "s3" {
		t.Errorf("StorageType = %q, want s3", cfg.StorageType)
	}
	if cfg.StoragePath != "/custom/storage" {
		t.Errorf("StoragePath = %q, want /custom/storage", cfg.StoragePath)
	}
	if cfg.JWTSecret != "my-custom-secret" {
		t.Errorf("JWTSecret = %q, want my-custom-secret", cfg.JWTSecret)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	configContent := []byte("grpc_port: 5555\ndb_driver: mysql\njwt_secret: file-secret\n")
	err := os.WriteFile(tmpDir+"/config.yaml", configContent, 0o644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCPort != 5555 {
		t.Errorf("GRPCPort = %d, want 5555", cfg.GRPCPort)
	}
	if cfg.DBDriver != "mysql" {
		t.Errorf("DBDriver = %q, want mysql", cfg.DBDriver)
	}
	if cfg.JWTSecret != "file-secret" {
		t.Errorf("JWTSecret = %q, want file-secret", cfg.JWTSecret)
	}
}
