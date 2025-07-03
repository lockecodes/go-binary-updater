package release

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

// AssetMatchingStrategy defines how to match release assets
type AssetMatchingStrategy int

const (
	// StandardStrategy uses the traditional {OS}_{ARCH} pattern
	StandardStrategy AssetMatchingStrategy = iota
	// FlexibleStrategy uses multiple patterns and fuzzy matching
	FlexibleStrategy
	// CustomStrategy uses user-defined patterns
	CustomStrategy
	// CDNStrategy downloads from external CDN instead of GitHub/GitLab releases
	CDNStrategy
	// HybridStrategy tries GitHub/GitLab first, then falls back to CDN
	HybridStrategy
)

// AssetMatchingConfig configures how assets are matched and handled
type AssetMatchingConfig struct {
	Strategy           AssetMatchingStrategy `json:"strategy"`
	CustomPatterns     []string              `json:"custom_patterns"`     // Custom regex patterns for asset matching
	IsDirectBinary     bool                  `json:"is_direct_binary"`    // True if asset is a direct binary, not an archive
	ProjectName        string                `json:"project_name"`        // Project name for pattern matching
	ArchitectureAliases map[string][]string  `json:"architecture_aliases"` // Custom architecture aliases
	OSAliases          map[string][]string   `json:"os_aliases"`          // Custom OS aliases
	FileExtensions     []string              `json:"file_extensions"`     // Expected file extensions

	// Enhanced filtering and CDN support
	ExcludePatterns     []string                 `json:"exclude_patterns"`     // Patterns to explicitly exclude (airgap, signatures)
	PriorityPatterns    []string                 `json:"priority_patterns"`    // Patterns that get higher priority scores
	CDNBaseURL          string                   `json:"cdn_base_url"`         // Base URL for CDN downloads (e.g., get.helm.sh)
	CDNPattern          string                   `json:"cdn_pattern"`          // URL pattern for CDN downloads with {version}, {os}, {arch} placeholders
	CDNVersionFormat    string                   `json:"cdn_version_format"`   // Version format for CDN: "as-is", "with-v", "without-v"
	CDNArchMapping      map[string]string        `json:"cdn_arch_mapping"`     // Custom architecture mapping for this CDN
	ExtractionConfig    *ExtractionConfig        `json:"extraction_config"`    // Configuration for complex archive extraction
}

// ExtractionConfig configures how binaries are extracted from archives
type ExtractionConfig struct {
	StripComponents int    `json:"strip_components"` // Number of directory components to strip (like tar --strip-components)
	BinaryPath      string `json:"binary_path"`      // Specific path to binary within archive (e.g., "linux-amd64/helm")
}

// DefaultAssetMatchingConfig returns a sensible default configuration
func DefaultAssetMatchingConfig() AssetMatchingConfig {
	return AssetMatchingConfig{
		Strategy:       FlexibleStrategy,
		IsDirectBinary: false,
		FileExtensions: []string{".tar.gz", ".zip", ".tgz", ".tar.bz2"},
		// Default exclusion patterns for common unwanted assets
		ExcludePatterns: []string{
			"airgap",     // Exclude airgap bundles (k0s)
			"\\.asc$",    // Exclude signature files
			"\\.sig$",    // Exclude signature files
			"\\.sha256$", // Exclude checksum files
			"\\.sha512$", // Exclude checksum files
			"\\.md5$",    // Exclude checksum files
		},
		ArchitectureAliases: map[string][]string{
			"amd64":   {"amd64", "x86_64", "x64"},
			"arm64":   {"arm64", "aarch64"},
			"arm":     {"arm", "armv6", "armv7", "armhf"},
			"386":     {"386", "i386", "i686", "x86"},
			"mips":    {"mips"},
			"mips64":  {"mips64"},
			"ppc64":   {"ppc64"},
			"ppc64le": {"ppc64le"},
			"s390x":   {"s390x"},
			"riscv64": {"riscv64"},
		},
		OSAliases: map[string][]string{
			"linux":   {"linux", "Linux"},
			"darwin":  {"darwin", "Darwin", "macos", "macOS", "osx", "OSX"},
			"windows": {"windows", "Windows", "win", "Win"},
			"freebsd": {"freebsd", "FreeBSD"},
			"openbsd": {"openbsd", "OpenBSD"},
			"netbsd":  {"netbsd", "NetBSD"},
		},
	}
}

