BINARY := quiver
TAGS := system_ladybug
LADYBUG_LIB := $(CURDIR)/cmd/quiver/lib-ladybug

# The rpath goes through -extldflags rather than CGO_LDFLAGS: the toolchain
# applies CGO_LDFLAGS both per-package and at the final link, and macOS ld
# warns about the resulting duplicate -rpath entries.
export CGO_CFLAGS := -I$(LADYBUG_LIB)
export CGO_LDFLAGS := -L$(LADYBUG_LIB)

.PHONY: build clean

build:
	go build -tags $(TAGS) -ldflags "-extldflags '-Wl,-rpath,$(LADYBUG_LIB)'" -o ./$(BINARY) ./cmd/quiver

clean:
	rm -f ./$(BINARY)
