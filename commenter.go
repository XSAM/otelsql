// Copyright Sam Xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelsql

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

type commentCarrier []string

var _ propagation.TextMapCarrier = (*commentCarrier)(nil)

func (c *commentCarrier) Keys() []string { return nil }

func (c *commentCarrier) Get(string) string { return "" }

func (c *commentCarrier) Set(key, value string) {
	*c = append(*c, fmt.Sprintf("%s='%s'", url.QueryEscape(key), url.QueryEscape(value)))
}

func (c *commentCarrier) Marshal() string {
	return " /*" + strings.Join(*c, ",") + "*/"
}

// Commenter an interface for that allows injecting comments into SQL statements.
type Commenter interface {
	Query(ctx context.Context, query string) string
	Exec(ctx context.Context, query string) string
	Prepare(ctx context.Context, query string) string
}

type noopCommenter struct{}

var _ Commenter = noopCommenter{}

func (n noopCommenter) Query(_ context.Context, query string) string {
	return query
}

func (n noopCommenter) Exec(_ context.Context, query string) string {
	return query
}

func (n noopCommenter) Prepare(_ context.Context, query string) string {
	return query
}

type propagationCommenter struct {
	propagator propagation.TextMapPropagator
}

var _ Commenter = (*propagationCommenter)(nil)

// NewPropagationCommenter returns a Commenter that will inject a comment into SQL statements using the propagator.
//
// e.g., a SQL query
//
//	SELECT * from FOO
//
// will become
//
//	SELECT * from FOO /*traceparent='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',tracestate='congo%3Dt61rcWkgMzE%2Crojo%3D00f067aa0ba902b7'*/
//
// Notice: This option is EXPERIMENTAL and may be changed or removed in a
// later release.
// NewPropagationCommenter returns a Commenter that will inject a comment into SQL statements using the propagator.
//
// e.g., a SQL query
//
//	SELECT * from FOO
//
// will become
//
//	SELECT * from FOO /*traceparent='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',tracestate='congo%3Dt61rcWkgMzE%2Crojo%3D00f067aa0ba902b7'*/
//
// Notice: This option is EXPERIMENTAL and may be changed or removed in a
// later release.
func NewPropagationCommenter(propagator propagation.TextMapPropagator) Commenter {
	if propagator == nil {
		propagator = otel.GetTextMapPropagator()
	}

	return &propagationCommenter{
		propagator: propagator,
	}
}

func (c *propagationCommenter) Query(ctx context.Context, query string) string {
	return c.withComment(ctx, query)
}

func (c *propagationCommenter) Exec(ctx context.Context, query string) string {
	return c.withComment(ctx, query)
}

func (c *propagationCommenter) Prepare(ctx context.Context, query string) string {
	return c.withComment(ctx, query)
}

func (c *propagationCommenter) withComment(ctx context.Context, query string) string {
	var cc commentCarrier

	c.propagator.Inject(ctx, &cc)

	if len(cc) == 0 {
		return query
	}

	return query + cc.Marshal()
}

type fixedCommenter string

// NewFixedCommenter returns a Commenter that will inject the attributes as a comment into SQL statements.
//
// Notice: This option is EXPERIMENTAL and may be changed or removed in a
// later release.
func NewFixedCommenter(attributes attribute.Set) Commenter {
	var cc commentCarrier

	i := attributes.Iter()
	for i.Next() {
		attr := i.Attribute()
		if attr.Valid() {
			cc.Set(string(attr.Key), attr.Value.AsString())
		}
	}

	if len(cc) == 0 {
		return noopCommenter{}
	}

	return fixedCommenter(cc.Marshal())
}

func (f fixedCommenter) Query(_ context.Context, query string) string {
	return query + string(f)
}

func (f fixedCommenter) Exec(_ context.Context, query string) string {
	return query + string(f)
}

func (f fixedCommenter) Prepare(_ context.Context, query string) string {
	return query + string(f)
}
