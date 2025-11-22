package domain

// LLVMVersion represents an LLVM/Clang version configuration
type LLVMVersion struct {
	Version     string
	DownloadURL string
}

// GetLLVMVersionForPHP returns the appropriate LLVM version for a given PHP version
func GetLLVMVersionForPHP(phpVersion Version) LLVMVersion {
	// PHP 8.3+ - Use latest LLVM
	if phpVersion.Major == 8 && phpVersion.Minor >= 3 {
		return LLVMVersion{
			Version:     "21.1.6",
			DownloadURL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-21.1.6/LLVM-21.1.6-Linux-X64.tar.xz",
		}
	}

	// PHP 8.0-8.2 - Use LLVM 18
	if phpVersion.Major == 8 {
		return LLVMVersion{
			Version:     "18.1.8",
			DownloadURL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-18.1.8/clang+llvm-18.1.8-x86_64-linux-gnu-ubuntu-18.04.tar.xz",
		}
	}

	// PHP 7.4 - Use LLVM 15
	if phpVersion.Major == 7 && phpVersion.Minor == 4 {
		return LLVMVersion{
			Version:     "15.0.6",
			DownloadURL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-15.0.6/clang+llvm-15.0.6-x86_64-linux-gnu-ubuntu-18.04.tar.xz",
		}
	}

	// PHP 7.0-7.3 - Use LLVM 12
	if phpVersion.Major == 7 {
		return LLVMVersion{
			Version:     "12.0.1",
			DownloadURL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-12.0.1/clang+llvm-12.0.1-x86_64-linux-gnu-ubuntu-16.04.tar.xz",
		}
	}

	// PHP 5.x - Use LLVM 8 (last version to support older C standards well)
	if phpVersion.Major == 5 {
		return LLVMVersion{
			Version:     "8.0.1",
			DownloadURL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-8.0.1/clang+llvm-8.0.1-x86_64-linux-gnu-ubuntu-14.04.tar.xz",
		}
	}

	// Default fallback - Use LLVM 21
	return LLVMVersion{
		Version:     "21.1.6",
		DownloadURL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-21.1.6/LLVM-21.1.6-Linux-X64.tar.xz",
	}
}
