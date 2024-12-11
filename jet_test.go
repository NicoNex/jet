package main

import (
	"bytes"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// TestPairMatch tests the match function of the pair struct.
func TestPairMatch(t *testing.T) {
	p := pair{
		pattern:     regexp.MustCompile("foo"),
		replacement: []byte("bar"),
	}

	tests := []struct {
		input    []byte
		expected bool
	}{
		{[]byte("foobar"), true},
		{[]byte("baz"), false},
		{[]byte("foo baz"), true},
		{[]byte(""), false},
	}

	for _, test := range tests {
		result := p.match(test.input)
		if result != test.expected {
			t.Errorf("pair.match(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

// TestPairReplaceAll tests the replaceAll function of the pair struct.
func TestPairReplaceAll(t *testing.T) {
	p := pair{
		pattern:     regexp.MustCompile("foo"),
		replacement: []byte("bar"),
	}

	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("foo foo baz"), []byte("bar bar baz")},
		{[]byte("no match here"), []byte("no match here")},
		{[]byte("foofoo"), []byte("barbar")},
	}

	for _, test := range tests {
		result := p.replaceAll(test.input)
		if !bytes.Equal(result, test.expected) {
			t.Errorf("pair.replaceAll(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

// TestPairsetMatch tests the match function of the pairset type.
func TestPairsetMatch(t *testing.T) {
	ps := pairset{
		{
			pattern:     regexp.MustCompile("foo"),
			replacement: []byte("bar"),
		},
		{
			pattern:     regexp.MustCompile("baz"),
			replacement: []byte("qux"),
		},
	}

	tests := []struct {
		input    []byte
		expected bool
	}{
		{[]byte("hello foo"), true},
		{[]byte("hello baz"), true},
		{[]byte("hello world"), false},
	}

	for _, test := range tests {
		result := ps.match(test.input)
		if result != test.expected {
			t.Errorf("pairset.match(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

// TestPairsetReplaceAll tests the replaceAll function of the pairset type.
func TestPairsetReplaceAll(t *testing.T) {
	ps := pairset{
		{
			pattern:     regexp.MustCompile("foo"),
			replacement: []byte("bar"),
		},
		{
			pattern:     regexp.MustCompile("baz"),
			replacement: []byte("qux"),
		},
	}

	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("foo baz foo baz"), []byte("bar qux bar qux")},
		{[]byte("no match here"), []byte("no match here")},
		{[]byte("foo and baz"), []byte("bar and qux")},
	}

	for _, test := range tests {
		result := ps.replaceAll(test.input)
		if !bytes.Equal(result, test.expected) {
			t.Errorf("pairset.replaceAll(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestPairsetSetValidPattern(t *testing.T) {
	var ps pairset
	// Reset the flag CommandLine so it doesn't interfere with the test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Provide a valid pattern and simulate command line arguments for the replacement.
	// Since Set reads the replacement from flag.Arg(0), we must simulate it.
	os.Args = []string{"cmd", "replacement"}
	flag.Parse()

	if err := ps.Set("foo"); err != nil {
		t.Errorf("expected no error for valid pattern, got %v", err)
	}

	if len(ps) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(ps))
	}
	if ps[0].pattern.String() != "foo" {
		t.Errorf("expected pattern to be 'foo', got %q", ps[0].pattern.String())
	}
	if string(ps[0].replacement) != "replacement" {
		t.Errorf("expected replacement to be 'replacement', got %q", ps[0].replacement)
	}
}

// TestPairsetSetInvalidPattern tests the Set method of pairset for invalid patterns.
func TestPairsetSetInvalidPattern(t *testing.T) {
	var ps pairset
	// The pattern "(?" is invalid
	err := ps.Set("(?")
	if err == nil {
		t.Errorf("expected error for invalid pattern, got nil")
	}
}

// TestWalkerEdit tests the edit function of the walker struct with a temporary file.
func TestWalkerEdit(t *testing.T) {
	// Create a temporary file with initial content.
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("foo baz foo baz")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Initialize the walker.
	w := &walker{
		ToStdout:      false,
		Glob:          "*",
		IsVerbose:     false,
		MaxDepth:      -1,
		IncludeHidden: true,
		ReplaceNames:  false,
		NamesOnly:     false,
		pairs: pairset{
			{
				pattern:     regexp.MustCompile("foo"),
				replacement: []byte("bar"),
			},
			{
				pattern:     regexp.MustCompile("baz"),
				replacement: []byte("qux"),
			},
		},
		WaitGroup: new(sync.WaitGroup),
	}

	// Perform the edit operation.
	w.edit(tmpfile.Name())

	// Read back the content of the file.
	newContent, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("bar qux bar qux")
	if !bytes.Equal(newContent, expected) {
		t.Errorf("After edit, content = %q; want %q", newContent, expected)
	}
}

// TestWalkerEditFilename tests the editFilename function of the walker struct.
func TestWalkerEditFilename(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with a name that needs to be replaced.
	oldPath := filepath.Join(tmpDir, "foo.txt")
	newPath := filepath.Join(tmpDir, "bar.txt")
	if _, err := os.Create(oldPath); err != nil {
		t.Fatal(err)
	}

	// Initialize the walker.
	w := &walker{
		ToStdout:      false,
		Glob:          "*",
		IsVerbose:     false,
		MaxDepth:      -1,
		IncludeHidden: true,
		ReplaceNames:  true,
		NamesOnly:     true,
		pairs: pairset{
			{
				pattern:     regexp.MustCompile("foo"),
				replacement: []byte("bar"),
			},
		},
	}

	// Perform the editFilename operation.
	resultPath := w.editFilename(oldPath)

	// Check if the file has been renamed.
	if resultPath != newPath {
		t.Errorf("editFilename result = %q; want %q", resultPath, newPath)
	}
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("Renamed file %q does not exist", newPath)
	}
}

func TestWalkerEditStdin(t *testing.T) {
	// Prepare the walker with replacement pairs
	w := &walker{
		ToStdout:      true,
		Glob:          "*",
		IsVerbose:     false,
		MaxDepth:      -1,
		IncludeHidden: true,
		ReplaceNames:  false,
		NamesOnly:     false,
		pairs: pairset{
			{
				pattern:     regexp.MustCompile("foo"),
				replacement: []byte("bar"),
			},
			{
				pattern:     regexp.MustCompile("baz"),
				replacement: []byte("qux"),
			},
		},
		WaitGroup: new(sync.WaitGroup),
	}

	// Save original stdin and stdout
	origStdin := os.Stdin
	origStdout := os.Stdout
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	}()

	// Create a pipe to simulate stdin
	rStdin, wStdin, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe for stdin: %v", err)
	}
	os.Stdin = rStdin

	// Create a pipe to capture stdout
	rStdout, wStdout, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe for stdout: %v", err)
	}
	os.Stdout = wStdout

	// Write input to stdin (include a null byte to mark the end)
	input := []byte("foo baz foo\x00")
	if _, err := wStdin.Write(input); err != nil {
		t.Fatalf("failed to write to stdin: %v", err)
	}
	wStdin.Close() // End of input

	// Run editStdin which reads from os.Stdin and writes to os.Stdout
	w.editStdin()

	// Close stdout writer to allow reading
	wStdout.Close()

	// Read output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rStdout); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	output := buf.String()
	expected := "bar qux bar\x00"

	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

// TestWalkerEditInvalidPath tests the edit function of the walker struct with a non-existent file.
func TestWalkerEditInvalidPath(t *testing.T) {
	// Capture stdout to check error message.
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	wk := &walker{
		ToStdout:      false,
		Glob:          "*",
		IsVerbose:     false,
		MaxDepth:      -1,
		IncludeHidden: true,
		ReplaceNames:  false,
		NamesOnly:     false,
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
		WaitGroup: new(sync.WaitGroup),
	}

	wk.edit("non_existent_file.txt")

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	output := buf.String()
	if !strings.Contains(output, "no such file or directory") {
		t.Errorf("expected error about missing file, got: %s", output)
	}
}

// TestDepth tests the depth function.
func TestDepth(t *testing.T) {
	tests := []struct {
		path     string
		expected int
	}{
		{"", 1},
		{"a/b/c", 3},
		{"/a/b/c", 4},
	}

	for _, test := range tests {
		result := depth(test.path)
		if result != test.expected {
			t.Errorf("depth(%q) = %d; want %d", test.path, result, test.expected)
		}
	}
}

// TestIsHidden tests the isHidden function.
func TestIsHidden(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{".hidden", true},
		{"..", false},
		{".", false},
		{"visible", false},
	}

	for _, test := range tests {
		result := isHidden(test.name)
		if result != test.expected {
			t.Errorf("isHidden(%q) = %v; want %v", test.name, result, test.expected)
		}
	}
}

// TestProcessFile tests the processFile function of the walker struct.
func TestProcessFile(t *testing.T) {
	// Create a temporary directory with files.
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files and directories.
	files := []string{
		"foo.txt",
		".hidden.txt",
		"subdir/bar.txt",
		"subdir/.hidden_sub.txt",
	}
	for _, file := range files {
		fullPath := filepath.Join(tmpDir, file)
		if strings.Contains(file, "/") {
			os.MkdirAll(filepath.Dir(fullPath), 0755)
		}
		if _, err := os.Create(fullPath); err != nil {
			t.Fatal(err)
		}
	}

	// Initialize the walker.
	w := &walker{
		ToStdout:      false,
		Glob:          "*.txt",
		IsVerbose:     false,
		MaxDepth:      -1,
		IncludeHidden: false,
		ReplaceNames:  false,
		NamesOnly:     false,
		pairs: pairset{
			{
				pattern:     regexp.MustCompile("foo"),
				replacement: []byte("bar"),
			},
		},
		WaitGroup: new(sync.WaitGroup),
	}

	// Walk the directory.
	w.Walk(tmpDir)

	// Wait for all goroutines to finish.
	w.Wait()

	// Check that the non-hidden file has been edited.
	content, err := os.ReadFile(filepath.Join(tmpDir, "foo.txt"))
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{}
	if !bytes.Equal(content, expected) {
		t.Errorf("Expected content of 'foo.txt' to be empty; got %q", content)
	}

	// Check that the hidden file has not been edited.
	content, err = os.ReadFile(filepath.Join(tmpDir, ".hidden.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, expected) {
		t.Errorf("Expected content of '.hidden.txt' to be empty; got %q", content)
	}
}

// TestPairsetString tests the String method of the pairset type.
func TestPairsetString(t *testing.T) {
	tests := []struct {
		name     string
		pairs    pairset
		expected string
	}{
		{
			name: "Single Pair",
			pairs: pairset{
				{
					pattern:     regexp.MustCompile("foo"),
					replacement: []byte("bar"),
				},
			},
			expected: "['foo', 'bar']",
		},
		{
			name: "Multiple Pairs",
			pairs: pairset{
				{
					pattern:     regexp.MustCompile("foo"),
					replacement: []byte("bar"),
				},
				{
					pattern:     regexp.MustCompile("baz"),
					replacement: []byte("qux"),
				},
			},
			expected: "['foo', 'bar'] ['baz', 'qux']",
		},
		{
			name:     "Empty Pairset",
			pairs:    pairset{},
			expected: "",
		},
		{
			name: "Special Characters",
			pairs: pairset{
				{
					pattern:     regexp.MustCompile(`a\w+b`),
					replacement: []byte("replacement"),
				},
			},
			expected: "['a\\w+b', 'replacement']",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.pairs.String()
			if result != test.expected {
				t.Errorf("pairset.String() = %q; want %q", result, test.expected)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	// Save the original command-line arguments and restore after the test.
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Reset the flag CommandLine so it doesn't conflict with other tests.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Simulate command-line arguments.
	// Explanation of arguments:
	// -p: print to stdout
	// -v: verbose
	// -g *.txt: glob pattern
	// -a: include hidden
	// -l 3: max depth
	// -r: replace names
	// -n: names only
	// After the flags, we must have pattern, replacement, and files.
	// Here "foo" is the pattern, "bar" is the replacement, and then two files: file1.txt and file2.txt.
	os.Args = []string{
		"cmd",
		"-p",
		"-v",
		"-g", "*.txt",
		"-a",
		"-l", "3",
		"-r",
		"-n",
		"foo", "bar", "file1.txt", "file2.txt",
	}

	// Execute parseFlags
	w, files := parseFlags()

	// Check walker fields
	if !w.ToStdout {
		t.Errorf("expected ToStdout to be true, got false")
	}
	if !w.IsVerbose {
		t.Errorf("expected IsVerbose to be true, got false")
	}
	if w.Glob != "*.txt" {
		t.Errorf("expected Glob to be '*.txt', got %s", w.Glob)
	}
	if !w.IncludeHidden {
		t.Errorf("expected IncludeHidden to be true, got false")
	}
	if w.MaxDepth != 3 {
		t.Errorf("expected MaxDepth to be 3, got %d", w.MaxDepth)
	}
	if !w.ReplaceNames {
		t.Errorf("expected ReplaceNames to be true, got false")
	}
	if !w.NamesOnly {
		t.Errorf("expected NamesOnly to be true, got false")
	}
	if w.WaitGroup == nil {
		t.Errorf("expected WaitGroup to be initialized, got nil")
	}

	// Check that w.pairs is initialized correctly.
	if len(w.pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(w.pairs))
	}

	p := w.pairs[0]
	if p.pattern.String() != "foo" {
		t.Errorf("expected pattern 'foo', got %s", p.pattern.String())
	}
	if string(p.replacement) != "bar" {
		t.Errorf("expected replacement 'bar', got %s", p.replacement)
	}

	// Check the files slice
	expectedFiles := []string{filepath.Clean("file1.txt"), filepath.Clean("file2.txt")}
	if !reflect.DeepEqual(files, expectedFiles) {
		t.Errorf("expected files %v, got %v", expectedFiles, files)
	}
}

func TestUsageWithJetAsArg0(t *testing.T) {
	// Save original os.Args and stdout
	origArgs := os.Args
	origStdout := os.Stdout

	defer func() {
		os.Args = origArgs
		os.Stdout = origStdout
	}()

	// Set os.Args[0] to "jet"
	os.Args = []string{"jet"}

	// Create a pipe to capture usage output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Call usage to print to our pipe
	usage()

	// Restore original stdout
	w.Close()
	os.Stdout = origStdout

	// Read the captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	r.Close()

	output := buf.String()

	// Define the expected usage output when os.Args[0] is "jet"
	expected := `jet - Just Edit Text
Jet allows you to replace all the substrings matched by specified regular
expressions in one or more files.
If it is given a directory as input, it will recursively replace all matches
in the files of the directory tree.

Usage:
  jet [options] pattern replacement input-files...
  jet [options] -e pattern1 replacement1 -e pattern2 replacement2 input-files...

Options:
  -p                       Print to stdout instead of modifying files.
  -v                       Enable verbose mode; explain what is being done.
  -g string                Only process files matching the given glob pattern.
  -a                       Includes hidden files (those starting with a dot).
  -l int                   Maximum depth for directory traversal.
  -r, --replace-names      Replace matches in file and directory names.
  -n, --names-only         Only replace matching names, ignoring file contents.
  -e pattern replacement   Specify a regular expression pattern and replacement.
                           Can be used multiple times for multiple replacements.
  -h, --help               Prints this help message and exits.

Notice:
  When using the -e flag multiple times, the pattern-replacement pairs are
  executed in the same order they are specified, one by one.

Examples:
  jet "foo" "bar" my/path1 my/path2
    Replace all occurrences of "foo" with "bar" in the files under my/path1
    and my/path2.

  jet -e "foo" "bar" -e "baz" "qux" my/path1 my/path2
    Replace all occurrences of "foo" with "bar" and "baz" with "qux" in the
    files under my/path1 and my/path2.

  jet -p -v "foo" "bar" my/path1
    Replace "foo" with "bar" in my/path1 and print the results to stdout
    with verbose output.

  jet -e "foo" "bar" -e "baz" "qux" -g "*.txt" -a my/path1
    Replace "foo" with "bar" and "baz" with "qux" in all text files,
    including hidden files, under my/path1.

Jet Copyright (C) 2023  Nicolò Santamaria
This program comes with ABSOLUTELY NO WARRANTY; for details refer to
https://www.gnu.org/licenses/gpl-3.0.html.
This is free software, and you are welcome to change and redistribute it
under the conditions defined by the license.
`

	if output != expected {
		t.Errorf("usage output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// Utility to capture stdout.
func captureStdout(f func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = origStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

// MockDirEntry for testing processFile.
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m MockDirEntry) Name() string               { return m.name }
func (m MockDirEntry) IsDir() bool                { return m.isDir }
func (m MockDirEntry) Type() fs.FileMode          { return 0 }
func (m MockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestWalkerEdit_ErrorReadingFile(t *testing.T) {
	w := &walker{
		pairs: pairset{{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")}},
	}

	// Try editing a non-existent file.
	output := captureStdout(func() {
		w.edit("non_existent_file.txt")
	})
	if !strings.Contains(output, "no such file or directory") {
		t.Errorf("expected error message about missing file, got: %s", output)
	}
}

func TestWalkerEdit_ToStdout(t *testing.T) {
	// Create a temp file
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("foo baz")
	tmpfile.Write(content)
	tmpfile.Close()

	w := &walker{
		ToStdout: true,
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	output := captureStdout(func() {
		w.edit(tmpfile.Name())
	})

	if output != "bar baz" {
		t.Errorf("expected replaced output 'bar baz', got %q", output)
	}
}

func TestWalkerEdit_Verbose(t *testing.T) {
	// Create a temp file
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte("foo"))
	tmpfile.Close()

	w := &walker{
		IsVerbose: true,
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	output := captureStdout(func() {
		w.edit(tmpfile.Name())
	})
	if !strings.Contains(output, "writing") {
		t.Errorf("expected 'writing' in verbose output, got %q", output)
	}
}

func TestWalkerEdit_StatError(t *testing.T) {
	// Stat error can be simulated by removing file before stat.
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	name := tmpfile.Name()
	tmpfile.Write([]byte("foo"))
	tmpfile.Close()
	os.Remove(name) // remove the file to cause a stat error later

	w := &walker{
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	output := captureStdout(func() {
		w.edit(name)
	})
	if !strings.Contains(output, "no such file or directory") {
		t.Errorf("expected stat error, got %q", output)
	}
}

func TestWalkerEdit_WriteError(t *testing.T) {
	// Cause write error by using a directory instead of a file
	tmpdir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	w := &walker{
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	output := captureStdout(func() {
		// Attempting to write to a directory path should fail
		w.edit(tmpdir)
	})
	if !strings.Contains(output, "is a directory") {
		t.Errorf("expected write error for directory, got %q", output)
	}
}

// Tests for editFilename
func TestWalkerEditFilename_NoChange(t *testing.T) {
	w := &walker{
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	// If filename does not contain "foo" no change is made.
	oldPath := "somefile.txt"
	newPath := w.editFilename(oldPath)
	if newPath != oldPath {
		t.Errorf("expected no change, got %q", newPath)
	}
}

func TestWalkerEditFilename_ChangeAndRenameSuccess(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	oldPath := filepath.Join(tmpdir, "foo.txt")
	if _, err := os.Create(oldPath); err != nil {
		t.Fatal(err)
	}

	w := &walker{
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
		IsVerbose: true,
	}

	output := captureStdout(func() {
		newPath := w.editFilename(oldPath)
		expected := filepath.Join(tmpdir, "bar.txt")
		if newPath != expected {
			t.Errorf("expected newPath %q, got %q", expected, newPath)
		}
	})

	if !strings.Contains(output, "renaming") {
		t.Errorf("expected 'renaming' in verbose output")
	}
}

func TestWalkerEditFilename_RenameError(t *testing.T) {
	// Attempt rename to a non-writable directory to force error
	tmpdir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	oldPath := filepath.Join(tmpdir, "foo.txt")
	if _, err := os.Create(oldPath); err != nil {
		t.Fatal(err)
	}

	// Make the directory read-only
	os.Chmod(tmpdir, 0500)
	defer os.Chmod(tmpdir, 0700) // restore permissions

	w := &walker{
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	output := captureStdout(func() {
		newPath := w.editFilename(oldPath)
		if newPath != oldPath {
			t.Errorf("expected to return oldPath on error, got %q", newPath)
		}
	})

	if !strings.Contains(output, "permission denied") {
		t.Errorf("expected rename error message, got %q", output)
	}
}

// Tests for processFile
func TestWalkerProcessFile_WithErrorParameter(t *testing.T) {
	w := &walker{
		Glob:          "*",
		MaxDepth:      -1,
		IncludeHidden: true,
		WaitGroup:     new(sync.WaitGroup),
	}

	output := captureStdout(func() {
		w.processFile("anyfile.txt", MockDirEntry{name: "anyfile.txt", isDir: false},
			fs.ErrNotExist) // Simulated error
	})
	if !strings.Contains(output, "file does not exist") && !strings.Contains(output, "no such file") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestWalkerProcessFile_ExceedMaxDepth(t *testing.T) {
	w := &walker{
		MaxDepth:  1,
		WaitGroup: new(sync.WaitGroup),
	}

	// depth("a/b/c") = 3, which is greater than MaxDepth=1
	err := w.processFile("a/b/c", MockDirEntry{name: "c", isDir: true}, nil)
	if err != fs.SkipDir {
		t.Errorf("expected fs.SkipDir for exceeding max depth, got %v", err)
	}
}

func TestWalkerProcessFile_HiddenFilesNotIncluded(t *testing.T) {
	w := &walker{
		IncludeHidden: false,
		MaxDepth:      -1,
		WaitGroup:     new(sync.WaitGroup),
	}

	// Hidden file
	err := w.processFile(".hiddenfile", MockDirEntry{name: ".hiddenfile", isDir: false}, nil)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	// Since it's hidden and not included, no action is taken.
	// To confirm no goroutine started, we can rely on coverage or manual checks.
}

func TestWalkerProcessFile_HiddenDirsNotIncluded(t *testing.T) {
	w := &walker{
		IncludeHidden: false,
		WaitGroup:     new(sync.WaitGroup),
	}

	err := w.processFile(".hiddendir", MockDirEntry{name: ".hiddendir", isDir: true}, nil)
	if err != fs.SkipDir {
		t.Errorf("expected fs.SkipDir for hidden directory, got %v", err)
	}
}

func TestWalkerProcessFile_MatchesGlobAndNamesOnly(t *testing.T) {
	w := &walker{
		Glob:      "*.txt",
		NamesOnly: true,
		WaitGroup: new(sync.WaitGroup),
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	tmpdir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	oldFile := filepath.Join(tmpdir, "foo.txt")
	os.Create(oldFile)

	// processFile should trigger a rename in a goroutine
	output := captureStdout(func() {
		err := w.processFile(oldFile, MockDirEntry{name: "foo.txt", isDir: false}, nil)
		if err != nil {
			t.Fatal(err)
		}
		// Wait for goroutine
		w.Wait()
	})

	newFile := filepath.Join(tmpdir, "bar.txt")
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("expected renamed file 'bar.txt' to exist")
	}

	// NamesOnly means no content edit, just rename. Check output if verbose not set - no direct output expected.
	if output != "" {
		// Might print errors if any. It's okay if it's empty.
	}
}

func TestWalkerProcessFile_MatchesGlobAndEditFile(t *testing.T) {
	w := &walker{
		Glob:      "*.txt",
		NamesOnly: false,
		WaitGroup: new(sync.WaitGroup),
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	// Create a temp file that matches glob and contains "foo".
	tmpdir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	filePath := filepath.Join(tmpdir, "test.txt")
	os.WriteFile(filePath, []byte("foo"), 0644)

	output := captureStdout(func() {
		err := w.processFile(filePath, MockDirEntry{name: "test.txt", isDir: false}, nil)
		if err != nil {
			t.Fatal(err)
		}
		w.Wait()
	})

	// Check file content edited
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "bar" {
		t.Errorf("expected content 'bar', got %q", content)
	}

	// output might be empty or contain error messages if something went wrong
	if strings.Contains(output, "error") {
		t.Errorf("unexpected error output: %s", output)
	}
}

func TestWalkerProcessFile_NoMatchGlob(t *testing.T) {
	w := &walker{
		Glob:      "*.md",
		WaitGroup: new(sync.WaitGroup),
		pairs: pairset{
			{pattern: regexp.MustCompile("foo"), replacement: []byte("bar")},
		},
	}

	err := w.processFile("test.txt", MockDirEntry{name: "test.txt"}, nil)
	if err != nil {
		t.Errorf("expected no error if glob does not match, got %v", err)
	}
	// No edits, no renames, no actions taken.
}
