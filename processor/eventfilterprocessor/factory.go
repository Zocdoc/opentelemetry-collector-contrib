package eventfilterprocessor

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

const (
	// The value of "type" key in configuration.
	typeStr = "eventwhitelist"
)

// Factory is the factory for processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *Factory) CreateDefaultConfig() configmodels.Processor {
	return generateDefaultConfig()
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateTraceProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	c configmodels.Processor,
) (component.TraceProcessor, error) {
	cfg := c.(*Config)
	return newTraceProcesor(params, nextConsumer, cfg), nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (component.MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

func generateDefaultConfig() *Config {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		EventWhiteList: []string{},
	}
}

