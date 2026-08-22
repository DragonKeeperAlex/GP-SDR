import Foundation

public struct SurveyObservation: Sendable, Equatable, Identifiable {
    public var id: String
    public var target: ScanTarget
    public var signalDBFS: Double
    public var noiseDBFS: Double
    public var signalToNoiseDB: Double
    public var observedAt: Date
    public var deviceID: String
    public var simulated: Bool
}

public struct SurveyProgress: Sendable, Equatable {
    public var completed: Int
    public var total: Int
    public var current: ScanTarget?
    public var observations: [SurveyObservation]
    public var finished: Bool

    public var fraction: Double {
        total == 0 ? 1 : Double(completed) / Double(total)
    }
}

public actor SurveyEngine {
    private var activeTask: Task<Void, Never>?

    public init() {}

    public func run(profile: ScanProfile, receiver: any SDRReceiver) throws
        -> AsyncThrowingStream<SurveyProgress, Error> {
        guard activeTask == nil else { throw ReceiverError.alreadyStreaming }
        let targets = try ScanPlanner().targets(for: profile)
        let stream = AsyncThrowingStream<SurveyProgress, Error> { continuation in
            activeTask = Task {
                var observations: [SurveyObservation] = []
                do {
                    for (index, target) in targets.enumerated() {
                        try Task.checkCancellation()
                        continuation.yield(SurveyProgress(
                            completed: index, total: targets.count, current: target,
                            observations: observations, finished: false
                        ))
                        let configuration = TunerConfiguration(
                            centerFrequencyHz: target.frequencyHz,
                            sampleRateHz: min(2_400_000, await receiver.descriptor.sampleRateLimit),
                            bandwidthHz: Int(target.bandwidthHz), modulation: target.mode,
                            ppmCorrection: 0, lnaGainDB: 16, vgaGainDB: 20,
                            amplifierEnabled: false, biasTEnabled: false
                        )
                        try await receiver.configure(configuration)
                        let frames = try await receiver.start()
                        var iterator = frames.makeAsyncIterator()
                        if let frame = try await iterator.next() {
                            let levels = SignalDetector().levels(frame)
                            let snr = levels.signalDBFS - levels.noiseDBFS
                            if snr >= profile.settings.noiseMarginDB {
                                let descriptor = await receiver.descriptor
                                observations.append(SurveyObservation(
                                    id: UUID().uuidString, target: target,
                                    signalDBFS: levels.signalDBFS, noiseDBFS: levels.noiseDBFS,
                                    signalToNoiseDB: snr, observedAt: .now,
                                    deviceID: descriptor.id, simulated: descriptor.kind == .simulator
                                ))
                            }
                        }
                        await receiver.stop()
                    }
                    continuation.yield(SurveyProgress(
                        completed: targets.count, total: targets.count, current: nil,
                        observations: observations, finished: true
                    ))
                    continuation.finish()
                } catch is CancellationError {
                    await receiver.stop()
                    continuation.finish()
                } catch {
                    await receiver.stop()
                    continuation.finish(throwing: error)
                }
                activeTask = nil
            }
        }
        return stream
    }

    public func stop() {
        activeTask?.cancel()
    }
}
