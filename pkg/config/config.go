package config

import "github.com/spf13/viper"

type Config struct {
	GRPCPort    int
	DBDriver    string
	DBPath      string
	StorageType string
	StoragePath string
	JWTSecret   string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")

	viper.SetDefault("grpc_port", 9090)
	viper.SetDefault("db_driver", "sqlite3")
	viper.SetDefault("db_path", "data/echovault.db")
	viper.SetDefault("storage_type", "local")
	viper.SetDefault("storage_path", "data/files")
	viper.SetDefault("jwt_secret", "change-me-in-production")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
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
