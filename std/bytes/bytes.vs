/// IEC binary byte scales. These are counts of bytes: `KiB` is 1024 bytes,
/// while decimal `kB` is 1000 bytes. There is deliberately no `KB` or `MB`
/// spelling, which is ambiguous between the two systems.

/// One byte.
public const B: i64 = 1

/// One kibibyte: 1024 bytes.
public const KiB: i64 = 1024

/// One mebibyte: 1024 kibibytes.
public const MiB: i64 = KiB * 1024

/// One gibibyte: 1024 mebibytes.
public const GiB: i64 = MiB * 1024

/// One tebibyte: 1024 gibibytes.
public const TiB: i64 = GiB * 1024

/// One pebibyte: 1024 tebibytes.
public const PiB: i64 = TiB * 1024

/// One exbibyte: 1024 pebibytes, the largest IEC scale that fits in `i64`.
public const EiB: i64 = PiB * 1024
