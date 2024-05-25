package config

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME"`
	Port        string `env:"PORT"`
	LogLevel    string `env:"LOG_LEVEL"`
}
