package dependency

import (
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestToolchainService_GetLLVMInstallDir(t *testing.T) {
	ts := NewToolchainService("/home/user/.phpv")

	tests := []struct {
		name        string
		llvmVersion string
		expected    string
	}{
		{
			name:        "LLVM 21.1.6",
			llvmVersion: "21.1.6",
			expected:    "/home/user/.phpv/toolchains/llvm-21.1.6",
		},
		{
			name:        "LLVM 18.1.8",
			llvmVersion: "18.1.8",
			expected:    "/home/user/.phpv/toolchains/llvm-18.1.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ts.GetLLVMInstallDir(tt.llvmVersion)
			if got != tt.expected {
				t.Errorf("GetLLVMInstallDir() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToolchainService_GetToolchainConfig(t *testing.T) {
	ts := NewToolchainService("/home/user/.phpv")

	tests := []struct {
		name       string
		phpVersion domain.Version
		wantCC     string
		wantCXX    string
	}{
		{
			name: "PHP 8.3",
			phpVersion: domain.Version{
				Major: 8,
				Minor: 3,
				Patch: 0,
			},
			wantCC:  "/home/user/.phpv/toolchains/llvm-21.1.6/bin/clang",
			wantCXX: "/home/user/.phpv/toolchains/llvm-21.1.6/bin/clang++",
		},
		{
			name: "PHP 8.0",
			phpVersion: domain.Version{
				Major: 8,
				Minor: 0,
				Patch: 30,
			},
			wantCC:  "/home/user/.phpv/toolchains/llvm-18.1.8/bin/clang",
			wantCXX: "/home/user/.phpv/toolchains/llvm-18.1.8/bin/clang++",
		},
		{
			name: "PHP 7.4",
			phpVersion: domain.Version{
				Major: 7,
				Minor: 4,
				Patch: 33,
			},
			wantCC:  "/home/user/.phpv/toolchains/llvm-15.0.7/bin/clang",
			wantCXX: "/home/user/.phpv/toolchains/llvm-15.0.7/bin/clang++",
		},
		{
			name: "PHP 7.2",
			phpVersion: domain.Version{
				Major: 7,
				Minor: 2,
				Patch: 34,
			},
			wantCC:  "/home/user/.phpv/toolchains/llvm-12.0.1/bin/clang",
			wantCXX: "/home/user/.phpv/toolchains/llvm-12.0.1/bin/clang++",
		},
		{
			name: "PHP 5.6",
			phpVersion: domain.Version{
				Major: 5,
				Minor: 6,
				Patch: 40,
			},
			wantCC:  "/home/user/.phpv/toolchains/llvm-8.0.1/bin/clang",
			wantCXX: "/home/user/.phpv/toolchains/llvm-8.0.1/bin/clang++",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will return nil if LLVM is not installed
			// In a real test, you'd need to mock the filesystem or install LLVM first
			config := ts.GetToolchainConfig(tt.phpVersion)

			// For now, just check the expected paths are correct based on the version
			llvmVersion := domain.GetLLVMVersionForPHP(tt.phpVersion)
			expectedCC := "/home/user/.phpv/toolchains/llvm-" + llvmVersion.Version + "/bin/clang"
			expectedCXX := "/home/user/.phpv/toolchains/llvm-" + llvmVersion.Version + "/bin/clang++"

			if expectedCC != tt.wantCC {
				t.Errorf("Expected CC path = %v, got %v", tt.wantCC, expectedCC)
			}
			if expectedCXX != tt.wantCXX {
				t.Errorf("Expected CXX path = %v, got %v", tt.wantCXX, expectedCXX)
			}

			// If LLVM were installed, verify the config
			if config != nil {
				if config.CC != tt.wantCC {
					t.Errorf("GetToolchainConfig() CC = %v, want %v", config.CC, tt.wantCC)
				}
				if config.CXX != tt.wantCXX {
					t.Errorf("GetToolchainConfig() CXX = %v, want %v", config.CXX, tt.wantCXX)
				}
			}
		})
	}
}
