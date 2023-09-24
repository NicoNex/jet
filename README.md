[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/) [![Go Report Card](https://goreportcard.com/badge/github.com/NicoNex/jet)](https://goreportcard.com/report/github.com/NicoNex/jet) [![License](http://img.shields.io/badge/license-GPL3.0-orange.svg?style=flat)](https://github.com/NicoNex/re/blob/master/LICENSE)

# jet - Just Edit Text
Jet is an intuitive and fast command-line tool for find & replace operations.

## Installation
You can install Jet using Go's package manager:
```bash
go install github.com/NicoNex/jet@latest
```

If you use [ArchLinux](https://archlinux.org/) you can install jet with [yay](https://github.com/Jguer/yay), [paru](https://github.com/morganamilo/paru) or any alternative AUR helper:
```bash
yay -S jet-edit
```
If you don't have an AUR helper installed:
```bash
git clone https://aur.archlinux.org/jet-edit.git
cd jet-edit
makepkg -si
```

Alternatively you can clone this repo and use the provided install.sh script that will install Jet alongside its man page for easy access:
```bash
./install.sh
```

## Usage
Jet allows you to replace all occurrences of a specified regex pattern in one or more files. It supports both single files and entire directories specified as input.

If you provide Jet with a directory as input, it will recursively find and replace text in all files within the directory tree.

You can also use `-` as the filename to read from stdin and write to stdout.

## Command
```bash
jet [options] "pattern" "replacement" input-files
```

### Options
- `-p`: Print the output to stdout instead of writing each file.
- `-v`: Enable verbose mode to explain the actions being performed.
- `-g string`: Add a glob pattern that file names must match to be edited.
- `-a`: Include hidden files (those starting with a dot).
- `-l int`: Set the maximum depth for directory tree processing. (Default: -1 for unlimited)
- `-h`: Display the help message and exit.
