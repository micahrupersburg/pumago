#!/bin/sh
if [ -f bin/llama-server ]; then
    echo "llama-server already exists"
    exit 0
fi

URL="https://github.com/ggerganov/llama.cpp/releases/download/b4042/llama-b4042-bin-macos-arm64.zip"
echo Downloading $URL
mkdir bin
# Download the zip file and extract the llama-server binary
curl -L $URL -o llama.zip
unzip -j llama.zip "build/bin/llama-server" -d bin
rm llama.zip

