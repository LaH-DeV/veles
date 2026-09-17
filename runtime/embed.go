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
