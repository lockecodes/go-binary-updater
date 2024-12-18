package release

// MapArch converts runtime.GOARCH values to desired arch formats (e.g., "amd64" -> "x86_64").
func MapArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	// Add more mappings as needed
	default:
		return arch
	}
}
