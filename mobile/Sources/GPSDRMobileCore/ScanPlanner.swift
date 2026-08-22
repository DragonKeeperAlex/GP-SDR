import Foundation

public struct ScanTarget: Sendable, Equatable, Identifiable {
    public var id: String
    public var name: String
    public var frequencyHz: Double
    public var bandwidthHz: Double
    public var mode: Modulation
    public var dwellMilliseconds: Int
    public var priority: Int
}

public struct ScanPlanner: Sendable {
    public static let maximumGeneratedTargets = 100_000

    public init() {}

    public func targets(for profile: ScanProfile) throws -> [ScanTarget] {
        try ProfileCodec().validate(profile)
        var targets = profile.channels.filter(\.enabled).map { channel in
            ScanTarget(id: channel.id, name: channel.name, frequencyHz: channel.frequencyHz,
                       bandwidthHz: channel.bandwidthHz,
                       mode: Modulation(rawValue: channel.mode.lowercased()) ?? .nfm,
                       dwellMilliseconds: 1_000, priority: channel.priority)
        }
        for range in profile.ranges where range.enabled {
            let rawCount = floor((range.endHz - range.startHz) / range.stepHz) + 1
            guard rawCount.isFinite, rawCount >= 0,
                  rawCount <= Double(Self.maximumGeneratedTargets - targets.count) else {
                throw ProfileCodecError.tooManyEntries
            }
            for index in 0..<Int(rawCount) {
                let frequency = range.startHz + Double(index) * range.stepHz
                targets.append(ScanTarget(
                    id: "\(range.id)-\(index)", name: "\(range.name) \(index + 1)",
                    frequencyHz: frequency, bandwidthHz: max(range.stepHz, 12_500),
                    mode: Modulation(rawValue: range.preferredMode.lowercased()) ?? .nfm,
                    dwellMilliseconds: max(50, range.dwellMilliseconds), priority: 0
                ))
            }
        }
        return targets.sorted {
            $0.priority == $1.priority ? $0.frequencyHz < $1.frequencyHz : $0.priority > $1.priority
        }
    }
}

public struct SignalDetector: Sendable {
    public init() {}

    public func levels(_ frame: IQFrame) -> (signalDBFS: Double, noiseDBFS: Double) {
        let count = frame.unsignedInterleavedIQ.count / 2
        guard count > 0 else { return (-120, -120) }
        var powers: [Double] = []
        powers.reserveCapacity(count)
        for index in 0..<count {
            let i = (Double(frame.unsignedInterleavedIQ[index * 2]) - 127.5) / 127.5
            let q = (Double(frame.unsignedInterleavedIQ[index * 2 + 1]) - 127.5) / 127.5
            powers.append(i * i + q * q)
        }
        powers.sort()
        let signal = powers.reduce(0, +) / Double(powers.count)
        let noiseIndex = min(powers.count - 1, powers.count / 5)
        let noise = max(1e-12, powers[noiseIndex])
        return (10 * log10(max(signal, 1e-12)), 10 * log10(noise))
    }
}
