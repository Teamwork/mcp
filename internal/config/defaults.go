package config

import "github.com/spf13/viper"

func defaults(viper *viper.Viper) {
	viper.SetDefault("server.address", "localhost:8012")
	viper.SetDefault("env", "dev")
}
