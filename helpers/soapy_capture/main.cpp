#include <algorithm>
#include <atomic>
#include <cmath>
#include <csignal>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#include <windows.h>
#else
#include <dlfcn.h>
#endif

// GP-SDR loads SoapySDR through its stable C ABI. This keeps the stream helper
// architecture-neutral at build time: the app can ship one Universal helper,
// while the user installs the matching Soapy runtime and device module.
struct SoapySDRDevice;
struct SoapySDRStream;
struct SoapySDRKwargs;

namespace {
constexpr int soapyRx = 1;
constexpr int soapyTimeout = -1;
constexpr const char *soapyCF32 = "CF32";
std::atomic<bool> running{true};

void stop(int) { running = false; }

std::string valueFor(int argc, char **argv, const std::string &name) {
    for (int i = 1; i + 1 < argc; ++i) {
        if (std::string(argv[i]) == name) return argv[i + 1];
    }
    throw std::runtime_error("missing " + name);
}

double optionalValue(int argc, char **argv, const std::string &name, double fallback) {
    for (int i = 1; i + 1 < argc; ++i) {
        if (std::string(argv[i]) == name) return std::stod(argv[i + 1]);
    }
    return fallback;
}

class SoapyAPI {
public:
    using Make = SoapySDRDevice *(*)(const char *);
    using Unmake = int (*)(SoapySDRDevice *);
    using LastError = const char *(*)();
    using ErrorText = const char *(*)(int);
    using SetRate = int (*)(SoapySDRDevice *, int, std::size_t, double);
    using SetFrequency = int (*)(SoapySDRDevice *, int, std::size_t, double, const SoapySDRKwargs *);
    using SetGain = int (*)(SoapySDRDevice *, int, std::size_t, double);
    using SetupStream = SoapySDRStream *(*)(SoapySDRDevice *, int, const char *, const std::size_t *, std::size_t, const SoapySDRKwargs *);
    using CloseStream = int (*)(SoapySDRDevice *, SoapySDRStream *);
    using StreamMTU = std::size_t (*)(const SoapySDRDevice *, SoapySDRStream *);
    using ActivateStream = int (*)(SoapySDRDevice *, SoapySDRStream *, int, long long, std::size_t);
    using DeactivateStream = int (*)(SoapySDRDevice *, SoapySDRStream *, int, long long);
    using ReadStream = int (*)(SoapySDRDevice *, SoapySDRStream *, void *const *, std::size_t, int *, long long *, long);

    SoapyAPI() {
#ifdef _WIN32
        const char *candidates[] = {"SoapySDR.dll", "libSoapySDR.dll"};
        for (const char *candidate : candidates) {
            handle = LoadLibraryA(candidate);
            if (handle) break;
        }
#elif __APPLE__
        const char *candidates[] = {
            "libSoapySDR.0.8.dylib", "libSoapySDR.dylib",
            "/opt/homebrew/lib/libSoapySDR.0.8.dylib", "/opt/homebrew/lib/libSoapySDR.dylib",
            "/usr/local/lib/libSoapySDR.0.8.dylib", "/usr/local/lib/libSoapySDR.dylib"
        };
        for (const char *candidate : candidates) {
            // Soapy hardware modules are loaded later by libSoapySDR and
            // resolve the C API from the process-wide symbol table.
            handle = dlopen(candidate, RTLD_NOW | RTLD_GLOBAL);
            if (handle) break;
        }
#else
        const char *candidates[] = {"libSoapySDR.so.0.8", "libSoapySDR.so"};
        for (const char *candidate : candidates) {
            handle = dlopen(candidate, RTLD_NOW | RTLD_LOCAL);
            if (handle) break;
        }
#endif
        if (!handle) throw std::runtime_error("SoapySDR runtime is not installed");
        make = symbol<Make>("SoapySDRDevice_makeStrArgs");
        unmake = symbol<Unmake>("SoapySDRDevice_unmake");
        lastError = symbol<LastError>("SoapySDRDevice_lastError");
        errorText = symbol<ErrorText>("SoapySDR_errToStr");
        setRate = symbol<SetRate>("SoapySDRDevice_setSampleRate");
        setFrequency = symbol<SetFrequency>("SoapySDRDevice_setFrequency");
        setGain = symbol<SetGain>("SoapySDRDevice_setGain");
        setupStream = symbol<SetupStream>("SoapySDRDevice_setupStream");
        closeStream = symbol<CloseStream>("SoapySDRDevice_closeStream");
        streamMTU = symbol<StreamMTU>("SoapySDRDevice_getStreamMTU");
        activateStream = symbol<ActivateStream>("SoapySDRDevice_activateStream");
        deactivateStream = symbol<DeactivateStream>("SoapySDRDevice_deactivateStream");
        readStream = symbol<ReadStream>("SoapySDRDevice_readStream");
    }

