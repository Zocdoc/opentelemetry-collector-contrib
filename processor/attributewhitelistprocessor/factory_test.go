package attributewhitelistprocessor

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateProcessor(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, exportertest.NewNopTraceExporter(), cfg)
	assert.NotNil(t, tp)
	assert.NoError(t, err, "cannot create trace processor")

	mp, err := factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, exportertest.NewNopMetricsExporter(), cfg)
	assert.Nil(t, mp)
	assert.Error(t, err, "should not be able to create metric processor")
}
