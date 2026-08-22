import Foundation

public struct HackRFTransportStats: Sendable, Equatable {
    public var receivedBytes: UInt64
    public var droppedBytes: UInt64

    public init(receivedBytes: UInt64, droppedBytes: UInt64) {
        self.receivedBytes = receivedBytes
        self.droppedBytes = droppedBytes
    }
}

/// Receive-only boundary for a future HackRF DriverKit user client. HackRF
/// provides signed 8-bit interleaved IQ; the app core uses unsigned IQ.
public protocol HackRFTransport: Actor {
    func open() async throws -> RadioDescriptor
    func configure(_ configuration: TunerConfiguration) async throws
    func startRX() async throws
    func readSignedIQ(maximumBytes: Int) async throws -> [Int8]
    func stats() async -> HackRFTransportStats
    func stopRX() async
    func close() async
}

public actor HackRFReceiver<Transport: HackRFTransport>: SDRReceiver {
    public private(set) var descriptor: RadioDescriptor
    private let transport: Transport
    private var configuration = TunerConfiguration.mobileDefault
    private var continuation: AsyncThrowingStream<IQFrame, Error>.Continuation?
    private var readerTask: Task<Void, Never>?
    private var sequence: UInt64 = 0
    private var currentTelemetry = ReceiverTelemetry.idle

    public init(transport: Transport) {
        self.transport = transport
        descriptor = RadioDescriptor(
            id: "hackrf-pending", name: "HackRF", kind: .hackRF,
            connected: false, sampleRateLimit: 10_000_000,
            note: "Receive-only DriverKit transport not opened"
        )
    }

    public func connect() async throws { descriptor = try await transport.open() }

    public func configure(_ configuration: TunerConfiguration) async throws {
        guard (2_000_000...descriptor.sampleRateLimit).contains(configuration.sampleRateHz) else {
            throw ReceiverError.invalidConfiguration("HackRF sample rate must be 2-10 MS/s on mobile")
        }
        try await transport.configure(configuration)
        self.configuration = configuration
    }

    public func start() async throws -> AsyncThrowingStream<IQFrame, Error> {
        guard readerTask == nil else { throw ReceiverError.alreadyStreaming }
        if !descriptor.connected { try await connect() }
        try await transport.startRX()
        let stream = AsyncThrowingStream<IQFrame, Error> { continuation in
            self.continuation = continuation
        }
        readerTask = Task { [weak self] in await self?.readLoop() }
        return stream
    }

    public func stop() async {
        readerTask?.cancel()
        readerTask = nil
        await transport.stopRX()
        continuation?.finish()
        continuation = nil
    }

    public func disconnect() async {
        await stop()
        await transport.close()
        descriptor.connected = false
    }

    public func telemetry() -> ReceiverTelemetry { currentTelemetry }

    private func readLoop() async {
        do {
            while !Task.isCancelled {
                let signed = try await transport.readSignedIQ(maximumBytes: 524_288)
                if signed.isEmpty {
                    try await Task.sleep(for: .milliseconds(5))
                    continue
                }
                let unsigned = signed.map { UInt8(bitPattern: $0) &+ 128 }
                let frame = IQFrame(unsignedInterleavedIQ: unsigned,
                                    sampleRateHz: configuration.sampleRateHz,
                                    centerFrequencyHz: configuration.centerFrequencyHz,
                                    sequence: sequence)
                sequence &+= 1
                let levels = SignalDetector().levels(frame)
                let stats = await transport.stats()
                currentTelemetry.receivedSamples = stats.receivedBytes / 2
                currentTelemetry.droppedSamples = stats.droppedBytes / 2
                currentTelemetry.signalDBFS = levels.signalDBFS
                currentTelemetry.noiseDBFS = levels.noiseDBFS
                currentTelemetry.updatedAt = .now
                continuation?.yield(frame)
            }
        } catch is CancellationError {
            continuation?.finish()
        } catch {
            continuation?.finish(throwing: error)
        }
    }
}
