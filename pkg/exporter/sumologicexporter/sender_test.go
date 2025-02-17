// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicexporter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

type senderTest struct {
	srv *httptest.Server
	s   *sender
}

// prepareSenderTest prepares sender test environment.
// The enclosed httptest.Server is closed automatically using test.Cleanup.
func prepareSenderTest(t *testing.T, cb []func(w http.ResponseWriter, req *http.Request)) *senderTest {
	var reqCounter int32
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if len(cb) == 0 {
			return
		}

		if c := int(atomic.LoadInt32(&reqCounter)); assert.Greater(t, len(cb), c) {
			cb[c](w, req)
			atomic.AddInt32(&reqCounter, 1)
		}
	}))
	t.Cleanup(func() { testServer.Close() })

	cfg := createDefaultConfig().(*Config)
	cfg.CompressEncoding = NoCompression
	cfg.HTTPClientSettings.Endpoint = testServer.URL
	cfg.LogFormat = TextFormat
	cfg.MetricFormat = Carbon2Format
	cfg.MaxRequestBodySize = 20_971_520

	f, err := newFilter(cfg.MetadataAttributes)
	require.NoError(t, err)

	c, err := newCompressor(cfg.CompressEncoding)
	require.NoError(t, err)

	pf, err := newPrometheusFormatter()
	require.NoError(t, err)

	gf, err := newGraphiteFormatter(cfg.GraphiteTemplate)
	require.NoError(t, err)

	return &senderTest{
		srv: testServer,
		s: newSender(
			cfg,
			&http.Client{
				Timeout: cfg.HTTPClientSettings.Timeout,
			},
			f,
			sourceFormats{
				host:     getTestSourceFormat(t, "source_host"),
				category: getTestSourceFormat(t, "source_category"),
				name:     getTestSourceFormat(t, "source_name"),
			},
			c,
			pf,
			gf,
			"",
			"",
			"",
		),
	}
}

// prepareOTLPSenderTest prepares sender test environment.
// The enclosed httptest.Server is closed automatically using test.Cleanup.
func prepareOTLPSenderTest(t *testing.T, cb []func(w http.ResponseWriter, req *http.Request)) *senderTest {
	var reqCounter int32
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if len(cb) == 0 {
			return
		}

		if c := int(atomic.LoadInt32(&reqCounter)); assert.Greater(t, len(cb), c) {
			cb[c](w, req)
			atomic.AddInt32(&reqCounter, 1)
		}
	}))
	t.Cleanup(func() { testServer.Close() })

	cfg := createDefaultConfig().(*Config)
	cfg.CompressEncoding = NoCompression
	cfg.HTTPClientSettings.Endpoint = testServer.URL

	f, err := newFilter(cfg.MetadataAttributes)
	require.NoError(t, err)

	c, err := newCompressor(cfg.CompressEncoding)
	require.NoError(t, err)

	pf, err := newPrometheusFormatter()
	require.NoError(t, err)

	gf, err := newGraphiteFormatter(cfg.GraphiteTemplate)
	require.NoError(t, err)

	return &senderTest{
		srv: testServer,
		s: newSender(
			cfg,
			&http.Client{
				Timeout: cfg.HTTPClientSettings.Timeout,
			},
			f,
			sourceFormats{
				host:     getTestSourceFormat(t, "source_host"),
				category: getTestSourceFormat(t, "source_category"),
				name:     getTestSourceFormat(t, "source_name"),
			},
			c,
			pf,
			gf,
			testServer.URL,
			testServer.URL,
			testServer.URL,
		),
	}
}

func extractBody(t *testing.T, req *http.Request) string {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, req.Body)
	require.NoError(t, err)
	return buf.String()
}

func exampleLog() []pdata.LogRecord {
	buffer := make([]pdata.LogRecord, 1)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].Body().SetStringVal("Example log")

	return buffer
}

func exampleTwoLogs() []pdata.LogRecord {
	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].Body().SetStringVal("Example log")
	buffer[0].Attributes().InsertString("key1", "value1")
	buffer[0].Attributes().InsertString("key2", "value2")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].Body().SetStringVal("Another example log")
	buffer[1].Attributes().InsertString("key1", "value1")
	buffer[1].Attributes().InsertString("key2", "value2")

	return buffer
}

