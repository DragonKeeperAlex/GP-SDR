#import <AppKit/AppKit.h>
#import <CoreLocation/CoreLocation.h>
#import <WebKit/WebKit.h>
#import <arpa/inet.h>
#import <netinet/in.h>
#import <sys/socket.h>
#import <unistd.h>

@interface GPSDRAppDelegate : NSObject <NSApplicationDelegate, NSWindowDelegate, WKNavigationDelegate, WKUIDelegate, WKScriptMessageHandler, CLLocationManagerDelegate>
@property(nonatomic, strong) NSWindow *window;
@property(nonatomic, strong) WKWebView *webView;
@property(nonatomic, strong) NSTask *serverProcess;
@property(nonatomic, strong) NSFileHandle *serverLogHandle;
@property(nonatomic) NSInteger serverPort;
@property(nonatomic, copy) NSString *serverToken;
@property(nonatomic, strong) CLLocationManager *locationManager;
@property(nonatomic) BOOL locationRequestPending;
@property(nonatomic, strong) id sleepActivity;
@end

@implementation GPSDRAppDelegate

- (instancetype)init {
    self = [super init];
    if (self) {
        _serverPort = 8073;
        _serverToken = [[[NSUUID UUID] UUIDString] stringByReplacingOccurrencesOfString:@"-" withString:@""];
        _locationManager = [[CLLocationManager alloc] init];
        _locationManager.delegate = self;
        _locationManager.desiredAccuracy = kCLLocationAccuracyHundredMeters;
    }
    return self;
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    self.sleepActivity = [NSProcessInfo.processInfo beginActivityWithOptions:NSActivityIdleSystemSleepDisabled
        reason:@"GP-SDR is monitoring radio activity."];
    [self buildMenus];
    [self buildWindow];
    self.serverPort = [self availableLoopbackPort];
    [self startBundledServer];
    [self loadConsoleWithAttempt:0];
    [NSApp activateIgnoringOtherApps:YES];
}

- (void)applicationWillTerminate:(NSNotification *)notification {
    if (self.serverProcess.running) {
        [self.serverProcess terminate];
        [self.serverProcess waitUntilExit];
    }
    [self.serverLogHandle closeFile];
    if (self.sleepActivity) {
        [NSProcessInfo.processInfo endActivity:self.sleepActivity];
        self.sleepActivity = nil;
    }
}

- (void)windowWillClose:(NSNotification *)notification { [NSApp terminate:nil]; }
- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender { return YES; }

- (void)buildMenus {
    NSMenu *main = [[NSMenu alloc] init];
    NSMenuItem *applicationItem = [[NSMenuItem alloc] init];
    NSMenu *applicationMenu = [[NSMenu alloc] init];
    [applicationMenu addItemWithTitle:@"About GP-SDR" action:@selector(orderFrontStandardAboutPanel:) keyEquivalent:@""];
    [applicationMenu addItem:[NSMenuItem separatorItem]];
    [applicationMenu addItemWithTitle:@"Quit GP-SDR" action:@selector(terminate:) keyEquivalent:@"q"];
    applicationItem.submenu = applicationMenu;
    [main addItem:applicationItem];

    NSMenuItem *viewItem = [[NSMenuItem alloc] init];
    viewItem.title = @"View";
    NSMenu *viewMenu = [[NSMenu alloc] initWithTitle:@"View"];
    NSArray<NSArray *> *views = @[
        @[@"Live", @"1", NSStringFromSelector(@selector(showLive))],
        @[@"Tuner", @"2", NSStringFromSelector(@selector(showTuner))],
        @[@"Activity", @"3", NSStringFromSelector(@selector(showActivity))],
        @[@"Mapper", @"4", NSStringFromSelector(@selector(showMapper))],
        @[@"Profiles", @"5", NSStringFromSelector(@selector(showProfiles))],
        @[@"Decoders", @"6", NSStringFromSelector(@selector(showDecoders))],
        @[@"Hardware", @"7", NSStringFromSelector(@selector(showHardware))],
        @[@"Settings", @"8", NSStringFromSelector(@selector(showSettings))]
    ];
    for (NSArray *entry in views) {
        [viewMenu addItemWithTitle:entry[0] action:NSSelectorFromString(entry[2]) keyEquivalent:entry[1]];
    }
    [viewMenu addItem:[NSMenuItem separatorItem]];
    [viewMenu addItemWithTitle:@"Reload Interface" action:@selector(reloadInterface) keyEquivalent:@"r"];
    viewItem.submenu = viewMenu;
    [main addItem:viewItem];

    NSMenuItem *windowItem = [[NSMenuItem alloc] init];
    windowItem.title = @"Window";
    NSMenu *windowMenu = [[NSMenu alloc] initWithTitle:@"Window"];
    [windowMenu addItemWithTitle:@"Minimize" action:@selector(performMiniaturize:) keyEquivalent:@"m"];
    [windowMenu addItemWithTitle:@"Zoom" action:@selector(performZoom:) keyEquivalent:@""];
    windowItem.submenu = windowMenu;
    [main addItem:windowItem];
    NSApp.mainMenu = main;
}

