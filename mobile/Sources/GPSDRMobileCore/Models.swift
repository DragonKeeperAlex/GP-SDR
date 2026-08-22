import Foundation

public enum RadioKind: String, Codable, Sendable, CaseIterable {
    case rtlSDR = "rtlsdr"
    case hackRF = "hackrf"
    case simulator
}

public enum Modulation: String, Codable, Sendable, CaseIterable {
    case am
    case nfm
    case wfm
    case usb
    case lsb
    case raw
}

public struct ScanRange: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var name: String
    public var startHz: Double
    public var endHz: Double
    public var stepHz: Double
    public var dwellMilliseconds: Int
    public var preferredMode: String
    public var decoder: String?
    public var enabled: Bool

    public init(id: String, name: String, startHz: Double, endHz: Double,
                stepHz: Double, dwellMilliseconds: Int, preferredMode: String,
                decoder: String? = nil, enabled: Bool = true) {
        self.id = id
        self.name = name
        self.startHz = startHz
        self.endHz = endHz
        self.stepHz = stepHz
        self.dwellMilliseconds = dwellMilliseconds
        self.preferredMode = preferredMode
        self.decoder = decoder
        self.enabled = enabled
    }
}

public struct ChannelDefinition: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var name: String
    public var frequencyHz: Double
    public var bandwidthHz: Double
    public var mode: String
    public var decoder: String?
    public var enabled: Bool
    public var priority: Int

    public init(id: String, name: String, frequencyHz: Double, bandwidthHz: Double,
                mode: String, decoder: String? = nil, enabled: Bool = true, priority: Int = 0) {
        self.id = id
        self.name = name
        self.frequencyHz = frequencyHz
        self.bandwidthHz = bandwidthHz
        self.mode = mode
        self.decoder = decoder
        self.enabled = enabled
        self.priority = priority
    }
}

public struct DeviceAssignment: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var deviceID: String?
    public var role: String
    public var target: String?
}

public struct TalkgroupDefinition: Codable, Equatable, Sendable, Identifiable {
    public var id: Int
    public var name: String
    public var mode: String
    public var encrypted: Bool
    public var enabled: Bool
}

public struct P25SystemConfig: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var name: String
    public var controlChannelsHz: [Double]
    public var nac: String
    public var wacn: String
    public var systemID: String
    public var tdmaControl: Bool
    public var talkgroups: [TalkgroupDefinition]
    public var enabled: Bool
}

public struct SurveySettings: Codable, Equatable, Sendable {
    public var noiseMarginDB: Double
    public var revisitSeconds: Int
    public var recordAudio: Bool
    public var recordIQForUnknown: Bool
    public var transcribeVoice: Bool
    public var maxRecordingDays: Int
    public var p25SampleRateHz: Int?

    public static let mobileDefault = SurveySettings(
        noiseMarginDB: 9, revisitSeconds: 30, recordAudio: true,
        recordIQForUnknown: false, transcribeVoice: false, maxRecordingDays: 14,
        p25SampleRateHz: 2_400_000
    )
}

public struct ScanProfile: Codable, Equatable, Sendable, Identifiable {
    public var schemaVersion: Int
    public var id: String
    public var name: String
    public var summary: String
    public var ranges: [ScanRange]
    public var channels: [ChannelDefinition]
    public var deviceAssignments: [DeviceAssignment]
    public var p25Systems: [P25SystemConfig]
    public var settings: SurveySettings
    public var builtIn: Bool

    public init(schemaVersion: Int = 1, id: String, name: String, summary: String = "",
                ranges: [ScanRange] = [], channels: [ChannelDefinition] = [],
                deviceAssignments: [DeviceAssignment] = [], p25Systems: [P25SystemConfig] = [],
                settings: SurveySettings = .mobileDefault, builtIn: Bool = false) {
        self.schemaVersion = schemaVersion
        self.id = id
        self.name = name
        self.summary = summary
        self.ranges = ranges
        self.channels = channels
        self.deviceAssignments = deviceAssignments
        self.p25Systems = p25Systems
        self.settings = settings
        self.builtIn = builtIn
    }
}