func exampleTwoDifferentLogs() []pdata.LogRecord {
	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].Body().SetStringVal("Example log")
	buffer[0].Attributes().InsertString("key1", "value1")
	buffer[0].Attributes().InsertString("key2", "value2")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].Body().SetStringVal("Another example log")
	buffer[1].Attributes().InsertString("key3", "value3")
	buffer[1].Attributes().InsertString("key4", "value4")

	return buffer
}

func exampleMultitypeLogs() []pdata.LogRecord {
	buffer := make([]pdata.LogRecord, 2)

	attVal := pdata.NewAttributeValueMap()
	attMap := attVal.MapVal()
	attMap.InsertString("lk1", "lv1")
	attMap.InsertInt("lk2", 13)

	buffer[0] = pdata.NewLogRecord()
	attVal.CopyTo(buffer[0].Body())

	buffer[0].Attributes().InsertString("key1", "value1")
	buffer[0].Attributes().InsertString("key2", "value2")

	buffer[1] = pdata.NewLogRecord()

	attVal = pdata.NewAttributeValueArray()
	attArr := attVal.ArrayVal()
	strVal := pdata.NewAttributeValueNull()
	strVal.SetStringVal("lv2")
	intVal := pdata.NewAttributeValueNull()
	intVal.SetIntVal(13)

	strVal.CopyTo(attArr.AppendEmpty())
	intVal.CopyTo(attArr.AppendEmpty())

	attVal.CopyTo(buffer[1].Body())
	buffer[1].Attributes().InsertString("key1", "value1")
	buffer[1].Attributes().InsertString("key2", "value2")

	return buffer
}

func exampleTrace() pdata.Traces {
	td := pdata.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().UpsertString("hostname", "testHost")
	rs.Resource().Attributes().UpsertString("_sourceHost", "source_host")
	rs.Resource().Attributes().UpsertString("_sourceName", "source_name")
	rs.Resource().Attributes().UpsertString("_sourceCategory", "source_category")
	span := rs.InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pdata.NewTraceID([16]byte{0x5B, 0x8E, 0xFF, 0xF7, 0x98, 0x3, 0x81, 0x3, 0xD2, 0x69, 0xB6, 0x33, 0x81, 0x3F, 0xC6, 0xC}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x73}))
	span.SetName("testSpan")
	span.SetStartTimestamp(1544712660000000000)
	span.SetEndTimestamp(1544712661000000000)
	span.Attributes().UpsertInt("attr1", 55)
	return td
}

func TestSendTrace(t *testing.T) {
	tracesMarshaler = otlp.NewProtobufTracesMarshaler()
	td := exampleTrace()
	traceBody, err := tracesMarshaler.MarshalTraces(td)
	assert.NoError(t, err)
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, string(traceBody), body)
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/x-protobuf", req.Header.Get("Content-Type"))
		},
	})

	err = test.s.sendTraces(context.Background(), td, fieldsFromMap(map[string]string{}))
	assert.NoError(t, err)

}

func TestSendLogs(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, "Example log\nAnother example log", body)
			assert.Equal(t, "key1=value, key2=value2", req.Header.Get("X-Sumo-Fields"))
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		},
	})

	test.s.logBuffer = exampleTwoLogs()

	_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "value", "key2": "value2"}))
	assert.NoError(t, err)
}

func TestSendLogsMultitype(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `{"lk1":"lv1","lk2":13}
["lv2",13]`
			assert.Equal(t, expected, body)
			assert.Equal(t, "key1=value, key2=value2", req.Header.Get("X-Sumo-Fields"))
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		},
	})

	test.s.logBuffer = exampleMultitypeLogs()

	_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "value", "key2": "value2"}))
	assert.NoError(t, err)
}

