package config

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME"`
	Host        string `env:"HOST"`
	Port        string `env:"PORT"`
	LogLevel    string `env:"LOG_LEVEL"`
	Telemetry   TelemetryConfig
}

type TelemetryConfig struct {
	ServiceNamespace string `env:"OTEL_SERVICE_NAMESPACE"`
	ServiceName      string `env:"OTEL_SERVICE_NAME"`
	ExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	Insecure         bool   `env:"OTEL_EXPORTER_OTLP_INSECURE"`
}
