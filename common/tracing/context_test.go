package tracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/0xPolygon/heimdall-v2/common/tracing"
)

func TestWithTracer(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("test")

	newCtx := tracing.WithTracer(ctx, tracer)
	require.NotNil(t, newCtx)

	// Verify tracer can be retrieved
	retrievedTracer := tracing.FromContext(newCtx)
	require.NotNil(t, retrievedTracer)
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		expectNil bool
	}{
		{
			name: "context with tracer",
			setupCtx: func() context.Context {
				ctx := context.Background()
				tracer := noop.NewTracerProvider().Tracer("test")
				return tracing.WithTracer(ctx, tracer)
			},
			expectNil: false,
		},
		{
			name: "context without tracer",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectNil: true,
		},
		{
			name: "nil context value",
			setupCtx: func() context.Context {
				// Context with the wrong key type
				return context.Background()
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			tracer := tracing.FromContext(ctx)

			if tt.expectNil {
				require.Nil(t, tracer)
			} else {
				require.NotNil(t, tracer)
			}
		})
	}
}

func TestStartSpan(t *testing.T) {
	tests := []struct {
		name          string
		setupCtx      func() context.Context
		spanName      string
		expectNilSpan bool
	}{
		{
			name: "start span with tracer",
			setupCtx: func() context.Context {
				ctx := context.Background()
				tracer := noop.NewTracerProvider().Tracer("test")
				return tracing.WithTracer(ctx, tracer)
			},
			spanName:      "test-span",
			expectNilSpan: false,
		},
		{
			name: "start span without tracer",
			setupCtx: func() context.Context {
				return context.Background()
			},
			spanName:      "test-span",
			expectNilSpan: true,
		},
		{
			name: "start span with empty name",
			setupCtx: func() context.Context {
				ctx := context.Background()
				tracer := noop.NewTracerProvider().Tracer("test")
				return tracing.WithTracer(ctx, tracer)
			},
			spanName:      "",
			expectNilSpan: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			newCtx, span := tracing.StartSpan(ctx, tt.spanName)

			require.NotNil(t, newCtx)

			if tt.expectNilSpan {
				require.Nil(t, span)
			} else {
				require.NotNil(t, span)
			}
		})
	}
}

func TestEndSpan(t *testing.T) {
	tests := []struct {
		name string
		span trace.Span
	}{
		{
			name: "end valid span",
			span: func() trace.Span {
				tracer := noop.NewTracerProvider().Tracer("test")
				_, span := tracer.Start(context.Background(), "test")
				return span
			}(),
		},
		{
			name: "end nil span",
			span: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			require.NotPanics(t, func() {
				tracing.EndSpan(tt.span)
			})
		})
	}
}

func TestSetAttributes(t *testing.T) {
	tests := []struct {
		name  string
		span  trace.Span
		attrs []attribute.KeyValue
	}{
		{
			name: "set attributes on valid span",
			span: func() trace.Span {
				tracer := noop.NewTracerProvider().Tracer("test")
				_, span := tracer.Start(context.Background(), "test")
				return span
			}(),
			attrs: []attribute.KeyValue{
				attribute.String("key1", "value1"),
				attribute.Int("key2", 42),
			},
		},
		{
			name: "set attributes on nil span",
			span: nil,
			attrs: []attribute.KeyValue{
				attribute.String("key", "value"),
			},
		},
		{
			name: "set empty attributes",
			span: func() trace.Span {
				tracer := noop.NewTracerProvider().Tracer("test")
				_, span := tracer.Start(context.Background(), "test")
				return span
			}(),
			attrs: []attribute.KeyValue{},
		},
		{
			name: "set multiple attributes",
			span: func() trace.Span {
				tracer := noop.NewTracerProvider().Tracer("test")
				_, span := tracer.Start(context.Background(), "test")
				return span
			}(),
			attrs: []attribute.KeyValue{
				attribute.String("attr1", "val1"),
				attribute.String("attr2", "val2"),
				attribute.Int("attr3", 123),
				attribute.Bool("attr4", true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			require.NotPanics(t, func() {
				tracing.SetAttributes(tt.span, tt.attrs...)
			})
		})
	}
}

func TestStartSpanPreservesTracer(t *testing.T) {
	// Verify that StartSpan preserves the tracer in the returned context
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx = tracing.WithTracer(ctx, tracer)

	newCtx, span := tracing.StartSpan(ctx, "test-span")
	require.NotNil(t, span)

	// Verify tracer is still available in the new context
	retrievedTracer := tracing.FromContext(newCtx)
	require.NotNil(t, retrievedTracer)
}

func TestWithTracerOverwrite(t *testing.T) {
	// Test that WithTracer can overwrite an existing tracer
	ctx := context.Background()
	tracer1 := noop.NewTracerProvider().Tracer("tracer1")
	tracer2 := noop.NewTracerProvider().Tracer("tracer2")

	ctx = tracing.WithTracer(ctx, tracer1)
	ctx = tracing.WithTracer(ctx, tracer2)

	retrieved := tracing.FromContext(ctx)
	require.NotNil(t, retrieved)
}

func TestEndSpanMultipleTimes(t *testing.T) {
	// Verify that calling EndSpan multiple times doesn't panic
	tracer := noop.NewTracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "test")

	require.NotPanics(t, func() {
		tracing.EndSpan(span)
		tracing.EndSpan(span)
		tracing.EndSpan(span)
	})
}

func TestSetAttributesDifferentTypes(t *testing.T) {
	// Test setting different attribute types
	tracer := noop.NewTracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "test")

	require.NotPanics(t, func() {
		tracing.SetAttributes(span,
			attribute.String("string_attr", "value"),
			attribute.Int("int_attr", 42),
			attribute.Int64("int64_attr", 12345),
			attribute.Float64("float_attr", 3.14),
			attribute.Bool("bool_attr", true),
		)
	})
}
