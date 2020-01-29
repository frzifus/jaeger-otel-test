package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	"go.opentelemetry.io/otel/exporter/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const nicerDicer string = "nicer.dicer/3000"

func main() {
	var (
		agentAddr = flag.String("address.agent", "127.0.0.1:6831", "jaeger-agent address")
	)
	flag.Parse()
	stdExporter, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}

	opts := []sdktrace.ProviderOption{
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(stdExporter),
	}

	if *agentAddr != "" {
		jaegerExporter, err := jaeger.NewExporter(jaeger.WithAgentEndpoint(*agentAddr))
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, sdktrace.WithSyncer(jaegerExporter))
	}

	tp, err := sdktrace.NewProvider(opts...)
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)

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
	tracer := global.TraceProvider().Tracer(nicerDicer)
	return tracer.WithSpan(ctx, "wrapper", workerStart(ctx, 1+rand.Intn(5)))
}

type worker func(context.Context) error

func workerStart(ctx context.Context, maxDepth int) worker {
	depth := maxDepth
	tracer := global.TraceProvider().Tracer(nicerDicer)
	var fn worker
	fn = func(ctx context.Context) error {
		time.Sleep(time.Second * time.Duration(1+rand.Intn(5)))
		log.Println(depth)
		if depth < 1 {
			return nil
		}
		depth--
		return tracer.WithSpan(ctx, "dive_"+strconv.Itoa(depth), fn)
	}
	return fn
}