func TestSendLogsSplit(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, "Example log", body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, "Another example log", body)
		},
	})
	test.s.config.MaxRequestBodySize = 10
	test.s.logBuffer = exampleTwoLogs()

	_, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.NoError(t, err)
}
func TestSendLogsSplitFailedOne(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(500)

			body := extractBody(t, req)
			assert.Equal(t, "Example log", body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, "Another example log", body)
		},
	})
	test.s.config.MaxRequestBodySize = 10
	test.s.config.LogFormat = TextFormat
	test.s.logBuffer = exampleTwoLogs()

	dropped, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, test.s.logBuffer[0:1], dropped)
}

func TestSendLogsSplitFailedAll(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(500)

			body := extractBody(t, req)
			assert.Equal(t, "Example log", body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(404)

			body := extractBody(t, req)
			assert.Equal(t, "Another example log", body)
		},
	})
	test.s.config.MaxRequestBodySize = 10
	test.s.config.LogFormat = TextFormat
	test.s.logBuffer = exampleTwoLogs()

	dropped, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(
		t,
		err,
		"[error during sending data: 500 Internal Server Error; error during sending data: 404 Not Found]",
	)
	assert.Equal(t, test.s.logBuffer[0:2], dropped)
}

func TestSendLogsJson(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `{"key1":"value1","key2":"value2","log":"Example log"}
{"key1":"value1","key2":"value2","log":"Another example log"}`
			assert.Equal(t, expected, body)
			assert.Equal(t, "key=value", req.Header.Get("X-Sumo-Fields"))
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		},
	})
	test.s.config.LogFormat = JSONFormat
	test.s.logBuffer = exampleTwoLogs()

	_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key": "value"}))
	assert.NoError(t, err)
}

func TestSendLogsJsonMultitype(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `{"key1":"value1","key2":"value2","log":{"lk1":"lv1","lk2":13}}
{"key1":"value1","key2":"value2","log":["lv2",13]}`
			assert.Equal(t, expected, body)
			assert.Equal(t, "key=value", req.Header.Get("X-Sumo-Fields"))
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		},
	})
	test.s.config.LogFormat = JSONFormat
	test.s.logBuffer = exampleMultitypeLogs()

	_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key": "value"}))
	assert.NoError(t, err)
}

func TestSendLogsJsonSplit(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, `{"key1":"value1","key2":"value2","log":"Example log"}`, body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, `{"key1":"value1","key2":"value2","log":"Another example log"}`, body)
		},
	})
	test.s.config.LogFormat = JSONFormat
	test.s.config.MaxRequestBodySize = 10
	test.s.logBuffer = exampleTwoLogs()

	_, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.NoError(t, err)
}

func TestSendLogsJsonSplitFailedOne(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(500)

			body := extractBody(t, req)
			assert.Equal(t, `{"key1":"value1","key2":"value2","log":"Example log"}`, body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, `{"key1":"value1","key2":"value2","log":"Another example log"}`, body)
		},
	})
	test.s.config.LogFormat = JSONFormat
	test.s.config.MaxRequestBodySize = 10
	test.s.logBuffer = exampleTwoLogs()

	dropped, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, test.s.logBuffer[0:1], dropped)
}

func TestSendLogsJsonSplitFailedAll(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(500)

			body := extractBody(t, req)
			assert.Equal(t, `{"key1":"value1","key2":"value2","log":"Example log"}`, body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(404)

			body := extractBody(t, req)
			assert.Equal(t, `{"key1":"value1","key2":"value2","log":"Another example log"}`, body)
		},
	})
	test.s.config.LogFormat = JSONFormat
	test.s.config.MaxRequestBodySize = 10
	test.s.logBuffer = exampleTwoLogs()

	dropped, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(
		t,
		err,
		"[error during sending data: 500 Internal Server Error; error during sending data: 404 Not Found]",
	)
	assert.Equal(t, test.s.logBuffer[0:2], dropped)
}

func TestSendLogsUnexpectedFormat(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
		},
	})
	test.s.config.LogFormat = "dummy"
	logs := exampleTwoLogs()
	test.s.logBuffer = logs

	dropped, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.Error(t, err)
	assert.Equal(t, logs, dropped)
}

