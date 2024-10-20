/*
 * jet - Just Edit Text
 * Copyright (C) 2023 Nicolò Santamaria
 *
 * Jet is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Jet is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type pair struct {
	pattern     *regexp.Regexp
	replacement []byte
}

func (p pair) match(src []byte) bool {
	return p.pattern.Match(src)
}

func (p pair) replaceAll(src []byte) []byte {
	return p.pattern.ReplaceAll(src, p.replacement)
}

type pairset []pair

func (p *pairset) Set(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	*p = append(*p, pair{pattern: re, replacement: []byte(flag.Arg(0))})
	flag.CommandLine.Parse(flag.Args()[1:])
	return nil
}

func (p pairset) String() string {
	var buf strings.Builder

	for i, pair := range p {
		buf.WriteString(fmt.Sprintf(
			"['%v', '%s']",
			pair.pattern,
			pair.replacement,
		))

		if i < len(p)-1 {
			buf.WriteByte(' ')
		}
	}
	return buf.String()
}

func (p pairset) match(src []byte) bool {
	for _, pair := range p {
		if pair.match(src) {
			return true
		}
	}
	return false
}

func (p pairset) replaceAll(src []byte) []byte {
	for _, pair := range p {
		src = pair.replaceAll(src)
	}
	return src
}

type walker struct {
	ToStdout      bool
	Glob          string
	IsVerbose     bool
	MaxDepth      int
	IncludeHidden bool
	ReplaceNames  bool
	NamesOnly     bool
	pairs         pairset
	*sync.WaitGroup
}

func (w *walker) matchGlob(path string) bool {
	ok, err := filepath.Match(w.Glob, filepath.Base(path))
	if err != nil {
		fmt.Println(err)
	}
	return ok
}

func (w *walker) edit(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	if w.ToStdout {
		fmt.Print(string(w.pairs.replaceAll(b)))
		return
	}

	if w.IsVerbose {
		fmt.Printf("writing %s\n", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.WriteFile(
		path,
		w.pairs.replaceAll(b),
		info.Mode().Perm(),
	)
	if err != nil {
		fmt.Println(err)
	}
}

func (w *walker) editStdin() {
	b, err := bufio.NewReader(os.Stdin).ReadBytes(0)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return
	}
	fmt.Print(string(w.pairs.replaceAll(b)))
}

func (w *walker) editFilename(path string) string {
	var (
		base    = filepath.Base(path)
		newbase = string(w.pairs.replaceAll([]byte(base)))
		newpath = filepath.Join(filepath.Dir(path), newbase)
	)

	// Return early if there are no changes in the name.
	if newbase == base {
		return path
	}

	if w.IsVerbose {
		fmt.Printf("renaming %s to %s", path, newpath)
	}

	if err := os.Rename(path, newpath); err != nil {
		fmt.Println(err)
		return path
	}
	return newpath
}

func isHidden(name string) bool {
	return name != "." && name != ".." && strings.HasPrefix(name, ".")
}

func (w *walker) processFile(path string, d fs.DirEntry, err error) error {
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// If the depth exceeds skip the entire directory.
	if d.IsDir() && w.MaxDepth >= 0 && depth(path) > w.MaxDepth {
		return fs.SkipDir
	}
	// Skip hidden files if not specified otherwise.
	if isHidden(d.Name()) && !w.IncludeHidden {
		if d.IsDir() {
			return fs.SkipDir
		}
		return nil
	}

	if w.matchGlob(path) {
		w.Add(1)
		go func() {
			defer w.Done()

			if w.NamesOnly || w.ReplaceNames {
				path = w.editFilename(path)
			}
			if !d.IsDir() && !w.NamesOnly && w.matchGlob(path) {
				w.edit(path)
			}
		}()
	}

	return nil
}

func (w *walker) Walk(paths ...string) {
	defer w.Wait()

	for _, p := range paths {
		if p == "-" {
			w.editStdin()
		} else {
			if err := filepath.WalkDir(p, w.processFile); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func depth(path string) int {
	return strings.Count(path, string(os.PathSeparator)) + 1
}

func main() {
	w, files := parseFlags()
	w.Walk(files...)
}

func containsDash(files []string) bool {
	for _, f := range files {
		if f == "-" {
			return true
		}
	}
	return false
}

func parseFlags() (w walker, files []string) {
	flag.Usage = usage
	flag.BoolVar(&w.ToStdout, "p", false, "Print to stdout.")
	flag.BoolVar(&w.IsVerbose, "v", false, "Verbose, explain what is being done.")
	flag.StringVar(&w.Glob, "g", "*", "Add a pattern the file names must match to be edited.")
	flag.BoolVar(&w.IncludeHidden, "a", false, "Includes hidden files (starting with a dot).")
	flag.IntVar(&w.MaxDepth, "l", -1, "Max depth.")
	flag.BoolVar(&w.ReplaceNames, "r", false, "Replace matches in file and directory names.")
	flag.BoolVar(&w.ReplaceNames, "replace-names", false, "Replace matches in file and directory names.")
	flag.BoolVar(&w.NamesOnly, "n", false, "Only replace matches in names, ignoring file contents.")
	flag.BoolVar(&w.NamesOnly, "names-only", false, "Only replace matches in names, ignoring file contents.")
	flag.Var(&w.pairs, "e", "Specify two arguments per flag usage for executing a replacement operation.")
	flag.Parse()

	w.WaitGroup = new(sync.WaitGroup)

	// Exit early if the pairs are set in the flags but no path is provided.
	if w.pairs != nil && flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// If no pair is provided using the -e flags:
	//   - Expect the pattern and replacement in the first two command-line arguments.
	//   - Process file paths starting from index 2.
	// Otherwise:
	//   - Process file paths starting from index 0.
	if w.pairs == nil {
		if flag.NArg() < 3 {
			flag.Usage()
			os.Exit(1)
		}

		re, err := regexp.Compile(flag.Arg(0))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		w.pairs = []pair{{pattern: re, replacement: []byte(flag.Arg(1))}}

		// Clean the user-provided paths.
		for _, f := range flag.Args()[2:] {
			files = append(files, filepath.Clean(f))
		}
	} else {
		for _, f := range flag.Args() {
			files = append(files, filepath.Clean(f))
		}
	}

	if len(files) > 1 && containsDash(files) {
		fmt.Println("cannot edit multiple files and stdin at the same time")
		os.Exit(1)
	}
	return
}

func usage() {
	fmt.Printf(
		`jet - Just Edit Text
Jet allows you to replace all the substrings matched by specified regular
expressions in one or more files.
If it is given a directory as input, it will recursively replace all matches
in the files of the directory tree.

Usage:
  %s [options] pattern replacement input-files...
  %s [options] -e pattern1 replacement1 -e pattern2 replacement2 input-files...

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
  %s "foo" "bar" my/path1 my/path2
    Replace all occurrences of "foo" with "bar" in the files under my/path1
    and my/path2.

  %s -e "foo" "bar" -e "baz" "qux" my/path1 my/path2
    Replace all occurrences of "foo" with "bar" and "baz" with "qux" in the
    files under my/path1 and my/path2.

  %s -p -v "foo" "bar" my/path1
    Replace "foo" with "bar" in my/path1 and print the results to stdout
    with verbose output.

  %s -e "foo" "bar" -e "baz" "qux" -g "*.txt" -a my/path1
    Replace "foo" with "bar" and "baz" with "qux" in all text files,
    including hidden files, under my/path1.

Jet Copyright (C) 2023  Nicolò Santamaria
This program comes with ABSOLUTELY NO WARRANTY; for details refer to
https://www.gnu.org/licenses/gpl-3.0.html.
This is free software, and you are welcome to change and redistribute it
under the conditions defined by the license.
`,
		os.Args[0],
		os.Args[0],
		os.Args[0],
		os.Args[0],
		os.Args[0],
		os.Args[0],
	)
}
