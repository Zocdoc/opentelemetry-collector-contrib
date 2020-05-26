package eventfilterprocessor

import (
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// white list for events to keep.
	EventWhiteList []string `mapstructure:"event_white_list,omitempty"`
}