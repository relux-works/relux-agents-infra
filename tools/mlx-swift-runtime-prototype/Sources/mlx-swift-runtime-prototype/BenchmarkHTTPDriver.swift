import Foundation
import MLXSwiftRuntimeContract

/// Every HTTP request a benchmark pass makes, and the receipt it leaves behind.
///
/// The driver and the transcript are the same code path on purpose. Revision 3
/// had a driver that measured and a gate that attested, and review put a
/// placeholder between them: two servers that answered `GET /v1/models`, real
/// attestations minted for them by production commands, and a set of
/// measurements typed by the caller. Nothing connected the numbers to the
/// process. Here a measurement cannot exist without the exchange it came from,
/// because the exchange record is produced by the same function that took the
/// timing.
enum BenchmarkHTTPDriver {
    typealias Exchange = RuntimeBenchmark.ScenarioTranscript.Exchange

    /// A driven streaming completion: what came back, and the receipt.
    struct StreamedCompletion {
        var exchange: Exchange
        /// Seconds to the first frame carrying generated text, reasoning
        /// included.
        ///
        /// Reasoning counts because this model's chat template opens a
        /// `<think>` block, so the first thing generated for any prompt is
        /// reasoning. Waiting for the first *content* delta would report the
        /// length of the model's thinking as runtime latency.
        var timeToFirstTokenSeconds: Double?
        var totalSeconds: Double
        var promptTokens: Int?
        var completionTokens: Int?
        var content: String
        var finishReason: String?
        var failure: String?
    }

    /// A driven non-streaming completion.
    struct Completion {
        var exchange: Exchange
        var totalSeconds: Double
        var status: Int
        var body: Data
        var failure: String?
    }

    /// A session whose timeouts are long enough for a 73k-token prefill.
    ///
    /// The capacity probe spends over twenty minutes before its first byte on
    /// the slower runtime. A default `URLSession` would cancel it and the pass
    /// would report a timeout as a runtime failure.
    static func session(requestTimeout: TimeInterval) -> URLSession {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.timeoutIntervalForRequest = requestTimeout
        configuration.timeoutIntervalForResource = requestTimeout + 600
        configuration.waitsForConnectivity = false
        return URLSession(configuration: configuration)
    }

    /// `GET`, with "the listener is not up yet" reported as status `0`.
    ///
    /// Status `0` is not an HTTP status and is never read as one: it means the
    /// question could not be asked. A refused connection during startup and a
    /// runtime that answered "not ready" are different facts, and only the
    /// second one is an answer.
    static func get(
        session: URLSession, url: URL, timeout: TimeInterval
    ) async -> (status: Int, body: Data) {
        var request = URLRequest(url: url)
        request.timeoutInterval = timeout
        do {
            let (data, response) = try await session.data(for: request)
            return ((response as? HTTPURLResponse)?.statusCode ?? 0, data)
        } catch {
            return (0, Data(String(describing: error).utf8))
        }
    }

    /// One non-streaming `POST /v1/chat/completions`, recorded.
    static func post(
        session: URLSession, endpoint: URL, path: String, body: Data, timeout: TimeInterval
    ) async -> Completion {
        var request = URLRequest(url: endpoint)
        request.httpMethod = "POST"
        request.httpBody = body
        request.timeoutInterval = timeout
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let sentAt = Date().timeIntervalSince1970
        do {
            let (data, response) = try await session.data(for: request)
            let now = Date().timeIntervalSince1970
            let status = (response as? HTTPURLResponse)?.statusCode ?? 0
            return Completion(
                exchange: Exchange(
                    method: "POST", path: path, requestDigest: digest(of: body),
                    requestByteCount: body.count, status: status,
                    responseDigest: digest(of: data), responseByteCount: data.count,
                    sentAtUnixSeconds: sentAt, firstByteAtUnixSeconds: data.isEmpty ? nil : now,
                    lastByteAtUnixSeconds: now),
                totalSeconds: now - sentAt, status: status, body: data,
                failure: status == 200 ? nil : "HTTP \(status)")
        } catch {
            let now = Date().timeIntervalSince1970
            return Completion(
                exchange: Exchange(
                    method: "POST", path: path, requestDigest: digest(of: body),
                    requestByteCount: body.count, status: 0,
                    responseDigest: digest(of: Data()), responseByteCount: 0,
                    sentAtUnixSeconds: sentAt, firstByteAtUnixSeconds: nil,
                    lastByteAtUnixSeconds: now),
                totalSeconds: now - sentAt, status: 0, body: Data(),
                failure: String(describing: error))
        }
    }

