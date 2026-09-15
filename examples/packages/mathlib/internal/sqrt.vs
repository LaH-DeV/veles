extern "C" { fun sqrt(x: f64): f64 }
pub fun root(x: f64): f64 = unsafe { sqrt(x) }
