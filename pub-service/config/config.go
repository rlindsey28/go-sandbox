package config

type AppConfig struct {
	ServiceName string           `env:"SERVICE_NAME"`
	Host        string           `env:"HOST"`
	Port        string           `env:"PORT"`
	LogLevel    string           `env:"LOG_LEVEL"`
	Kafka       *KafkaConfig     `env:", prefix=KAFKA_"`
	Telemetry   *TelemetryConfig `env:", prefix=OTEL_"`
}

type TelemetryConfig struct {
	ServiceNamespace string `env:"SERVICE_NAMESPACE"`
	ServiceName      string `env:"SERVICE_NAME"`
	ExporterEndpoint string `env:"EXPORTER_OTLP_ENDPOINT"`
	Insecure         bool   `env:"EXPORTER_OTLP_INSECURE"`
}

type KafkaConfig struct {
	Brokers []string `env:"BROKERS, delimiter=;"`
	Topic   string   `env:"TOPIC"`
}
