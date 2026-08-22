import Foundation

public struct IQFrame: Sendable, Equatable {
    public var unsignedInterleavedIQ: [UInt8]
    public var sampleRateHz: Int
    public var centerFrequencyHz: Double
    public var sequence: UInt64
    public var receivedAt: Date

    public init(unsignedInterleavedIQ: [UInt8], sampleRateHz: Int,
                centerFrequencyHz: Double, sequence: UInt64, receivedAt: Date = .now) {
        self.unsignedInterleavedIQ = unsignedInterleavedIQ
        self.sampleRateHz = sampleRateHz
        self.centerFrequencyHz = centerFrequencyHz
        self.sequence = sequence
        self.receivedAt = receivedAt
    }
}

public enum ReceiverError: Error, Equatable {
    case unavailable(String)
    case alreadyStreaming
    case notStreaming
    case invalidConfiguration(String)
    case transport(String)
}

public protocol SDRReceiver: Actor {
    var descriptor: RadioDescriptor { get }
    func configure(_ configuration: TunerConfiguration) async throws
    func start() async throws -> AsyncThrowingStream<IQFrame, Error>
    func stop() async
    func telemetry() async -> ReceiverTelemetry
}

public actor ReceiverRegistry {
    private var receivers: [String: any SDRReceiver] = [:]

    public init() {}

    public func register(_ receiver: any SDRReceiver) async {
        receivers[await receiver.descriptor.id] = receiver
    }

    public func remove(id: String) {
        receivers.removeValue(forKey: id)
    }

    public func descriptors() async -> [RadioDescriptor] {
        var result: [RadioDescriptor] = []
        for receiver in receivers.values {
            result.append(await receiver.descriptor)
        }
        return result.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
    }

    public func receiver(id: String) -> (any SDRReceiver)? {
        receivers[id]
    }
}
