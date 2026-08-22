import Foundation

public enum ProfileCodecError: Error, Equatable {
    case unsupportedSchema(Int)
    case invalidFrequency
    case invalidRange
    case tooManyEntries
}

public struct ProfileCodec: Sendable {
    public static let maximumEntries = 100_000

    public init() {}

    public func decode(_ data: Data) throws -> ScanProfile {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        let profile = try decoder.decode(ScanProfile.self, from: data)
        try validate(profile)
        return profile
    }

    public func encode(_ profile: ScanProfile) throws -> Data {
        try validate(profile)
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys, .withoutEscapingSlashes]
        return try encoder.encode(profile)
    }

    public func validate(_ profile: ScanProfile) throws {
        guard profile.schemaVersion == 1 else {
            throw ProfileCodecError.unsupportedSchema(profile.schemaVersion)
        }
        guard profile.channels.count + profile.ranges.count <= Self.maximumEntries else {
            throw ProfileCodecError.tooManyEntries
        }
        for channel in profile.channels {
            guard channel.frequencyHz.isFinite, channel.frequencyHz > 0,
                  channel.frequencyHz <= 7_500_000_000 else {
                throw ProfileCodecError.invalidFrequency
            }
        }
        for range in profile.ranges {
            guard range.startHz.isFinite, range.endHz.isFinite, range.stepHz.isFinite,
                  range.startHz > 0, range.endHz >= range.startHz, range.stepHz > 0,
                  range.endHz <= 7_500_000_000 else {
                throw ProfileCodecError.invalidRange
            }
        }
    }
}
