import Foundation
import MLXSwiftRuntimeContract
import NIOCore
import NIOHTTP1
import NIOPosix

/// A thread-safe handle for writing a response from outside the event loop.
///
/// `Channel.writeAndFlush` hops to the channel's event loop internally, so the
/// generation task can write SSE frames directly as tokens arrive.
struct ResponseWriter: @unchecked Sendable {
    let channel: any Channel

    func head(status: Int, headers: HTTPHeaders) async throws {
        let head = HTTPResponseHead(
            version: .http1_1,
            status: HTTPResponseStatus(statusCode: status),
            headers: headers)
        try await channel.writeAndFlush(HTTPServerResponsePart.head(head)).get()
    }

    func body(_ text: String) async throws {
        var buffer = channel.allocator.buffer(capacity: text.utf8.count)
        buffer.writeString(text)
        try await channel.writeAndFlush(HTTPServerResponsePart.body(.byteBuffer(buffer))).get()
    }

    func end() async throws {
        try await channel.writeAndFlush(HTTPServerResponsePart.end(nil)).get()
    }
}

/// Accumulates one HTTP request and hands it to the router.
final class RuntimeHTTPHandler: ChannelInboundHandler, @unchecked Sendable {
    typealias InboundIn = HTTPServerRequestPart
    typealias OutboundOut = HTTPServerResponsePart

    /// A request body larger than this is refused rather than buffered. A chat
    /// transcript at this profile's 75k-token context window is far smaller.
    private static let maximumBodyBytes = 64 * 1024 * 1024

    private let router: Router
    private var head: HTTPRequestHead?
    private var body = Data()
    private var overflowed = false

    init(router: Router) {
        self.router = router
    }

    func channelRead(context: ChannelHandlerContext, data: NIOAny) {
        switch unwrapInboundIn(data) {
        case .head(let head):
            self.head = head
            body.removeAll(keepingCapacity: true)
            overflowed = false
        case .body(var buffer):
            guard !overflowed else { return }
            if body.count + buffer.readableBytes > Self.maximumBodyBytes {
                overflowed = true
                body.removeAll(keepingCapacity: false)
                return
            }
            if let bytes = buffer.readBytes(length: buffer.readableBytes) {
                body.append(contentsOf: bytes)
            }
        case .end:
            guard let head else { return }
            let writer = ResponseWriter(channel: context.channel)
            let method = head.method.rawValue
            let uri = head.uri
            let payload = body
            let router = self.router
            let refused = overflowed
            self.head = nil
            body.removeAll(keepingCapacity: true)
            overflowed = false

            Task {
                if refused {
                    await Self.sendJSON(
                        writer: writer, status: 413,
                        body: ChatCompletionResponseBuilder.error(
                            message: "request body exceeds \(Self.maximumBodyBytes) bytes",
                            type: "invalid_request_error", code: "payload_too_large"))
                    return
                }
                let outcome = await router.route(method: method, path: uri, body: payload)
                switch outcome {
                case .json(let status, let body):
                    await Self.sendJSON(writer: writer, status: status, body: body)
                case .stream(let request, let engine, let builder):
                    await Self.sendStream(
                        writer: writer, router: router, request: request, engine: engine,
                        builder: builder)
                }
            }
        }
    }

    private static func sendJSON(writer: ResponseWriter, status: Int, body: JSONValue) async {
        do {
            let data = try JSONEncoding.data(body)
            var headers = HTTPHeaders()
            headers.add(name: "Content-Type", value: "application/json")
            headers.add(name: "Content-Length", value: String(data.count))
            try await writer.head(status: status, headers: headers)
            try await writer.body(String(decoding: data, as: UTF8.self))
            try await writer.end()
        } catch {
            StandardOutput.shared.log("failed to write response: \(error)")
        }
    }

    private static func sendStream(
        writer: ResponseWriter,
        router: Router,
        request: AdmittedChatCompletion,
        engine: GenerationEngine,
        builder: ChatCompletionResponseBuilder
    ) async {
        var headers = HTTPHeaders()
        headers.add(name: "Content-Type", value: "text/event-stream")
        headers.add(name: "Cache-Control", value: "no-cache")
        headers.add(name: "Transfer-Encoding", value: "chunked")
        do {
            try await writer.head(status: 200, headers: headers)
        } catch {
            StandardOutput.shared.log("failed to write stream head: \(error)")
            return
        }

        do {
            try await engine.generate(request) { event in
                switch event {
                case .delta(let reasoning, let content):
                    let chunk = builder.chunk(
                        content: content, reasoning: reasoning, toolCalls: [], finishReason: nil)
                    try await writer.body(try ServerSentEvent.data(chunk))
                case .toolCall(let call):
                    let chunk = builder.chunk(
                        content: "", reasoning: "", toolCalls: [call], finishReason: nil)
                    try await writer.body(try ServerSentEvent.data(chunk))
                case .completed(let finishReason, let usage):
                    let final = builder.chunk(
                        content: "", reasoning: "", toolCalls: [], finishReason: finishReason)
                    try await writer.body(try ServerSentEvent.data(final))
                    if request.includeUsage {
                        try await writer.body(
                            try ServerSentEvent.data(builder.streamingUsage(usage)))
                    }
                }
            }
            try await writer.body(ServerSentEvent.done)
        } catch {
            // The head is already on the wire, so the failure is reported as a
            // terminal SSE frame rather than as an HTTP status the client can no
            // longer receive. Health is updated first: the client's next move
            // after a broken stream is usually to poll `/health`, and it must
            // not find a `200` there.
            await router.recordGenerationFailure(error)
            StandardOutput.shared.log("stream failed: \(error)")
            let payload = ChatCompletionResponseBuilder.error(
                message: String(describing: error), type: "server_error",
                code: "generation_failed")
            if let frame = try? ServerSentEvent.data(payload) {
                try? await writer.body(frame)
            }
            try? await writer.body(ServerSentEvent.done)
        }

        try? await writer.end()
    }
}

/// The loopback HTTP listener.
final class RuntimeHTTPServer: @unchecked Sendable {
    private let group: MultiThreadedEventLoopGroup
    private var channel: (any Channel)?

    init() {
        group = MultiThreadedEventLoopGroup(numberOfThreads: 2)
    }

    /// Bind and start accepting.
    ///
    /// Called *before* the model finishes loading: the managed launcher polls
    /// `/v1/models` and treats a refused connection as "not started yet" but a
    /// `503` as "still loading", so binding early is what makes the startup
    /// timeout meaningful.
    func start(router: Router, host: String, port: Int) async throws {
        let bootstrap = ServerBootstrap(group: group)
            .serverChannelOption(.backlog, value: 64)
            .serverChannelOption(.socketOption(.so_reuseaddr), value: 1)
            .childChannelInitializer { channel in
                channel.pipeline.configureHTTPServerPipeline(withErrorHandling: true)
                    .flatMap {
                        channel.pipeline.addHandler(RuntimeHTTPHandler(router: router))
                    }
            }
            .childChannelOption(.socketOption(.so_reuseaddr), value: 1)
            .childChannelOption(.maxMessagesPerRead, value: 16)

        channel = try await bootstrap.bind(host: host, port: port).get()
    }

    var boundPort: Int? { channel?.localAddress?.port }

    func shutdown() async {
        if let channel {
            try? await channel.close().get()
        }
        try? await group.shutdownGracefully()
    }
}
