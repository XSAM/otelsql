package otelsql

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

type TextAttributesPropagator struct {
	Attributes map[string]string
}

var _ propagation.TextMapPropagator = TextAttributesPropagator{}

func (p TextAttributesPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	for k, v := range p.Attributes {
		carrier.Set(k, v)
	}
}

func (p TextAttributesPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	for _, k := range carrier.Keys() {
		if _, ok := p.Attributes[k]; ok {
			p.Attributes[k] = carrier.Get(k)
		}
	}
	return ctx
}

func (p TextAttributesPropagator) Fields() []string {
	var keys []string
	for k := range p.Attributes {
		keys = append(keys, k)
	}
	return keys
}
