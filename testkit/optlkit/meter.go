package optlkit

import (
	"context"
	"fmt"
	"time"

	"github.com/testground/sdk-go/run"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.11.0"
)

func SetupMeter(
	ctx context.Context,
	tgCtx *run.InitContext,
	testCaseName,
	instanceRole string,
	opts []otlpmetrichttp.Option,
) (provider *metric.MeterProvider, stopFn func(context.Context) error, err error) {
	exp, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	provider = metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exp, metric.WithTimeout(2*time.Second))),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(fmt.Sprintf("Celestia-Testground-Case-%s", testCaseName)),
			semconv.ServiceInstanceIDKey.String(fmt.Sprintf("%s-Instance-%d", instanceRole, tgCtx.GlobalSeq)),
		)),
	)
	stopFn = func(startCtx context.Context) error {
		return provider.Shutdown(startCtx)
	}

	return provider, stopFn, nil
}
