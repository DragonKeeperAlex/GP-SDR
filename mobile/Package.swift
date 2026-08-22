// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "GPSDRMobile",
    platforms: [.iOS(.v16), .macOS(.v13)],
    products: [
        .library(name: "GPSDRMobileCore", targets: ["GPSDRMobileCore"]),
        .executable(name: "GPSDRMobileApp", targets: ["GPSDRMobileApp"])
    ],
    targets: [
        .target(name: "GPSDRMobileCore"),
        .executableTarget(name: "GPSDRMobileApp", dependencies: ["GPSDRMobileCore"]),
        .testTarget(name: "GPSDRMobileCoreTests", dependencies: ["GPSDRMobileCore"])
    ]
)