// AssetMatcher provides flexible asset matching capabilities
type AssetMatcher struct {
	config AssetMatchingConfig
	os     string
	arch   string
}

// NewAssetMatcher creates a new asset matcher with the given configuration
func NewAssetMatcher(config AssetMatchingConfig) *AssetMatcher {
	return &AssetMatcher{
		config: config,
		os:     runtime.GOOS,
		arch:   runtime.GOARCH,
	}
}

// FindBestMatch finds the best matching asset from a list of asset names
func (am *AssetMatcher) FindBestMatch(assetNames []string) (string, error) {
	if len(assetNames) == 0 {
		return "", fmt.Errorf("no assets provided")
	}

	// Filter out excluded assets first
	filteredAssets := am.filterExcludedAssets(assetNames)
	if len(filteredAssets) == 0 {
		return "", fmt.Errorf("no assets remaining after applying exclusion filters. Original assets: %v, Excluded patterns: %v",
			assetNames, am.config.ExcludePatterns)
	}

	switch am.config.Strategy {
	case StandardStrategy:
		return am.findStandardMatch(filteredAssets)
	case FlexibleStrategy:
		return am.findFlexibleMatch(filteredAssets)
	case CustomStrategy:
		return am.findCustomMatch(filteredAssets)
	case CDNStrategy:
		return am.findCDNMatch()
	case HybridStrategy:
		return am.findHybridMatch(filteredAssets)
	default:
		return am.findFlexibleMatch(filteredAssets)
	}
}

// findStandardMatch uses the traditional {OS}_{ARCH} pattern
func (am *AssetMatcher) findStandardMatch(assetNames []string) (string, error) {
	mappedArch := MapArch(am.arch)
	osTitle := strings.Title(strings.ToLower(am.os))
	searchKey := fmt.Sprintf("%s_%s", osTitle, mappedArch)

	for _, name := range assetNames {
		if strings.Contains(name, searchKey) {
			return name, nil
		}
	}

	return "", fmt.Errorf("no asset found matching pattern %s", searchKey)
}

// findFlexibleMatch uses multiple patterns and fuzzy matching
func (am *AssetMatcher) findFlexibleMatch(assetNames []string) (string, error) {
	// Get all possible aliases for current platform
	osAliases := am.getOSAliases(am.os)
	archAliases := am.getArchAliases(am.arch)

	// Score each asset and find the best match
	bestScore := 0
	bestMatch := ""

	for _, assetName := range assetNames {
		score := am.scoreAsset(assetName, osAliases, archAliases)
		if score > bestScore {
			bestScore = score
			bestMatch = assetName
		}
	}

	if bestScore == 0 {
		return "", fmt.Errorf("no suitable asset found for platform %s/%s", am.os, am.arch)
	}

	return bestMatch, nil
}

