package cprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

type logEnricherProcessor struct {
	cfg  *Config
	next consumer.Logs
}

var _ consumer.Logs = (*logEnricherProcessor)(nil)

func (p *logEnricherProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *logEnricherProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (p *logEnricherProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (p *logEnricherProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	modifyLogs(ld, p.cfg)
	return p.next.ConsumeLogs(ctx, ld)
}

func modifyLogs(ld plog.Logs, cfg *Config) {
	resourceLogs := ld.ResourceLogs()

	for i := 0; i < resourceLogs.Len(); i++ {
		rl := resourceLogs.At(i)

		if cfg.Environment != "" {
			rl.Resource().Attributes().PutStr("deployment.environment", cfg.Environment)
		}

		scopeLogs := rl.ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			logs := scopeLogs.At(j).LogRecords()

			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)

				if cfg.AddLogAttribute {
					log.Attributes().PutBool("collector.processed", true)
				}

				if cfg.RedactUserEmail {
					attr, ok := log.Attributes().Get("user.email")
					if ok && attr.Str() != "" {
						log.Attributes().PutStr("user.email", "[REDACTED]")
					}
				}
			}
		}
	}

	print("Processed logs")
}