- (void)buildWindow {
    WKWebViewConfiguration *configuration = [[WKWebViewConfiguration alloc] init];
    configuration.websiteDataStore = WKWebsiteDataStore.defaultDataStore;
    [configuration.userContentController addScriptMessageHandler:self name:@"gpsdrNative"];
    WKUserScript *capabilities = [[WKUserScript alloc]
        initWithSource:@"window.gpsdrNativeCapabilities=['location','localDatabaseFolder'];"
        injectionTime:WKUserScriptInjectionTimeAtDocumentStart
        forMainFrameOnly:YES];
    [configuration.userContentController addUserScript:capabilities];
    configuration.mediaTypesRequiringUserActionForPlayback = WKAudiovisualMediaTypeNone;
    configuration.allowsAirPlayForMediaPlayback = YES;
    self.webView = [[WKWebView alloc] initWithFrame:NSZeroRect configuration:configuration];
    self.webView.navigationDelegate = self;
    self.webView.UIDelegate = self;
    [self.webView setValue:@NO forKey:@"drawsBackground"];

    NSWindowStyleMask style = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
        NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable;
    self.window = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 1240, 820)
        styleMask:style backing:NSBackingStoreBuffered defer:NO];
    self.window.title = @"GP-SDR";
    self.window.titlebarAppearsTransparent = NO;
    self.window.titleVisibility = NSWindowTitleVisible;
    self.window.backgroundColor = [NSColor colorWithRed:0.043 green:0.051 blue:0.063 alpha:1];
    self.window.minSize = NSMakeSize(880, 620);
    self.window.contentView = self.webView;
    self.window.delegate = self;
    [self.window center];
    [self.window makeKeyAndOrderFront:nil];
}

- (void)webView:(WKWebView *)webView
    runOpenPanelWithParameters:(WKOpenPanelParameters *)parameters
    initiatedByFrame:(WKFrameInfo *)frame
    completionHandler:(void (^)(NSArray<NSURL *> * _Nullable URLs))completionHandler {
    NSOpenPanel *panel = [[NSOpenPanel alloc] init];
    panel.canChooseDirectories = parameters.allowsDirectories;
    panel.canChooseFiles = YES;
    panel.allowsMultipleSelection = parameters.allowsMultipleSelection;
    [panel beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse response) {
        completionHandler(response == NSModalResponseOK ? panel.URLs : nil);
    }];
}

- (void)startBundledServer {
    NSString *override = NSProcessInfo.processInfo.environment[@"GPSDR_SERVER"];
    NSString *bundled = [[NSBundle.mainBundle.resourceURL URLByAppendingPathComponent:@"bin/gpsdr-server"] path];
    NSString *executable = override ?: bundled;
    if (![NSFileManager.defaultManager isExecutableFileAtPath:executable]) {
        [self showFailure:@"The receiver service is missing from this application bundle."];
        return;
    }

    NSTask *process = [[NSTask alloc] init];
    process.executableURL = [NSURL fileURLWithPath:executable];
    process.arguments = @[@"-listen", @"0.0.0.0", @"-port", [NSString stringWithFormat:@"%ld", (long)self.serverPort], @"-token", self.serverToken];
    NSMutableDictionary *environment = [NSProcessInfo.processInfo.environment mutableCopy];
    NSString *helperDirectory = [[NSBundle.mainBundle.resourceURL URLByAppendingPathComponent:@"bin"] path];
    if (helperDirectory) environment[@"GPSDR_HELPERS"] = helperDirectory;
    process.environment = environment;
    process.standardInput = NSFileHandle.fileHandleWithNullDevice;

    NSURL *logDirectory = [[NSFileManager.defaultManager URLsForDirectory:NSApplicationSupportDirectory inDomains:NSUserDomainMask].firstObject URLByAppendingPathComponent:@"GP-SDR" isDirectory:YES];
    [NSFileManager.defaultManager createDirectoryAtURL:logDirectory withIntermediateDirectories:YES attributes:nil error:nil];
    NSString *logPath = [[logDirectory URLByAppendingPathComponent:@"server.log"] path];
    if (![NSFileManager.defaultManager fileExistsAtPath:logPath]) {
        [NSFileManager.defaultManager createFileAtPath:logPath contents:nil attributes:nil];
    }
    self.serverLogHandle = [NSFileHandle fileHandleForWritingAtPath:logPath];
    [self.serverLogHandle seekToEndOfFile];
    process.standardOutput = self.serverLogHandle;
    process.standardError = self.serverLogHandle;

    NSError *error = nil;
    if ([process launchAndReturnError:&error]) {
        self.serverProcess = process;
    } else {
        [self showFailure:[NSString stringWithFormat:@"The receiver service could not start: %@", error.localizedDescription]];
    }
}

