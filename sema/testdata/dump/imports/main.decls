module main
  struct TooBig  package
    field limit: i64
  fun check: fun(Rect): f64 throws  package  [throws TooBig]
  fun main: fun() throws  package  [throws TooBig]
  implement Error for TooBig
module geo
  struct Rect  package
    field w: f64
    field h: f64
  fun area: fun(Rect): f64  package
  fun hidden: fun(): i64  package
module geo/units
  fun cm: fun(f64): f64  package