    /// One streaming `POST /v1/chat/completions`, timed from the client side and
    /// recorded byte for byte.
    ///
    /// The response digest covers the raw SSE bytes rather than the assembled
    /// text, so the receipt is over what the runtime sent rather than over what
    /// this driver made of it.
    static func stream(
        session: URLSession, endpoint: URL, path: String, body: Data, timeout: TimeInterval
    ) async -> StreamedCompletion {
        var request = URLRequest(url: endpoint)
        request.httpMethod = "POST"
        request.httpBody = body
        request.timeoutInterval = timeout
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let sentAt = Date().timeIntervalSince1970
        var raw = Data()
        var firstByteAt: Double?
        var timeToFirstToken: Double?
        var promptTokens: Int?
        var completionTokens: Int?
        var content = ""
        var finishReason: String?
        var failure: String?
        var status = 0

        func finish(_ lastByteAt: Double) -> StreamedCompletion {
            StreamedCompletion(
                exchange: Exchange(
                    method: "POST", path: path, requestDigest: digest(of: body),
                    requestByteCount: body.count, status: status,
                    responseDigest: digest(of: raw), responseByteCount: raw.count,
                    sentAtUnixSeconds: sentAt, firstByteAtUnixSeconds: firstByteAt,
                    lastByteAtUnixSeconds: lastByteAt),
                timeToFirstTokenSeconds: timeToFirstToken, totalSeconds: lastByteAt - sentAt,
                promptTokens: promptTokens, completionTokens: completionTokens,
                content: content, finishReason: finishReason, failure: failure)
        }

        do {
            let (bytes, response) = try await session.bytes(for: request)
            status = (response as? HTTPURLResponse)?.statusCode ?? 0
            var line = Data()
            for try await byte in bytes {
                if firstByteAt == nil { firstByteAt = Date().timeIntervalSince1970 }
                raw.append(byte)
                guard byte == UInt8(ascii: "\n") else {
                    line.append(byte)
                    continue
                }
                let text = String(decoding: line, as: UTF8.self)
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                line = Data()
                guard text.hasPrefix("data:") else { continue }
                let payload = text.dropFirst("data:".count)
                    .trimmingCharacters(in: .whitespaces)
                if payload == "[DONE]" { break }
                guard let frameData = payload.data(using: .utf8),
                    let frame = try? JSONSerialization.jsonObject(with: frameData)
                        as? [String: Any]
                else {
                    failure = "undecodable SSE frame"
                    break
                }
                if let usage = frame["usage"] as? [String: Any] {
                    promptTokens = usage["prompt_tokens"] as? Int ?? promptTokens
                    completionTokens = usage["completion_tokens"] as? Int ?? completionTokens
                }
                for choice in (frame["choices"] as? [[String: Any]]) ?? [] {
                    let delta = (choice["delta"] as? [String: Any]) ?? [:]
                    let text =
                        ((delta["content"] as? String) ?? "")
                        + ((delta["reasoning"] as? String) ?? "")
                    if !text.isEmpty, timeToFirstToken == nil {
                        timeToFirstToken = Date().timeIntervalSince1970 - sentAt
                    }
                    content += (delta["content"] as? String) ?? ""
                    if let reason = choice["finish_reason"] as? String { finishReason = reason }
                }
            }
            if status != 200 {
                failure =
                    "HTTP \(status): "
                    + String(decoding: raw.prefix(500), as: UTF8.self)
            }
        } catch {
            failure = String(describing: error)
        }
        return finish(Date().timeIntervalSince1970)
    }
}
