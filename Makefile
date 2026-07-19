BINARY := quiver
TAGS := system_ladybug
LADYBUG_LIB := $(CURDIR)/cmd/quiver/lib-ladybug

# The rpath goes through -extldflags rather than CGO_LDFLAGS: the toolchain
# applies CGO_LDFLAGS both per-package and at the final link, and macOS ld
# warns about the resulting duplicate -rpath entries.
export CGO_CFLAGS := -I$(LADYBUG_LIB)
export CGO_LDFLAGS := -L$(LADYBUG_LIB)

.PHONY: build clean test-pure

build:
	go build -tags $(TAGS) -ldflags "-extldflags '-Wl,-rpath,$(LADYBUG_LIB)'" -o ./$(BINARY) ./cmd/quiver

# Tests for every package that doesn't touch LadyBug; needs no native
# library, build tags, or CGO flags.
test-pure:
	go test ./model/... ./schema/... ./goparse/... ./dump/... ./store ./query/... ./internal/testutil

clean:
	rm -f ./$(BINARY)
