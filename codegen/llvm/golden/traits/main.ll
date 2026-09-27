%S.main.Square = type { double }
%S.main.Circle = type { double }
%S.main.Point = type { i64, i64 }
@desc.main.Square = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 0, i64 0, [0 x i64] [] }
@desc.main.Circle = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 0, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a2 = alloca { ptr, ptr }
  %a11 = alloca ptr
  %a13 = alloca ptr
  %a14 = alloca i64
  %a33 = alloca { ptr, ptr }
  %a45 = alloca i64
  %a47 = alloca i64
  %a52 = alloca %str
  %a58 = alloca %str
  %a61 = alloca %str
  %a63 = alloca %str
  %a72 = alloca %str
  %a78 = alloca %str
  %a87 = alloca %str
  %t1 = call ptr @veles_list_new(ptr @adesc.Shape, i64 2)
  %t3 = insertvalue %S.main.Square undef, double 0x4000000000000000, 0
  %t4 = call ptr @veles_gc_alloc(ptr @desc.main.Square, i64 8)
  store %S.main.Square %t3, ptr %t4
  %t5 = insertvalue { ptr, ptr } undef, ptr %t4, 0
  %t6 = insertvalue { ptr, ptr } %t5, ptr @vt.Shape.main.Square, 1
  store { ptr, ptr } %t6, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  %t7 = insertvalue %S.main.Circle undef, double 0x3FF0000000000000, 0
  %t8 = call ptr @veles_gc_alloc(ptr @desc.main.Circle, i64 8)
  store %S.main.Circle %t7, ptr %t8
  %t9 = insertvalue { ptr, ptr } undef, ptr %t8, 0
  %t10 = insertvalue { ptr, ptr } %t9, ptr @vt.Shape.main.Circle, 1
  store { ptr, ptr } %t10, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store ptr %t1, ptr %a11
  %t12 = load ptr, ptr %a11
  store ptr %t12, ptr %a13
  store i64 0, ptr %a14
  br label %loop.cond.1
loop.cond.1:
  %t15 = load i64, ptr %a14
  %t16 = load ptr, ptr %a13
  %t17 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t16, i32 0, i32 1
  %t18 = load i64, ptr %t17
  %t19 = icmp slt i64 %t15, %t18
  br i1 %t19, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t20 = load ptr, ptr %a13
  %t21 = load i64, ptr %a14
  %t22 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t20, i32 0, i32 1
  %t23 = load i64, ptr %t22
  %t24 = icmp ult i64 %t21, %t23
  br i1 %t24, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t25 = extractvalue %str { ptr @.str.1, i64 12 }, 0
  %t26 = extractvalue %str { ptr @.str.1, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t20, i64 %t21, ptr %t25, i64 %t26)
  unreachable
idx.ok.5:
  %t27 = load ptr, ptr %t20
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t20, i32 0, i32 3
  %t29 = load i64, ptr %t28
  %t30 = mul i64 %t29, %t21
  %t31 = getelementptr inbounds i8, ptr %t27, i64 %t30
  %t32 = load { ptr, ptr }, ptr %t31
  store { ptr, ptr } %t32, ptr %a33
  %t34 = load { ptr, ptr }, ptr %a33
  %t35 = extractvalue { ptr, ptr } %t34, 0
  %t36 = extractvalue { ptr, ptr } %t34, 1
  %t37 = getelementptr ptr, ptr %t36, i64 1
  %t38 = load ptr, ptr %t37
  %t39 = call %str %t38(ptr %t35)
  call void @v_std.io.println(%str %t39)
  br label %loop.post.2
loop.post.2:
  %t40 = load i64, ptr %a14
  %t41 = add i64 %t40, 1
  store i64 %t41, ptr %a14
  %t42 = load volatile i32, ptr @veles_stop_requested, align 4
  %t43 = icmp ne i32 %t42, 0
  br i1 %t43, label %safepoint.7, label %safepoint.on.8, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.7:
  call void @veles_gc_park()
  br label %safepoint.on.8
safepoint.on.8:
  br label %loop.cond.1
loop.end.3:
  %t44 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a45
  call void @veles_list_push(ptr %t44, ptr %a45)
  store i64 9, ptr %a45
  call void @veles_list_push(ptr %t44, ptr %a45)
  store i64 4, ptr %a45
  call void @veles_list_push(ptr %t44, ptr %a45)
  %t46 = call { i1, i64 } @v_main.largest__i64(ptr %t44)
  %t49 = extractvalue { i1, i64 } %t46, 0
  %t48 = xor i1 %t49, true
  br i1 %t48, label %elvis.default.9, label %elvis.some.10
elvis.some.10:
  %t50 = extractvalue { i1, i64 } %t46, 1
  store i64 %t50, ptr %a47
  br label %elvis.end.11
elvis.default.9:
  store i64 0, ptr %a47
  br label %elvis.end.11
