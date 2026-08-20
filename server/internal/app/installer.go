package app

import (
	"bytes"
	"errors"
	"fmt"
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
			Guide:    "Installs whisper.cpp. After installation, choose or download a local ggml model and set GPSDR_WHISPER_MODEL before starting GP-SDR.",
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
			Guide:    "Install DSD-FME from its upstream project, then place dsd-fme on PATH or in GPSDR_HELPERS. Builds and dependencies vary by operating system.",
			GuideURL: "https://github.com/lwvmobile/dsd-fme"}, tools: []string{"dsd-fme"}},
		{component: SetupComponent{ID: "dump1090", Name: "dump1090", Category: "decoder",
			Guide:    "Install a maintained dump1090 build for your operating system and place dump1090 or dump1090-fa on PATH.",
			GuideURL: "https://github.com/flightaware/dump1090"}, tools: []string{"dump1090", "dump1090-fa"}},
		{component: SetupComponent{ID: "multimon-ng", Name: "multimon-ng", Category: "decoder",
			Guide:    "Build or install multimon-ng for pager and signaling decoders, then place multimon-ng on PATH.",
			GuideURL: "https://github.com/EliasOenal/multimon-ng"}, tools: []string{"multimon-ng"}},
		{component: SetupComponent{ID: "acarsdec", Name: "acarsdec", Category: "decoder",
			Guide:    "Build or install acarsdec for ACARS, then place acarsdec on PATH.",
			GuideURL: "https://github.com/TLeconte/acarsdec"}, tools: []string{"acarsdec"}},
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
		ready := recipeReady(recipe)
		if component.ID == "p25-voice" {
			ready = findJMBELibrary(installer.dataDirectory, filepath.Join(userHomeDirectory(), "SDRTrunk")) != ""
		}
		if ready {
			component.State, component.Action, component.Note = "ready", "Ready", "Installed and discoverable by GP-SDR."
		} else {
			component.State = "setup"
			component.Installable = runtime.GOOS == "darwin" && brewErr == nil && len(recipe.formulae) > 0
			if component.ID == "p25-voice" {
				_, creatorErr := findJMBECreator()
				component.Installable = creatorErr == nil
			}
			if component.Installable {
				component.Action = "Install"
				component.Note = "Can be installed with Homebrew from this screen."
				component.Command = brew + " install " + strings.Join(recipe.formulae, " ")
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
	if componentID == "p25-voice" {
		ready = findJMBELibrary(installer.dataDirectory, filepath.Join(userHomeDirectory(), "SDRTrunk")) != ""
	}
	if ready {
		return InstallJob{}, errors.New("component is already installed")
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
