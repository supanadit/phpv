package memory

import "github.com/supanadit/phpv/domain"

type ConfigureFlagRule = domain.ConfigureFlagRule

var ConfigureFlagRules = map[string][]ConfigureFlagRule{
	"openssl": {
		{MaxVer: "1.0.2", Flags: []string{"shared", "no-ssl3"}},
		{MinVer: "1.1.0", Flags: []string{"shared", "no-ssl3", "no-tests"}},
	},
	"m4": {
		{Flags: []string{"--disable-maintainer-mode"}},
	},
	"libxml2": {
		{Flags: []string{"--disable-shared", "--enable-static", "--without-lzma", "--without-python", "--disable-dependency-tracking", "--with-zlib"}},
	},
	"curl": {
		{Flags: []string{"--with-ssl", "--without-brotli", "--disable-ldap", "--without-libpsl", "--without-libidn2", "--without-zstd", "--without-nghttp2", "--without-zlib"}},
	},
	"icu": {
		{Flags: []string{"--disable-extras", "--disable-samples"}},
	},
	"libzip": {
		{Flags: []string{
			// libzip 1.7.x declares cmake_minimum_required(VERSION 3.1),
			// which CMake 4.x refuses without this policy override.
			"-DCMAKE_POLICY_VERSION_MINIMUM=3.5",
			"-DBUILD_SHARED_LIBS=ON",
			"-DBUILD_TOOLS=OFF",
			"-DBUILD_EXAMPLES=OFF",
			"-DBUILD_DOC=OFF",
			"-DENABLE_BZIP2=OFF",
			"-DENABLE_LZMA=OFF",
			"-DENABLE_ZSTD=OFF",
		}},
	},
	"nginx": {
		{Flags: []string{
			// nginx compiles PCRE2 and zlib from their source trees (not
			// install prefixes) — both placeholders resolve to the actual
			// extracted source dir containing ./configure.
			"--with-pcre={{source:pcre2}}",
			"--with-zlib={{source:zlib}}",
			"--with-http_gzip_static_module",
			"--with-http_stub_status_module",
		}},
	},
	"httpd": {
		{Flags: []string{
			// apr/apr-util are vendored into srclib/ by the assembler
			// (see AssembleServer) before configure runs. httpd needs
			// installed pcre2 (pcre2-config) and expat (vendored apr-util's
			// xml support links it).
			"--with-included-apr",
			"--with-pcre={{dep:pcre2}}",
			"--with-expat={{dep:expat}}",
			"--enable-so",
			"--enable-mpms-shared=event prefork",
			"--with-mpm=prefork",
			"--enable-proxy",
			"--enable-proxy-fcgi",
			"--disable-userdir",
		}},
	},
}

// DefaultExtensionCandidates is the recommended default extension set for a
// typical PHP install. Extensions are filtered by MinPHPVersion/MaxPHPVersion
// at runtime via DefaultExtensions().
var DefaultExtensionCandidates = []string{
	"bcmath", "curl", "dom", "fileinfo", "filter", "gd",
	"iconv", "intl", "json", "mbstring", "openssl", "opcache",
	"pdo", "pdo_mysql", "pdo_sqlite", "phar", "session",
	"simplexml", "sqlite3", "tokenizer", "xml", "xmlreader",
	"xmlwriter", "zip", "zlib",
}

func (f *FlagCatalog) ConfigureFlagRules(name string) []ConfigureFlagRule {
	return ConfigureFlagRules[name]
}
func (f *FlagCatalog) DefaultExtensionCandidates() []string {
	return append([]string(nil), DefaultExtensionCandidates...)
}
