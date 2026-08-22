import Foundation

public actor SimulatedReceiver: SDRReceiver {
    public nonisolated let descriptor = RadioDescriptor(
        id: "mobile-simulator", name: "Mobile simulator", kind: .simulator,
        serial: nil, connected: true, sampleRateLimit: 2_400_000,
        note: "Synthetic IQ; never reported as received RF."
    )

    private var configuration: TunerConfiguration
    private var streaming = false
    private var continuation: AsyncThrowingStream<IQFrame, Error>.Continuation?
    private var producer: Task<Void, Never>?
    private var status = ReceiverTelemetry.idle

    public init(configuration: TunerConfiguration = .mobileDefault) {
        self.configuration = configuration
    }

    public func configure(_ configuration: TunerConfiguration) throws {
        guard configuration.sampleRateHz > 0,
              configuration.sampleRateHz <= descriptor.sampleRateLimit else {
            throw ReceiverError.invalidConfiguration("Unsupported simulator sample rate")
        }
        self.configuration = configuration
    }

    public func start() throws -> AsyncThrowingStream<IQFrame, Error> {
        guard !streaming else { throw ReceiverError.alreadyStreaming }
        streaming = true
        let stream = AsyncThrowingStream<IQFrame, Error> { continuation in
            self.continuation = continuation
        }
        producer = Task { [weak self] in
            var sequence: UInt64 = 0
            while !Task.isCancelled {
                guard let self else { break }
                let frame = await self.makeFrame(sequence: sequence)
                await self.publish(frame)
                sequence &+= 1
                try? await Task.sleep(for: .milliseconds(40))
            }
        }
        return stream
    }

    public func stop() {
        producer?.cancel()
        producer = nil
        continuation?.finish()
        continuation = nil
        streaming = false
    }

    public func telemetry() -> ReceiverTelemetry { status }

    private func publish(_ frame: IQFrame) {
        status.receivedSamples &+= UInt64(frame.unsignedInterleavedIQ.count / 2)
        status.transferRateBytesPerSecond = Double(frame.unsignedInterleavedIQ.count) / 0.04
        status.signalDBFS = -28
        status.noiseDBFS = -72
        status.updatedAt = .now
        continuation?.yield(frame)
    }

    private func makeFrame(sequence: UInt64) -> IQFrame {
        let sampleCount = 4_096
        let toneHz = 12_500.0
        var bytes = [UInt8](repeating: 0, count: sampleCount * 2)
        for index in 0..<sampleCount {
            let phase = 2 * Double.pi * toneHz * Double(index) / Double(configuration.sampleRateHz)
            bytes[index * 2] = UInt8(clamping: Int(127.5 + cos(phase) * 92))
            bytes[index * 2 + 1] = UInt8(clamping: Int(127.5 + sin(phase) * 92))
        }
        return IQFrame(unsignedInterleavedIQ: bytes, sampleRateHz: configuration.sampleRateHz,
                       centerFrequencyHz: configuration.centerFrequencyHz,
                       sequence: sequence)
    }
}
