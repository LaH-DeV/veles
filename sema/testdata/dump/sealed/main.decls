module main
  variant of Shape Circle  package
    field r: f64
  variant of Shape Rect  package
    field w: f64
    field h: f64
  sealed Shape  package
    variant Circle
    variant Rect
  fun area: fun(Shape): f64  package
  fun first: fun(List<string>): string?  package
  fun main: fun()  package
