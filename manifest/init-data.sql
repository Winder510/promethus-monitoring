CREATE DATABASE IF NOT EXISTS otel_traces;

CREATE TABLE otel_traces.spans (
    Timestamp DateTime64(9),
    TraceId String,
    SpanId String,
    ParentSpanId String,
    ServiceName LowCardinality(String),
    SpanName String,
    DurationNano UInt64,
    StatusCode Int32
) ENGINE = MergeTree()
ORDER BY (ServiceName, Timestamp, TraceId);