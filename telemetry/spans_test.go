// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

func BenchmarkSpansJSON(b *testing.B) {
	// This benchmark tests the overhead of turning spans into JSON.
	batch := &SpanBatch{}
	numSpans := 10 * 1000
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	for i := 0; i < numSpans; i++ {
		batch.Spans = append(batch.Spans, Span{
			ID:             "myid",
			TraceID:        "mytraceid",
			Name:           "myname",
			ParentID:       "myparent",
			Timestamp:      tm,
			Duration:       2 * time.Second,
			ServiceName:    "myentity",
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if bts := batch.Bytes(); nil == bts || len(bts) == 0 {
			b.Fatal(string(bts))
		}
	}
}

func testHarvesterSpans(t testing.TB, h *Harvester, expect string) {
	reqs := h.swapOutSpans()
	if nil == reqs {
		if expect != "null" {
			t.Error("nil spans", expect)
		}
		return
	}
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	if u := reqs[0].URL.String(); u != defaultSpanURL {
		t.Fatal(u)
	}
	bodyReader, _ := reqs[0].GetBody()
	compressedBytes, _ := ioutil.ReadAll(bodyReader)
	uncompressedBytes, _ := internal.Uncompress(compressedBytes)
	js := string(uncompressedBytes)
	actual := string(js)
	if th, ok := t.(interface{ Helper() }); ok {
		th.Helper()
	}
	compactExpect := compactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestSpan(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, _ := NewHarvester(configTesting)
	h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes: map[string]interface{}{
			"zip": "zap",
		},
	})
	expect := `[{"spans":[{
		"id":"myid",
		"trace.id":"mytraceid",
		"timestamp":1417136460000,
		"attributes": {
			"name":"myname",
			"parent.id":"myparent",
			"duration.ms":2000,
			"service.name":"myentity",
			"zip":"zap"
		}
	}]}]`
	testHarvesterSpans(t, h, expect)
}

func TestSpanInvalidAttribute(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, _ := NewHarvester(configTesting)
	h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes: map[string]interface{}{
			"weird-things-get-turned-to-strings": struct{}{},
			"nil-gets-removed":                   nil,
		},
	})
	expect := `[{"spans":[{
		"id":"myid",
		"trace.id":"mytraceid",
		"timestamp":1417136460000,
		"attributes": {
			"name":"myname",
			"parent.id":"myparent",
			"duration.ms":2000,
			"service.name":"myentity",
			"weird-things-get-turned-to-strings":"struct {}"
		}
	}]}]`
	testHarvesterSpans(t, h, expect)
}

func TestRecordSpanNilHarvester(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	var h *Harvester
	err := h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes: map[string]interface{}{
			"zip": "zap",
			"zop": 123,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSpanWithEvents(t *testing.T) {
	tm := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	h, _ := NewHarvester(configTesting)
	h.RecordSpan(Span{
		ID:          "myid",
		TraceID:     "mytraceid",
		Name:        "myname",
		ParentID:    "myparent",
		Timestamp:   tm,
		Duration:    2 * time.Second,
		ServiceName: "myentity",
		Attributes:  map[string]interface{}{},
		Events: []Event{
			{
				EventType: "exception",
				Timestamp: tm,
				Attributes: map[string]interface{}{
					"exception.message": "Everything is fine!",
				},
			},
		},
	})
	expect := `[{"spans":[{
		"id":"myid",
		"trace.id":"mytraceid",
		"timestamp":1417136460000,
		"attributes": {
			"name":"myname",
			"parent.id":"myparent",
			"duration.ms":2000,
			"service.name":"myentity"
		},
		"events": [
			{
				"name": "exception",
				"timestamp": 1417136460000,
				"attributes": {
					"exception.message": "Everything is fine!"
				}
			}
		]
	}]}]`
	testHarvesterSpans(t, h, expect)
}
