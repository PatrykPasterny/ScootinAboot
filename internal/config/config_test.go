//go:build unit

package config

import (
	"context"
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := map[string]struct {
		configPath string
		want       *Config
		wantErr    bool
	}{
		"successful run": {
			configPath: "test_vars/valid_vars.env",
			want: &Config{
				HTTP: 8081,
				Name: "scootin_aboot",
			},
			wantErr: false,
		},
		"failed run": {
			configPath: "test_vars/invalid_vars.env",
			want:       nil,
			wantErr:    true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewConfig(context.Background(), tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
