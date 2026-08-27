import Foundation

/// The launcher profile a benchmark pass will actually exec.
///
/// Read from the same `model-harness` config bytes the launcher is handed, with
/// the launcher's own `{host}` / `{port}` substitution applied, so the recorded
/// argv is the tokens the runtime process receives rather than a template
/// nobody ran. `RuntimeBenchmark.contextPolicy(derivedFrom:)` is then a reading
/// of that argv, which is what makes the `contextPolicy` pin a fact about the
/// launch instead of a claim beside it.
///
/// Parsed in-process rather than shelled out to Python. That is not a
/// preference: the whole point of revision 4 is that launch, measurement,
/// record construction and judgement happen in one invocation, and a config
/// read by a separate program is a place where the launch that was recorded and
/// the launch that happened can drift apart.
///
/// The parser covers the subset `model-harness` profiles use — table headers,
/// string values, and arrays of strings — and refuses everything else by name
/// rather than skipping it. A silently ignored key in a launcher config is a
/// condition that did not make it to the process under test.
enum BenchmarkLaunchConfig {
    struct Profile {
        let name: String
        let executable: String
        /// Argv with `{host}` and `{port}` already substituted.
        let argv: [String]
    }

    enum ConfigError: Error, CustomStringConvertible {
        case unreadable(path: String)
        case syntax(path: String, detail: String)
        case noSuchProfile(path: String, profile: String, available: [String])
        case noExecutable(profile: String)

        var description: String {
            switch self {
            case .unreadable(let path):
                return "could not read launcher config \(path.debugDescription)"
            case .syntax(let path, let detail):
                return "launcher config \(path.debugDescription) is not parseable: \(detail)"
            case .noSuchProfile(let path, let profile, let available):
                return
                    "\(path.debugDescription) has no profile \(profile.debugDescription); it "
                    + "declares \(available)"
            case .noExecutable(let profile):
                return "profile \(profile.debugDescription) names no executable"
            }
        }
    }

