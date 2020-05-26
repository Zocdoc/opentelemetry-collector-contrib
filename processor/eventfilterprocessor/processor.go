package eventfilterprocessor

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

type eventfiltertraceprocessor struct {
	nextConsumer consumer.TraceConsumer
	eventWhitelist []string
}

func newTraceProcesor(
	params component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	config Config) component.TraceProcessor {
	proc := &eventfiltertraceprocessor{
		nextConsumer: nextConsumer,
		eventWhitelist: config.EventWhiteList,
	}
	return proc
}


func (wp *eventfiltertraceprocessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return nil
}

func (wp *eventfiltertraceprocessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (wp *eventfiltertraceprocessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (wp *eventfiltertraceprocessor) Shutdown(context.Context) error {
	return nil
}