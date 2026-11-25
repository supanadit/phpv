package disk

import "testing"

// Regression: compatibility patches must be selected for exactly the
// versions whose generated sources they fit. An over-broad range makes
// `patch` fail the whole build (PHP 5.0/5.1 once received the 5.6 curl
// configure hunk); an under-broad one leaves old toolchains broken.
func TestLegacyPatchRanges(t *testing.T) {
	repo := NewPatchRepository(repoPatchesDir(t), "")

	cases := []struct {
		pkg, version, want string
	}{
		{"php", "4.0.6", "0001-drop-traditional-cpp"},
		{"php", "4.1.2", "0001-drop-traditional-cpp"},
		{"php", "4.1.2", "0001-legacy-libtool-objext"},
		{"php", "4.2.3", "0001-legacy-libtool-objext"},
		{"php", "4.3.11", "0001-legacy-arm64-compat"},
		{"php", "4.4.9", "0001-legacy-arm64-compat"},
		{"php", "4.4.9", "0001-legacy-arm64-compat"},
		{"php", "5.2.17", "0001-configure-curl-cflags"},
		{"php", "5.6.40", "0001-configure-curl-cflags"},
		{"openssl", "1.0.1u", "0001-add-darwin64-arm64-target"},
		{"icu", "58.2", "0001-gcc15-compat"},
	}
	for _, tc := range cases {
		found := false
		for _, p := range repo.PatchesFor(tc.pkg, tc.version) {
			if p.Name == tc.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s@%s did not select %s", tc.pkg, tc.version, tc.want)
		}
	}

	for _, tc := range []struct{ pkg, version, unwanted string }{
		{"php", "5.0.5", "0001-configure-curl-cflags"},
		{"php", "5.1.6", "0001-configure-curl-cflags"},
	} {
		for _, p := range repo.PatchesFor(tc.pkg, tc.version) {
			if p.Name == tc.unwanted {
				t.Errorf("%s@%s unexpectedly selected %s", tc.pkg, tc.version, tc.unwanted)
			}
		}
	}
}