elvis.end.11:
  %t51 = load i64, ptr %a47
  call void @veles_i64_to_string(ptr %a52, i64 %t51)
  %t53 = load %str, ptr %a52
  %t54 = extractvalue %str %t53, 0
  %t55 = extractvalue %str %t53, 1
  %t56 = extractvalue %str { ptr @.str.2, i64 1 }, 0
  %t57 = extractvalue %str { ptr @.str.2, i64 1 }, 1
  call void @veles_string_concat(ptr %a58, ptr %t54, i64 %t55, ptr %t56, i64 %t57)
  %t59 = load %str, ptr %a58
  %t60 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.3, i64 1 }, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  store %str { ptr @.str.4, i64 1 }, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  %t62 = call { i1, %str } @v_main.largest__string(ptr %t60)
  %t65 = extractvalue { i1, %str } %t62, 0
  %t64 = xor i1 %t65, true
  br i1 %t64, label %elvis.default.12, label %elvis.some.13
elvis.some.13:
  %t66 = extractvalue { i1, %str } %t62, 1
  store %str %t66, ptr %a63
  br label %elvis.end.14
elvis.default.12:
  store %str { ptr @.str.5, i64 0 }, ptr %a63
  br label %elvis.end.14
elvis.end.14:
  %t67 = load %str, ptr %a63
  %t68 = extractvalue %str %t59, 0
  %t69 = extractvalue %str %t59, 1
  %t70 = extractvalue %str %t67, 0
  %t71 = extractvalue %str %t67, 1
  call void @veles_string_concat(ptr %a72, ptr %t68, i64 %t69, ptr %t70, i64 %t71)
  %t73 = load %str, ptr %a72
  %t74 = extractvalue %str %t73, 0
  %t75 = extractvalue %str %t73, 1
  %t76 = extractvalue %str { ptr @.str.2, i64 1 }, 0
  %t77 = extractvalue %str { ptr @.str.2, i64 1 }, 1
  call void @veles_string_concat(ptr %a78, ptr %t74, i64 %t75, ptr %t76, i64 %t77)
  %t79 = load %str, ptr %a78
  %t80 = insertvalue %S.main.Point undef, i64 1, 0
  %t81 = insertvalue %S.main.Point %t80, i64 2, 1
  %t82 = call %str @show.main.Point(%S.main.Point %t81)
  %t83 = extractvalue %str %t79, 0
  %t84 = extractvalue %str %t79, 1
  %t85 = extractvalue %str %t82, 0
  %t86 = extractvalue %str %t82, 1
  call void @veles_string_concat(ptr %a87, ptr %t83, i64 %t84, ptr %t85, i64 %t86)
  %t88 = load %str, ptr %a87
  call void @v_std.io.println(%str %t88)
  ret void
}

define double @v_main.Shape.Square.area(ptr %p0) {
entry:
  %a1 = alloca ptr
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds %S.main.Square, ptr %t2, i32 0, i32 0
  %t4 = load double, ptr %t3
  %t5 = load ptr, ptr %a1
  %t6 = getelementptr inbounds %S.main.Square, ptr %t5, i32 0, i32 0
  %t7 = load double, ptr %t6
  %t8 = fmul double %t4, %t7
  ret double %t8
}

define double @v_main.Shape.Circle.area(ptr %p0) {
entry:
  %a1 = alloca ptr
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds %S.main.Circle, ptr %t2, i32 0, i32 0
  %t4 = load double, ptr %t3
  %t5 = fmul double 0x4008000000000000, %t4
  %t6 = load ptr, ptr %a1
  %t7 = getelementptr inbounds %S.main.Circle, ptr %t6, i32 0, i32 0
  %t8 = load double, ptr %t7
  %t9 = fmul double %t5, %t8
  ret double %t9
}

define %str @v_main.Shape.Circle.describe(ptr %p0) {
entry:
  %a1 = alloca ptr
  store ptr %p0, ptr %a1
  ret %str { ptr @.str.6, i64 6 }
}

