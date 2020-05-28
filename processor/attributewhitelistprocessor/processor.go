package attributewhitelistprocessor

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"regexp"
)

type attributewhitelistprocessor struct {
	nextConsumer    consumer.TraceConsumer
	regexpWhiteList []*regexp.Regexp
}

func newTraceProcesor(
	params component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	config Config) (component.TraceProcessor, error) {
	var regexlist []*regexp.Regexp
	for _, whitelist := range config.AttributeWhiteList {
		r, err := regexp.Compile(whitelist)
		if err != nil{
			return nil, err
		}
		regexlist = append(regexlist, r)
	}

	proc := &attributewhitelistprocessor{
		nextConsumer:    nextConsumer,
		regexpWhiteList: regexlist,
	}
	return proc, nil
}


func (wp *attributewhitelistprocessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	// drop tags when they don't match the white list
	resourcespans := td.ResourceSpans()
	for i := 0; i < resourcespans.Len(); i++ {
		rs := resourcespans.At(i)
		if rs.IsNil() {
			continue
		}
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			if ils.IsNil() {
				continue
			}
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				s := spans.At(k)
				if s.IsNil() {
					continue
				}
				attributes := s.Attributes()

				var attributesToDelete []string
				findAttrsToDelete := func(k string, v pdata.AttributeValue) {
					if wp.shouldDeleteTag(k) {
						attributesToDelete = append(attributesToDelete, k)
					}
				}
				attributes.ForEach(findAttrsToDelete)
				for _, k := range attributesToDelete {
					attributes.Delete(k)
				}
			}
		}
	}
	return wp.nextConsumer.ConsumeTraces(ctx, td)
}

func (wp *attributewhitelistprocessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (wp *attributewhitelistprocessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (wp *attributewhitelistprocessor) Shutdown(context.Context) error {
	return nil
}

func (wp *attributewhitelistprocessor) shouldDeleteTag(tagName string) bool {
	for _, r := range wp.regexpWhiteList {
		if r.MatchString(tagName) {
			return false
		}
	}
	return true
}