- (NSInteger)availableLoopbackPort {
    int descriptor = socket(AF_INET, SOCK_STREAM, 0);
    if (descriptor < 0) return 8073;
    struct sockaddr_in address = {0};
    address.sin_len = sizeof(address);
    address.sin_family = AF_INET;
    address.sin_port = 0;
    address.sin_addr.s_addr = inet_addr("127.0.0.1");
    if (bind(descriptor, (struct sockaddr *)&address, sizeof(address)) != 0) { close(descriptor); return 8073; }
    socklen_t length = sizeof(address);
    if (getsockname(descriptor, (struct sockaddr *)&address, &length) != 0) { close(descriptor); return 8073; }
    close(descriptor);
    return ntohs(address.sin_port);
}

- (void)loadConsoleWithAttempt:(NSInteger)attempt {
    NSURL *url = [NSURL URLWithString:[NSString stringWithFormat:@"http://127.0.0.1:%ld/?token=%@", (long)self.serverPort, self.serverToken]];
    [[[NSURLSession sharedSession] dataTaskWithURL:url completionHandler:^(NSData *data, NSURLResponse *response, NSError *error) {
        (void)data;
        (void)error;
        BOOL ready = [(NSHTTPURLResponse *)response statusCode] == 200;
        dispatch_async(dispatch_get_main_queue(), ^{
            if (ready) {
                [self.webView loadRequest:[NSURLRequest requestWithURL:url]];
            } else if (attempt < 120) {
                dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.25 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{ [self loadConsoleWithAttempt:attempt + 1]; });
            } else {
                [self showFailure:@"GP-SDR could not connect to its local receiver service."];
            }
        });
    }] resume];
}

- (void)showFailure:(NSString *)message {
    NSString *html = [NSString stringWithFormat:@"<html><body style='margin:0;background:#0b0d10;color:#edf1f5;font:14px -apple-system;display:grid;place-items:center;height:100vh'><div style='max-width:430px;text-align:center'><h2>GP-SDR</h2><p style='color:#87909c;line-height:1.5'>%@</p><button onclick=\"window.webkit.messageHandlers.gpsdrNative.postMessage({action:'retryInterface'})\" style='margin-top:12px;padding:9px 16px;border:1px solid #2c3540;border-radius:7px;background:#151a20;color:#edf1f5;font:600 13px -apple-system;cursor:pointer'>Retry</button></div></body></html>", message];
    [self.webView loadHTMLString:html baseURL:nil];
}

- (void)showView:(NSString *)view { [self.webView evaluateJavaScript:[NSString stringWithFormat:@"setView('%@')", view] completionHandler:nil]; }
- (void)showLive { [self showView:@"live"]; }
- (void)showTuner { [self showView:@"tuner"]; }
- (void)showActivity { [self showView:@"activity"]; }
- (void)showMapper { [self showView:@"mapper"]; }
- (void)showProfiles { [self showView:@"profiles"]; }
- (void)showDecoders { [self showView:@"decoders"]; }
- (void)showHardware { [self showView:@"hardware"]; }
- (void)showSettings { [self showView:@"settings"]; }
- (void)reloadInterface { [self loadConsoleWithAttempt:0]; }

- (void)userContentController:(WKUserContentController *)userContentController didReceiveScriptMessage:(WKScriptMessage *)message {
    if (![message.name isEqualToString:@"gpsdrNative"] || ![message.body isKindOfClass:NSDictionary.class]) return;
    NSString *action = [(NSDictionary *)message.body objectForKey:@"action"];
    if ([action isEqualToString:@"chooseLocalDatabaseFolder"]) [self chooseLocalDatabaseFolder];
    else if ([action isEqualToString:@"requestLocation"]) [self requestCurrentLocation];
    else if ([action isEqualToString:@"openLocationSettings"]) [self openLocationSettings];
    else if ([action isEqualToString:@"retryInterface"]) [self loadConsoleWithAttempt:0];
}

