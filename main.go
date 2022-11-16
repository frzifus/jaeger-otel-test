package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_generated_traces_total",
		Help: "The total number generated traces",
	})
)

const nicerDicer string = "nicer.dicer/3000"

func main() {
	var (
		jaegerAgentHost = flag.String("jaeger.agent.host", "", "jaeger-agent address")
		jaegerAgentPort = flag.String("jaeger.agent.port", "6831", "jaeger-port address")

		otelAgentHost = flag.String("otel.agent.host", "", "otel collector address")
		otelAgentPort = flag.String("otel.agent.port", "4317", "otel grpc port")
	)
	flag.Parse()
	stdExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSyncer(stdExporter),
	}

	if *jaegerAgentHost != "" && *jaegerAgentPort != "" {
		log.Printf("Jaeger: host %s, port: %s", *jaegerAgentHost, *jaegerAgentPort)
		jaegerExporter, err := jaeger.New(jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(*jaegerAgentHost),
			jaeger.WithAgentPort(*jaegerAgentPort),
		))
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, sdktrace.WithSyncer(jaegerExporter))
	}

	if *otelAgentHost != "" && *otelAgentPort != "" {
		log.Printf("Otel: host %s, port: %s", *otelAgentHost, *otelAgentPort)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock()}
		target := fmt.Sprintf("%s:%s", *otelAgentHost, *otelAgentPort)
		conn, err := grpc.DialContext(ctx, target, grpcOptions...)
		if err != nil {
			log.Fatalf("failed to create gRPC connection to collector: %w", err)
		}
		defer conn.Close()

		// Set up a trace exporter
		otelExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			log.Fatalf("failed to create trace exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithSyncer(otelExporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)

	otel.SetTracerProvider(tp)

	gopherit := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := run(ctx); err != nil {
			log.Println(err)
		}
	}

	go func() {
		for {
			gopherit()
			opsProcessed.Inc()
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	const timeout = 30 * time.Second
	srv := http.Server{
		Addr:         "0.0.0.0:2112",
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  timeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
	log.Println("shutdown...")
}

func run(ctx context.Context) error {
	tracer := otel.GetTracerProvider().Tracer(nicerDicer)
	ctx, span := tracer.Start(ctx, "wrapper", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	return workerStart(ctx, 1+rand.Intn(5))(ctx)
}

type worker func(context.Context) error

func workerStart(ctx context.Context, maxDepth int) worker {
	depth := maxDepth
	tracer := otel.GetTracerProvider().Tracer(nicerDicer)
	var fn worker
	fn = func(ctx context.Context) error {
		time.Sleep(time.Second * time.Duration(1+rand.Intn(5)))
		log.Println(depth)
		if depth < 1 {
			return nil
		}
		depth--
		ctx, span := tracer.Start(ctx, "dive_"+strconv.Itoa(depth), trace.WithSpanKind(trace.SpanKindInternal))
		defer span.End()
		return fn(ctx)
	}
	return fn
}