func TestSendLogsOTLP(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			//nolint:lll
			assert.Equal(t, "\n\xe2\x01\nb\n\x1c\n\v_sourceHost\x12\r\n\vsource_host\n\x1c\n\v_sourceName\x12\r\n\vsource_name\n$\n\x0f_sourceCategory\x12\x11\n\x0fsource_category\x12|\n\x00\x127*\r\n\vExample log2\x10\n\x04key1\x12\b\n\x06value12\x10\n\x04key2\x12\b\n\x06value2J\x00R\x00\x12?*\x15\n\x13Another example log2\x10\n\x04key1\x12\b\n\x06value12\x10\n\x04key2\x12\b\n\x06value2J\x00R\x00", body)
			assert.Equal(t, "key1=value, key2=value2", req.Header.Get("X-Sumo-Fields"))
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/x-protobuf", req.Header.Get("Content-Type"))
		},
	})

	test.s.logBuffer = exampleTwoLogs()
	test.s.config.LogFormat = "otlp"

	_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "value", "key2": "value2"}))
	assert.NoError(t, err)
}

func TestOverrideSourceName(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
			func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "Test source name/test_name", req.Header.Get("X-Sumo-Name"))
			},
		})

		test.s.sources.name = getTestSourceFormat(t, "Test source name/%{key1}")
		test.s.logBuffer = exampleLog()

		_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "test_name"}))
		assert.NoError(t, err)
	})

	t.Run("otlp", func(t *testing.T) {
		test := prepareOTLPSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
			func(w http.ResponseWriter, req *http.Request) {
				unmarshaller := otlp.NewProtobufLogsUnmarshaler()
				b, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				l, err := unmarshaller.UnmarshalLogs(b)
				require.NoError(t, err)

				require.Equal(t, l.ResourceLogs().Len(), 1)
				sourceCategory, ok := l.ResourceLogs().At(0).Resource().Attributes().Get("_sourceName")
				require.True(t, ok)
				require.Equal(t, pdata.AttributeValueTypeString, sourceCategory.Type())
				require.Equal(t, "Test source name/test_name", sourceCategory.StringVal())
			},
		})

		test.s.sources.name = getTestSourceFormat(t, "Test source name/%{key1}")
		test.s.logBuffer = exampleLog()

		_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "test_name"}))
		assert.NoError(t, err)
	})
}

func TestOverrideSourceCategory(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
			func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "Test source category/test_name", req.Header.Get("X-Sumo-Category"))
			},
		})

		test.s.sources.category = getTestSourceFormat(t, "Test source category/%{key1}")
		test.s.logBuffer = exampleLog()

		_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "test_name"}))
		assert.NoError(t, err)
	})

	t.Run("otlp", func(t *testing.T) {
		test := prepareOTLPSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
			func(w http.ResponseWriter, req *http.Request) {
				unmarshaller := otlp.NewProtobufLogsUnmarshaler()
				b, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				l, err := unmarshaller.UnmarshalLogs(b)
				require.NoError(t, err)

				require.Equal(t, l.ResourceLogs().Len(), 1)
				sourceCategory, ok := l.ResourceLogs().At(0).Resource().Attributes().Get("_sourceCategory")
				require.True(t, ok)
				require.Equal(t, pdata.AttributeValueTypeString, sourceCategory.Type())
				require.Equal(t, "Test source category/test_name", sourceCategory.StringVal())
			},
		})

		test.s.sources.category = getTestSourceFormat(t, "Test source category/%{key1}")
		test.s.logBuffer = exampleLog()

		_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "test_name"}))
		assert.NoError(t, err)
	})
}

func TestOverrideSourceHost(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
			func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "Test source host/test_name", req.Header.Get("X-Sumo-Host"))
			},
		})

		test.s.sources.host = getTestSourceFormat(t, "Test source host/%{key1}")
		test.s.logBuffer = exampleLog()

		_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "test_name"}))
		assert.NoError(t, err)
	})

	t.Run("otlp", func(t *testing.T) {
		test := prepareOTLPSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
			func(w http.ResponseWriter, req *http.Request) {
				unmarshaller := otlp.NewProtobufLogsUnmarshaler()
				b, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				l, err := unmarshaller.UnmarshalLogs(b)
				require.NoError(t, err)

				require.Equal(t, l.ResourceLogs().Len(), 1)
				sourceHost, ok := l.ResourceLogs().At(0).Resource().Attributes().Get("_sourceHost")
				require.True(t, ok)
				require.Equal(t, pdata.AttributeValueTypeString, sourceHost.Type())
				require.Equal(t, "Test source host/test_name", sourceHost.StringVal())
			},
		})

		test.s.sources.host = getTestSourceFormat(t, "Test source host/%{key1}")
		test.s.logBuffer = exampleLog()

		_, err := test.s.sendLogs(context.Background(), fieldsFromMap(map[string]string{"key1": "test_name"}))
		assert.NoError(t, err)
	})
}

