package common_otel

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	gtracing "gorm.io/plugin/opentelemetry/tracing"
)

type Metric struct {
	Name        string
	Unit        string
	Description string
}

type TelemetryProvider interface {
	GetServiceName() string
	Logger() *logrus.Logger
	LogInfo(args ...interface{})
	LogErrorln(args ...interface{})
	LogFatalln(args ...interface{})
	LogInfof(format string, args ...interface{})
	LogErrorlnf(format string, args ...interface{})
	LogFatallnf(format string, args ...interface{})
	MeterInt64Histogram(metric Metric) (otelmetric.Int64Histogram, error)
	MeterInt64Counter(metric Metric) (otelmetric.Int64Counter, error)
	MeterInt64UpDownCounter(metric Metric) (otelmetric.Int64UpDownCounter, error)
	TraceStart(ctx context.Context, name string) (context.Context, oteltrace.Span)
	LogRequest(next http.Handler) http.Handler
	MeterRequestStatus(next http.Handler) http.Handler
	MeterRequestDuration(next http.Handler) http.Handler
	MeterRequestInFlight(next http.Handler) http.Handler
	MuxMiddleware(next http.Handler) http.Handler
	UseGormPlugin(db *gorm.DB) error
	UseRedisPlugin(rdb redis.UniversalClient) error
	Close(ctx context.Context)
}

type TelemetryConfig struct {
	ServiceName    string
	ServiceVersion string
}

type Telemetry struct {
	lp          *log.LoggerProvider
	mp          *metric.MeterProvider
	tp          *trace.TracerProvider
	log         *logrus.Logger
	meter       otelmetric.Meter
	propagators propagation.TextMapPropagator
	tracer      oteltrace.Tracer
	gormPlugin  gorm.Plugin
	cfg         TelemetryConfig
}

func NewTelemetry(ctx context.Context, cfg TelemetryConfig) (*Telemetry, error) {
	rp := newResource(cfg.ServiceName, cfg.ServiceVersion)

	lp, err := newLoggerProvider(ctx, rp)
	if err != nil {
		return nil, err
	}

	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.TextFormatter{})
	hook := otellogrus.NewHook(cfg.ServiceName, otellogrus.WithLoggerProvider(lp))
	log.AddHook(hook)

	mp, err := newMeterProvider(ctx, rp)
	if err != nil {
		return nil, err
	}

	if err := otelruntime.Start(otelruntime.WithMeterProvider(mp)); err != nil {
		return nil, fmt.Errorf("failed to start runtime instrumentation: %w", err)
	}

	propagators := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	tp, err := newTracerProvider(ctx, rp, propagators)
	if err != nil {
		return nil, err
	}

	gormPlugin := gtracing.NewPlugin(
		gtracing.WithTracerProvider(tp),
	)

	return &Telemetry{
		lp:          lp,
		mp:          mp,
		tp:          tp,
		log:         log,
		meter:       mp.Meter(cfg.ServiceName),
		tracer:      tp.Tracer(cfg.ServiceName),
		gormPlugin:  gormPlugin,
		cfg:         cfg,
		propagators: propagators,
	}, nil
}

func (t *Telemetry) GetServiceName() string { return t.cfg.ServiceName }

func (t *Telemetry) Logger() *logrus.Logger                         { return t.log }
func (t *Telemetry) LogInfo(args ...interface{})                    { t.log.Info(args...) }
func (t *Telemetry) LogErrorln(args ...interface{})                 { t.log.Errorln(args...) }
func (t *Telemetry) LogFatalln(args ...interface{})                 { t.log.Fatalln(args...) }
func (t *Telemetry) LogInfof(format string, args ...interface{})    { t.log.Infof(format, args...) }
func (t *Telemetry) LogErrorlnf(format string, args ...interface{}) { t.log.Errorf(format, args...) }
func (t *Telemetry) LogFatallnf(format string, args ...interface{}) { t.log.Fatalf(format, args...) }

func (t *Telemetry) MeterInt64Histogram(metric Metric) (otelmetric.Int64Histogram, error) {
	histogram, err := t.meter.Int64Histogram(
		metric.Name,
		otelmetric.WithUnit(metric.Unit),
		otelmetric.WithDescription(metric.Description),
	)
	if err != nil {
		return nil, err
	}
	return histogram, nil
}

func (t *Telemetry) MeterInt64Counter(metric Metric) (otelmetric.Int64Counter, error) {
	counter, err := t.meter.Int64Counter(
		metric.Name,
		otelmetric.WithUnit(metric.Unit),
		otelmetric.WithDescription(metric.Description),
	)
	if err != nil {
		return nil, err
	}
	return counter, nil
}

func (t *Telemetry) MeterInt64UpDownCounter(metric Metric) (otelmetric.Int64UpDownCounter, error) {
	counter, err := t.meter.Int64UpDownCounter(
		metric.Name,
		otelmetric.WithUnit(metric.Unit),
		otelmetric.WithDescription(metric.Description),
	)
	if err != nil {
		return nil, err
	}
	return counter, nil
}

func (t *Telemetry) TraceStart(ctx context.Context, name string) (context.Context, oteltrace.Span) {
	return t.tracer.Start(ctx, name)
}

func (t *Telemetry) UseGormPlugin(db *gorm.DB) error {
	return db.Use(t.gormPlugin)
}

func (t *Telemetry) UseRedisPlugin(rdb redis.UniversalClient) error {
	return errors.Join(redisotel.InstrumentTracing(rdb, redisotel.WithTracerProvider(t.tp)),
		redisotel.InstrumentMetrics(rdb, redisotel.WithMeterProvider(t.mp)))
}

func (t *Telemetry) Close(ctx context.Context) {
	t.lp.Shutdown(ctx)
	t.mp.Shutdown(ctx)
	t.tp.Shutdown(ctx)
}
