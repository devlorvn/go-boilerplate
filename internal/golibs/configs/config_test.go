package configs

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_FileService(t *testing.T) {
	rootDir, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatalf("failed to get root dir: %v", err)
	}

	cfg, err := LoadConfig(rootDir, ".env.file")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.App.Name != "fileservice" {
		t.Errorf("expected App.Name 'fileservice', got '%s'", cfg.App.Name)
	}

	if cfg.GRPC.Port != 50052 {
		t.Errorf("expected GRPC.Port 50052, got %d", cfg.GRPC.Port)
	}

	if cfg.DB.DBName != "file_db" {
		t.Errorf("expected DB.DBName 'file_db', got '%s'", cfg.DB.DBName)
	}

	if cfg.Metrics.Port != 2113 {
		t.Errorf("expected Metrics.Port 2113, got %d", cfg.Metrics.Port)
	}

	if cfg.Provider.AWS.Region != "HCM03" {
		t.Errorf("expected Provider.AWS.Region 'HCM03', got '%s'", cfg.Provider.AWS.Region)
	}
}

func TestLoadConfig_AuthService(t *testing.T) {
	rootDir, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatalf("failed to get root dir: %v", err)
	}

	cfg, err := LoadConfig(rootDir, ".env.auth")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.App.Name != "authservice" {
		t.Errorf("expected App.Name 'authservice', got '%s'", cfg.App.Name)
	}

	if cfg.GRPC.Port != 50051 {
		t.Errorf("expected GRPC.Port 50051, got %d", cfg.GRPC.Port)
	}

	if cfg.DB.DBName != "auth_db" {
		t.Errorf("expected DB.DBName 'auth_db', got '%s'", cfg.DB.DBName)
	}

	if cfg.Metrics.Port != 2112 {
		t.Errorf("expected Metrics.Port 2112, got %d", cfg.Metrics.Port)
	}
}

func TestLoadConfig_GatewayService(t *testing.T) {
	rootDir, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatalf("failed to get root dir: %v", err)
	}

	cfg, err := LoadConfig(rootDir, ".env.gateway")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.App.Name != "api-gateway" {
		t.Errorf("expected App.Name 'api-gateway', got '%s'", cfg.App.Name)
	}

	if cfg.HTTP.Port != 8000 {
		t.Errorf("expected HTTP.Port 8000, got %d", cfg.HTTP.Port)
	}

	if cfg.HTTP.AuthSvcAddr != "localhost:50051" {
		t.Errorf("expected HTTP.AuthSvcAddr 'localhost:50051', got '%s'", cfg.HTTP.AuthSvcAddr)
	}

	if cfg.HTTP.FileSvcAddr != "localhost:50052" {
		t.Errorf("expected HTTP.FileSvcAddr 'localhost:50052', got '%s'", cfg.HTTP.FileSvcAddr)
	}

	if cfg.Metrics.Port != 2114 {
		t.Errorf("expected Metrics.Port 2114, got %d", cfg.Metrics.Port)
	}
}
