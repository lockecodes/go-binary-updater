package release

import (
	"strings"
)

// MapArch converts runtime.GOARCH values to common release asset naming conventions.
// It handles both Go architecture names and provides fallback logic for unmapped architectures.
func MapArch(arch string) string {
	// Normalize input to lowercase for consistent matching
	normalizedArch := strings.ToLower(strings.TrimSpace(arch))

	switch normalizedArch {
	// AMD64 / x86_64 variants
	case "amd64", "x86_64", "x64":
		return "x86_64"

	// ARM64 variants
	case "arm64", "aarch64":
		return "arm64"

	// ARM 32-bit variants
	case "arm", "armv6", "armv7", "armhf":
		return "arm"

	// 386 / i386 variants
	case "386", "i386", "i686", "x86":
		return "i386"

	// MIPS variants
	case "mips":
		return "mips"
	case "mipsle":
		return "mipsle"
	case "mips64":
		return "mips64"
	case "mips64le":
		return "mips64le"

	// PowerPC variants
	case "ppc64":
		return "ppc64"
	case "ppc64le":
		return "ppc64le"

	// IBM System z
	case "s390x":
		return "s390x"

	// RISC-V
	case "riscv64":
		return "riscv64"

	// WebAssembly
	case "wasm":
		return "wasm"

	// Fallback: return the original architecture if no mapping found
	// This ensures compatibility with future or uncommon architectures
	default:
		return arch
	}
}

// GetArchVariants returns common variants for a given architecture.
// This can be used for fuzzy matching when exact architecture match fails.
func GetArchVariants(arch string) []string {
	normalizedArch := strings.ToLower(strings.TrimSpace(arch))

	switch normalizedArch {
	case "amd64", "x86_64", "x64":
		return []string{"x86_64", "amd64", "x64"}
	case "arm64", "aarch64":
		return []string{"arm64", "aarch64"}
	case "arm", "armv6", "armv7", "armhf":
		return []string{"arm", "armv6", "armv7", "armhf"}
	case "386", "i386", "i686", "x86":
		return []string{"i386", "386", "i686", "x86"}
	case "mips":
		return []string{"mips"}
	case "mipsle":
		return []string{"mipsle"}
	case "mips64":
		return []string{"mips64"}
	case "mips64le":
		return []string{"mips64le"}
	case "ppc64":
		return []string{"ppc64"}
	case "ppc64le":
		return []string{"ppc64le"}
	case "s390x":
		return []string{"s390x"}
	case "riscv64":
		return []string{"riscv64"}
	case "wasm":
		return []string{"wasm"}
	default:
		return []string{arch}
	}
}