- (void)chooseLocalDatabaseFolder {
    NSOpenPanel *panel = [NSOpenPanel openPanel];
    panel.title = @"Choose Local Radio Database Folder";
    panel.prompt = @"Choose";
    panel.canChooseDirectories = YES;
    panel.canChooseFiles = NO;
    panel.allowsMultipleSelection = NO;
    panel.canCreateDirectories = YES;
    if ([panel runModal] != NSModalResponseOK || !panel.URL.path) return;
    NSData *data = [NSJSONSerialization dataWithJSONObject:panel.URL.path options:NSJSONWritingFragmentsAllowed error:nil];
    NSString *json = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    [self.webView evaluateJavaScript:[NSString stringWithFormat:@"window.setLocalDatabaseFolder(%@)", json] completionHandler:nil];
}

- (void)requestCurrentLocation {
    if (![CLLocationManager locationServicesEnabled]) {
        [self sendLocationResult:@{@"status": @"error", @"message": @"Location Services are off. Enable them in System Settings.", @"settings": @YES}];
        return;
    }
    switch (self.locationManager.authorizationStatus) {
        case kCLAuthorizationStatusNotDetermined:
            self.locationRequestPending = YES;
            [self.locationManager requestWhenInUseAuthorization];
            break;
        case kCLAuthorizationStatusAuthorizedAlways:
            self.locationRequestPending = YES;
            [self.locationManager requestLocation];
            break;
        case kCLAuthorizationStatusDenied:
        case kCLAuthorizationStatusRestricted:
            [self sendLocationResult:@{@"status": @"error", @"message": @"Location access is off for GP-SDR. Enable it in System Settings > Privacy & Security > Location Services.", @"settings": @YES}];
            break;
    }
}

- (void)locationManagerDidChangeAuthorization:(CLLocationManager *)manager {
    if (!self.locationRequestPending) return;
    switch (manager.authorizationStatus) {
        case kCLAuthorizationStatusAuthorizedAlways:
            [manager requestLocation];
            break;
        case kCLAuthorizationStatusDenied:
        case kCLAuthorizationStatusRestricted:
            self.locationRequestPending = NO;
            [self sendLocationResult:@{@"status": @"error", @"message": @"Location access is off for GP-SDR. Enable it in System Settings > Privacy & Security > Location Services.", @"settings": @YES}];
            break;
        default:
            break;
    }
}

- (void)locationManager:(CLLocationManager *)manager didUpdateLocations:(NSArray<CLLocation *> *)locations {
    CLLocation *location = locations.lastObject;
    if (!self.locationRequestPending || !location) return;
    self.locationRequestPending = NO;
    [self sendLocationResult:@{
        @"status": @"success",
        @"latitude": @(location.coordinate.latitude),
        @"longitude": @(location.coordinate.longitude),
        @"accuracy": @(location.horizontalAccuracy)
    }];
}

- (void)locationManager:(CLLocationManager *)manager didFailWithError:(NSError *)error {
    if (!self.locationRequestPending) return;
    self.locationRequestPending = NO;
    [self sendLocationResult:@{@"status": @"error", @"message": [NSString stringWithFormat:@"Could not read the current location: %@", error.localizedDescription]}];
}

- (void)sendLocationResult:(NSDictionary *)result {
    NSData *data = [NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
    NSString *json = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    dispatch_async(dispatch_get_main_queue(), ^{
        [self.webView evaluateJavaScript:[NSString stringWithFormat:@"window.gpsdrNativeLocationResult(%@)", json] completionHandler:nil];
    });
}

- (void)openLocationSettings {
    NSURL *url = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices"];
    if (url) [NSWorkspace.sharedWorkspace openURL:url];
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationAction:(WKNavigationAction *)navigationAction decisionHandler:(void (^)(WKNavigationActionPolicy))decisionHandler {
    NSString *host = navigationAction.request.URL.host;
    if (host && ![host isEqualToString:@"127.0.0.1"] && ![host isEqualToString:@"localhost"]) {
        if (navigationAction.request.URL) [NSWorkspace.sharedWorkspace openURL:navigationAction.request.URL];
        decisionHandler(WKNavigationActionPolicyCancel);
    } else {
        decisionHandler(WKNavigationActionPolicyAllow);
    }
}

@end

int main(int argc, const char *argv[]) {
    (void)argc;
    (void)argv;
    @autoreleasepool {
        NSApplication *application = NSApplication.sharedApplication;
        GPSDRAppDelegate *delegate = [[GPSDRAppDelegate alloc] init];
        application.delegate = delegate;
        [application run];
    }
    return 0;
}
