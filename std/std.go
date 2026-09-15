// Package std embeds the Veles standard library sources so that the
// compiler binary is self-contained.
package std

import "embed"

//go:embed */*.vs
var FS embed.FS
