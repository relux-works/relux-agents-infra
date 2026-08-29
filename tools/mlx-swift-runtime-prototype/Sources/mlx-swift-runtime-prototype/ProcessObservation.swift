import CryptoKit
import Darwin
import Foundation

/// What the kernel says about a live process.
///
/// Every reading here comes from the kernel rather than from an argument, which
/// is the only reason an attestation is worth more than the record it checks.
/// Since revision 4 the pid handed to this type is always one `benchmark-run`
/// spawned itself, so there is no longer a caller who can point it at a process
/// of their choosing — the defect review exploited by having the gate attest
/// two placeholder HTTP servers.
enum ProcessObservation {
    struct Reading {
        let executablePath: String
        let startUnixSeconds: Double
    }

    static func of(pid: Int) -> Reading? {
        guard let path = executablePath(pid: pid), let start = startTime(pid: pid) else {
            return nil
        }
        return Reading(executablePath: path, startUnixSeconds: start)
    }

    private static func executablePath(pid: Int) -> String? {
        var buffer = [CChar](repeating: 0, count: 4096)
        let length = proc_pidpath(Int32(pid), &buffer, UInt32(buffer.count))
        guard length > 0 else { return nil }
        return String(cString: buffer)
    }

    /// `kinfo_proc.kp_proc.p_starttime`, as Unix seconds with microseconds.
    ///
    /// Recorded so a recycled pid cannot be closed over: the number is fixed at
    /// exec and no later process under the same pid can reproduce it.
    private static func startTime(pid: Int) -> Double? {
        var name: [Int32] = [CTL_KERN, KERN_PROC, KERN_PROC_PID, Int32(pid)]
        var info = kinfo_proc()
        var size = MemoryLayout<kinfo_proc>.stride
        let status = sysctl(&name, UInt32(name.count), &info, &size, nil, 0)
        guard status == 0, size > 0, info.kp_proc.p_pid == Int32(pid) else { return nil }
        let start = info.kp_proc.p_un.__p_starttime
        return Double(start.tv_sec) + Double(start.tv_usec) / 1_000_000
    }
}

/// The binary making the current decision.
enum GateBinary {
    static func digest() -> String? {
        guard let path = path() else { return nil }
        return fileDigest(path)
    }

    static func path() -> String? {
        if let executable = Bundle.main.executablePath { return executable }
        return CommandLine.arguments.first
    }
}

/// SHA-256 over a file's bytes, or `nil` when they could not be read.
///
/// `nil` is never turned into an empty digest by a caller: a file that could not
/// be read and a file with nothing in it are different facts, and every caller
/// here refuses on the first.
func fileDigest(_ path: String) -> String? {
    guard let data = try? Data(contentsOf: URL(fileURLWithPath: path), options: [.mappedIfSafe])
    else { return nil }
    return SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
}

func digest(of data: Data) -> String {
    SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
}
