import Foundation

public actor EventStore {
    private let directory: URL
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder

    public init(directory: URL) {
        self.directory = directory
        self.encoder = JSONEncoder()
        self.decoder = JSONDecoder()
        encoder.dateEncodingStrategy = .iso8601
        decoder.dateDecodingStrategy = .iso8601
    }

    public func append(_ event: TransmissionEvent) throws {
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        let url = directory.appendingPathComponent("events.jsonl")
        var line = try encoder.encode(event)
        line.append(0x0A)
        if !FileManager.default.fileExists(atPath: url.path) {
            guard FileManager.default.createFile(atPath: url.path, contents: line) else {
                throw CocoaError(.fileWriteUnknown)
            }
            return
        }
        let handle = try FileHandle(forWritingTo: url)
        defer { try? handle.close() }
        try handle.seekToEnd()
        try handle.write(contentsOf: line)
    }

    public func recent(limit: Int = 500) throws -> [TransmissionEvent] {
        let url = directory.appendingPathComponent("events.jsonl")
        guard let data = try? Data(contentsOf: url), !data.isEmpty else { return [] }
        return data.split(separator: 0x0A)
            .suffix(max(0, min(limit, 10_000)))
            .compactMap { try? decoder.decode(TransmissionEvent.self, from: Data($0)) }
            .reversed()
    }

    public func clear() throws {
        let url = directory.appendingPathComponent("events.jsonl")
        if FileManager.default.fileExists(atPath: url.path) {
            try FileManager.default.removeItem(at: url)
        }
    }
}
