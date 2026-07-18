//go:build system_ladybug

package main

// Downloads the native LadyBug shared library into lib-ladybug/ so it can be
// linked with the system_ladybug build tag. See:
// https://github.com/LadybugDB/go-ladybug#option-2-add-the-compiled-libraries-to-your-project
//
//go:generate sh -c "curl -fsSL https://raw.githubusercontent.com/LadybugDB/ladybug/refs/heads/main/scripts/download-liblbug.sh | LBUG_TARGET_DIR=$(pwd)/lib-ladybug bash"
//go:generate sh -c "[ \"$(uname)\" = Darwin ] && ln -sf liblbug.dylib lib-ladybug/liblbug.0.dylib || ln -sf liblbug.so lib-ladybug/liblbug.so.0"
//
// Compiler/linker flags pointing at lib-ladybug/ are supplied via CGO_CFLAGS and
// CGO_LDFLAGS (see the Makefile) rather than #cgo pragmas here, because pragmas
// in this package would not propagate to go-ladybug's own cgo compilation anyway.
