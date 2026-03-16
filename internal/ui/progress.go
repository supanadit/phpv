package ui

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
)

type Spinner struct {
	frames    []string
	frame     int
	mu        sync.Mutex
	running   bool
	message   string
	stopChan  chan struct{}
	startTime time.Time
}

func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
		frame:    0,
		running:  false,
		stopChan: make(chan struct{}),
	}
}

func (s *Spinner) Start(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.message = message
	s.running = true
	s.startTime = time.Now()
	s.stopChan = make(chan struct{})

	go func() {
		tick := time.NewTicker(80 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				s.mu.Lock()
				s.frame++
				s.mu.Unlock()
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
}

func (s *Spinner) StartWithDisplay(message string) {
	s.Start(message)
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				running := s.running
				s.mu.Unlock()
				if !running {
					return
				}
				fmt.Printf("\r%s", s.View())
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *Spinner) StopWithClear() {
	s.Stop()
	fmt.Print("\r\033[K")
}

func (s *Spinner) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startTime.IsZero() {
		return 0
	}
	return time.Since(s.startTime)
}

func (s *Spinner) View() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ""
	}

	frame := s.frames[s.frame%len(s.frames)]
	elapsed := time.Since(s.startTime)
	timer := formatDuration(elapsed)

	return fmt.Sprintf("%s %s %s", InfoStyle.Render(frame), s.message, DimStyle.Render("["+timer+"]"))
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = msg
}

type ProgressBar struct {
	total       int64
	current     int64
	width       int
	showPercent bool
	prefix      string
	suffix      string
	mu          sync.Mutex
}

func NewProgressBar(total int64) *ProgressBar {
	return &ProgressBar{
		total:       total,
		current:     0,
		width:       40,
		showPercent: true,
	}
}

func (p *ProgressBar) SetTotal(total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
}

func (p *ProgressBar) SetCurrent(current int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
}

func (p *ProgressBar) Add(delta int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current += delta
}

func (p *ProgressBar) View() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.total == 0 {
		return ""
	}

	percent := float64(p.current) / float64(p.total)
	if percent > 1 {
		percent = 1
	}

	filled := int(float64(p.width) * percent)
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	result := fmt.Sprintf("%s[%s]%s", p.prefix, bar, p.suffix)

	if p.showPercent {
		result += fmt.Sprintf(" %d%%", int(percent*100))
	}

	return result
}

type MultiProgress struct {
	bars   []*ProgressBar
	mu     sync.Mutex
	active int
	title  string
}

func NewMultiProgress(title string) *MultiProgress {
	return &MultiProgress{
		title:  title,
		active: -1,
	}
}

func (m *MultiProgress) AddBar(total int64) *ProgressBar {
	m.mu.Lock()
	defer m.mu.Unlock()

	bar := NewProgressBar(total)
	m.bars = append(m.bars, bar)

	if m.active == -1 {
		m.active = 0
	}

	return bar
}

func (m *MultiProgress) SetActive(index int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= 0 && index < len(m.bars) {
		m.active = index
	}
}

func (m *MultiProgress) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.title != "" {
		result := m.title + "\n"
		for i, bar := range m.bars {
			prefix := "  "
			if i == m.active {
				prefix = "▸ "
			}
			bar.prefix = prefix
			result += bar.View() + "\n"
		}
		return strings.TrimSuffix(result, "\n")
	}

	var result string
	for i, bar := range m.bars {
		prefix := "  "
		if i == m.active {
			prefix = "▸ "
		}
		bar.prefix = prefix
		result += bar.View() + "\n"
	}

	return strings.TrimSuffix(result, "\n")
}

type StatusIndicator struct {
	mu    sync.Mutex
	state string
}

func NewStatusIndicator() *StatusIndicator {
	return &StatusIndicator{}
}

func (s *StatusIndicator) Set(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *StatusIndicator) View() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case "success":
		return SuccessStyle.Render("✓")
	case "error":
		return ErrorStyle.Render("✗")
	case "warning":
		return WarningStyle.Render("⚠")
	case "info":
		return InfoStyle.Render("ℹ")
	default:
		return "○"
	}
}

func Truncate(s string, maxLen int) string {
	if runewidth.StringWidth(s) <= maxLen {
		return s
	}

	truncated := runewidth.Truncate(s, maxLen-3, "...")
	return truncated
}

type ByteCount int64

func (b ByteCount) String() string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func ParseByteCount(s string) (int64, error) {
	s = strings.TrimSpace(s)

	s = strings.ToUpper(s)

	multipliers := map[string]int64{
		"B":  1,
		"K":  1024,
		"KB": 1024,
		"M":  1024 * 1024,
		"MB": 1024 * 1024,
		"G":  1024 * 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}

	for unit, mult := range multipliers {
		if strings.HasSuffix(s, unit) {
			numStr := strings.TrimSuffix(s, unit)
			numStr = strings.TrimSpace(numStr)

			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, err
			}

			return int64(val * float64(mult)), nil
		}
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}

	return val, nil
}

func GetByteCountString(bytes int64, decimals int) string {
	if bytes < 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	format := "%." + fmt.Sprintf("%d", decimals) + "f %cB"
	return fmt.Sprintf(format, float64(bytes)/float64(div), "KMGTPE"[exp])
}

var DiscardOutput io.Writer = io.Discard