func TestLogsBuffer(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	assert.Equal(t, test.s.countLogs(), 0)
	logs := exampleTwoLogs()

	droppedLogs, err := test.s.batchLog(context.Background(), logs[0], newFields(pdata.NewAttributeMap()))
	require.NoError(t, err)
	assert.Nil(t, droppedLogs)
	assert.Equal(t, 1, test.s.countLogs())
	assert.Equal(t, []pdata.LogRecord{logs[0]}, test.s.logBuffer)

	droppedLogs, err = test.s.batchLog(context.Background(), logs[1], newFields(pdata.NewAttributeMap()))
	require.NoError(t, err)
	assert.Nil(t, droppedLogs)
	assert.Equal(t, 2, test.s.countLogs())
	assert.Equal(t, logs, test.s.logBuffer)

	test.s.cleanLogsBuffer()
	assert.Equal(t, 0, test.s.countLogs())
	assert.Equal(t, []pdata.LogRecord{}, test.s.logBuffer)
}

func TestInvalidEndpoint(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.config.HTTPClientSettings.Endpoint = ":"
	test.s.logBuffer = exampleLog()

	_, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, `parse ":": missing protocol scheme`)
}

func TestInvalidPostRequest(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.config.HTTPClientSettings.Endpoint = ""
	test.s.logBuffer = exampleLog()

	_, err := test.s.sendLogs(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, `Post "": unsupported protocol scheme ""`)
}

func TestLogsBufferOverflow(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.config.HTTPClientSettings.Endpoint = ":"
	log := exampleLog()
	flds := newFields(pdata.NewAttributeMap())

	for test.s.countLogs() < maxBufferSize-1 {
		_, err := test.s.batchLog(context.Background(), log[0], flds)
		require.NoError(t, err)
	}

	_, err := test.s.batchLog(context.Background(), log[0], flds)
	assert.EqualError(t, err, `parse ":": missing protocol scheme`)
	assert.Equal(t, 0, test.s.countLogs())
}

func TestInvalidMetricFormat(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.config.MetricFormat = "invalid"

	err := test.s.send(context.Background(), MetricsPipeline, strings.NewReader(""), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, `unsupported metrics format: invalid`)
}

func TestInvalidPipeline(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	err := test.s.send(context.Background(), "invalidPipeline", strings.NewReader(""), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, `unexpected pipeline: invalidPipeline`)
}

func TestSendCompressGzip(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)
			if _, err := res.Write([]byte("")); err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				assert.FailNow(t, "err: %v", err)
				return
			}
			body := decodeGzip(t, req.Body)
			assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
			assert.Equal(t, "Some example log", body)
		},
	})

	test.s.config.CompressEncoding = "gzip"

	c, err := newCompressor("gzip")
	require.NoError(t, err)

	test.s.compressor = c
	reader := strings.NewReader("Some example log")

	err = test.s.send(context.Background(), LogsPipeline, reader, newFields(pdata.NewAttributeMap()))
	require.NoError(t, err)
}

func TestSendCompressDeflate(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)

			if _, err := res.Write([]byte("")); err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				assert.FailNow(t, "err: %v", err)
				return
			}
			body := decodeDeflate(t, req.Body)
			assert.Equal(t, "deflate", req.Header.Get("Content-Encoding"))
			assert.Equal(t, "Some example log", body)
		},
	})

	test.s.config.CompressEncoding = "deflate"

	c, err := newCompressor("deflate")
	require.NoError(t, err)

	test.s.compressor = c
	reader := strings.NewReader("Some example log")

	err = test.s.send(context.Background(), LogsPipeline, reader, newFields(pdata.NewAttributeMap()))
	require.NoError(t, err)
}

