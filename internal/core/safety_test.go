package core

import (
	"runtime"
	"strings"
	"testing"

	"github.com/lakshaymaurya-felt/purewin/internal/config"
)

// ---------------------------------------------------------------------------
// ValidatePath tests
// ---------------------------------------------------------------------------

func TestValidatePath_RejectsEmpty(t *testing.T) {
	for _, p := range []string{"", "   ", "\t"} {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) should reject empty/blank path", p)
		}
	}
}

func TestValidatePath_RejectsRelative(t *testing.T) {
	paths := []string{"relative/path", ".", ".."}
	if runtime.GOOS == "windows" {
		paths = append(paths, `foo\bar`)
	}
	for _, p := range paths {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) should reject relative path", p)
		}
	}
}

func TestValidatePath_RejectsDriveRoots(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive root test is Windows-specific")
	}
	for _, p := range []string{`C:\`, `D:\`, `C:`, `c:\`, `E:\`} {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) should reject drive root", p)
		}
	}
}

func TestValidatePath_RejectsTraversal(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("traversal test paths are Windows-specific")
	}
	for _, p := range []string{
		`C:\Users\..\..\..\Windows\System32`,
		`C:\Users\test\..\..\Windows`,
		`C:\foo\bar\..\..\..\baz`,
	} {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) should reject path with traversal (..) component", p)
		}
	}
}

func TestValidatePath_RejectsControlChars(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("control char test paths are Windows-specific")
	}
	for _, p := range []string{
		"C:\\Users\\test\x00file",
		"C:\\dir\x01name\\file",
		"C:\\test\npath",
	} {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) should reject control characters", p)
		}
	}
}

func TestValidatePath_RejectsNeverDeletePaths(t *testing.T) {
	// Every path returned by GetNeverDeletePaths must be rejected.
	for _, p := range config.GetNeverDeletePaths() {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) MUST reject NEVER_DELETE path", p)
		}
	}
}

func TestValidatePath_AcceptsValidPaths(t *testing.T) {
	var paths []string
	if runtime.GOOS == "windows" {
		paths = []string{
			`C:\SomeSafeDir\SubDir\file.tmp`,
			`D:\Projects\build\output.zip`,
			`C:\Workspace\tools\binary.exe`,
		}
	} else {
		paths = []string{
			`/tmp/safe_test_dir/file.tmp`,
			`/home/user/projects/build/output.zip`,
		}
	}
	for _, p := range paths {
		if err := ValidatePath(p); err != nil {
			t.Errorf("ValidatePath(%q) should accept valid path, got: %v", p, err)
		}
	}
}

// ---------------------------------------------------------------------------
// IsSafePath tests
// ---------------------------------------------------------------------------

func TestIsSafePath_ProtectsAllNeverDeletePaths(t *testing.T) {
	// Every path in GetNeverDeletePaths() must return false from IsSafePath.
	for _, p := range config.GetNeverDeletePaths() {
		if IsSafePath(p) {
			t.Errorf("IsSafePath(%q) MUST return false for NEVER_DELETE path", p)
		}
	}
}

func TestIsSafePath_ProtectsSubdirectories(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("subdirectory protection test uses Windows paths")
	}
	for _, p := range []string{
		`C:\Windows\System32\drivers`,
		`C:\Windows\System32\config\SAM`,
		`C:\Windows\WinSxS\Manifests`,
		`C:\Program Files\Common Files`,
		`C:\Program Files (x86)\SomeApp`,
		`C:\Windows\Installer\patches`,
	} {
		if IsSafePath(p) {
			t.Errorf("IsSafePath(%q) must return false — subdirectory of NEVER_DELETE", p)
		}
	}
}

func TestIsSafePath_CaseInsensitive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case-insensitive test is Windows-specific")
	}
	for _, p := range []string{
		`c:\windows`,
		`C:\WINDOWS`,
		`c:\windows\system32`,
		`C:\PROGRAM FILES`,
	} {
		if IsSafePath(p) {
			t.Errorf("IsSafePath(%q) must be case-insensitive and reject this path", p)
		}
	}
}

func TestIsSafePath_AllowsSafePaths(t *testing.T) {
	var paths []string
	if runtime.GOOS == "windows" {
		paths = []string{
			`C:\SomeSafeDir\SubDir`,
			`D:\Projects\build`,
			`C:\Workspace\output`,
		}
	} else {
		paths = []string{
			`/tmp/safe_test`,
			`/home/user/projects`,
			`/opt/workspace`,
		}
	}
	for _, p := range paths {
		if !IsSafePath(p) {
			t.Errorf("IsSafePath(%q) should return true for non-protected path", p)
		}
	}
}

func TestValidatePath_ErrorMessages(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("error message test uses Windows paths")
	}
	tests := []struct {
		path     string
		contains string
	}{
		{"", "empty"},
		{"relative", "absolute"},
		{`C:\`, "drive root"},
		{`C:\Windows`, "protected"},
	}
	for _, tc := range tests {
		err := ValidatePath(tc.path)
		if err == nil {
			t.Errorf("ValidatePath(%q) should return error", tc.path)
			continue
		}
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.contains)) {
			t.Errorf("ValidatePath(%q) error should contain %q, got: %v", tc.path, tc.contains, err)
		}
	}
}