define %str @v_main.Display.Point.toString(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a5 = alloca %str
  %a11 = alloca %str
  %a17 = alloca %str
  %a22 = alloca %str
  %a28 = alloca %str
  %a34 = alloca %str
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds %S.main.Point, ptr %t2, i32 0, i32 0
  %t4 = load i64, ptr %t3
  call void @veles_i64_to_string(ptr %a5, i64 %t4)
  %t6 = load %str, ptr %a5
  %t7 = extractvalue %str { ptr @.str.7, i64 1 }, 0
  %t8 = extractvalue %str { ptr @.str.7, i64 1 }, 1
  %t9 = extractvalue %str %t6, 0
  %t10 = extractvalue %str %t6, 1
  call void @veles_string_concat(ptr %a11, ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  %t12 = load %str, ptr %a11
  %t13 = extractvalue %str %t12, 0
  %t14 = extractvalue %str %t12, 1
  %t15 = extractvalue %str { ptr @.str.8, i64 2 }, 0
  %t16 = extractvalue %str { ptr @.str.8, i64 2 }, 1
  call void @veles_string_concat(ptr %a17, ptr %t13, i64 %t14, ptr %t15, i64 %t16)
  %t18 = load %str, ptr %a17
  %t19 = load ptr, ptr %a1
  %t20 = getelementptr inbounds %S.main.Point, ptr %t19, i32 0, i32 1
  %t21 = load i64, ptr %t20
  call void @veles_i64_to_string(ptr %a22, i64 %t21)
  %t23 = load %str, ptr %a22
  %t24 = extractvalue %str %t18, 0
  %t25 = extractvalue %str %t18, 1
  %t26 = extractvalue %str %t23, 0
  %t27 = extractvalue %str %t23, 1
  call void @veles_string_concat(ptr %a28, ptr %t24, i64 %t25, ptr %t26, i64 %t27)
  %t29 = load %str, ptr %a28
  %t30 = extractvalue %str %t29, 0
  %t31 = extractvalue %str %t29, 1
  %t32 = extractvalue %str { ptr @.str.9, i64 1 }, 0
  %t33 = extractvalue %str { ptr @.str.9, i64 1 }, 1
  call void @veles_string_concat(ptr %a34, ptr %t30, i64 %t31, ptr %t32, i64 %t33)
  %t35 = load %str, ptr %a34
  ret %str %t35
}

define %str @v_main.Shape.describe_Self_main.Square_(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a4 = alloca %str
  %a10 = alloca %str
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = call double @v_main.Shape.Square.area(ptr %t2)
  call void @veles_f64_to_string(ptr %a4, double %t3)
  %t5 = load %str, ptr %a4
  %t6 = extractvalue %str { ptr @.str.10, i64 5 }, 0
  %t7 = extractvalue %str { ptr @.str.10, i64 5 }, 1
  %t8 = extractvalue %str %t5, 0
  %t9 = extractvalue %str %t5, 1
  call void @veles_string_concat(ptr %a10, ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  %t11 = load %str, ptr %a10
  ret %str %t11
}

define { i1, i64 } @v_main.largest__i64(ptr %p1) {
entry:
  %a1 = alloca ptr
  %a3 = alloca ptr
  store ptr %p1, ptr %a1
  %t2 = load ptr, ptr %a1
  store ptr %t2, ptr %a3
  %t4 = call { i1, i64 } @v_std.prelude.extend.List_T.max_T_i64_(ptr %a3)
  ret { i1, i64 } %t4
}

define { i1, %str } @v_main.largest__string(ptr %p1) {
entry:
  %a1 = alloca ptr
  %a3 = alloca ptr
  store ptr %p1, ptr %a1
  %t2 = load ptr, ptr %a1
  store ptr %t2, ptr %a3
  %t4 = call { i1, %str } @v_std.prelude.extend.List_T.max_T_string_(ptr %a3)
  ret { i1, %str } %t4
}

define internal double @vt.Shape.main.Square.0(ptr %self) {
entry:
  %t1 = call double @v_main.Shape.Square.area(ptr %self)
  ret double %t1
}

define internal %str @vt.Shape.main.Square.1(ptr %self) {
entry:
  %t1 = call %str @v_main.Shape.describe_Self_main.Square_(ptr %self)
  ret %str %t1
}

@vt.Shape.main.Square = internal constant [2 x ptr] [ptr @vt.Shape.main.Square.0, ptr @vt.Shape.main.Square.1]
define internal double @vt.Shape.main.Circle.0(ptr %self) {
entry:
  %t1 = call double @v_main.Shape.Circle.area(ptr %self)
  ret double %t1
}

define internal %str @vt.Shape.main.Circle.1(ptr %self) {
entry:
  %t1 = call %str @v_main.Shape.Circle.describe(ptr %self)
  ret %str %t1
}

@vt.Shape.main.Circle = internal constant [2 x ptr] [ptr @vt.Shape.main.Circle.0, ptr @vt.Shape.main.Circle.1]
define internal %str @show.main.Point(%S.main.Point %v) {
entry:
  %a2 = alloca %S.main.Point
  store %S.main.Point %v, ptr %a2
  %t1 = call %str @v_main.Display.Point.toString(ptr %a2)
  ret %str %t1
}

@.str.1 = private unnamed_addr constant [13 x i8] c"main.vs:33:3\00"
@.str.2 = private unnamed_addr constant [2 x i8] c" \00"
@.str.3 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.4 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.5 = private unnamed_addr constant [1 x i8] c"\00"
@.str.6 = private unnamed_addr constant [7 x i8] c"circle\00"
@.str.7 = private unnamed_addr constant [2 x i8] c"(\00"
@.str.8 = private unnamed_addr constant [3 x i8] c", \00"
@.str.9 = private unnamed_addr constant [2 x i8] c")\00"
@.str.10 = private unnamed_addr constant [6 x i8] c"area \00"
