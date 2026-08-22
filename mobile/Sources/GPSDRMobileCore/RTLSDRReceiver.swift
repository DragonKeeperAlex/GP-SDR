import Foundation

public struct RTLSDRTransportStats: Sendable, Equatable {
    public var writtenBytes: UInt64
    public var droppedBytes: UInt64

    public init(writtenBytes: UInt64, droppedBytes: UInt64) {
        self.writtenBytes = writtenBytes
        self.droppedBytes = droppedBytes
    }
}

/// Small app-facing boundary around the GPL DriverKit implementation. The live
/// adapter wraps its IOKit user client; tests use an in-memory implementation.
public protocol RTLSDRTransport: Actor {
    func open() async throws -> RadioDescriptor
    func configure(_ configuration: TunerConfiguration) async throws
    func startStream() async throws
    func readIQ(maximumBytes: Int) async throws -> [UInt8]
    func stats() async -> RTLSDRTransportStats
    func stopStream() async
    func close() async
}

public actor RTLSDRReceiver<Transport: RTLSDRTransport>: SDRReceiver {
    public private(set) var descriptor: RadioDescriptor
    private let transport: Transport
    private var configuration = TunerConfiguration.mobileDefault
    private var continuation: AsyncThrowingStream<IQFrame, Error>.Continuation?
    private var readerTask: Task<Void, Never>?
    private var sequence: UInt64 = 0
    private var currentTelemetry = ReceiverTelemetry.idle

    public init(transport: Transport) {
        self.transport = transport
        self.descriptor = RadioDescriptor(
            id: "rtlsdr-pending", name: "RTL-SDR", kind: .rtlSDR,
            connected: false, sampleRateLimit: 3_200_000, note: "Driver not opened"
        )
    }

    public func connect() async throws {
        descriptor = try await transport.open()
    }

    public func configure(_ configuration: TunerConfiguration) async throws {
        guard configuration.sampleRateHz > 0,
              configuration.sampleRateHz <= descriptor.sampleRateLimit else {
            throw ReceiverError.invalidConfiguration("RTL-SDR sample rate is outside the device limit")
        }
        try await transport.configure(configuration)
        self.configuration = configuration
    }

    public func start() async throws -> AsyncThrowingStream<IQFrame, Error> {
        guard readerTask == nil else { throw ReceiverError.alreadyStreaming }
        if !descriptor.connected { try await connect() }
        try await transport.startStream()
        let stream = AsyncThrowingStream<IQFrame, Error> { continuation in
            self.continuation = continuation
        }
        readerTask = Task { [weak self] in
            guard let self else { return }
            await self.readLoop()
        }
        return stream
    }

    public func stop() async {
        readerTask?.cancel()
        readerTask = nil
        await transport.stopStream()
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
                let bytes = try await transport.readIQ(maximumBytes: 262_144)
                if bytes.isEmpty {
                    try await Task.sleep(for: .milliseconds(5))
                    continue
                }
                let frame = IQFrame(unsignedInterleavedIQ: bytes,
                                    sampleRateHz: configuration.sampleRateHz,
                                    centerFrequencyHz: configuration.centerFrequencyHz,
                                    sequence: sequence)
                sequence &+= 1
                let levels = SignalDetector().levels(frame)
                let stats = await transport.stats()
                currentTelemetry.receivedSamples = stats.writtenBytes / 2
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
