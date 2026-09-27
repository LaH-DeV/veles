/// The one bound on recursion depth (D75): `maxRecursionDepth` (1000), the
/// sentence every limit in the standard library reports
/// (`tooDeepMessage(limit)`), and `Depth`, the counter for a walk whose
/// depth is only its call frames:
///
/// ```veles
/// use recursion
///
/// struct Parser {
///   private depth: recursion.Depth = recursion.Depth(limit: 500)
/// }
/// ```
///
/// Veles runs on the C stack and does not grow it, so an unbounded
/// recursive descent over input someone else wrote is a crash rather than
/// an error. The declarations are written with the prelude, whose decoders
/// use them; this module is where a program finds them.
