.PHONY: all getllama build download
LLAMA_URL := https://github.com/ggerganov/llama.cpp/releases/download/b4042/llama-b4042-bin-macos-arm64.zip
BIN_DIR := bin
MODEL_NAME := gte-base_fp32.gguf
MODEL_REPO := ChristianAzinn/gte-base-gguf
CONFIG_DIR := ~/.config/puma

all: statics build llama model
build:
	@go build -o $(BIN_DIR)/

clean: clean-db clean-bin
clean-bin:
	rm -rf $(BIN_DIR)

model:
	@huggingface-cli download --local-dir $(CONFIG_DIR) $(MODEL_REPO) $(MODEL_NAME)
	ln -Fs $(MODEL_NAME) $(CONFIG_DIR)/model.gguf

llama:
	@scripts/getllama.sh

clean-db:
	rm -rf ~/.config/puma/db.sqlite ~/.config/puma/vectors.db

statics:
	cp resources/credentials.json $(BIN_DIR)/
