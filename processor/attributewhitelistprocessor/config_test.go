package attributewhitelistprocessor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"path"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	require.NoError(t, err)
	factory := &Factory{}
	factories.Processors[configmodels.Type(typeStr)] = factory

	err = configcheck.ValidateConfig(factory.CreateDefaultConfig())
	require.NoError(t, err)

	config, err := config.LoadConfigFile(
		t,
		path.Join(".", "testdata", "testConfig.yaml"),
		factories)

	require.Nil(t, err)
	require.NotNil(t, config)

	proc := config.Processors["attributewhitelist"]
	assert.Equal(t, proc,
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "attributewhitelist",
				NameVal: "attributewhitelist",
			},
			AttributeWhiteList: []string {
				"\\bsomething\\b",
				"^http\\.*",
			},
		})
}