// findCustomMatch uses user-defined regex patterns
func (am *AssetMatcher) findCustomMatch(assetNames []string) (string, error) {
	if len(am.config.CustomPatterns) == 0 {
		return "", fmt.Errorf("no custom patterns defined")
	}

	osAliases := am.getOSAliases(am.os)
	archAliases := am.getArchAliases(am.arch)

	for _, pattern := range am.config.CustomPatterns {
		// Replace placeholders in pattern
		expandedPattern := am.expandPattern(pattern, osAliases, archAliases)
		
		regex, err := regexp.Compile(expandedPattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, assetName := range assetNames {
			if regex.MatchString(assetName) {
				return assetName, nil
			}
		}
	}

	return "", fmt.Errorf("no asset matched custom patterns")
}

// scoreAsset scores an asset name based on how well it matches the current platform
func (am *AssetMatcher) scoreAsset(assetName string, osAliases, archAliases []string) int {
	score := 0
	lowerName := strings.ToLower(assetName)

	// Check for OS matches
	osMatched := false
	for _, osAlias := range osAliases {
		if strings.Contains(lowerName, strings.ToLower(osAlias)) {
			score += 10
			osMatched = true
			break
		}
	}

	// Check for architecture matches
	archMatched := false
	for _, archAlias := range archAliases {
		if strings.Contains(lowerName, strings.ToLower(archAlias)) {
			score += 10
			archMatched = true
			break
		}
	}

	// Bonus points for having both OS and arch
	if osMatched && archMatched {
		score += 5
	}

	// For projects like k0s that don't include OS in asset names,
	// give bonus points if arch matches and no wrong OS is detected
	if !osMatched && archMatched && !am.containsWrongOS(lowerName, osAliases) {
		score += 8 // High score for arch-only matches when no wrong OS detected
	}

	// Check for common patterns
	if am.matchesCommonPatterns(lowerName, osAliases, archAliases) {
		score += 3
	}

	// Bonus for priority patterns
	for _, priorityPattern := range am.config.PriorityPatterns {
		if matched, _ := regexp.MatchString(strings.ToLower(priorityPattern), lowerName); matched {
			score += 15 // High bonus for priority patterns
			break
		}
	}

	// Penalty for wrong OS/arch
	if am.containsWrongPlatform(lowerName, osAliases, archAliases) {
		score -= 20
	}

	// Bonus for expected file extensions (if not direct binary)
	if !am.config.IsDirectBinary {
		for _, ext := range am.config.FileExtensions {
			if strings.HasSuffix(lowerName, ext) {
				score += 2
				break
			}
		}
	}

	return score
}

// matchesCommonPatterns checks for common naming patterns
func (am *AssetMatcher) matchesCommonPatterns(assetName string, osAliases, archAliases []string) bool {
	// Pattern: {project}-{version}-{arch} (like k0s)
	if am.config.ProjectName != "" {
		for _, archAlias := range archAliases {
			projectPattern := fmt.Sprintf("%s-.*-%s", strings.ToLower(am.config.ProjectName), strings.ToLower(archAlias))
			if matched, _ := regexp.MatchString(projectPattern, assetName); matched {
				return true
			}
		}
	}

	// Pattern: {os}-{arch} or {arch}-{os}
	for _, osAlias := range osAliases {
		for _, archAlias := range archAliases {
			pattern1 := fmt.Sprintf("%s.*%s", strings.ToLower(osAlias), strings.ToLower(archAlias))
			pattern2 := fmt.Sprintf("%s.*%s", strings.ToLower(archAlias), strings.ToLower(osAlias))

			if matched, _ := regexp.MatchString(pattern1, assetName); matched {
				return true
			}
			if matched, _ := regexp.MatchString(pattern2, assetName); matched {
				return true
			}
		}
	}

	return false
}

// containsWrongPlatform checks if the asset contains indicators for wrong platforms
func (am *AssetMatcher) containsWrongPlatform(assetName string, osAliases, archAliases []string) bool {
	// Check for wrong OS
	allOSAliases := []string{"linux", "darwin", "windows", "freebsd", "openbsd", "netbsd", "macos", "osx", "win"}
	for _, wrongOS := range allOSAliases {
		if strings.Contains(assetName, wrongOS) {
			// Check if this is actually our OS
			isOurOS := false
			for _, ourOS := range osAliases {
				if strings.EqualFold(wrongOS, ourOS) {
					isOurOS = true
					break
				}
			}
			if !isOurOS {
				return true
			}
		}
	}

	// Check for wrong architecture
	allArchAliases := []string{"amd64", "x86_64", "arm64", "aarch64", "arm", "386", "i386", "mips", "ppc64"}
	for _, wrongArch := range allArchAliases {
		if strings.Contains(assetName, wrongArch) {
			// Check if this is actually our arch
			isOurArch := false
			for _, ourArch := range archAliases {
				if strings.EqualFold(wrongArch, ourArch) {
					isOurArch = true
					break
				}
			}
			if !isOurArch {
				return true
			}
		}
	}

	return false
}

// containsWrongOS checks if the asset contains indicators for wrong OS
func (am *AssetMatcher) containsWrongOS(assetName string, osAliases []string) bool {
	// Check for wrong OS
	allOSAliases := []string{"linux", "darwin", "windows", "freebsd", "openbsd", "netbsd", "macos", "osx", "win"}
	for _, wrongOS := range allOSAliases {
		if strings.Contains(assetName, wrongOS) {
			// Check if this is actually our OS
			isOurOS := false
			for _, ourOS := range osAliases {
				if strings.EqualFold(wrongOS, ourOS) {
					isOurOS = true
					break
				}
			}
			if !isOurOS {
				return true
			}
		}
	}
	return false
}

// getOSAliases returns all aliases for the given OS
func (am *AssetMatcher) getOSAliases(os string) []string {
	if aliases, exists := am.config.OSAliases[os]; exists {
		return aliases
	}
	return []string{os}
}

// getArchAliases returns all aliases for the given architecture
func (am *AssetMatcher) getArchAliases(arch string) []string {
	mappedArch := MapArch(arch)
	if aliases, exists := am.config.ArchitectureAliases[mappedArch]; exists {
		return aliases
	}
	return []string{arch, mappedArch}
}

// expandPattern expands pattern placeholders with actual values
func (am *AssetMatcher) expandPattern(pattern string, osAliases, archAliases []string) string {
	// Replace {OS} with OS alternatives
	osPattern := strings.Join(osAliases, "|")
	pattern = strings.ReplaceAll(pattern, "{OS}", fmt.Sprintf("(%s)", osPattern))

	// Replace {ARCH} with architecture alternatives
	archPattern := strings.Join(archAliases, "|")
	pattern = strings.ReplaceAll(pattern, "{ARCH}", fmt.Sprintf("(%s)", archPattern))

	// Replace {PROJECT} with project name
	if am.config.ProjectName != "" {
		pattern = strings.ReplaceAll(pattern, "{PROJECT}", am.config.ProjectName)
	}

	return pattern
}

// filterExcludedAssets removes assets that match exclusion patterns
func (am *AssetMatcher) filterExcludedAssets(assetNames []string) []string {
	if len(am.config.ExcludePatterns) == 0 {
		return assetNames
	}

	var filtered []string
	for _, assetName := range assetNames {
		excluded := false
		lowerName := strings.ToLower(assetName)

		for _, excludePattern := range am.config.ExcludePatterns {
			if matched, _ := regexp.MatchString(strings.ToLower(excludePattern), lowerName); matched {
				excluded = true
				break
			}
		}

		if !excluded {
			filtered = append(filtered, assetName)
		}
	}

	return filtered
}

// findCDNMatch constructs a CDN download URL instead of matching assets
func (am *AssetMatcher) findCDNMatch() (string, error) {
	if am.config.CDNBaseURL == "" || am.config.CDNPattern == "" {
		return "", fmt.Errorf("CDN strategy requires CDNBaseURL and CDNPattern to be configured")
	}

	osAliases := am.getOSAliases(am.os)
	archAliases := am.getArchAliases(am.arch)

	// Use the first (primary) alias for URL construction
	osName := osAliases[0]
	archName := archAliases[0]

	// Construct CDN URL with placeholders
	cdnURL := am.config.CDNBaseURL + am.config.CDNPattern
	cdnURL = strings.ReplaceAll(cdnURL, "{os}", osName)
	cdnURL = strings.ReplaceAll(cdnURL, "{arch}", archName)

	// Note: {version} will be replaced by the calling code that has version information
	return cdnURL, nil
}

// findHybridMatch tries flexible matching first, then falls back to CDN
func (am *AssetMatcher) findHybridMatch(assetNames []string) (string, error) {
	// Try flexible matching first
	if len(assetNames) > 0 {
		result, err := am.findFlexibleMatch(assetNames)
		if err == nil {
			return result, nil
		}
	}

	// Fall back to CDN if flexible matching fails
	return am.findCDNMatch()
}
