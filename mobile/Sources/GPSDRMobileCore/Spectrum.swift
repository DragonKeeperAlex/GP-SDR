import Foundation

public struct SpectrumPoint: Equatable, Sendable, Identifiable {
    public var id: Int { bin }
    public var bin: Int
    public var frequencyHz: Double
    public var powerDBFS: Double
}

public struct SpectrumAnalyzer: Sendable {
    public init() {}

    public func analyze(_ frame: IQFrame, bins requestedBins: Int = 256) -> [SpectrumPoint] {
        let complexCount = frame.unsignedInterleavedIQ.count / 2
        guard complexCount >= 8 else { return [] }
        let bins = max(8, min(requestedBins, complexCount))
        let stride = max(1, complexCount / bins)
        var output: [SpectrumPoint] = []
        output.reserveCapacity(bins)

        for bin in 0..<bins {
            let start = min(bin * stride, complexCount - 1)
            let end = min(start + stride, complexCount)
            var energy = 0.0
            for sample in start..<end {
                let i = (Double(frame.unsignedInterleavedIQ[sample * 2]) - 127.5) / 127.5
                let q = (Double(frame.unsignedInterleavedIQ[sample * 2 + 1]) - 127.5) / 127.5
                energy += i * i + q * q
            }
            let mean = energy / Double(max(1, end - start))
            let db = 10 * log10(max(mean, 1e-12))
            let offset = (Double(bin) / Double(bins) - 0.5) * Double(frame.sampleRateHz)
            output.append(SpectrumPoint(bin: bin,
                                        frequencyHz: frame.centerFrequencyHz + offset,
                                        powerDBFS: db))
        }
        return output
    }
}
