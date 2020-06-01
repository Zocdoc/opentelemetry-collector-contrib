package attributewhitelistprocessor

import (
	"go.opentelemetry.io/collector/config/configmodels"
)

type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// white list for events to keep.
	AttributeWhiteList []string `mapstructure:"event_white_list,omitempty"`
}