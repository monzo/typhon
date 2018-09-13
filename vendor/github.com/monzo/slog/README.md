# `slog`
**S**tructured **log**ging.

slog is a library for capturing structured log information. In contrast to "traditional" logging libraries, slog:

* captures a Context for each logging event
* captures arbitrary key-value metadata on each log event

Currently, slog forwards messages to [seelog](https://github.com/cihub/seelog) by default. However, it is easy to write customised outputs which make use of the context and metadata.

At [Monzo](https://monzo.com/), slog captures events both on a per-service and a per-request basis (using the context information). This lets us view all the logs for a given request across all the micro-services it touches.