    static func profile(
        named name: String, in path: String, host: String, port: Int
    ) throws -> Profile {
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)),
            let text = String(data: data, encoding: .utf8)
        else { throw ConfigError.unreadable(path: path) }
        let tables: [String: [String: Value]]
        do {
            tables = try parse(text)
        } catch let error as ConfigError {
            throw error
        } catch {
            throw ConfigError.syntax(path: path, detail: String(describing: error))
        }
        let key = "profiles.\(name)"
        guard let table = tables[key] else {
            let available = tables.keys
                .filter { $0.hasPrefix("profiles.") }
                .map { String($0.dropFirst("profiles.".count)) }
                .sorted()
            throw ConfigError.noSuchProfile(path: path, profile: name, available: available)
        }
        guard case .string(let executable)? = table["executable"], !executable.isEmpty else {
            throw ConfigError.noExecutable(profile: name)
        }
        var argv: [String] = []
        if case .array(let tokens)? = table["argv"] {
            argv = tokens.map {
                $0.replacingOccurrences(of: "{host}", with: host)
                    .replacingOccurrences(of: "{port}", with: String(port))
            }
        }
        return Profile(name: name, executable: executable, argv: argv)
    }

    enum Value: Equatable {
        case string(String)
        case array([String])
    }

    /// Table name to key/value pairs. Only the shapes a launcher profile uses.
    static func parse(_ text: String) throws -> [String: [String: Value]] {
        var tables: [String: [String: Value]] = [:]
        var current = ""
        var scanner = Scanner(text)
        while let character = scanner.peek() {
            if character == "#" {
                scanner.skipToEndOfLine()
                continue
            }
            if character.isWhitespace {
                scanner.advance()
                continue
            }
            if character == "[" {
                current = try scanner.readTableHeader()
                if tables[current] == nil { tables[current] = [:] }
                continue
            }
            let (key, value) = try scanner.readAssignment()
            tables[current, default: [:]][key] = value
        }
        return tables
    }

    /// A character cursor over the config text.
    ///
    /// Written out rather than done with regular expressions because the values
    /// that matter here contain quotes: the Python profile passes
    /// `--chat-template-args '{"reasoning_effort": "medium"}'` as a single TOML
    /// string with escaped quotes inside it, and that is exactly the token whose
    /// loss review measured as a 1.93x prompt-token skew.
    struct Scanner {
        private let characters: [Character]
        private var index: Int = 0

        init(_ text: String) { characters = Array(text) }

        func peek() -> Character? { index < characters.count ? characters[index] : nil }

        mutating func advance() { index += 1 }

        mutating func skipToEndOfLine() {
            while index < characters.count, characters[index] != "\n" { index += 1 }
        }

        mutating func skipTrivia() {
            while index < characters.count {
                let character = characters[index]
                if character == "#" {
                    skipToEndOfLine()
                } else if character.isWhitespace {
                    index += 1
                } else {
                    return
                }
            }
        }

        mutating func readTableHeader() throws -> String {
            advance()  // [
            var name = ""
            while index < characters.count, characters[index] != "]" {
                name.append(characters[index])
                index += 1
            }
            guard index < characters.count else {
                throw ConfigError.syntax(path: "<config>", detail: "unterminated table header")
            }
            advance()  // ]
            return name.trimmingCharacters(in: .whitespaces)
        }

        mutating func readAssignment() throws -> (String, Value) {
            var key = ""
            while index < characters.count, characters[index] != "=", !characters[index].isNewline {
                key.append(characters[index])
                index += 1
            }
            guard index < characters.count, characters[index] == "=" else {
                throw ConfigError.syntax(
                    path: "<config>",
                    detail: "key \(key.trimmingCharacters(in: .whitespaces).debugDescription) has "
                        + "no value")
            }
            advance()  // =
            skipTrivia()
            let value = try readValue()
            return (key.trimmingCharacters(in: .whitespaces), value)
        }

        mutating func readValue() throws -> Value {
            guard let character = peek() else {
                throw ConfigError.syntax(path: "<config>", detail: "value ended early")
            }
            if character == "\"" { return .string(try readString()) }
            if character == "[" { return .array(try readStringArray()) }
            // Bare values — numbers, booleans — are refused rather than skipped.
            // No launcher profile field the benchmark reads is one, and a
            // silently dropped key in a launcher config is a condition the
            // process under test never received.
            var token = ""
            while index < characters.count, !characters[index].isNewline {
                token.append(characters[index])
                index += 1
            }
            let shown = token.trimmingCharacters(in: .whitespaces).debugDescription
            throw ConfigError.syntax(
                path: "<config>",
                detail: "unsupported value \(shown); this parser reads strings and arrays of "
                    + "strings")
        }

        mutating func readString() throws -> String {
            advance()  // opening quote
            var value = ""
            while index < characters.count {
                let character = characters[index]
                if character == "\\" {
                    index += 1
                    guard index < characters.count else { break }
                    switch characters[index] {
                    case "n": value.append("\n")
                    case "t": value.append("\t")
                    case "r": value.append("\r")
                    case "\"": value.append("\"")
                    case "\\": value.append("\\")
                    case let other: value.append(other)
                    }
                    index += 1
                    continue
                }
                if character == "\"" {
                    advance()
                    return value
                }
                value.append(character)
                index += 1
            }
            throw ConfigError.syntax(path: "<config>", detail: "unterminated string")
        }

        mutating func readStringArray() throws -> [String] {
            advance()  // [
            var values: [String] = []
            while true {
                skipTrivia()
                guard let character = peek() else {
                    throw ConfigError.syntax(path: "<config>", detail: "unterminated array")
                }
                if character == "]" {
                    advance()
                    return values
                }
                if character == "," {
                    advance()
                    continue
                }
                guard character == "\"" else {
                    throw ConfigError.syntax(
                        path: "<config>",
                        detail: "arrays in a launcher profile hold strings, found "
                            + String(character).debugDescription)
                }
                values.append(try readString())
            }
        }
    }
}
