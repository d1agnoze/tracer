package tracer

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Span struct {
	Ctx   context.Context
	Span  trace.Span
	Attrs spanAttributes
}

type spanAttributes struct {
	Str   map[string]string
	Bool  map[string]bool
	Slice map[string][]string
	Int   map[string]int
	Float map[string]float64
}

type spanEvents struct {
	msg       string
	span      trace.Span
	attrs     *spanAttributes
	timestamp *time.Time
}

var (
	kindMap = map[string]trace.SpanKind{
		"internal": trace.SpanKindInternal,
		"server":   trace.SpanKindServer,
		"client":   trace.SpanKindClient,
		"producer": trace.SpanKindProducer,
		"consumer": trace.SpanKindConsumer,
	}
)

type startOptions struct {
	Kind       string
	TracerName string
}

func NewAttrs() *spanAttributes {
	return &spanAttributes{}
}

func New(ctx context.Context, spanName string, tracerArgs ...startOptions) *Span {
	opt := startOptions{}
	kind := trace.SpanKindInternal

	if len(tracerArgs) > 0 {
		opt = tracerArgs[0]
		if spanKind, ok := kindMap[opt.Kind]; ok {
			kind = spanKind
		}
	}

	ctx, span := otel.Tracer(opt.TracerName).Start(ctx, spanName, trace.WithSpanKind(kind))
	return &Span{Ctx: ctx, Span: span}
}

func (s *Span) TraceID() string {
	return s.Span.SpanContext().TraceID().String()
}

func (s *Span) SpanID() string {
	return s.Span.SpanContext().SpanID().String()
}

func (s *Span) Event(msg string) *spanEvents {
	return &spanEvents{span: s.Span, msg: msg}
}

func (s *Span) AddLink(ctx context.Context, attrs ...spanAttributes) {
	attr := []attribute.KeyValue{}
	if len(attrs) > 0 {
		attr = attrs[0].Parse()
	}

	s.Span.AddLink(trace.LinkFromContext(ctx, attr...))
}

// Span.Extract process all current attribute into otel Span instance
func (s *Span) Extract() {
	s.Span.SetAttributes(s.Attrs.Parse()...)
}

func (s *Span) End() {
	if r := recover(); r != nil {
		err := fmt.Errorf("recovered from panic: %v", r)
		s.Span.RecordError(err, trace.WithStackTrace(true))
		s.Error(err)
		s.Span.End()

		// NOTE: add your custom panic handling logic here, e.g., logging

		return
	}

	s.Extract()
	s.Span.End()
}

func (s *Span) OK(msg ...string) {
	description := ""
	if len(msg) > 0 {
		description = msg[0]
	}
	s.Span.SetStatus(codes.Ok, description)
}

func (s *Span) SError(msg string) {
	if msg == "" {
		return
	}

	s.Span.SetStatus(codes.Error, msg)
}

func (s *Span) Error(err error, recordError ...bool) {
	if err != nil {
		if len(recordError) > 0 && recordError[0] {
			s.Span.RecordError(err)
		}

		s.Span.SetStatus(codes.Error, err.Error())
		// NOTE: add your custom error handling logic here
	}
}

func (a *spanAttributes) Parse() []attribute.KeyValue {
	out := []attribute.KeyValue{}
	for k, v := range a.Str {
		out = append(out, attribute.String(k, v))
	}
	for k, v := range a.Bool {
		out = append(out, attribute.Bool(k, v))
	}
	for k, v := range a.Int {
		out = append(out, attribute.Int(k, v))
	}
	for k, v := range a.Float {
		out = append(out, attribute.Float64(k, v))
	}
	for k, v := range a.Slice {
		out = append(out, attribute.StringSlice(k, v))
	}
	return out
}

func (a *spanAttributes) StrKV(k string, v string) *spanAttributes {
	if a.Str == nil {
		a.Str = map[string]string{}
	}
	a.Str[k] = v
	return a
}

func (a *spanAttributes) BoolKV(k string, v bool) *spanAttributes {
	if a.Bool == nil {
		a.Bool = map[string]bool{}
	}
	a.Bool[k] = v
	return a
}

func (a *spanAttributes) IntKV(k string, v int) *spanAttributes {
	if a.Int == nil {
		a.Int = map[string]int{}
	}
	a.Int[k] = v
	return a
}

func (a *spanAttributes) FloatKV(k string, v float64) *spanAttributes {
	if a.Float == nil {
		a.Float = map[string]float64{}
	}
	a.Float[k] = v
	return a
}

func (a *spanAttributes) SliceKV(k string, v []string) *spanAttributes {
	if a.Slice == nil {
		a.Slice = map[string][]string{}
	}
	a.Slice[k] = v
	return a
}

func (a *spanAttributes) ErrorKV(k string, v error) *spanAttributes {
	if a.Str == nil {
		a.Str = map[string]string{}
	}

	if v == nil {
		a.Str[k] = "<nil>"
	} else {
		a.Str[k] = v.Error()
	}

	return a
}

func (e *spanEvents) Timestamp(input time.Time) *spanEvents {
	e.timestamp = &input
	return e
}

func (e *spanEvents) Attributes(input *spanAttributes) *spanEvents {
	e.attrs = input
	return e
}

func (e *spanEvents) Add() {
	opts := []trace.EventOption{}
	if e.timestamp != nil {
		opts = append(opts, trace.WithTimestamp(*e.timestamp))
	}
	if e.attrs != nil {
		opts = append(opts, trace.WithAttributes(e.attrs.Parse()...))
	}
	e.span.AddEvent(e.msg, opts...)
}
