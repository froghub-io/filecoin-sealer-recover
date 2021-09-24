SHELL=/usr/bin/env bash

all: build-deps build

unexport GOFLAGS

.PHONY: all build

LOTUS_PATH:=./extern/lotus/
TARGET=./sealer-recover

build-deps:
	git submodule update --init --recursive
	make -C ${LOTUS_PATH} deps

build: build-deps
	go build -o $(TARGET)

install:
	install -C $(TARGET) /usr/local/bin/sealer-recover

.PHONY: clean switch-interop switch-master

clean:
	-rm -f $(TARGET)
	-make -C $(LOTUS_PATH) clean

switch-master:
	git submodule set-branch --branch master $(TARGET)

switch-interop:
	git submodule set-branch --branch interopnet $(TARGET)


