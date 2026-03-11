package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName    string
	Environment    string
	JaegerEndpoint string
}

//Инициализирует трассировщик (tracer).
//Возвращает функцию для закрытия (shutdown) и ошибку, если что-то пошло не так.
//Что делает:

//Создаёт экспортер (отправитель данных) через newExporter.
//Создаёт провайдер трассировок через newTraceProvider.
//Устанавливает глобальный провайдер (otel.SetTracerProvider).
//Устанавливает механизм распространения контекста (propagator) — как передавать трассировки между сервисами.
//Возвращает функцию завершения работы (Shutdown) и возможную ошибку.

func InitTracer(cfg Config) (func(context.Context) error, error) {
	// Exporter
	traceExporter, err := newExporter(cfg.JaegerEndpoint)
	if err != nil {
		return nil, err
	}
	// Trace Provider
	traceProvider, err := newTraceProvider(cfg, traceExporter)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(traceProvider)

	// Propagator
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	return traceProvider.Shutdown, nil
}

func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// Что делает: создаёт экспортер для отправки данных в Jaeger через его Collector API.
// Использует jaeger.New(), указывая адрес collector (endpoint).
// Пример: если endpoint — это http://localhost:14268/api/traces, трассировки будут отправляться туда.
func newExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

//Обеспечивает поддержку двух стандартных способов распространения контекста:

// TraceContext — передача информации о трассировке по HTTP-заголовкам (W3C Trace-Context).
// Baggage — передачу метаданных или дополнительных данных.
// Объединяет их в один для многослойной работы.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

//Создаёт коллекцию ресурса — метаданных, связанных с сервисом:

//ServiceName — название сервиса.
//DeploymentEnvironment — окружение (например, production, staging).
//Это помогает при анализе в Jaeger или другом инструменте.

//Создаёт TracerProvider — основной компонент для выдачи трассировок:

// Использует WithBatcher(exporter) — собирает и отправляет данные пакетом для эффективности.
// Использует WithResource(res) — добавляет метаданные.
func newTraceProvider(cfg Config, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	return traceProvider, nil
}
