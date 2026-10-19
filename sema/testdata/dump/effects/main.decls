module main
  struct Failed  package
    field why: string
  fun main: fun() throws Failed  package
  fun risky: fun(i64): i64 throws Failed  package
  fun twice<T>: fun(T, fun(T): T): T  package
  implement Error for Failed
