package release

import (
	"testing"
)

func TestMapArch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// AMD64 variants
		{"AMD Architecture Mapping", "amd64", "x86_64"},
		{"x86_64", "x86_64", "x86_64"},
		{"x64", "x64", "x86_64"},

		// ARM64 variants
		{"ARM Architecture Mapping", "arm64", "arm64"},
		{"aarch64", "aarch64", "arm64"},

		// ARM 32-bit variants
		{"ARM", "arm", "arm"},
		{"ARMv6", "armv6", "arm"},
		{"ARMv7", "armv7", "arm"},
		{"ARMHF", "armhf", "arm"},

		// 386 variants
		{"386", "386", "i386"},
		{"i386", "i386", "i386"},
		{"i686", "i686", "i386"},
		{"x86", "x86", "i386"},

		// MIPS variants
		{"MIPS", "mips", "mips"},
		{"MIPSLE", "mipsle", "mipsle"},
		{"MIPS64", "mips64", "mips64"},
		{"MIPS64LE", "mips64le", "mips64le"},

		// PowerPC variants
		{"PPC64", "ppc64", "ppc64"},
		{"PPC64LE", "ppc64le", "ppc64le"},

		// IBM System z
		{"S390X", "s390x", "s390x"},

		// RISC-V
		{"RISCV64", "riscv64", "riscv64"},

		// WebAssembly
		{"WASM", "wasm", "wasm"},

		// Case insensitive
		{"AMD64 uppercase", "AMD64", "x86_64"},
		{"ARM64 uppercase", "ARM64", "arm64"},
		{"Mixed case", "X86_64", "x86_64"},

		// Whitespace handling
		{"With spaces", " amd64 ", "x86_64"},
		{"With tabs", "\tarm64\t", "arm64"},

		// Unknown architecture (fallback)
		{"Unknown Architecture Mapping", "unknown", "unknown"},
		{"Custom arch", "custom-arch", "custom-arch"},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapArch(tt.input)
			if result != tt.expected {
				t.Errorf("MapArch(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetArchVariants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "AMD64 variants",
			input:    "amd64",
			expected: []string{"x86_64", "amd64", "x64"},
		},
		{
			name:     "ARM64 variants",
			input:    "arm64",
			expected: []string{"arm64", "aarch64"},
		},
		{
			name:     "ARM variants",
			input:    "arm",
			expected: []string{"arm", "armv6", "armv7", "armhf"},
		},
		{
			name:     "386 variants",
			input:    "386",
			expected: []string{"i386", "386", "i686", "x86"},
		},
		{
			name:     "MIPS variants",
			input:    "mips",
			expected: []string{"mips"},
		},
		{
			name:     "Unknown architecture",
			input:    "unknown",
			expected: []string{"unknown"},
		},
		{
			name:     "Case insensitive",
			input:    "AMD64",
			expected: []string{"x86_64", "amd64", "x64"},
		},
		{
			name:     "With whitespace",
			input:    " arm64 ",
			expected: []string{"arm64", "aarch64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetArchVariants(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("GetArchVariants(%q) returned %d variants, want %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i, variant := range result {
				if variant != tt.expected[i] {
					t.Errorf("GetArchVariants(%q)[%d] = %q, want %q", tt.input, i, variant, tt.expected[i])
				}
			}
		})
	}
}
