import SwiftUI
import GPSDRMobileCore

@main
struct GPSDRMobileApp: App {
    @StateObject private var model = MobileAppModel()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(model)
                .preferredColorScheme(.dark)
        }
    }
}
