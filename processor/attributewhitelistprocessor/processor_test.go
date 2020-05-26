package attributewhitelistprocessor

import (
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
)


func TestTraceProcessor(t *testing.T) {
	cfg := Config{
		AttributeWhiteList: []string{"something"},
	}
	_, err := newTraceProcesor(component.ProcessorCreateParams{Logger: zap.NewNop()}, exportertest.NewNopTraceExporter(), cfg)
	require.NoError(t, err)
}
