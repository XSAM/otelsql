package otelsql

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

type ServiceNamePropagator struct {
	ServiceName string
}

var _ propagation.TextMapPropagator = ServiceNamePropagator{}

func (p ServiceNamePropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	carrier.Set("service.name", p.ServiceName)
}

func (p ServiceNamePropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return ctx
}

func (p ServiceNamePropagator) Fields() []string {
	return []string{"service.name"}
}
