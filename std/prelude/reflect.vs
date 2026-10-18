// Prelude — reflection over a struct's fields (D174). `T.fields` is a
// constant list of these, known while the program is compiled; a library
// reads it in a `const fun` to build tables, or walks it with an
// `inline loop` to write code that touches each field with its own type.

/// A field of a struct, as the compiler describes it while compiling.
/// `Options.fields` is a `List<FieldInfo>` in declaration order. The
/// annotations on the field are read with `f.attr<A>()` (an `A?`) and
/// `f.has<A>()`, in a `const fun` or in an `inline loop`.
public struct FieldInfo {
  /// The name as declared.
  public name: string
  /// The doc comment above the field, without the `///`.
  public doc: string = ""
  /// The type as the compiler spells it: `i64`, `List<string>`.
  public typeName: string = ""
  /// Whether the field has a default value.
  public hasDefault: bool = false
  /// The position in declaration order, from 0.
  public index: i64 = 0
  /// Which struct this describes, for `attr` and `has` (the compiler's number).
  public owner: i64 = -1
}
