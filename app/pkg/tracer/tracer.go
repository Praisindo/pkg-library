package tracer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	TracingTool        string
	OTLPEndpoint       string
	GoogleCloudProject string
	JaegerEndpoint     string
	TracerSamplingRate string
}

func InitTracer(ctx context.Context, serviceName, environment, moduleName string, config Config, ginEngine *gin.Engine) (*sdktrace.TracerProvider, error) {
	if strings.TrimSpace(config.TracingTool) == "" {
		fmt.Println("TracingTool is empty, skipping tracer initialization")
		return nil, errors.New("tracing tool not configured")
	}

	sampler := initializeTraceSampler(config.TracerSamplingRate)
	var tp *sdktrace.TracerProvider
	switch {
	case strings.Contains(config.TracingTool, "GCP") && config.GoogleCloudProject != "":
		exporter, err := texporter.New(texporter.WithProjectID(config.GoogleCloudProject))
		if err != nil {
			return nil, err
		}

		// Identify your application using resource detection
		res, err := resource.New(ctx,
			// Use the GCP resource detector to detect information about the GCP platform
			resource.WithDetectors(gcp.NewDetector()),
			// Keep the default detectors
			resource.WithTelemetrySDK(),
			// Add your own custom attributes to identify your application
			resource.WithAttributes(
				semconv.ServiceNameKey.String(serviceName),
				attribute.String("environment", environment),
				attribute.String("module", moduleName),
			),
		)
		if err != nil {
			fmt.Println("Failed to create GCP tracer resource:", err)
			return nil, err
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		if tp == nil {
			fmt.Println("Failed to create GCP tracer provider:", err)
			return nil, errors.New("failed to create GCP tracer provider")
		} else {
			fmt.Println("GCP Tracer Provider created successfully")
		}
	case strings.Contains(config.TracingTool, "STDOUT"):
		fmt.Println("infrastructureconfiguration.TracingTool CCC: ", config.TracingTool)
		stdoutExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return &sdktrace.TracerProvider{}, err
		}

		resources := resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", environment),
			attribute.String("module", moduleName),
		)

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithBatcher(stdoutExporter),
			sdktrace.WithResource(resources),
		)
	case strings.Contains(config.TracingTool, "JAEGER"):
		jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sampler),
			sdktrace.WithBatcher(jaegerExporter),
			sdktrace.WithResource(
				resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String(serviceName),
					attribute.String("environment", environment),
					attribute.String("module", moduleName),
				)),
		)
	}

	if tp != nil {
		// Set global provider
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			),
		)
		// Test the tracer
		tr := tp.Tracer("InitializeTracer")
		_, span := tr.Start(context.Background(), "InitializeTracerSpan")
		span.AddEvent("Tracer initialized successfully")
		span.End()
	}

	if ginEngine != nil && tp != nil {
		// Tambahkan middleware OpenTelemetry
		ginEngine.Use(otelgin.Middleware("gin-server"))

		// Middleware tambahan untuk menambahkan full URL ke trace
		ginEngine.Use(func(c *gin.Context) {
			span := trace.SpanFromContext(c.Request.Context())
			if span != nil {
				span.SetAttributes(attribute.String("http.full_url", c.Request.URL.String()))
			}
			c.Next()
		})
	}

	return tp, nil
}

func initializeTraceSampler(TracerSamplingRate string) sdktrace.Sampler {
	sampler := sdktrace.AlwaysSample()
	if TracerSamplingRate != "" {
		var samplingRate float64
		_, err := fmt.Sscanf(TracerSamplingRate, "%f", &samplingRate)
		if err != nil {
			fmt.Println("Invalid TracerSamplingRate, using AlwaysSample:", err)
		} else {
			if samplingRate >= 1.0 {
				samplingRate = 1.0
			} else if samplingRate <= 0.0 {
				samplingRate = 0.0
			}
			sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRate))
			fmt.Printf("Using TraceIDRatioBased sampler with rate: %f\n", samplingRate)
		}
	}
	return sampler
}
