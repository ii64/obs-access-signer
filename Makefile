all: build

export DOCKER_BUILDKIT=1

# use zig cc/c++ to statically link deps
TARGET_TRIPLE := x86_64-linux

CFLAGS ?=
CFLAGS += -target $(TARGET_TRIPLE)
CXXFLAGS ?=
CXXFLAGS += -target $(TARGET_TRIPLE)
GOFLAGS ?=
GOFLAGS += -x -trimpath

.PHONY: dep
dep:
	go mod download

build:
	CGO_ENABLED=0 CC="zig cc $(CFLAGS)" CXX="zig c++ $(CXXFLAGS)" go build $(GOFLAGS) .