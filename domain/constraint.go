package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ConstraintType int

const (
	ConstraintExact ConstraintType = iota
	ConstraintTilde
	ConstraintRange
	ConstraintEmpty
)

type DependencyConstraint struct {
	Type        ConstraintType
	Min         string
	Max         string
	Recommended string
	Raw         string
	Optional    bool
	Exclusions  []string
}

type DependencyVersionSpec struct {
	ConstraintStr string
	Constraint    *DependencyConstraint
	Optional      bool
}

func (s DependencyVersionSpec) IsOptional() bool {
	return s.Optional
}

func (s DependencyVersionSpec) GetRecommended() string {
	if s.Constraint != nil {
		return s.Constraint.Recommended
	}
	return ""
}

func (s DependencyVersionSpec) GetMin() string {
	if s.Constraint != nil {
		return s.Constraint.Min
	}
	return ""
}

func (s DependencyVersionSpec) GetMax() string {
	if s.Constraint != nil {
		return s.Constraint.Max
	}
	return ""
}

func (s DependencyVersionSpec) IsValid() bool {
	if s.Optional && s.ConstraintStr == "" {
		return true
	}
	if s.Constraint == nil {
		return false
	}
	return true
}

func ParseConstraint(input string) (*DependencyConstraint, error) {
	constraint := &DependencyConstraint{
		Raw: input,
	}

	if input == "" {
		constraint.Type = ConstraintEmpty
		constraint.Optional = true
		return constraint, nil
	}

	parts := strings.Split(input, "|")
	var recommended, constraintStr string

	if len(parts) == 1 {
		recommended = parts[0]
		constraintStr = parts[0]
	} else if len(parts) == 2 {
		recommended = parts[0]
		constraintStr = parts[1]
	} else {
		return nil, fmt.Errorf("invalid constraint format: %s", input)
	}

	constraint.Recommended = recommended
	constraintStr = strings.TrimSpace(constraintStr)

	if constraintStr == "" {
		constraint.Type = ConstraintEmpty
		constraint.Optional = true
		return constraint, nil
	}

	constraint.Exclusions = parseExclusions(constraintStr)
	constraintStr = removeExclusions(constraintStr)

	switch {
	case strings.HasPrefix(constraintStr, "~"):
		constraint.Type = ConstraintTilde
		min, max := parseTilde(constraintStr)
		constraint.Min = min
		constraint.Max = max
	case containsOperators(constraintStr):
		constraint.Type = ConstraintRange
		min, max := parseRange(constraintStr)
		constraint.Min = min
		constraint.Max = max
	default:
		constraint.Type = ConstraintExact
		constraint.Min = constraintStr
		constraint.Max = constraintStr
	}

	return constraint, nil
}

func parseExclusions(constraintStr string) []string {
	var exclusions []string
	exclusionRegex := regexp.MustCompile(`!=\s*([0-9a-zA-Z.\-_]+)`)
	matches := exclusionRegex.FindAllStringSubmatch(constraintStr, -1)
	for _, match := range matches {
		if len(match) > 1 {
			exclusions = append(exclusions, match[1])
		}
	}
	return exclusions
}

func removeExclusions(constraintStr string) string {
	exclusionRegex := regexp.MustCompile(`,?\s*!=[^,]+`)
	return exclusionRegex.ReplaceAllString(constraintStr, "")
}

func containsOperators(s string) bool {
	operators := []string{">=", "<=", ">", "<", ","}
	for _, op := range operators {
		if strings.Contains(s, op) {
			return true
		}
	}
	return false
}

func parseTilde(tilde string) (min, max string) {
	version := strings.TrimPrefix(tilde, "~")
	version = strings.TrimSpace(version)

	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		major, _ := strconv.Atoi(parts[0])
		minor, _ := strconv.Atoi(parts[1])
		min = version
		max = fmt.Sprintf("%d.%d.0", major, minor+1)
	} else if len(parts) == 1 {
		major, _ := strconv.Atoi(parts[0])
		min = version
		max = fmt.Sprintf("%d.0.0", major+1)
	}

	return min, max
}

func parseRange(constraintStr string) (min, max string) {
	parts := strings.Split(constraintStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, ">=") {
			min = strings.TrimSpace(strings.TrimPrefix(part, ">="))
		} else if strings.HasPrefix(part, ">") {
			v := strings.TrimSpace(strings.TrimPrefix(part, ">"))
			min = incrementPatch(v)
		} else if strings.HasPrefix(part, "<=") {
			max = strings.TrimSpace(strings.TrimPrefix(part, "<="))
		} else if strings.HasPrefix(part, "<") {
			max = strings.TrimSpace(strings.TrimPrefix(part, "<"))
		}
	}
	return min, max
}

func incrementPatch(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 3 {
		patch, _ := strconv.Atoi(parts[2])
		parts[2] = strconv.Itoa(patch + 1)
		return strings.Join(parts, ".")
	}
	return version
}