func TestCompressionError(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.compressor = getTestCompressor(errors.New("read error"), nil)
	reader := strings.NewReader("Some example log")

	err := test.s.send(context.Background(), LogsPipeline, reader, newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, "read error")
}

func TestInvalidContentEncoding(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.config.CompressEncoding = "test"
	reader := strings.NewReader("Some example log")

	err := test.s.send(context.Background(), LogsPipeline, reader, newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, "invalid content encoding: test")
}

func TestSendMetrics(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `test_metric_data{test="test_value",test2="second_value"} 14500 1605534165000
gauge_metric_name{foo="bar",remote_name="156920",url="http://example_url"} 124 1608124661166
gauge_metric_name{foo="bar",remote_name="156955",url="http://another_url"} 245 1608124662166`
			assert.Equal(t, expected, body)
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/vnd.sumologic.prometheus", req.Header.Get("Content-Type"))
		},
	})
	flds := fieldsFromMap(map[string]string{
		"key1": "value",
		"key2": "value2",
	})

	test.s.config.MetricFormat = PrometheusFormat
	test.s.metricBuffer = []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}
	_, err := test.s.sendMetrics(context.Background(), flds)
	assert.NoError(t, err)
}

func TestSendMetricsSplit(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `test_metric_data{test="test_value",test2="second_value"} 14500 1605534165000`
			assert.Equal(t, expected, body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `gauge_metric_name{foo="bar",remote_name="156920",url="http://example_url"} 124 1608124661166
gauge_metric_name{foo="bar",remote_name="156955",url="http://another_url"} 245 1608124662166`
			assert.Equal(t, expected, body)
		},
	})
	test.s.config.MaxRequestBodySize = 10
	test.s.config.MetricFormat = PrometheusFormat
	test.s.metricBuffer = []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}

	_, err := test.s.sendMetrics(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.NoError(t, err)
}

func TestSendMetricsSplitFailedOne(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(500)

			body := extractBody(t, req)
			expected := `test_metric_data{test="test_value",test2="second_value"} 14500 1605534165000`
			assert.Equal(t, expected, body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `gauge_metric_name{foo="bar",remote_name="156920",url="http://example_url"} 124 1608124661166
gauge_metric_name{foo="bar",remote_name="156955",url="http://another_url"} 245 1608124662166`
			assert.Equal(t, expected, body)
		},
	})
	test.s.config.MaxRequestBodySize = 10
	test.s.config.MetricFormat = PrometheusFormat
	test.s.metricBuffer = []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}

	dropped, err := test.s.sendMetrics(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, test.s.metricBuffer[0:1], dropped)
}

func TestSendMetricsSplitFailedAll(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(500)

			body := extractBody(t, req)
			expected := `test_metric_data{test="test_value",test2="second_value"} 14500 1605534165000`
			assert.Equal(t, expected, body)
		},
		func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(404)

			body := extractBody(t, req)
			expected := `gauge_metric_name{foo="bar",remote_name="156920",url="http://example_url"} 124 1608124661166
gauge_metric_name{foo="bar",remote_name="156955",url="http://another_url"} 245 1608124662166`
			assert.Equal(t, expected, body)
		},
	})
	test.s.config.MaxRequestBodySize = 10
	test.s.config.MetricFormat = PrometheusFormat
	test.s.metricBuffer = []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}

	dropped, err := test.s.sendMetrics(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(
		t,
		err,
		"[error during sending data: 500 Internal Server Error; error during sending data: 404 Not Found]",
	)
	assert.Equal(t, test.s.metricBuffer[0:2], dropped)
}

func TestSendMetricsUnexpectedFormat(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
		},
	})
	test.s.config.MetricFormat = "invalid"
	metrics := []metricPair{
		exampleIntMetric(),
	}
	test.s.metricBuffer = metrics

	dropped, err := test.s.sendMetrics(context.Background(), newFields(pdata.NewAttributeMap()))
	assert.EqualError(t, err, "unexpected metric format: invalid")
	assert.Equal(t, dropped, metrics)
}

