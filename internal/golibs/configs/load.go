package configs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// Load builds the config of service `name` from dir, later layers winning:
//
//  1. dir/common.yaml          shared by all services (required)
//  2. dir/<name>.yaml          service specific (required)
//  3. dir/secrets/<name>.yaml  local secrets, gitignored (optional)
//  4. environment variables    key path with "." -> "_", upper-cased,
//     e.g. postgres.password -> POSTGRES_PASSWORD, jwt.secret -> JWT_SECRET
//
// Every field of T can be overridden from env, even if no YAML layer mentions
// it. This keeps one image deployable anywhere: in the cloud, secrets arrive
// as env vars (ECS/EKS secret injection).
func Load[T any](name, dir string) (*T, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	layers := []struct {
		path     string
		optional bool
	}{
		{filepath.Join(dir, "common.yaml"), false},
		{filepath.Join(dir, name+".yaml"), false},
		{filepath.Join(dir, "secrets", name+".yaml"), true},
	}
	for _, l := range layers {
		f, err := os.Open(l.path)
		if err != nil {
			if l.optional && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("open config %s: %w", l.path, err)
		}
		err = v.MergeConfig(f)
		_ = f.Close() // read-only file, nothing to flush
		if err != nil {
			return nil, fmt.Errorf("parse config %s: %w", l.path, err)
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()
	bindEnvs(v, reflect.TypeOf((*T)(nil)).Elem(), "")
	v.Set("common.name", name)

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config for %s: %w", name, err)
	}
	return &cfg, nil
}

// bindEnvs registers every leaf key of t (following mapstructure tags and
// ",squash") so viper.Unmarshal sees env values for keys absent from YAML.
func bindEnvs(v *viper.Viper, t reflect.Type, prefix string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, opts, _ := strings.Cut(f.Tag.Get("mapstructure"), ",")
		ft := f.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		isStruct := ft.Kind() == reflect.Struct && ft.PkgPath() != "time"
		if strings.Contains(opts, "squash") && isStruct {
			bindEnvs(v, ft, prefix)
			continue
		}
		if tag == "" || tag == "-" {
			tag = strings.ToLower(f.Name)
		}
		key := prefix + tag
		if isStruct {
			bindEnvs(v, ft, key+".")
			continue
		}
		_ = v.BindEnv(key)
	}
}
