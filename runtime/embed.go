// Package runtime embeds the C runtime that every Veles executable links.
package runtime

import _ "embed"

//go:embed c/veles_rt.c
var Source string

//go:embed c/veles_gc.c
var GCSource string

//go:embed c/veles_task.c
var TaskSource string

//go:embed c/veles_os.c
var OSSource string

//go:embed c/veles_net.c
var NetSource string

//go:embed c/veles_ffi.c
var FFISource string

//go:embed c/veles_sync.c
var SyncSource string

// TLSHeader is the per-thread block every runtime file includes.
//
//go:embed c/veles_tls.h
var TLSHeader string

//go:embed c/veles_stack.c
var StackSource string

//go:embed c/veles_tlsio.c
var TLSIOSource string
