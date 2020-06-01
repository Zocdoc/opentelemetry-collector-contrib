package attributewhitelistprocessor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/translator/conventions"
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

func TestAttributeWhiteListProcessor_ConsumeTraces(t *testing.T) {
	testCases := []testCase{
		{
			Name:          "works for nil attrs",
			Attrs:         nil,
			ExpectedAttrs: nil,
			WhiteList:     []string{
				"\\bsomething\\b",
			},
		},{
			Name:          "works for matches",
			Attrs:         map[string]pdata.AttributeValue{
				"ShouldBeKept": pdata.NewAttributeValueInt(123),
				"shouldBeDropped": pdata.NewAttributeValueInt(123),
				"http.port.should.be.kept": pdata.NewAttributeValueInt(123),
				"http.ipv4.should.be.kept": pdata.NewAttributeValueInt(123),
				"http.x.should.be.kept": pdata.NewAttributeValueInt(123),
				"x.http.should.be.dropped" : pdata.NewAttributeValueBool(true),
			},
			ExpectedAttrs: map[string]pdata.AttributeValue{
				"ShouldBeKept": pdata.NewAttributeValueInt(123),
				"http.port.should.be.kept": pdata.NewAttributeValueInt(123),
				"http.ipv4.should.be.kept": pdata.NewAttributeValueInt(123),
				"http.x.should.be.kept": pdata.NewAttributeValueInt(123),
			},
			WhiteList:     []string{
				"\\bShouldBeKept\\b",
				"^http\\.*",
			},
		},
	}

	for _, test := range testCases {
		runConsumeTracesTest(t, test)
	}
}

type testCase struct {
	Name string
	Attrs map[string]pdata.AttributeValue
	ExpectedAttrs map[string]pdata.AttributeValue
	WhiteList []string
}

func generateTraceData(inputName string, attrs map[string]pdata.AttributeValue) pdata.Traces {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	rs := td.ResourceSpans().At(0)
	rs.Resource().InitEmpty()
	rs.Resource().Attributes().UpsertString(conventions.AttributeServiceName, "test-service")
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	spans := ils.Spans()
	spans.Resize(1)
	spans.At(0).SetName(inputName)
	spans.At(0).Attributes().InitFromMap(attrs).Sort()
	return td
}

func runConsumeTracesTest(t *testing.T, test testCase) {
	// generate data
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.AttributeWhiteList = test.WhiteList

	p, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, p)

	t.Run(test.Name, func(t *testing.T) {
		td := generateTraceData(test.Name, test.Attrs)
		assert.NoError(t, p.ConsumeTraces(context.Background(), td))
		// td is modified now
		expected := generateTraceData(test.Name, test.ExpectedAttrs)
		assert.EqualValues(t, expected, td)
	})
}
