package cprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

var typeStr = component.MustNewType("cprocessor")

func createDefaultConfig() component.Config {
	return &Config{
		Environment:      "dev",
		AddLogAttribute: true,
		RedactUserEmail:  true,
	}
}

func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, component.StabilityLevelAlpha),
	)
}

func createLogsProcessor(_ context.Context, _ processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	return &logEnricherProcessor{
		cfg:  cfg.(*Config),
		next: next,
	}, nil
}
