// Package config provides application configuration loading.
package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

const defaultGRPCPort = 9090

// Config holds application configuration.
type Config struct {
	GRPCPort    int
	DBDriver    string
	DBPath      string
	StorageType string
	StoragePath string
	JWTSecret   string
}

// Load reads configuration from file and environment.
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")

	viper.SetDefault("grpc_port", defaultGRPCPort)
	viper.SetDefault("db_driver", "sqlite3")
	viper.SetDefault("db_path", "data/echovault.db")
	viper.SetDefault("storage_type", "local")
	viper.SetDefault("storage_path", "data/files")
	viper.SetDefault("jwt_secret", "change-me-in-production")

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return &Config{
		GRPCPort:    viper.GetInt("grpc_port"),
		DBDriver:    viper.GetString("db_driver"),
		DBPath:      viper.GetString("db_path"),
		StorageType: viper.GetString("storage_type"),
		StoragePath: viper.GetString("storage_path"),
		JWTSecret:   viper.GetString("jwt_secret"),
	}, nil
}
