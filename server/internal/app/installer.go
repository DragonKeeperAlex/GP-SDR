package app

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type SetupComponent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	State       string `json:"state"`
	Note        string `json:"note"`
	Installable bool   `json:"installable"`
	Action      string `json:"action"`
	Guide       string `json:"guide"`
	GuideURL    string `json:"guideURL,omitempty"`
	Command     string `json:"command,omitempty"`
}

type InstallJob struct {
	ComponentID string     `json:"componentID"`
	State       string     `json:"state"`
	Message     string     `json:"message"`
	Output      string     `json:"output,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}

type SetupOverview struct {
	Platform       string           `json:"platform"`
	PackageManager string           `json:"packageManager"`
	Components     []SetupComponent `json:"components"`
	Job            *InstallJob      `json:"job,omitempty"`
}

type installerRecipe struct {
	component SetupComponent
	tools     []string
	formulae  []string
}

func isDecoderSuiteComponent(id string) bool {
	switch id {
	case "dsd-fme", "dump1090", "multimon-ng", "acarsdec", "ais":
		return true
	default:
		return false
	}
}

type Installer struct {
	mu            sync.Mutex
	job           *InstallJob
	onRefresh     func()
	dataDirectory string
}

func NewInstaller(dataDirectory string, onRefresh func()) *Installer {
	return &Installer{dataDirectory: dataDirectory, onRefresh: onRefresh}
}

func setupRecipes() []installerRecipe {
	return []installerRecipe{
		{component: SetupComponent{ID: "hackrf", Name: "HackRF tools", Category: "receiver",
			Guide:    "Installs the official open-source HackRF host tools. Connect the radio in HackRF mode, install, then press Refresh.",
			GuideURL: "https://formulae.brew.sh/formula/hackrf"}, tools: []string{"hackrf_info", "hackrf_transfer"}, formulae: []string{"hackrf"}},
		{component: SetupComponent{ID: "rtlsdr", Name: "RTL-SDR tools", Category: "receiver",
			Guide:    "Installs librtlsdr and the rtl_test/rtl_sdr tools used for discovery and live IQ capture.",
			GuideURL: "https://formulae.brew.sh/formula/librtlsdr"}, tools: []string{"rtl_test", "rtl_sdr"}, formulae: []string{"librtlsdr"}},
		{component: SetupComponent{ID: "soapysdr", Name: "SoapySDR", Category: "integration",
			Guide:    "Installs the vendor-neutral SoapySDR runtime. A matching device module is also required. The stream helper is already included in macOS packages; native HackRF and RTL-SDR paths do not require SoapySDR.",
			GuideURL: "https://github.com/pothosware/SoapySDR"}, tools: []string{"SoapySDRUtil"}, formulae: []string{"soapysdr"}},
		{component: SetupComponent{ID: "transcription", Name: "Transcription", Category: "integration",
			Guide:    "Installs whisper.cpp and downloads GP-SDR's checksum-pinned English base model. Processing stays on this computer; no account or API key is required.",
			GuideURL: "https://github.com/ggml-org/whisper.cpp"}, tools: []string{"whisper-cli"}, formulae: []string{"whisper-cpp"}},
		{component: SetupComponent{ID: "rtl-433", Name: "rtl_433", Category: "decoder",
			Guide:    "Installs rtl_433 for compatible weather stations, TPMS devices, and ISM-band sensors.",
			GuideURL: "https://github.com/merbanan/rtl_433"}, tools: []string{"rtl_433"}, formulae: []string{"rtl_433"}},
		{component: SetupComponent{ID: "p25", Name: "P25 Phase 1/2", Category: "decoder",
			Guide:    "SDRTrunk is the GP-SDR P25 engine and is included in complete packages. It provides tested HackRF and RTL-SDR input, Phase 1/2 control decoding, trunk following, call logging, and audio. JMBE must be created once to decode voice; GP-SDR detects an existing SDRTrunk JMBE library.",
			GuideURL: "https://github.com/DSheirer/sdrtrunk"}, tools: []string{"sdr-trunk"}},
		{component: SetupComponent{ID: "p25-voice", Name: "P25 voice codec", Category: "decoder",
			Guide:    "Creates the open-source JMBE voice library locally. The creator downloads and compiles the codec after showing the upstream patent notice; check the rules that apply where you use it.",
			GuideURL: "https://github.com/DSheirer/jmbe"}},
		{component: SetupComponent{ID: "dsd-fme", Name: "DSD-FME", Category: "decoder",
			Guide:    "Install DSD-FME from its upstream project, then place dsd-fme on PATH or in GPSDR_HELPERS. The GP-SDR install guide includes the tested macOS dependency and build commands.",
			GuideURL: "https://github.com/lwvmobile/dsd-fme"}, tools: []string{"dsd-fme"}},
		{component: SetupComponent{ID: "dump1090", Name: "dump1090", Category: "decoder",
			Guide:    "Install a maintained dump1090 build for your operating system and place dump1090 or dump1090-fa on PATH.",
			GuideURL: "https://github.com/flightaware/dump1090"}, tools: []string{"dump1090", "dump1090-fa"}},
		{component: SetupComponent{ID: "multimon-ng", Name: "multimon-ng", Category: "decoder",
			Guide:    "Build or install multimon-ng for pager and signaling decoders, then place multimon-ng on PATH.",
			GuideURL: "https://github.com/EliasOenal/multimon-ng"}, tools: []string{"multimon-ng"}},
		{component: SetupComponent{ID: "acarsdec", Name: "acarsdec", Category: "decoder",
			Guide:    "Build or install acarsdec for ACARS, then place acarsdec on PATH.",
			GuideURL: "https://github.com/f00b4r0/acarsdec"}, tools: []string{"acarsdec"}},
		{component: SetupComponent{ID: "ais", Name: "AIS-catcher", Category: "decoder",
			Guide:    "Install AIS-catcher from its upstream releases or package instructions, then place AIS-catcher on PATH.",
			GuideURL: "https://github.com/jvde-github/AIS-catcher"}, tools: []string{"AIS-catcher", "ais-catcher"}},
		{component: SetupComponent{ID: "radioreference", Name: "RadioReference", Category: "integration",
			Guide:    "A Premium subscription and an approved application API key are required. On macOS, save them to Keychain from Settings. Credentials are never stored in shared profiles or Mapper exports.",
			GuideURL: "https://www.radioreference.com/account/api/apply"}},
	}
}

func recipeReady(recipe installerRecipe) bool {
	if recipe.component.ID == "p25" {
		_, err := findSDRTrunk()
		return err == nil
	}
	if recipe.component.ID == "p25-voice" {
		return findJMBELibrary("", filepath.Join(userHomeDirectory(), "SDRTrunk")) != ""
	}
	if len(recipe.tools) == 0 {
		return false
	}
	if recipe.component.ID == "dump1090" || recipe.component.ID == "ais" {
		for _, tool := range recipe.tools {
			if _, err := findTool(tool); err == nil {
				return true
			}
		}
		return false
	}
	for _, tool := range recipe.tools {
		if _, err := findTool(tool); err != nil {
			return false
		}
	}
	return true
}

func (installer *Installer) Overview() SetupOverview {
	brew, brewErr := findHomebrew()
	overview := SetupOverview{Platform: runtime.GOOS, PackageManager: "manual"}
	if runtime.GOOS == "darwin" && brewErr == nil {
		overview.PackageManager = brew
	}
	for _, recipe := range setupRecipes() {
		component := recipe.component
		applyPlatformSetupGuidance(&component)
		ready := recipeReady(recipe)
		if component.ID == "transcription" {
			status := NewTranscriber(installer.dataDirectory).Status()
			ready = status.State == "ready"
			component.Note = status.Note
		}
		if component.ID == "p25-voice" {
			ready = findJMBELibrary(installer.dataDirectory, filepath.Join(userHomeDirectory(), "SDRTrunk")) != ""
		}
		if ready {
			component.State, component.Action = "ready", "Ready"
			if component.Note == "" {
				component.Note = "Installed and discoverable by GP-SDR."
			}
		} else {
			component.State = "setup"
			component.Installable = runtime.GOOS == "darwin" && brewErr == nil && len(recipe.formulae) > 0
			usesSuiteInstaller := false
			if runtime.GOOS == "darwin" && brewErr == nil && isDecoderSuiteComponent(component.ID) {
				if _, scriptErr := findOptionalDecoderInstaller(); scriptErr == nil {
					usesSuiteInstaller = true
					component.Installable = true
					component.Note = "GP-SDR can install the complete optional decoder suite from maintained upstream sources."
					component.Command = "GP-SDR → Hardware → Install"
				}
			}
			if component.ID == "p25-voice" {
				_, creatorErr := findJMBECreator()
				component.Installable = creatorErr == nil
			}
			if component.Installable {
				component.Action = "Install"
				if !usesSuiteInstaller {
					component.Note = "Can be installed with Homebrew from this screen."
					component.Command = brew + " install " + strings.Join(recipe.formulae, " ")
				}
			} else {
				component.Action = "How to"
				component.Note = "Manual setup is required on this platform."
			}
		}
		overview.Components = append(overview.Components, component)
	}
	installer.mu.Lock()
	if installer.job != nil {
		copy := *installer.job
		overview.Job = &copy
	}
	installer.mu.Unlock()
	return overview
}

func applyPlatformSetupGuidance(component *SetupComponent) {
	if component == nil || runtime.GOOS == "darwin" {
		return
	}
	if runtime.GOOS == "linux" {
		switch component.ID {
		case "hackrf":
			component.Guide = "Install the distribution HackRF user-space tools and udev rules, reconnect the receiver, then press Refresh."
			component.Command = "sudo apt install hackrf"
		case "rtlsdr":
			component.Guide = "Install rtl-sdr and its udev rules, add your user to the required device-access group if your distribution uses one, reconnect, then press Refresh."
			component.Command = "sudo apt install rtl-sdr"
		case "soapysdr":
			component.Guide = "Install SoapySDR plus the hardware module for the exact receiver, then press Refresh."
			component.Command = "sudo apt install soapysdr-tools"
		case "rtl-433":
			component.Guide = "Install rtl_433 from your distribution, then press Refresh."
			component.Command = "sudo apt install rtl-433"
		}
		return
	}
	if runtime.GOOS == "windows" {
		switch component.ID {
		case "hackrf":
			component.Guide = "Install the official HackRF Windows host tools. If Windows cannot claim the radio, use Zadig only on the exact HackRF interface to select WinUSB, reconnect, then press Refresh."
			component.GuideURL = "https://github.com/greatscottgadgets/hackrf/releases"
		case "rtlsdr":
			component.Guide = "Use Zadig on Bulk-In Interface 0 of the exact RTL-SDR and select WinUSB. Do not replace a keyboard, mouse, storage, or unrelated USB driver. Then add rtl_sdr.exe and rtl_test.exe beside GP-SDR and press Refresh."
			component.GuideURL = "https://zadig.akeo.ie/"
		case "soapysdr":
			component.Guide = "Install the official PothosSDR Windows bundle and the module for the exact receiver, then press Refresh."
			component.GuideURL = "https://downloads.myriadrf.org/builds/PothosSDR/"
		}
	}
}

func (installer *Installer) Start(componentID string) (InstallJob, error) {
	var selected *installerRecipe
	for _, recipe := range setupRecipes() {
		if recipe.component.ID == componentID {
			copy := recipe
			selected = &copy
			break
		}
	}
	if selected == nil {
		return InstallJob{}, errors.New("unknown setup component")
	}
	ready := recipeReady(*selected)
	if componentID == "transcription" {
		ready = NewTranscriber(installer.dataDirectory).Status().State == "ready"
	}
	if componentID == "p25-voice" {
		ready = findJMBELibrary(installer.dataDirectory, filepath.Join(userHomeDirectory(), "SDRTrunk")) != ""
	}
	if ready {
		return InstallJob{}, errors.New("component is already installed")
	}
	if isDecoderSuiteComponent(componentID) && runtime.GOOS == "darwin" {
		script, scriptErr := findOptionalDecoderInstaller()
		if scriptErr != nil {
			return InstallJob{}, errors.New("the bundled optional-decoder installer is missing; reinstall GP-SDR or open How to")
		}
		installer.mu.Lock()
		if installer.job != nil && installer.job.State == "installing" {
			installer.mu.Unlock()
			return InstallJob{}, errors.New("another component is already installing")
		}
		job := &InstallJob{ComponentID: componentID, State: "installing", Message: "Installing the optional decoder suite…", StartedAt: time.Now()}
		installer.job = job
		installer.mu.Unlock()
		go installer.runOptionalDecoderInstaller(script, job)
		return *job, nil
	}
	if componentID == "p25-voice" {
		creator, creatorErr := findJMBECreator()
		if creatorErr != nil {
			return InstallJob{}, errors.New("the JMBE creator is missing; reinstall the complete GP-SDR package")
		}
		installer.mu.Lock()
		if installer.job != nil && installer.job.State == "installing" {
			installer.mu.Unlock()
			return InstallJob{}, errors.New("another component is already installing")
		}
		job := &InstallJob{ComponentID: componentID, State: "installing", Message: "Creating the P25 voice codec…", StartedAt: time.Now()}
		installer.job = job
		installer.mu.Unlock()
		go installer.runJMBECreator(creator, job)
		return *job, nil
	}
	brew, err := findHomebrew()
	if runtime.GOOS != "darwin" || err != nil || len(selected.formulae) == 0 {
		return InstallJob{}, errors.New("automatic installation is unavailable; open How to for platform instructions")
	}
	installer.mu.Lock()
	if installer.job != nil && installer.job.State == "installing" {
		installer.mu.Unlock()
		return InstallJob{}, errors.New("another component is already installing")
	}
	job := &InstallJob{ComponentID: componentID, State: "installing", Message: "Installing " + selected.component.Name + "…", StartedAt: time.Now()}
	installer.job = job
	installer.mu.Unlock()
	go installer.run(*selected, brew, job)
	copy := *job
	return copy, nil
}

func findOptionalDecoderInstaller() (string, error) {
	var candidates []string
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(base, "..", "Resources", "Scripts", "install_optional_decoders_macos.sh"))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(workingDirectory, "Scripts", "install_optional_decoders_macos.sh"),
			filepath.Join(workingDirectory, "..", "Scripts", "install_optional_decoders_macos.sh"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", errors.New("optional decoder installer not found")
}

func (installer *Installer) runOptionalDecoderInstaller(script string, job *InstallJob) {
	command := exec.Command("/bin/sh", script)
	command.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_INSTALL_CLEANUP=1",
		"PATH="+strings.Join([]string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}, string(os.PathListSeparator)))
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	err := command.Run()
	finished := time.Now()
	message, state := "Optional decoder suite installed. Refreshing hardware…", "ready"
	if err != nil {
		state, message = "error", fmt.Sprintf("Decoder installation failed: %v", err)
	}
	text := output.String()
	if len(text) > 24_000 {
		text = text[len(text)-24_000:]
	}
	installer.mu.Lock()
	job.State, job.Message, job.Output, job.FinishedAt = state, message, text, &finished
	installer.mu.Unlock()
	if err == nil && installer.onRefresh != nil {
		installer.onRefresh()
	}
}

func userHomeDirectory() string {
	home, _ := os.UserHomeDir()
	return home
}

func findJMBECreator() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("GPSDR_JMBE_CREATOR")); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured, nil
		}
	}
	name := "creator"
	if runtime.GOOS == "windows" {
		name = "creator.bat"
	}
	var candidates []string
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(base, "..", "jmbe-creator-"+runtime.GOARCH, "bin", name),
			filepath.Join(base, "..", "jmbe-creator", "bin", name),
			filepath.Join(base, "jmbe-creator", "bin", name),
		)
	}
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		candidates = append(candidates, filepath.Join("/usr/lib/gp-sdr/jmbe-creator/bin", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Clean(candidate)); err == nil && !info.IsDir() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", errors.New("JMBE creator not found")
}

func (installer *Installer) runJMBECreator(creator string, job *InstallJob) {
	directory := filepath.Join(installer.dataDirectory, "Components", "JMBE")
	_ = os.MkdirAll(directory, 0o700)
	destination := filepath.Join(directory, "jmbe-1.0.9.jar")
	command := platformCommand(creator, destination)
	command.Env = os.Environ()
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	err := command.Run()
	finished := time.Now()
	state, message := "ready", "P25 voice codec created."
	if err != nil {
		state, message = "error", fmt.Sprintf("P25 voice setup failed: %v", err)
	}
	text := output.String()
	if len(text) > 24_000 {
		text = text[len(text)-24_000:]
	}
	installer.mu.Lock()
	job.State, job.Message, job.Output, job.FinishedAt = state, message, text, &finished
	installer.mu.Unlock()
	if err == nil && installer.onRefresh != nil {
		installer.onRefresh()
	}
}

func (installer *Installer) run(recipe installerRecipe, brew string, job *InstallJob) {
	arguments := append([]string{"install"}, recipe.formulae...)
	command := exec.Command(brew, arguments...)
	command.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_INSTALL_CLEANUP=1",
		"PATH="+strings.Join([]string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}, string(os.PathListSeparator)))
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	err := command.Run()
	if err == nil && recipe.component.ID == "transcription" {
		installer.mu.Lock()
		job.Message = "Downloading the offline English speech model…"
		installer.mu.Unlock()
		if modelErr := installer.downloadWhisperModel(); modelErr != nil {
			err = modelErr
		}
	}
	finished := time.Now()
	message, state := recipe.component.Name+" installed. Refreshing hardware…", "ready"
	if err != nil {
		state = "error"
		message = fmt.Sprintf("Installation failed: %v", err)
	}
	text := output.String()
	if len(text) > 24_000 {
		text = text[len(text)-24_000:]
	}
	installer.mu.Lock()
	job.State, job.Message, job.Output, job.FinishedAt = state, message, text, &finished
	installer.mu.Unlock()
	if err == nil && installer.onRefresh != nil {
		installer.onRefresh()
	}
}

const whisperBaseEnglishModelURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin"
const whisperBaseEnglishModelSHA256 = "a03779c86df3323075f5e796cb2ce5029f00ec8869eee3fdfb897afe36c6d002"

func (installer *Installer) downloadWhisperModel() error {
	directory := filepath.Join(installer.dataDirectory, "Components", "Whisper")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	destination := filepath.Join(directory, "ggml-base.en.bin")
	if info, err := os.Stat(destination); err == nil && info.Size() > 100*1024*1024 {
		return nil
	}
	request, err := http.NewRequest(http.MethodGet, whisperBaseEnglishModelURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 20 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("offline speech model download failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("offline speech model download returned HTTP %d", response.StatusCode)
	}
	temporary := destination + ".part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, 1024*1024*1024))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written < 100*1024*1024 {
		_ = os.Remove(temporary)
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return fmt.Errorf("offline speech model download was incomplete")
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != whisperBaseEnglishModelSHA256 {
		_ = os.Remove(temporary)
		return fmt.Errorf("offline speech model checksum did not match the pinned release")
	}
	return os.Rename(temporary, destination)
}
