(module
	(type $t0 (func (param i32 i32) (result i32)))
	(func $f1 (type $t0) (i32.add (local.get 0) (local.get 1)))
	(export "add" (func $f1))
)