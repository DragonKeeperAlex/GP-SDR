package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	mu        sync.Mutex
	job       *InstallJob
	onRefresh func()
}

func NewInstaller(onRefresh func()) *Installer {
	return &Installer{onRefresh: onRefresh}
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
			Guide:    "Installs whisper.cpp. After installation, choose or download a local ggml model and set GPSDR_WHISPER_MODEL before starting GP-SDR. The older SIGNALHARBOR_WHISPER_MODEL name remains compatible.",
			GuideURL: "https://github.com/ggml-org/whisper.cpp"}, tools: []string{"whisper-cli"}, formulae: []string{"whisper-cpp"}},
		{component: SetupComponent{ID: "rtl-433", Name: "rtl_433", Category: "decoder",
			Guide:    "Installs rtl_433 for compatible weather stations, TPMS devices, and ISM-band sensors.",
			GuideURL: "https://github.com/merbanan/rtl_433"}, tools: []string{"rtl_433"}, formulae: []string{"rtl_433"}},
		{component: SetupComponent{ID: "p25", Name: "P25 Phase 1/2", Category: "decoder",
			Guide:    "The complete GopherTrunk P25 Phase 1/2 stack is included in GP-SDR packages. It provides native HackRF and RTL-SDR input, control-channel following, voice decoding, recording, and per-talkgroup audio. If this card is not Ready, reinstall the complete GP-SDR package.",
			GuideURL: "https://github.com/MattCheramie/GopherTrunk"}, tools: []string{"gophertrunk"}},
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
			Guide:    "A Premium subscription and an approved developer API key are required. Set GPSDR_RR_USERNAME, GPSDR_RR_PASSWORD, and GPSDR_RR_APP_KEY before launching GP-SDR. Credentials are never stored in shared profiles.",
			GuideURL: "https://wiki.radioreference.com/index.php/RadioReference.com_Web_Service"}},
	}
}

func recipeReady(recipe installerRecipe) bool {
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
		if recipeReady(recipe) {
			component.State, component.Action, component.Note = "ready", "Ready", "Installed and discoverable by GP-SDR."
		} else {
			component.State = "setup"
			component.Installable = runtime.GOOS == "darwin" && brewErr == nil && len(recipe.formulae) > 0
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
	if recipeReady(*selected) {
		return InstallJob{}, errors.New("component is already installed")
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
