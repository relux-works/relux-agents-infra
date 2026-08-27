import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("reasoning splitter")
struct ReasoningSplitterTests {
    static func drain(_ chunks: [String], mode: ReasoningSplitter.Mode = .startsInReasoning)
        -> (reasoning: String, content: String)
    {
        var splitter = ReasoningSplitter(mode: mode)
        var reasoning = ""
        var content = ""
        for chunk in chunks {
            let output = splitter.consume(chunk)
            reasoning += output.reasoning
            content += output.content
        }
        let tail = splitter.flush()
        return (reasoning + tail.reasoning, content + tail.content)
    }

    @Test("generation that starts inside <think> splits on the closing marker")
    func splitsOnClosingMarker() {
        let result = Self.drain(["let me think", "</think>", "\n\nThe answer is 4."])
        #expect(result.reasoning == "let me think")
        #expect(result.content == "\n\nThe answer is 4.")
    }

    @Test("a marker split across chunks is still recognized")
    func handlesSplitMarker() {
        // Token boundaries do not respect markers: `</think>` regularly arrives
        // as several chunks. Emitting the fragments as reasoning text would
        // leak `</think` into the visible answer.
        let result = Self.drain(["thinking", "</", "th", "ink", ">", "answer"])
        #expect(result.reasoning == "thinking")
        #expect(result.content == "answer")
    }

    @Test(
        "marker characters are never emitted",
        arguments: [
            ["reasoning</think>answer"],
            ["reasoning", "</think>", "answer"],
            ["reasoning</", "think>answer"],
            ["reasoning</thin", "k>answer"],
        ])
    func neverLeaksMarker(chunks: [String]) {
        let result = Self.drain(chunks)
        #expect(!result.reasoning.contains("</think"))
        #expect(!result.content.contains("</think"))
        #expect(!result.content.contains("think>"))
        #expect(result.content == "answer")
    }

    @Test("a near-miss that never completes is emitted, not swallowed")
    func emitsIncompleteMarkerText() {
        // Held-back bytes are real model output. If generation stops mid-marker
        // (a `length` finish), dropping the buffer would silently truncate.
        let result = Self.drain(["done</thin"])
        #expect(result.reasoning == "done</thin")
        #expect(result.content.isEmpty)
    }

    @Test("text resembling the marker but diverging is emitted")
    func emitsDivergentText() {
        let result = Self.drain(["a</thought>b</think>c"])
        #expect(result.reasoning == "a</thought>b")
        #expect(result.content == "c")
    }

    @Test("everything before a closing marker is reasoning even with no answer")
    func handlesReasoningOnly() {
        let result = Self.drain(["only thinking here"])
        #expect(result.reasoning == "only thinking here")
        #expect(result.content.isEmpty)
    }

    @Test("with thinking disabled the stream starts as content")
    func startsInContentWhenThinkingOff() {
        let result = Self.drain(["plain answer"], mode: .startsInContent)
        #expect(result.reasoning.isEmpty)
        #expect(result.content == "plain answer")
    }

    @Test("a re-opened think block returns to reasoning")
    func reentersReasoning() {
        // Mirrors mlx_lm's state machine, which keeps a `normal -> <think>`
        // edge; a model that opens a second block must not have it counted as
        // answer text.
        let result = Self.drain(["r1</think>a1<think>r2</think>a2"])
        #expect(result.reasoning == "r1r2")
        #expect(result.content == "a1a2")
    }

    @Test("character-by-character delivery matches whole-string delivery")
    func chunkingIsIrrelevant() {
        let whole = "step one</think>final words"
        let byCharacter = whole.map(String.init)
        #expect(Self.drain([whole]) == Self.drain(byCharacter))
    }

    @Test("suffix hold-back length is the longest marker prefix")
    func computesHoldBack() {
        let marker = ReasoningSplitter.thinkEnd
        #expect(ReasoningSplitter.longestMarkerPrefixSuffix(of: "abc</thin", marker: marker) == 6)
        #expect(ReasoningSplitter.longestMarkerPrefixSuffix(of: "abc<", marker: marker) == 1)
        #expect(ReasoningSplitter.longestMarkerPrefixSuffix(of: "abc", marker: marker) == 0)
        // A complete marker is handled by the match path, never held back.
        #expect(ReasoningSplitter.longestMarkerPrefixSuffix(of: "</think>", marker: marker) == 0)
    }
}
