package pattern

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

type PatternRepository interface {
	RegisterPatterns(patterns []domain.URLPattern)
	MatchPattern(name string, v *domain.Version) (domain.URLPattern, error)
	MatchPatterns(name string, v *domain.Version) ([]domain.URLPattern, error)
	MatchPatternByType(name, sourceType, targetOS, targetArch string, v *domain.Version) (domain.URLPattern, error)
	BuildURL(pattern domain.URLPattern, v *domain.Version) (string, error)
	BuildURLs(pattern domain.URLPattern, v *domain.Version) ([]string, error)
	BuildURLByType(name, version, sourceType string) (string, error)
}

type Service struct {
	index map[string][]domain.URLPattern
}

func NewService() *Service {
	return &Service{
		index: make(map[string][]domain.URLPattern),
	}
}

func (s *Service) RegisterPatterns(patterns []domain.URLPattern) {
	for _, p := range patterns {
		s.index[p.Name] = append(s.index[p.Name], p)
	}
}

func (s *Service) MatchPattern(name string, v *domain.Version) (domain.URLPattern, error) {
	patterns, err := s.MatchPatterns(name, v)
	if err != nil {
		return domain.URLPattern{}, err
	}
	if len(patterns) == 0 {
		return domain.URLPattern{}, fmt.Errorf("no matching URL pattern for %s@%s", name, v.Raw)
	}
	return patterns[0], nil
}

func (s *Service) MatchPatterns(name string, v *domain.Version) ([]domain.URLPattern, error) {
	patterns, ok := s.index[name]
	if !ok {
		return nil, fmt.Errorf("no URL pattern found for %s", name)
	}
	var matches []domain.URLPattern
	for _, p := range patterns {
		if p.Constraint(v) {
			matches = append(matches, p)
		}
	}
	return matches, nil
}

func (s *Service) MatchPatternByType(name, sourceType, targetOS, targetArch string, v *domain.Version) (domain.URLPattern, error) {
	patterns, ok := s.index[name]
	if !ok {
		return domain.URLPattern{}, fmt.Errorf("no URL pattern found for %s", name)
	}

	var bestMatch domain.URLPattern
	var exactMatch domain.URLPattern
	var fallbackMatch domain.URLPattern

	for _, p := range patterns {
		if p.Type != sourceType {
			continue
		}
		if !p.Constraint(v) {
			continue
		}

		if p.OS == "" && p.Arch == "" {
			fallbackMatch = p
			continue
		}

		if p.OS == targetOS && p.Arch == targetArch {
			exactMatch = p
			break
		}

		if p.OS == targetOS && p.Arch == "" {
			bestMatch = p
		}
	}

	if exactMatch.Name != "" {
		return exactMatch, nil
	}
	if bestMatch.Name != "" {
		return bestMatch, nil
	}
	if fallbackMatch.Name != "" {
		return fallbackMatch, nil
	}

	return domain.URLPattern{}, fmt.Errorf("no matching URL pattern for %s@%s type=%s os=%s arch=%s", name, v.Raw, sourceType, targetOS, targetArch)
}

func (s *Service) BuildURL(pattern domain.URLPattern, v *domain.Version) (string, error) {
	url := pattern.Template
	url = strings.ReplaceAll(url, "{version}", v.Raw)

	majorMinor := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	url = strings.ReplaceAll(url, "{major}.{minor}", majorMinor)

	major := fmt.Sprintf("%d", v.Major)
	url = strings.ReplaceAll(url, "{major}", major)

	minor := fmt.Sprintf("%d", v.Minor)
	url = strings.ReplaceAll(url, "{minor}", minor)

	if pattern.ExtensionFunc != nil {
		ext := pattern.ExtensionFunc(v)
		url = strings.ReplaceAll(url, "{ext}", ext)
	}

	return url, nil
}

func (s *Service) BuildURLs(pattern domain.URLPattern, v *domain.Version) ([]string, error) {
	var urls []string

	url, err := s.BuildURL(pattern, v)
	if err != nil {
		return nil, err
	}
	urls = append(urls, url)

	for _, fallback := range pattern.Fallbacks {
		fbURL := fallback
		fbURL = strings.ReplaceAll(fbURL, "{version}", v.Raw)
		fbURL = strings.ReplaceAll(fbURL, "{major}.{minor}", fmt.Sprintf("%d.%d", v.Major, v.Minor))
		if pattern.ExtensionFunc != nil {
			ext := pattern.ExtensionFunc(v)
			fbURL = strings.ReplaceAll(fbURL, "{ext}", ext)
		}
		urls = append(urls, fbURL)
	}

	return urls, nil
}

func (s *Service) BuildURLByType(name, version, sourceType string) (string, error) {
	targetOS := viper.GetString("PHPV_TARGET_OS")
	targetArch := viper.GetString("PHPV_TARGET_ARCH")

	if targetOS == "" {
		targetOS = "linux"
	}
	if targetArch == "" {
		targetArch = "x86_64"
	}

	v := utils.ParseVersion(version)
	pattern, err := s.MatchPatternByType(name, sourceType, targetOS, targetArch, v)
	if err != nil {
		return "", err
	}

	return s.BuildURL(pattern, v)
}
