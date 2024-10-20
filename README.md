[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/) [![Go Report Card](https://goreportcard.com/badge/github.com/NicoNex/jet)](https://goreportcard.com/report/github.com/NicoNex/jet) [![License](http://img.shields.io/badge/license-GPL3.0-green.svg?style=flat)](https://github.com/NicoNex/jet/blob/master/LICENSE)

# jet - Just Edit Text
Jet is an intuitive and fast command-line tool for find & replace operations using regular expressions.
Jet allows you to replace all substrings matched by specified regular expressions in one or more files and directories.
It can process single files or entire directories specified as input.
When given a directory, Jet recursively finds and replaces matches in all files and directory names.

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
Jet allows you to replace all substrings matched by specified regular expressions in one or more files and directories. It supports both single files and entire directories specified as input.

If you provide Jet with a directory as input, it will recursively find and replace text in all files within the directory tree optionally replacing the file or directory names as well.

You can also use `-` as the filename to read from stdin and write to stdout.

## Command
```bash
jet [options] pattern replacement input-files
jet [options] -e pattern1 replacement1 -e pattern2 replacement2 input-files...
```

### Options
- `-p`: Print the output to stdout instead of writing each file.
- `-v`: Enable verbose mode; explain what is being done.
- `-g string`: Only process files matching the given glob pattern.
- `-a`: Include hidden files (those starting with a dot).
- `-l int`: Maximum depth for directory traversal. (Default: -1 for unlimited)
- `-r`, `--replace-names`: Replace matches in file and directory names.
- `-n`, `--names-only`: Only replace matches in names, ignoring file contents.
- `-e pattern replacement`: Specify a regular expression pattern and replacement. Can be used multiple times for multiple replacements.
- `-h`, `--help`: Prints the help message and exit.

## Examples

- **Replace all occurrences of "foo" with "bar" in the files under `my/path1` and `my/path2`:**

  ```bash
  jet "foo" "bar" my/path1 my/path2
  ```

- **Replace all occurrences of "foo" with "bar" and "baz" with "qux" in the files under `my/path1` and `my/path2`:**

  ```bash
  jet -e "foo" "bar" -e "baz" "qux" my/path1 my/path2
  ```

- **Replace "foo" with "bar" in `my/path1` and print the results to stdout with verbose output:**

  ```bash
  jet -p -v "foo" "bar" my/path1
  ```

- **Rename files and directories by replacing "foo" with "bar" in their names under `my/path1`, without modifying file contents:**

  ```bash
  jet -n "foo" "bar" my/path1
  ```

- **Replace "foo" with "bar" and "baz" with "qux" in all text files, including hidden files, under `my/path1`:**

  ```bash
  jet -g "*.txt" -a -e "foo" "bar" -e "baz" "qux" my/path1
  ```

## License

Jet is licensed under the GNU General Public License v3.0. See [LICENSE](https://github.com/NicoNex/jet/blob/master/LICENSE) for more information.
