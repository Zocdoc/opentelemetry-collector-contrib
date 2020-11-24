// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package honeycombexporter

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type honeycombData struct {
	Data map[string]interface{} `json:"data"`
}

func testingServer(callback func(data []honeycombData)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		uncompressed, err := zstd.NewReader(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer req.Body.Close()
		b, err := ioutil.ReadAll(uncompressed)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		var data []honeycombData
		err = json.Unmarshal(b, &data)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		callback(data)
		rw.Write([]byte(`OK`))
	}))
}

func testTraceExporter(td pdata.Traces, t *testing.T, cfg *Config) []honeycombData {
	var got []honeycombData
	server := testingServer(func(data []honeycombData) {
		got = append(got, data...)
	})
	defer server.Close()

	cfg.APIURL = server.URL

	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	exporter, err := createTraceExporter(context.Background(), params, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = exporter.ConsumeTraces(ctx, td)
	require.NoError(t, err)
	exporter.Shutdown(context.Background())

	return got
}

func baseConfig() *Config {
	return &Config{
		APIKey:              "test",
		Dataset:             "test",
		Debug:               false,
		SampleRateAttribute: "",
	}
}

func TestExporter(t *testing.T) {
	td := consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "test_service"},
			Attributes: map[string]string{
				"A": "B",
			},
		},
		Resource: &resourcepb.Resource{
			Type: "foobar",
			Labels: map[string]string{
				"B": "C",
			},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x02},
				Name:                    &tracepb.TruncatableString{Value: "root"},
				Kind:                    tracepb.Span_SERVER,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"span_attr_name": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "Span Attribute"},
							},
						},
					},
				},
				Resource: &resourcepb.Resource{
					Type: "override",
					Labels: map[string]string{
						"B": "D",
					},
				},
				TimeEvents: &tracepb.Span_TimeEvents{
					TimeEvent: []*tracepb.Span_TimeEvent{
						{
							Time: &timestamppb.Timestamp{
								Seconds: 0,
								Nanos:   0,
							},
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Description: &tracepb.TruncatableString{Value: "Some Description"},
									Attributes: &tracepb.Span_Attributes{
										AttributeMap: map[string]*tracepb.AttributeValue{
											"attribute_name": {
												Value: &tracepb.AttributeValue_StringValue{
													StringValue: &tracepb.TruncatableString{Value: "Hello MessageEvent"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x03},
				ParentSpanId:            []byte{0x02},
				Name:                    &tracepb.TruncatableString{Value: "client"},
				Kind:                    tracepb.Span_CLIENT,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
				Links: &tracepb.Span_Links{
					Link: []*tracepb.Span_Link{
						{
							TraceId: []byte{0x04},
							SpanId:  []byte{0x05},
							Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
							Attributes: &tracepb.Span_Attributes{
								AttributeMap: map[string]*tracepb.AttributeValue{
									"span_link_attr": {
										Value: &tracepb.AttributeValue_IntValue{
											IntValue: 12345,
										},
									},
								},
							},
						},
					},
					DroppedLinksCount: 0,
				},
			},
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x04},
				ParentSpanId:            []byte{0x03},
				Name:                    &tracepb.TruncatableString{Value: "server"},
				Kind:                    tracepb.Span_SERVER,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: false},
			},
		},
	}
	got := testTraceExporter(internaldata.OCToTraceData(td), t, baseConfig())
	want := []honeycombData{
		{
			Data: map[string]interface{}{
				"meta.annotation_type": "link",
				"span_link_attr":       float64(12345),
				"trace.trace_id":       "01000000000000000000000000000000",
				"trace.parent_id":      "0300000000000000",
				"trace.link.span_id":   "0500000000000000",
				"trace.link.trace_id":  "04000000000000000000000000000000",
			},
		},
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"name":                                   "client",
				"service.name":                           "test_service",
				"span_kind":                              "client",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.parent_id":                        "0200000000000000",
				"trace.span_id":                          "0300000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.resourcetype":                "foobar",
				"opencensus.same_process_as_parent_span": true,
				"A":                                      "B",
				"B":                                      "C",
			},
		},
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"name":                                   "server",
				"service.name":                           "test_service",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.parent_id":                        "0300000000000000",
				"trace.span_id":                          "0400000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"A":                                      "B",
				"B":                                      "C",
				"opencensus.resourcetype":                "foobar",
				"opencensus.same_process_as_parent_span": false,
			},
		},
		{
			Data: map[string]interface{}{
				"A":                       "B",
				"B":                       "D",
				"attribute_name":          "Hello MessageEvent",
				"meta.annotation_type":    "span_event",
				"name":                    "Some Description",
				"opencensus.resourcetype": "override",
				"service.name":            "test_service",
				"trace.parent_id":         "0200000000000000",
				"trace.parent_name":       "root",
				"trace.trace_id":          "01000000000000000000000000000000",
			},
		},
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"name":                                   "root",
				"service.name":                           "test_service",
				"span_attr_name":                         "Span Attribute",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"A":                                      "B",
				"B":                                      "D",
				"opencensus.resourcetype":                "override",
				"opencensus.same_process_as_parent_span": true,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("otel span: (-want +got):\n%s", diff)
	}
}

