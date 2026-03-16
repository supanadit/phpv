package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

type UI struct {
	logger   *Logger
	renderer *Renderer
	spinner  *Spinner
	mu       sync.RWMutex
	quiet    bool
	verbose  bool
}

var (
	defaultUI *UI
	uiOnce    sync.Once
)

func GetUI() *UI {
	uiOnce.Do(func() {
		defaultUI = NewUI()
	})
	return defaultUI
}

func NewUI() *UI {
	quiet := viper.GetBool("PHPV_QUIET")
	verbose := viper.GetBool("PHPV_VERBOSE")

	ui := &UI{
		logger:   NewLogger(),
		renderer: NewRenderer(),
		spinner:  NewSpinner(),
		quiet:    quiet,
		verbose:  verbose,
	}

	ui.logger.SetQuiet(quiet)
	ui.logger.SetVerbose(verbose)

	return ui
}

func (u *UI) Logger() *Logger {
	return u.logger
}

func (u *UI) Renderer() *Renderer {
	return u.renderer
}

func (u *UI) Spinner() *Spinner {
	return u.spinner
}

func (u *UI) SetVerbose(verbose bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.verbose = verbose
	u.logger.SetVerbose(verbose)
}

func (u *UI) SetQuiet(quiet bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.quiet = quiet
	u.logger.SetQuiet(quiet)
}

func (u *UI) IsVerbose() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.verbose
}

func (u *UI) IsQuiet() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.quiet
}

func (u *UI) StartSpinner(message string) {
	if u.IsQuiet() {
		return
	}
	u.spinner.Start(message)
}

func (u *UI) StopSpinner() {
	u.spinner.Stop()
}

func (u *UI) SpinnerView() string {
	return u.spinner.View()
}

func (u *UI) RenderMarkdown(markdown string) string {
	output, err := u.renderer.Render(markdown)
	if err != nil {
		return markdown
	}
	return output
}

func (u *UI) RenderMarkdownf(format string, args ...interface{}) string {
	return u.RenderMarkdown(fmt.Sprintf(format, args...))
}

func (u *UI) PrintBox(title, content string) {
	if u.IsQuiet() {
		return
	}

	box := BoxStyle.Render(title + "\n" + content)
	fmt.Println(box)
}

func (u *UI) PrintHeader(title string) {
	if u.IsQuiet() {
		return
	}
	fmt.Println(HeaderStyle.Render(title))
}

func (u *UI) PrintSubheader(title string) {
	if u.IsQuiet() {
		return
	}
	fmt.Println(SubheaderStyle.Render(title))
}

func (u *UI) PrintSuccess(message string) {
	if u.IsQuiet() {
		return
	}
	fmt.Println(SuccessStyle.Render("✓ ") + message)
}

func (u *UI) PrintError(message string) {
	fmt.Println(ErrorStyle.Render("✗ ") + message)
}

func (u *UI) PrintWarning(message string) {
	if u.IsQuiet() {
		return
	}
	fmt.Println(WarningStyle.Render("⚠ ") + message)
}

func (u *UI) PrintInfo(message string) {
	if u.IsQuiet() {
		return
	}
	fmt.Println(InfoStyle.Render("ℹ ") + message)
}

func (u *UI) PrintDim(message string) {
	if u.IsQuiet() {
		return
	}
	fmt.Println(DimStyle.Render(message))
}

func (u *UI) Println() {
	if u.IsQuiet() {
		return
	}
	fmt.Println()
}

func (u *UI) PrintCheckList(items []string, title string) {
	if u.IsQuiet() {
		return
	}

	if title != "" {
		fmt.Println(SubheaderStyle.Render(title))
	}

	for _, item := range items {
		if strings.HasPrefix(item, "✓") || strings.HasPrefix(item, "✔") {
			fmt.Println(CheckMarkStyle.Render(item))
		} else if strings.HasPrefix(item, "✗") || strings.HasPrefix(item, "✘") {
			fmt.Println(ErrorStyle.Render(item))
		} else {
			fmt.Println("  " + item)
		}
	}
}

func (u *UI) PrintDependencyStatus(depName string, systemVersion string, requirement string, met bool) {
	if u.IsQuiet() {
		return
	}

	if met {
		fmt.Printf("%s %s: system %s (meets requirement ≥%s)\n",
			CheckMarkStyle.Render("✓"),
			depName,
			SuccessStyle.Render(systemVersion),
			requirement)
	} else {
		fmt.Printf("%s %s: system %s (meets requirement ≥%s)\n",
			CheckMarkStyle.Render("✓"),
			depName,
			InfoStyle.Render(systemVersion),
			requirement)
	}
}

func (u *UI) PrintStep(stepNum int, totalSteps int, description string) {
	if u.IsQuiet() {
		return
	}

	stepStr := fmt.Sprintf("Step %d/%d:", stepNum, totalSteps)
	fmt.Printf("\n%s %s\n\n", HeaderStyle.Render(stepStr), description)
}

func (u *UI) PrintSection(title string) {
	if u.IsQuiet() {
		return
	}

	fmt.Printf("\n=== %s ===\n\n", title)
}

func (u *UI) PrintAction(action, target string) {
	if u.IsQuiet() {
		return
	}

	fmt.Printf("%s %s: %s\n", ArrowStyle.Render("→"), action, target)
}

func (u *UI) PrintAlreadyBuilt(name, version string) {
	if u.IsQuiet() {
		return
	}

	fmt.Printf("%s %s %s already built, skipping\n",
		ArrowStyle.Render("→"),
		name,
		DimStyle.Render(version))
}

func (u *UI) PrintBuildComplete(name, version, location string) {
	if u.IsQuiet() {
		return
	}

	fmt.Printf("\n%s Successfully built and installed %s %s to %s\n",
		SuccessStyle.Render("✓"),
		name,
		version,
		CodeStyle.Render(location))
}

func (u *UI) PrintBuildInfo(title string, info map[string]string) {
	if u.IsQuiet() {
		return
	}

	content := ""
	for key, value := range info {
		content += fmt.Sprintf("  %s: %s\n", key, CodeStyle.Render(value))
	}

	u.PrintBox(title, content)
}

var (
	UI_Default       = GetUI
	Logger_Default   = GetLogger
	Renderer_Default = func() *Renderer { return GetRenderer() }
)

func init() {
	viper.SetDefault("PHPV_QUIET", false)
	viper.SetDefault("PHPV_VERBOSE", false)
	viper.SetDefault("PHPV_THEME", "dark")
}
