package domain

import "testing"

func TestGetLLVMVersionForPHP(t *testing.T) {
	tests := []struct {
		name       string
		phpVersion Version
		wantLLVM   string
		wantURL    string
	}{
		{
			name: "PHP 8.4 uses LLVM 21",
			phpVersion: Version{
				Major: 8,
				Minor: 4,
				Patch: 14,
			},
			wantLLVM: "21.1.6",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-21.1.6/LLVM-21.1.6-Linux-X64.tar.xz",
		},
		{
			name: "PHP 8.3 uses LLVM 21",
			phpVersion: Version{
				Major: 8,
				Minor: 3,
				Patch: 0,
			},
			wantLLVM: "21.1.6",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-21.1.6/LLVM-21.1.6-Linux-X64.tar.xz",
		},
		{
			name: "PHP 8.2 uses LLVM 18",
			phpVersion: Version{
				Major: 8,
				Minor: 2,
				Patch: 25,
			},
			wantLLVM: "18.1.8",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-18.1.8/LLVM-18.1.8-Linux-X64.tar.xz",
		},
		{
			name: "PHP 8.0 uses LLVM 18",
			phpVersion: Version{
				Major: 8,
				Minor: 0,
				Patch: 30,
			},
			wantLLVM: "18.1.8",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-18.1.8/LLVM-18.1.8-Linux-X64.tar.xz",
		},
		{
			name: "PHP 7.4 uses LLVM 15",
			phpVersion: Version{
				Major: 7,
				Minor: 4,
				Patch: 33,
			},
			wantLLVM: "15.0.7",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-15.0.7/LLVM-15.0.7-Linux-X64.tar.xz",
		},
		{
			name: "PHP 7.3 uses LLVM 12",
			phpVersion: Version{
				Major: 7,
				Minor: 3,
				Patch: 33,
			},
			wantLLVM: "12.0.1",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-12.0.1/LLVM-12.0.1-Linux-X64.tar.xz",
		},
		{
			name: "PHP 7.0 uses LLVM 12",
			phpVersion: Version{
				Major: 7,
				Minor: 0,
				Patch: 33,
			},
			wantLLVM: "12.0.1",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-12.0.1/LLVM-12.0.1-Linux-X64.tar.xz",
		},
		{
			name: "PHP 5.6 uses LLVM 8",
			phpVersion: Version{
				Major: 5,
				Minor: 6,
				Patch: 40,
			},
			wantLLVM: "8.0.1",
			wantURL:  "https://github.com/llvm/llvm-project/releases/download/llvmorg-8.0.1/LLVM-8.0.1-Linux-X64.tar.xz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLLVMVersionForPHP(tt.phpVersion)
			if got.Version != tt.wantLLVM {
				t.Errorf("GetLLVMVersionForPHP() Version = %v, want %v", got.Version, tt.wantLLVM)
			}
			if got.DownloadURL != tt.wantURL {
				t.Errorf("GetLLVMVersionForPHP() DownloadURL = %v, want %v", got.DownloadURL, tt.wantURL)
			}
		})
	}
}