func TestSampleRateAttribute(t *testing.T) {
	td := consumerdata.TraceData{
		Node: nil,
		Spans: []*tracepb.Span{
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x02},
				Name:                    &tracepb.TruncatableString{Value: "root"},
				Kind:                    tracepb.Span_SERVER,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"some_attribute": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "A value"},
							},
						},
						"hc.sample.rate": {
							Value: &tracepb.AttributeValue_IntValue{
								IntValue: 13,
							},
						},
					},
				},
			},
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x02},
				Name:                    &tracepb.TruncatableString{Value: "root"},
				Kind:                    tracepb.Span_SERVER,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"no_sample_rate": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "gets_default"},
							},
						},
					},
				},
			},
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x02},
				Name:                    &tracepb.TruncatableString{Value: "root"},
				Kind:                    tracepb.Span_SERVER,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"hc.sample.rate": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "wrong_type"},
							},
						},
					},
				},
			},
		},
	}

	cfg := baseConfig()
	cfg.SampleRateAttribute = "hc.sample.rate"

	got := testTraceExporter(internaldata.OCToTraceData(td), t, cfg)

	want := []honeycombData{
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"hc.sample.rate":                         float64(13),
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.same_process_as_parent_span": true,
				"some_attribute":                         "A value",
			},
		},
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.same_process_as_parent_span": true,
				"no_sample_rate":                         "gets_default",
			},
		},
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"hc.sample.rate":                         "wrong_type",
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.same_process_as_parent_span": true,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("otel span: (-want +got):\n%s", diff)
	}
}

func TestEmptyNode(t *testing.T) {
	td := consumerdata.TraceData{
		Node: nil,
		Spans: []*tracepb.Span{
			{
				TraceId:                 []byte{0x01},
				SpanId:                  []byte{0x02},
				Name:                    &tracepb.TruncatableString{Value: "root"},
				Kind:                    tracepb.Span_SERVER,
				SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
			},
		},
	}

	got := testTraceExporter(internaldata.OCToTraceData(td), t, baseConfig())

	want := []honeycombData{
		{
			Data: map[string]interface{}{
				"duration_ms":                            float64(0),
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.same_process_as_parent_span": true,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("otel span: (-want +got):\n%s", diff)
	}
}

type testNodeCase struct {
	name       string
	identifier *commonpb.ProcessIdentifier
	expected   map[string]interface{}
}

func TestNode(t *testing.T) {

	testcases := []testNodeCase{
		{
			name: "all_information",
			identifier: &commonpb.ProcessIdentifier{
				HostName: "my-host",
				Pid:      123,
				StartTimestamp: &timestamppb.Timestamp{
					Seconds: 1599596112,
					Nanos:   0,
				},
			},
			expected: map[string]interface{}{
				"B":                                      "C",
				"duration_ms":                            float64(0),
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.resourcetype":                "container",
				"opencensus.same_process_as_parent_span": true,
				"opencensus.starttime":                   "2020-09-08T20:15:12Z",
				"host.name":                              "my-host",
				"opencensus.pid":                         float64(123),
				"service.name":                           "test_service",
			},
		},
		{
			name: "missing_pid_and_time",
			identifier: &commonpb.ProcessIdentifier{
				HostName: "my-host",
			},
			expected: map[string]interface{}{
				"B":                                      "C",
				"duration_ms":                            float64(0),
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.resourcetype":                "container",
				"opencensus.same_process_as_parent_span": true,
				"host.name":                              "my-host",
				"service.name":                           "test_service",
			},
		},
		{
			name:       "nil_identifier",
			identifier: nil,
			expected: map[string]interface{}{
				"B":                                      "C",
				"duration_ms":                            float64(0),
				"name":                                   "root",
				"span_kind":                              "server",
				"status.code":                            float64(0),
				"status.message":                         "OK",
				"trace.span_id":                          "0200000000000000",
				"trace.trace_id":                         "01000000000000000000000000000000",
				"opencensus.resourcetype":                "container",
				"opencensus.same_process_as_parent_span": true,
				"service.name":                           "test_service",
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			td := consumerdata.TraceData{
				Node: &commonpb.Node{
					ServiceInfo: &commonpb.ServiceInfo{Name: "test_service"},
					Identifier:  test.identifier,
				},
				Resource: &resourcepb.Resource{
					Type: "container",
					Labels: map[string]string{
						"B": "C",
					},
				},
				Spans: []*tracepb.Span{
					{
						TraceId:                 []byte{0x01},
						SpanId:                  []byte{0x02},
						Name:                    &tracepb.TruncatableString{Value: "root"},
						Kind:                    tracepb.Span_SERVER,
						SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
					},
				},
			}

			got := testTraceExporter(internaldata.OCToTraceData(td), t, baseConfig())

			want := []honeycombData{
				{
					Data: test.expected,
				},
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("otel span: (-want +got):\n%s", diff)
			}
		})
	}

}