func CompareVersions(a, b string) int {
	a = normalizeVersion(a)
	b = normalizeVersion(b)

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		aNum, aSuffix := extractNumericAndSuffix(aParts, i)
		bNum, bSuffix := extractNumericAndSuffix(bParts, i)

		if aNum != bNum {
			if aNum < bNum {
				return -1
			}
			return 1
		}

		if aSuffix != bSuffix {
			return compareSuffixes(aSuffix, bSuffix)
		}
	}

	return 0
}

func extractNumericAndSuffix(parts []string, index int) (int, string) {
	if index >= len(parts) {
		return 0, ""
	}

	part := parts[index]
	numStr := ""
	suffix := ""

	for i, char := range part {
		if char >= '0' && char <= '9' {
			numStr += string(char)
		} else {
			suffix = part[i:]
			break
		}
	}

	num := 0
	if numStr != "" {
		num, _ = strconv.Atoi(numStr)
	}

	return num, suffix
}

func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimSpace(version)

	suffixes := []string{"-alpha", "-beta", "-rc", "-p"}
	for _, suffix := range suffixes {
		if idx := strings.Index(version, suffix); idx != -1 {
			normalized := version[:idx] + "." + strings.ReplaceAll(version[idx+len(suffix):], ".", "")
			return normalized
		}
	}

	version = strings.ReplaceAll(version, "-", ".")
	version = strings.ReplaceAll(version, "_", ".")

	return version
}

func compareSuffixes(a, b string) int {
	order := map[string]int{
		"":      0,
		"alpha": 1,
		"a":     1,
		"beta":  2,
		"b":     2,
		"rc":    3,
	}

	aKey := strings.ToLower(strings.TrimLeft(a, "0123456789"))
	bKey := strings.ToLower(strings.TrimLeft(b, "0123456789"))

	aOrder, aHas := order[aKey]
	bOrder, bHas := order[bKey]

	if !aHas && !bHas {
		return 0
	}
	if !aHas {
		return 1
	}
	if !bHas {
		return -1
	}

	if aOrder != bOrder {
		if aOrder < bOrder {
			return -1
		}
		return 1
	}

	return 0
}

func (c *DependencyConstraint) Matches(version string) bool {
	if c.Type == ConstraintEmpty || c.Optional {
		return true
	}

	for _, exclusion := range c.Exclusions {
		if version == exclusion {
			return false
		}
	}

	if c.Type == ConstraintExact {
		return CompareVersions(version, c.Min) == 0
	}

	if c.Min != "" && CompareVersions(version, c.Min) < 0 {
		return false
	}

	if c.Max != "" && CompareVersions(version, c.Max) >= 0 {
		return false
	}

	return true
}

func (c *DependencyConstraint) String() string {
	if c.Type == ConstraintEmpty {
		return "(empty/optional)"
	}

	if c.Type == ConstraintExact {
		return fmt.Sprintf("exact: %s", c.Min)
	}

	if c.Type == ConstraintTilde {
		return fmt.Sprintf("~%s (>=%s, <%s)", c.Min, c.Min, c.Max)
	}

	if c.Type == ConstraintRange {
		constraints := ""
		if c.Min != "" {
			constraints += ">=" + c.Min
		}
		if c.Max != "" {
			if constraints != "" {
				constraints += ", "
			}
			constraints += "<" + c.Max
		}
		if len(c.Exclusions) > 0 {
			for _, ex := range c.Exclusions {
				constraints += fmt.Sprintf(", !=%s", ex)
			}
		}
		return constraints
	}

	return c.Raw
}

func ValidateDependencyVersion(depName, version string, spec DependencyVersionSpec) (bool, []string, error) {
	var warnings []string

	if spec.Optional && version == "" {
		return true, nil, nil
	}

	if spec.Constraint == nil {
		return false, nil, fmt.Errorf("constraint not parsed for %s", depName)
	}

	constraint := spec.Constraint

	if !constraint.Matches(version) {
		warnings = append(warnings, fmt.Sprintf(
			"%s version %s is outside allowed range (constraint: %s)",
			depName, version, constraint.String(),
		))
		return false, warnings, nil
	}

	return true, warnings, nil
}

type ValidationWarning struct {
	Dependency  string
	Current     string
	Constraint  string
	Recommended string
	Message     string
}

func ValidateAllDependencies(specs map[string]DependencyVersionSpec, currentVersions map[string]string) []ValidationWarning {
	var warnings []ValidationWarning

	for name, spec := range specs {
		if spec.Optional && (spec.ConstraintStr == "" || currentVersions[name] == "") {
			continue
		}

		if spec.Constraint == nil {
			continue
		}

		currentVersion := currentVersions[name]
		if currentVersion == "" {
			currentVersion = "(not installed)"
		}

		if !spec.Constraint.Matches(currentVersion) {
			warning := ValidationWarning{
				Dependency:  name,
				Current:     currentVersion,
				Constraint:  spec.Constraint.String(),
				Recommended: spec.Constraint.Recommended,
				Message: fmt.Sprintf(
					"%s version %s does not match constraint (%s). Recommended: %s",
					name, currentVersion, spec.Constraint.String(), spec.Constraint.Recommended,
				),
			}
			warnings = append(warnings, warning)
		}
	}

	return warnings
}
