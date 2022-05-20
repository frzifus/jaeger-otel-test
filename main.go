package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const nicerDicer string = "nicer.dicer/3000"

func main() {
	var (
		agentHost = flag.String("address.host", "localhost", "jaeger-agent address")
		agentPort = flag.String("address.port", "6831", "jaeger-port address")
	)
	flag.Parse()
	stdExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSyncer(stdExporter),
	}

	if *agentHost != "" && *agentPort != "" {
		log.Printf("Host: %s, Port: %s", *agentHost, *agentPort)
		jaegerExporter, err := jaeger.New(jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(*agentHost),
			jaeger.WithAgentPort(*agentPort),
		),
		)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, sdktrace.WithSyncer(jaegerExporter))
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

	for {
		gopherit()
	}
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
