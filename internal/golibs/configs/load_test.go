package configs

import (
	"testing"
	"time"
)

type svcConfig struct {
	Base    `mapstructure:",squash"`
	Storage struct {
		Bucket string `mapstructure:"bucket"`
	} `mapstructure:"storage"`
}

func TestLoad_Layers(t *testing.T) {
	cfg, err := Load[svcConfig]("svc", "testdata")
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.GetBase()
	checks := []struct {
		name      string
		got, want any
	}{
		{"name from argument", b.Common.Name, "svc"},
		{"common layer", b.Common.Env, "local"},
		{"duration", b.Common.ShutdownTimeout, 5 * time.Second},
		{"service layer overrides", b.Common.GRPCPort, 50052},
		{"service-only infra key", b.Postgres.DBName, "svc_db"},
		{"common infra key kept", b.Postgres.Host, "localhost"},
		{"secret layer", b.JWT.Secret, "from-secret-file"},
		{"service-specific section", cfg.Storage.Bucket, "my-bucket"},
		{"list", len(b.Kafka.Brokers), 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("JWT_SECRET", "from-env")
	t.Setenv("POSTGRES_PASSWORD", "pw")
	t.Setenv("COMMON_GRPC_PORT", "6000")
	t.Setenv("KAFKA_BROKERS", "") // empty disables kafka
	t.Setenv("COMMON_NAME", "hijack")

	cfg, err := Load[svcConfig]("svc", "testdata")
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.GetBase()
	if b.JWT.Secret != "from-env" || b.Postgres.Password != "pw" || b.Common.GRPCPort != 6000 {
		t.Errorf("env not applied: %+v", b)
	}
	if len(b.Kafka.Brokers) != 0 {
		t.Errorf("expected no brokers, got %v", b.Kafka.Brokers)
	}
	if b.Common.Name != "svc" {
		t.Errorf("name must come from argument, got %q", b.Common.Name)
	}
}

func TestLoad_OptionalSecretsAndRequiredFiles(t *testing.T) {
	if _, err := Load[svcConfig]("nosecret", "testdata"); err != nil {
		t.Fatalf("missing secrets file must be fine: %v", err)
	}
	if _, err := Load[svcConfig]("missing", "testdata"); err == nil {
		t.Fatal("missing service file must fail")
	}
}

func TestLoad_EnvForKeyMissingFromYAML(t *testing.T) {
	t.Setenv("COMMON_HTTP_PORT", "9999") // http_port appears in no YAML layer
	t.Setenv("REDIS_PASSWORD", "rp")     // redis.password neither
	cfg, err := Load[svcConfig]("svc", "testdata")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Common.HTTPPort != 9999 || cfg.Redis.Password != "rp" {
		t.Fatalf("env not bound: http_port=%d redis.password=%q", cfg.Common.HTTPPort, cfg.Redis.Password)
	}
}
