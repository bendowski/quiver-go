//go:build system_ladybug

package main

// Downloads the native LadyBug shared library into lib-ladybug/ so it can be
// linked with the system_ladybug build tag. See:
// https://github.com/LadybugDB/go-ladybug#option-2-add-the-compiled-libraries-to-your-project
//
//go:generate sh -c "curl -fsSL https://raw.githubusercontent.com/LadybugDB/ladybug/refs/heads/main/scripts/download-liblbug.sh | LBUG_TARGET_DIR=$(pwd)/lib-ladybug bash"
//go:generate sh -c "[ \"$(uname)\" = Darwin ] && ln -sf liblbug.dylib lib-ladybug/liblbug.0.dylib || ln -sf liblbug.so lib-ladybug/liblbug.so.0"

/*
#cgo CFLAGS: -I${SRCDIR}/lib-ladybug
#cgo darwin LDFLAGS: -L${SRCDIR}/lib-ladybug -Wl,-rpath,${SRCDIR}/lib-ladybug
#cgo linux LDFLAGS: -L${SRCDIR}/lib-ladybug -Wl,-rpath,${SRCDIR}/lib-ladybug
#cgo windows LDFLAGS: -L${SRCDIR}/lib-ladybug
*/
import "C"

import (
	_ "github.com/LadybugDB/go-ladybug"
)