public struct TransmissionEvent: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var startedAt: Date
    public var durationSeconds: Double
    public var frequencyHz: Double
    public var bandwidthHz: Double
    public var signalDBFS: Double
    public var noiseDBFS: Double
    public var modulation: String
    public var protocolName: String?
    public var label: String?
    public var deviceID: String
    public var systemName: String?
    public var talkgroupID: UInt32?
    public var sourceRadioID: UInt32?
    public var encrypted: Bool
    public var transcript: String?
    public var callsigns: [String]
    public var ctcssHz: Double?
    public var confidence: Double
    public var audioPath: String?
    public var iqPath: String?
    public var simulated: Bool

    public init(id: String = UUID().uuidString, startedAt: Date = .now,
                durationSeconds: Double, frequencyHz: Double, bandwidthHz: Double,
                signalDBFS: Double, noiseDBFS: Double, modulation: String,
                protocolName: String? = nil, label: String? = nil, deviceID: String,
                systemName: String? = nil, talkgroupID: UInt32? = nil,
                sourceRadioID: UInt32? = nil, encrypted: Bool = false,
                transcript: String? = nil, callsigns: [String] = [], ctcssHz: Double? = nil,
                confidence: Double = 0, audioPath: String? = nil, iqPath: String? = nil,
                simulated: Bool = false) {
        self.id = id
        self.startedAt = startedAt
        self.durationSeconds = durationSeconds
        self.frequencyHz = frequencyHz
        self.bandwidthHz = bandwidthHz
        self.signalDBFS = signalDBFS
        self.noiseDBFS = noiseDBFS
        self.modulation = modulation
        self.protocolName = protocolName
        self.label = label
        self.deviceID = deviceID
        self.systemName = systemName
        self.talkgroupID = talkgroupID
        self.sourceRadioID = sourceRadioID
        self.encrypted = encrypted
        self.transcript = transcript
        self.callsigns = callsigns
        self.ctcssHz = ctcssHz
        self.confidence = confidence
        self.audioPath = audioPath
        self.iqPath = iqPath
        self.simulated = simulated
    }
}

public struct RadioDescriptor: Codable, Equatable, Sendable, Identifiable {
    public var id: String
    public var name: String
    public var kind: RadioKind
    public var serial: String?
    public var connected: Bool
    public var sampleRateLimit: Int
    public var note: String?

    public init(id: String, name: String, kind: RadioKind, serial: String? = nil,
                connected: Bool, sampleRateLimit: Int, note: String? = nil) {
        self.id = id
        self.name = name
        self.kind = kind
        self.serial = serial
        self.connected = connected
        self.sampleRateLimit = sampleRateLimit
        self.note = note
    }
}

public struct TunerConfiguration: Codable, Equatable, Sendable {
    public var centerFrequencyHz: Double
    public var sampleRateHz: Int
    public var bandwidthHz: Int
    public var modulation: Modulation
    public var ppmCorrection: Int
    public var lnaGainDB: Int
    public var vgaGainDB: Int
    public var amplifierEnabled: Bool
    public var biasTEnabled: Bool

    public static let mobileDefault = TunerConfiguration(
        centerFrequencyHz: 462_675_000, sampleRateHz: 2_400_000,
        bandwidthHz: 12_500, modulation: .nfm, ppmCorrection: 0,
        lnaGainDB: 16, vgaGainDB: 20, amplifierEnabled: false, biasTEnabled: false
    )
}

public struct ReceiverTelemetry: Codable, Equatable, Sendable {
    public var receivedSamples: UInt64
    public var droppedSamples: UInt64
    public var transferRateBytesPerSecond: Double
    public var signalDBFS: Double
    public var noiseDBFS: Double
    public var temperatureState: String
    public var updatedAt: Date

    public static let idle = ReceiverTelemetry(
        receivedSamples: 0, droppedSamples: 0, transferRateBytesPerSecond: 0,
        signalDBFS: -120, noiseDBFS: -120, temperatureState: "nominal", updatedAt: .now
    )
}
