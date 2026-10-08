// The OTLP/HTTP exporter (D126): what `std/otel` sends its protobuf through.
// It lives here, not in `otel`, because `http` already uses `otel` (a span per
// request) and a module cannot import one that imports it.
use compress as gz
use otel
use tls

struct OtlpExporter {
  endpoint: string
  headers:  Map<string, Secret<string>>
  gzip:     bool
  client:   Client

  implement otel.Exporter {
    fun export(signal: otel.Signal, body: List<u8>) suspends throws otel.ExportError {
      val url = this.endpoint + otel.signalPath(signal)
      val headers: MutableMap<string, string> = [:]
      loop ((name, value) in this.headers.entries()) {
        headers.set(name, value.expose())
      }
      val sent = if (this.gzip) gz.gzip(body) else body
      if (this.gzip) headers.set("content-encoding", "gzip")
      when (this.client.post(url, body: Payload.bytes(sent, contentType: MediaType(name: "application/x-protobuf")), headers: headers.toMap())) {
        is Ok(res) => {
          with answer = res
          // a small answer is read and dropped, so the connection serves the next export
          answer.discard()
          if (answer.ok) return
          // a collector that is busy may take it a moment later; one that
          // refuses it will refuse it again
          val busy = answer.status.code == 429 || retryableStatus(answer.status)
          throw otel.ExportError(message: "the collector answered ${answer.status}", retryable: busy)
        }
        is Err(e)  => throw otel.ExportError(message: e.message(), retryable: retryable(e))
      }
    }
  }
}

/// The exporter for a collector that speaks OTLP over HTTP — the standard
/// port is 4318 — to hand to `otel.start`: protobuf bodies, gzipped, posted to
/// `<endpoint>/v1/traces`, `/v1/metrics` and `/v1/logs`. A busy collector
/// (429, 502, 503, 504) or a lost connection is retried by the pipeline; any
/// other refusal is not.
///
/// `headers` carry what the backend asks for — an API key — as `Secret`s, so
/// they never appear in a log or a failure message. An `https://` endpoint is
/// verified against the system's trusted roots; `tlsOptions` names a private
/// authority's roots instead.
///
/// ```veles
/// val exporter = http.otlp(endpoint: "http://localhost:4318", headers: ["authorization": Secret.of(token)])
/// with tel = try otel.start(service: "notes", exporter: exporter)
/// ```
public fun otlp(endpoint: string, headers: Map<string, Secret<string>> = [:], timeout: Duration = Duration.seconds(10), gzip: bool = true, tlsOptions: tls.Options = tls.Options()): otel.Exporter {
  val trimmed = if (endpoint.endsWith("/")) (endpoint.substring(0, endpoint.len() - 1) ?: endpoint) else endpoint
  val lower: MutableMap<string, Secret<string>> = [:]
  loop ((name, value) in headers.entries()) {
    lower.set(name.toLower(), value)
  }
  OtlpExporter(endpoint: trimmed, headers: lower.toMap(), gzip, client: Client.untraced(timeout, tlsOptions))
}
