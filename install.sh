#!/bin/sh
echo "compiling.."
go build

echo "copying jet to /usr/bin"
sudo cp jet /usr/bin/

echo "installing jet manual.."
gzip -c jet.1 > jet.1.gz
sudo cp jet.1.gz /usr/share/man/man1/

echo "done"
