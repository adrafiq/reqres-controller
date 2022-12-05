package config

import (
	"github.com/spf13/viper"
)

func New() *viper.Viper {
	var envConfig = viper.New()
	// if env file, then that else os.env
	envConfig.Set("REQRES_ROOT_URL", nil)
	envConfig.AutomaticEnv()
	return envConfig
}
