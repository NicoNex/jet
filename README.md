[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/) [![Go Report Card](https://goreportcard.com/badge/github.com/NicoNex/jet)](https://goreportcard.com/report/github.com/NicoNex/jet) [![License](http://img.shields.io/badge/license-GPL3.0-orange.svg?style=flat)](https://github.com/NicoNex/re/blob/master/LICENSE)

# jet
Just Edit Text

Jet is an intuitive and fast find & replace cli.
Jet replaces all matches of a specified regex with the provided replacement text.
It can be run over single files or entire directories specified in the command line argument.
When jet encounters a directory it recursively finds and replaces text in all the files in the directory tree.

Install it with `go install github.com/NicoNex/jet@latest`.
Run `jet -h` for more options.

## Usage
```
jet - Just Edit Text
Jet allows you to replace all the substrings matched by a specified regex in
one or more files.
If it is given a directory as input, it will recursively replace all the
matches in the files of the directory tree.

Usage:
    jet [options] "pattern" "replacement" input-files

Options:
    -p           Print to stdout instead of writing each file.
    -v           Verbose, explain what is being done.
    -g string    Add a glob the file names must match to be edited.
    -a           Includes hidden files (starting with a dot).
    -l int       Max depth in a directory tree.
    -h           Prints this help message and exits.
```
