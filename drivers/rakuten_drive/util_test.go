package rakuten_drive

import "testing"

func TestFilePathAndNameFromNestedAPIPath(t *testing.T) {
	f := File{
		Path:     "2/01 - wind.flac",
		IsFolder: false,
	}
	f.parentPath = "2/"
	f.filePath = resolveFilePath(f.parentPath, f.Path, f.IsFolder)

	if got := f.GetName(); got != "01 - wind.flac" {
		t.Fatalf("GetName() = %q, want %q", got, "01 - wind.flac")
	}
	if got := f.GetPath(); got != "2/01 - wind.flac" {
		t.Fatalf("GetPath() = %q, want %q", got, "2/01 - wind.flac")
	}
	if got := f.apiParentPath(); got != "2/" {
		t.Fatalf("apiParentPath() = %q, want %q", got, "2/")
	}
	if got := f.apiPath(); got != "01 - wind.flac" {
		t.Fatalf("apiPath() = %q, want %q", got, "01 - wind.flac")
	}
}

func TestResolveFilePathFromRelativeAPIPath(t *testing.T) {
	if got := resolveFilePath("2/", "01 - wind.flac", false); got != "2/01 - wind.flac" {
		t.Fatalf("resolveFilePath() = %q, want %q", got, "2/01 - wind.flac")
	}
}
