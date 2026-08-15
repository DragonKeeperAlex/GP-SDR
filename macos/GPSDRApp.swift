import AppKit
import Darwin
import WebKit

final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate, WKNavigationDelegate {
    private var window: NSWindow!
    private var webView: WKWebView!
    private var serverProcess: Process?
    private var serverPort = 8073

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        buildMenus()
        buildWindow()
        serverPort = availableLoopbackPort()
        startBundledServer()
        loadConsole(attempt: 0)
        NSApp.activate(ignoringOtherApps: true)
    }

    func applicationWillTerminate(_ notification: Notification) {
        if let process = serverProcess, process.isRunning {
            process.terminate()
            process.waitUntilExit()
        }
    }

    func windowWillClose(_ notification: Notification) { NSApp.terminate(nil) }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }

    private func buildMenus() {
        let main = NSMenu()
        let applicationItem = NSMenuItem()
        let applicationMenu = NSMenu()
        applicationMenu.addItem(withTitle: "About GP-SDR", action: #selector(NSApplication.orderFrontStandardAboutPanel(_:)), keyEquivalent: "")
        applicationMenu.addItem(.separator())
        applicationMenu.addItem(withTitle: "Quit GP-SDR", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        applicationItem.submenu = applicationMenu
        main.addItem(applicationItem)

        let viewItem = NSMenuItem()
        viewItem.title = "View"
        let viewMenu = NSMenu(title: "View")
        for (title, key, action) in [
            ("Live", "1", #selector(showLive)), ("Tuner", "2", #selector(showTuner)),
            ("Activity", "3", #selector(showActivity)), ("Profiles", "4", #selector(showProfiles)),
            ("Decoders", "5", #selector(showDecoders)), ("Hardware", "6", #selector(showHardware)),
            ("Settings", "7", #selector(showSettings))
        ] {
            viewMenu.addItem(withTitle: title, action: action, keyEquivalent: key)
        }
        viewMenu.addItem(.separator())
        viewMenu.addItem(withTitle: "Reload Interface", action: #selector(reloadInterface), keyEquivalent: "r")
        viewItem.submenu = viewMenu
        main.addItem(viewItem)

        let windowItem = NSMenuItem()
        windowItem.title = "Window"
        let windowMenu = NSMenu(title: "Window")
        windowMenu.addItem(withTitle: "Minimize", action: #selector(NSWindow.performMiniaturize(_:)), keyEquivalent: "m")
        windowMenu.addItem(withTitle: "Zoom", action: #selector(NSWindow.performZoom(_:)), keyEquivalent: "")
        windowItem.submenu = windowMenu
        main.addItem(windowItem)
        NSApp.mainMenu = main
    }

    private func buildWindow() {
        let configuration = WKWebViewConfiguration()
        configuration.websiteDataStore = .default()
        webView = WKWebView(frame: .zero, configuration: configuration)
        webView.navigationDelegate = self
        webView.setValue(false, forKey: "drawsBackground")

        window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1240, height: 820),
            styleMask: [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.title = "GP-SDR"
        window.titlebarAppearsTransparent = true
        window.titleVisibility = .hidden
        window.backgroundColor = NSColor(red: 0.043, green: 0.051, blue: 0.063, alpha: 1)
        window.minSize = NSSize(width: 880, height: 620)
        window.contentView = webView
        window.delegate = self
        window.center()
        window.makeKeyAndOrderFront(nil)
    }

    private func startBundledServer() {
        let environmentOverride = ProcessInfo.processInfo.environment["GPSDR_SERVER"]
        let bundled = Bundle.main.resourceURL?.appendingPathComponent("bin/gpsdr-server").path
        guard let executable = environmentOverride ?? bundled,
              FileManager.default.isExecutableFile(atPath: executable) else {
            showFailure("The receiver service is missing from this application bundle.")
            return
        }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = ["-listen", "127.0.0.1", "-port", String(serverPort)]
        process.standardInput = FileHandle.nullDevice
        var environment = ProcessInfo.processInfo.environment
        if let helperDirectory = Bundle.main.resourceURL?.appendingPathComponent("bin").path {
            environment["GPSDR_HELPERS"] = helperDirectory
        }
        process.environment = environment

        let logDirectory = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            .appendingPathComponent("GP-SDR", isDirectory: true)
        try? FileManager.default.createDirectory(at: logDirectory, withIntermediateDirectories: true)
        let logURL = logDirectory.appendingPathComponent("server.log")
        if !FileManager.default.fileExists(atPath: logURL.path) {
            FileManager.default.createFile(atPath: logURL.path, contents: nil)
        }
        if let handle = try? FileHandle(forWritingTo: logURL) {
            _ = try? handle.seekToEnd()
            process.standardOutput = handle
            process.standardError = handle
        }

        do {
            try process.run()
            serverProcess = process
        } catch {
            showFailure("The receiver service could not start: \(error.localizedDescription)")
        }
    }

    private func availableLoopbackPort() -> Int {
        let descriptor = Darwin.socket(AF_INET, SOCK_STREAM, 0)
        guard descriptor >= 0 else { return 8073 }
        defer { Darwin.close(descriptor) }
        var address = sockaddr_in()
        address.sin_len = UInt8(MemoryLayout<sockaddr_in>.size)
        address.sin_family = sa_family_t(AF_INET)
        address.sin_port = 0
        address.sin_addr = in_addr(s_addr: inet_addr("127.0.0.1"))
        let bound = withUnsafePointer(to: &address) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.bind(descriptor, $0, socklen_t(MemoryLayout<sockaddr_in>.size))
            }
        }
        guard bound == 0 else { return 8073 }
        var length = socklen_t(MemoryLayout<sockaddr_in>.size)
        let named = withUnsafeMutablePointer(to: &address) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.getsockname(descriptor, $0, &length)
            }
        }
        guard named == 0 else { return 8073 }
        return Int(UInt16(bigEndian: address.sin_port))
    }

    private func loadConsole(attempt: Int) {
        guard let url = URL(string: "http://127.0.0.1:\(serverPort)/") else { return }
        URLSession.shared.dataTask(with: url) { [weak self] _, response, _ in
            let ready = (response as? HTTPURLResponse)?.statusCode == 200
            DispatchQueue.main.async {
                guard let self else { return }
                if ready {
                    self.webView.load(URLRequest(url: url))
                } else if attempt < 30 {
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) { self.loadConsole(attempt: attempt + 1) }
                } else {
                    self.showFailure("GP-SDR could not connect to its local receiver service.")
                }
            }
        }.resume()
    }

    private func showFailure(_ message: String) {
        let html = """
        <html><body style="margin:0;background:#0b0d10;color:#edf1f5;font:14px -apple-system;display:grid;place-items:center;height:100vh">
        <div style="max-width:430px;text-align:center"><h2>GP-SDR</h2><p style="color:#87909c;line-height:1.5">\(message)</p></div>
        </body></html>
        """
        webView?.loadHTMLString(html, baseURL: nil)
    }

    private func show(view: String) {
        webView?.evaluateJavaScript("setView('\(view)')")
    }

    @objc private func showLive() { show(view: "live") }
    @objc private func showTuner() { show(view: "tuner") }
    @objc private func showActivity() { show(view: "activity") }
    @objc private func showProfiles() { show(view: "profiles") }
    @objc private func showDecoders() { show(view: "decoders") }
    @objc private func showHardware() { show(view: "hardware") }
    @objc private func showSettings() { show(view: "settings") }
    @objc private func reloadInterface() { webView?.reload() }

    func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction,
                 decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        if let host = navigationAction.request.url?.host,
           host != "127.0.0.1" && host != "localhost" {
            if let url = navigationAction.request.url { NSWorkspace.shared.open(url) }
            decisionHandler(.cancel)
        } else {
            decisionHandler(.allow)
        }
    }
}

@main
struct GPSDRApplication {
    static func main() {
        let application = NSApplication.shared
        let delegate = AppDelegate()
        application.delegate = delegate
        application.run()
    }
}
