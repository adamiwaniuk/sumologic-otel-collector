# Comparison between the Installed Collector and the OpenTelemetry Collector

- [Syslog](#syslog)
  - [Syslog Receiver](#syslog-receiver)
  - [TCPlog/UDPlog Receiver and Sumo Logic Syslog Processor](#tcplogudplog-receiver-and-sumo-logic-syslog-processor)

## Syslog

The OpenTelemetry Collector offers two approaches for syslog processing.

### Syslog Receiver

Syslog Receiver is a perfect solution if you are sending logs using a certain RFC protocol.
There are two supported formats: `rfc3164` and `rfc5424`.
Parsing is strict, if you send `rfc5424` logs to the `rfc3164` endpoint,
it will fail with an error and the log (and timestamp as well) won't be parsed.

For example, with the following configuration:

```yaml
receivers:
  syslog:
    tcp:
      listen_address: "0.0.0.0:54526"
    protocol: rfc5424
exporters:
  logging:
    logLevel: debug
service:
  pipelines:
    logs:
      receivers: [syslog]
      exporters: [logging]
```

and the following example logs:

```text
<14>Apr  19 09:50:00 mymachineICIP su: RFC3164
<34>1 2021-04-09T07:54:14.001Z mymachine.example.com su - - - RFC5424 | TCP
```

it will produce the following logs:

```text
...
2021-08-24T12:55:43.323+0200    error   Failed to process entry {"kind": "receiver", "name": "syslog", "operator_id": "$..syslog_parser", "operator_type": "syslog_parser", "error": "expecting a version value in the range 1-999 [col 4]", "action": "send", "entry": {"timestamp":"2021-08-24T12:55:43.323699582+02:00","body":"<14>Apr  19 09:50:00 mymachineICIP su: RFC3164","severity":0}}
2021-08-24T12:55:43.374+0200    INFO    loggingexporter/logging_exporter.go:71  LogsExporter    {"#logs": 1}
2021-08-24T12:55:43.374+0200    DEBUG   loggingexporter/logging_exporter.go:81  ResourceLog #0
InstrumentationLibraryLogs #0
InstrumentationLibrary
LogRecord #0
Timestamp: 2021-08-24 10:55:43.323699582 +0000 UTC
Severity: Undefined
ShortName:
Body: <14>Apr  19 09:50:00 mymachineICIP su: RFC3164

2021-08-24T12:55:55.173+0200    INFO    loggingexporter/logging_exporter.go:71  LogsExporter    {"#logs": 1}
2021-08-24T12:55:55.174+0200    DEBUG   loggingexporter/logging_exporter.go:81  ResourceLog #0
InstrumentationLibraryLogs #0
InstrumentationLibrary
LogRecord #0
Timestamp: 2021-04-09 07:54:14.001 +0000 UTC
Severity: Error2
ShortName:
Body: {
     -> appname: STRING(su)
     -> facility: INT(4)
     -> hostname: STRING(mymachine.example.com)
     -> message: STRING(RFC5424 | TCP)
     -> priority: INT(34)
     -> version: INT(1)
}
...
```

### TCPlog/UDPlog Receiver and Sumo Logic Syslog Processor

This second approach is compatible with the current Installed Collector behavior.
It doesn't parse out the fields on the collector side,
but extracts the facility name and sends it as the `Source Name`.
In addition, it doesn't verify the protocol of incoming logs,
so every format is treated the same.

For example, with the following configuration:

```yaml
extensions:
  sumologic:
    access_id: <access_id>
    access_key: <access_key>
receivers:
  tcplog:
    listen_address: "0.0.0.0:54526"
    add_attributes: true

processors:
  sumologic_syslog: {}
  groupbyattrs:
    keys:
    - net.peer.name
    - facility

exporters:
  sumologic:
    ## Set Source Name to facility name
    source_name: "%{facility}"
    ## Set Source Host to client hostname
    source_host: "%{net.peer.name}"
    metadata_attributes: [facility, net.peer.name]
  logging:
    logLevel: debug
service:
  extensions: [sumologic]
  pipelines:
    logs:
      receivers: [tcplog]
      processors: [sumologic_syslog, groupbyattrs]
      exporters: [logging, sumologic]
```

and the following example logs:

```text
<14>Apr  19 09:50:00 mymachineICIP su: RFC3164
<34>1 2021-04-09T07:54:14.001Z mymachine.example.com su - - - RFC5424 | TCP
```

it will produce the following logs:

```text
2021-08-24T13:18:41.464+0200    INFO    loggingexporter/logging_exporter.go:71  LogsExporter    {"#logs": 1}
2021-08-24T13:18:41.464+0200    DEBUG   loggingexporter/logging_exporter.go:81  ResourceLog #0
Resource labels:
     -> net.peer.name: STRING(localhost)
     -> facility: STRING(user-level messages)
InstrumentationLibraryLogs #0
InstrumentationLibrary
LogRecord #0
Timestamp: 2021-08-24 11:18:41.394337919 +0000 UTC
Severity: Undefined
ShortName:
Body: <14>Apr  19 09:50:00 mymachineICIP su: RFC3164
Attributes:
     -> net.transport: STRING(IP.TCP)
     -> net.peer.ip: STRING(127.0.0.1)
     -> net.peer.port: STRING(56790)
     -> net.host.name: STRING(localhost)
     -> net.host.ip: STRING(127.0.0.1)
     -> net.host.port: STRING(54526)

2021-08-24T13:18:42.854+0200    INFO    loggingexporter/logging_exporter.go:71  LogsExporter    {"#logs": 1}
2021-08-24T13:18:42.854+0200    DEBUG   loggingexporter/logging_exporter.go:81  ResourceLog #0
Resource labels:
     -> net.peer.name: STRING(localhost)
     -> facility: STRING(security/authorization messages)
InstrumentationLibraryLogs #0
InstrumentationLibrary
LogRecord #0
Timestamp: 2021-08-24 11:18:41.653010477 +0000 UTC
Severity: Undefined
ShortName:
Body: <34>1 2021-04-09T07:54:14.001Z mymachine.example.com su - - - RFC5424 | TCP
Attributes:
     -> net.transport: STRING(IP.TCP)
     -> net.peer.ip: STRING(127.0.0.1)
     -> net.peer.port: STRING(56790)
     -> net.host.name: STRING(localhost)
     -> net.host.ip: STRING(127.0.0.1)
     -> net.host.port: STRING(54526)
```
