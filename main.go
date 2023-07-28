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

type walker struct {
	ToStdout      bool
	Replacement   string
	Glob          string
	IsVerbose     bool
	MaxDepth      int
	IncludeHidden bool
	*regexp.Regexp
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
	defer w.Done()

	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	if w.ToStdout {
		fmt.Print(string(w.ReplaceAll(b, []byte(w.Replacement))))
	} else if w.Match(b) {
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
			w.ReplaceAll(b, []byte(w.Replacement)),
			info.Mode().Perm(),
		)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (w *walker) editStdin() {
	b, err := bufio.NewReader(os.Stdin).ReadBytes(0)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return
	}
	fmt.Print(string(w.ReplaceAll(b, []byte(w.Replacement))))
}

func (w *walker) walkDir(path string, d fs.DirEntry, err error) error {
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// If the depth exceeds skip the entire directory.
	if d.IsDir() && w.MaxDepth >= 0 && depth(path) > w.MaxDepth {
		return fs.SkipDir
	}
	// Skip hidden files if not specified otherwise.
	if d.Name()[0] == '.' && !w.IncludeHidden {
		return nil
	}
	if !d.IsDir() && w.matchGlob(path) {
		w.Add(1)
		go w.edit(path)
	}

	return nil
}

func (w *walker) Walk(paths ...string) {
	defer w.Wait()

	for _, p := range paths {
		if p == "-" {
			w.editStdin()
		} else {
			if err := filepath.WalkDir(p, w.walkDir); err != nil {
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
	flag.Parse()

	if flag.NArg() < 3 {
		flag.Usage()
		os.Exit(1)
	}

	re, err := regexp.Compile(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	w.WaitGroup = new(sync.WaitGroup)
	w.Regexp = re
	w.Replacement = flag.Arg(1)

	// Clean the user-provided paths.
	for _, f := range flag.Args()[2:] {
		files = append(files, filepath.Clean(f))
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
Jet allows you to replace all the substrings matched by a specified regex in
one or more files.
If it is given a directory as input, it will recursively replace all the
matches in the files of the directory tree.

Usage:
    %s [options] "pattern" "replacement" input-files

Options:
    -p           Print to stdout instead of writing each file.
    -v           Verbose, explain what is being done.
    -g string    Add a glob the file names must match to be edited.
    -a           Includes hidden files (starting with a dot).
    -l int       Max depth in a directory tree.
    -h           Prints this help message and exits.

Jet Copyright (C) 2023  Nicolò Santamaria
This program comes with ABSOLUTELY NO WARRANTY; for details refer to
https://www.gnu.org/licenses/gpl-3.0.html.
This is free software, and you are welcome to change and redistribute it
under the conditions defined by the license.
`,
		os.Args[0],
	)
}
