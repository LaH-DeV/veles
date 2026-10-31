package selfhost

import "flag"

// update is shared by tests that regenerate self-hosted Veles sources.
var update = flag.Bool("update", false, "regenerate selfhost-generated sources")