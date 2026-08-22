import Foundation

public struct AudioFrame: Sendable, Equatable {
    public var samples: [Int16]
    public var sampleRateHz: Int
    public var channelID: String

    public init(samples: [Int16], sampleRateHz: Int, channelID: String) {
        self.samples = samples
        self.sampleRateHz = sampleRateHz
        self.channelID = channelID
    }
}

public struct AnalogDemodulator: Sendable {
    public init() {}

    public func demodulate(_ frame: IQFrame, mode: Modulation,
                           outputRateHz requestedOutputRate: Int = 48_000,
                           channelID: String = "quick-tune") -> AudioFrame {
        let count = frame.unsignedInterleavedIQ.count / 2
        guard count > 1, frame.sampleRateHz > 0 else {
            return AudioFrame(samples: [], sampleRateHz: requestedOutputRate, channelID: channelID)
        }
        let decimation = max(1, frame.sampleRateHz / max(1, requestedOutputRate))
        let actualRate = frame.sampleRateHz / decimation
        var output: [Int16] = []
        output.reserveCapacity(count / decimation + 1)
        var accumulator = 0.0
        var accumulated = 0
        var previousI = 0.0
        var previousQ = 0.0
        var amDC = 0.0

        for index in 0..<count {
            let i = (Double(frame.unsignedInterleavedIQ[index * 2]) - 127.5) / 127.5
            let q = (Double(frame.unsignedInterleavedIQ[index * 2 + 1]) - 127.5) / 127.5
            let value: Double
            switch mode {
            case .am:
                let magnitude = hypot(i, q)
                amDC = amDC * 0.9995 + magnitude * 0.0005
                value = (magnitude - amDC) * 3.2
            case .nfm, .wfm:
                if index == 0 {
                    value = 0
                } else {
                    let cross = previousI * q - previousQ * i
                    let dot = previousI * i + previousQ * q
                    let discriminator = atan2(cross, dot)
                    value = discriminator * (mode == .wfm ? 0.52 : 1.8)
                }
            case .usb:
                value = i - q
            case .lsb:
                value = i + q
            case .raw:
                value = i
            }
            previousI = i
            previousQ = q
            accumulator += value
            accumulated += 1
            if accumulated == decimation {
                let averaged = accumulator / Double(accumulated)
                output.append(Int16(clamping: Int(averaged.clamped(to: -1...1) * 30_000)))
                accumulator = 0
                accumulated = 0
            }
        }
        return AudioFrame(samples: output, sampleRateHz: actualRate, channelID: channelID)
    }
}

private extension Double {
    func clamped(to range: ClosedRange<Double>) -> Double {
        min(range.upperBound, max(range.lowerBound, self))
    }
}