    ~SoapyAPI() {
#ifdef _WIN32
        if (handle) FreeLibrary(handle);
#else
        if (handle) dlclose(handle);
#endif
    }

    SoapyAPI(const SoapyAPI &) = delete;
    SoapyAPI &operator=(const SoapyAPI &) = delete;

    void check(int status, const std::string &operation) const {
        if (status == 0) return;
        const char *detail = lastError ? lastError() : nullptr;
        if (!detail || !*detail) detail = errorText ? errorText(status) : "unknown error";
        throw std::runtime_error(operation + ": " + detail);
    }

    Make make{};
    Unmake unmake{};
    LastError lastError{};
    ErrorText errorText{};
    SetRate setRate{};
    SetFrequency setFrequency{};
    SetGain setGain{};
    SetupStream setupStream{};
    CloseStream closeStream{};
    StreamMTU streamMTU{};
    ActivateStream activateStream{};
    DeactivateStream deactivateStream{};
    ReadStream readStream{};

private:
#ifdef _WIN32
    HMODULE handle{};
#else
    void *handle{};
#endif

    template <typename T> T symbol(const char *name) {
#ifdef _WIN32
        void *address = reinterpret_cast<void *>(GetProcAddress(handle, name));
#else
        void *address = dlsym(handle, name);
#endif
        if (!address) throw std::runtime_error(std::string("SoapySDR is missing ") + name);
        return reinterpret_cast<T>(address);
    }
};
} // namespace

int main(int argc, char **argv) {
    if (argc == 2 && std::string(argv[1]) == "--version") {
        std::cout << "gpsdr-soapy 2\n";
        return 0;
    }
    try {
        const std::string arguments = valueFor(argc, argv, "--device");
        const double frequency = std::stod(valueFor(argc, argv, "--frequency"));
        const double rate = std::stod(valueFor(argc, argv, "--rate"));
        const double gain = optionalValue(argc, argv, "--gain", -1.0);
        SoapyAPI api;

        std::signal(SIGINT, stop);
        std::signal(SIGTERM, stop);
#ifdef _WIN32
        _setmode(_fileno(stdout), _O_BINARY);
#endif

        SoapySDRDevice *device = api.make(arguments.c_str());
        if (!device) {
            const char *message = api.lastError();
            throw std::runtime_error(message && *message ? message : "SoapySDR could not open " + arguments);
        }
        SoapySDRStream *stream = nullptr;
        try {
            api.check(api.setRate(device, soapyRx, 0, rate), "set sample rate");
            api.check(api.setFrequency(device, soapyRx, 0, frequency, nullptr), "set frequency");
            if (gain >= 0.0) api.check(api.setGain(device, soapyRx, 0, gain), "set gain");
            stream = api.setupStream(device, soapyRx, soapyCF32, nullptr, 0, nullptr);
            if (!stream) {
                const char *message = api.lastError();
                throw std::runtime_error(message && *message ? message : "SoapySDR did not create a receive stream");
            }
            api.check(api.activateStream(device, stream, 0, 0, 0), "activate stream");

            const std::size_t mtu = std::max<std::size_t>(1024, api.streamMTU(device, stream));
            std::vector<float> iq(mtu * 2);
            std::vector<std::int8_t> bytes(mtu * 2);
            void *buffers[] = {iq.data()};
            while (running) {
                int flags = 0;
                long long timestamp = 0;
                const int count = api.readStream(device, stream, buffers, mtu, &flags, &timestamp, 500000);
                if (count == soapyTimeout) continue;
                if (count < 0) api.check(count, "read stream");
                for (int i = 0; i < count * 2; ++i) {
                    const float sample = std::clamp(iq[static_cast<std::size_t>(i)], -1.0f, 1.0f);
                    bytes[static_cast<std::size_t>(i)] = static_cast<std::int8_t>(std::lrint(sample * 127.0f));
                }
                if (std::fwrite(bytes.data(), 2, static_cast<std::size_t>(count), stdout) != static_cast<std::size_t>(count)) break;
            }
            api.deactivateStream(device, stream, 0, 0);
            api.closeStream(device, stream);
            api.unmake(device);
            return 0;
        } catch (...) {
            if (stream) {
                api.deactivateStream(device, stream, 0, 0);
                api.closeStream(device, stream);
            }
            api.unmake(device);
            throw;
        }
    } catch (const std::exception &error) {
        std::cerr << "gpsdr-soapy: " << error.what() << '\n';
        return 2;
    }
}
