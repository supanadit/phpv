package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/viper"
)

type Renderer struct {
	renderer *glamour.TermRenderer
	theme    Theme
	isTTY    bool
}

func NewRenderer() *Renderer {
	theme := Theme(viper.GetString("PHPV_THEME"))
	if theme == "" {
		theme = ThemeDark
	}

	isTTY := checkIsTTY()

	opts := []glamour.TermRendererOption{
		glamour.WithAutoStyle(),
	}

	if theme == ThemeDark {
		opts = append(opts, glamour.WithStandardStyle("dark"))
	} else if theme == ThemeLight {
		opts = append(opts, glamour.WithStandardStyle("light"))
	}

	renderer, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		renderer, _ = glamour.NewTermRenderer(glamour.WithAutoStyle())
	}

	return &Renderer{
		renderer: renderer,
		theme:    theme,
		isTTY:    isTTY,
	}
}

func checkIsTTY() bool {
	return isTerminal()
}

func isTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func (r *Renderer) Render(markdown string) (string, error) {
	if !r.isTTY {
		markdown = stripMarkdown(markdown)
		return markdown, nil
	}

	output, err := r.renderer.Render(markdown)
	if err != nil {
		return markdown, err
	}

	return strings.TrimSuffix(output, "\n"), nil
}

func (r *Renderer) RenderBytes(markdown []byte) ([]byte, error) {
	output, err := r.Render(string(markdown))
	if err != nil {
		return markdown, err
	}
	return []byte(output), nil
}

func (r *Renderer) Renderf(format string, args ...interface{}) (string, error) {
	return r.Render(fmt.Sprintf(format, args...))
}

func stripMarkdown(markdown string) string {
	replacer := strings.NewReplacer(
		"**", "",
		"*", "",
		"`", "",
		"```", "",
		"#", "",
		"##", "",
		"###", "",
		"- ", "",
		"+ ", "",
		"> ", "",
		"---", "",
		"___", "",
	)
	return replacer.Replace(markdown)
}

func (r *Renderer) Theme() Theme {
	return r.theme
}

func (r *Renderer) IsTTY() bool {
	return r.isTTY
}

var defaultRenderer *Renderer

func GetRenderer() *Renderer {
	if defaultRenderer == nil {
		defaultRenderer = NewRenderer()
	}
	return defaultRenderer
}

func Render(markdown string) string {
	output, _ := GetRenderer().Render(markdown)
	return output
}

func Renderf(format string, args ...interface{}) string {
	output, _ := GetRenderer().Renderf(format, args...)
	return output
}