func TestMetricsBuffer(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	assert.Equal(t, test.s.countMetrics(), 0)
	metrics := []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}

	droppedMetrics, err := test.s.batchMetric(context.Background(), metrics[0], newFields(pdata.NewAttributeMap()))
	require.NoError(t, err)
	assert.Nil(t, droppedMetrics)
	assert.Equal(t, 1, test.s.countMetrics())
	assert.Equal(t, metrics[0:1], test.s.metricBuffer)

	droppedMetrics, err = test.s.batchMetric(context.Background(), metrics[1], newFields(pdata.NewAttributeMap()))
	require.NoError(t, err)
	assert.Nil(t, droppedMetrics)
	assert.Equal(t, 2, test.s.countMetrics())
	assert.Equal(t, metrics, test.s.metricBuffer)

	test.s.cleanMetricBuffer()
	assert.Equal(t, 0, test.s.countMetrics())
	assert.Equal(t, []metricPair{}, test.s.metricBuffer)
}

func TestMetricsBufferOverflow(t *testing.T) {
	t.Skip("Skip test due to prometheus format complexity. Execution can take over 30s")
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){})

	test.s.config.HTTPClientSettings.Endpoint = ":"
	test.s.config.MetricFormat = PrometheusFormat
	test.s.config.MaxRequestBodySize = 1024 * 1024 * 1024 * 1024
	metric := exampleIntMetric()
	flds := newFields(pdata.NewAttributeMap())

	for test.s.countMetrics() < maxBufferSize-1 {
		_, err := test.s.batchMetric(context.Background(), metric, flds)
		require.NoError(t, err)
	}

	_, err := test.s.batchMetric(context.Background(), metric, flds)
	assert.EqualError(t, err, `parse ":": missing protocol scheme`)
	assert.Equal(t, 0, test.s.countMetrics())
}

func TestSendCarbon2Metrics(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			//nolint:lll
			expected := `test=test_value test2=second_value _unit=m/s escape_me=:invalid_ metric=true metric=test.metric.data unit=bytes  14500 1605534165
foo=bar metric=gauge_metric_name  124 1608124661
foo=bar metric=gauge_metric_name  245 1608124662`
			assert.Equal(t, expected, body)
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/vnd.sumologic.carbon2", req.Header.Get("Content-Type"))
		},
	})

	test.s.config.MetricFormat = Carbon2Format
	test.s.metricBuffer = []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}

	flds := fieldsFromMap(map[string]string{
		"key1": "value",
		"key2": "value2",
	})

	test.s.metricBuffer[0].attributes.InsertString("unit", "m/s")
	test.s.metricBuffer[0].attributes.InsertString("escape me", "=invalid\n")
	test.s.metricBuffer[0].attributes.InsertBool("metric", true)

	_, err := test.s.sendMetrics(context.Background(), flds)
	assert.NoError(t, err)
}

func TestSendGraphiteMetrics(t *testing.T) {
	test := prepareSenderTest(t, []func(w http.ResponseWriter, req *http.Request){
		func(w http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			expected := `test_metric_data.true.m/s 14500 1605534165
gauge_metric_name.. 124 1608124661
gauge_metric_name.. 245 1608124662`
			assert.Equal(t, expected, body)
			assert.Equal(t, "otelcol", req.Header.Get("X-Sumo-Client"))
			assert.Equal(t, "application/vnd.sumologic.graphite", req.Header.Get("Content-Type"))
		},
	})

	gf, err := newGraphiteFormatter("%{_metric_}.%{metric}.%{unit}")
	require.NoError(t, err)
	test.s.graphiteFormatter = gf

	test.s.config.MetricFormat = GraphiteFormat
	test.s.metricBuffer = []metricPair{
		exampleIntMetric(),
		exampleIntGaugeMetric(),
	}

	flds := fieldsFromMap(map[string]string{
		"key1": "value",
		"key2": "value2",
	})

	test.s.metricBuffer[0].attributes.InsertString("unit", "m/s")
	test.s.metricBuffer[0].attributes.InsertBool("metric", true)

	_, err = test.s.sendMetrics(context.Background(), flds)
	assert.NoError(t, err)
}
