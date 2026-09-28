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
  %a47 = alloca i64
  %a51 = alloca i64
  %a56 = alloca [21 x i8]
  %a61 = alloca %str
  %a65 = alloca %str
  %a73 = alloca %str
  %a74 = alloca [5 x %str]
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
  %t40 = extractvalue %str { ptr @.str.2, i64 21 }, 0
  %t41 = extractvalue %str { ptr @.str.2, i64 21 }, 1
  call void @veles_call_push(ptr %t40)
  call void @v_std.io.println(%str %t39)
  call void @veles_call_pop()
  br label %loop.post.2
loop.post.2:
  %t42 = load i64, ptr %a14
  %t43 = add i64 %t42, 1
  store i64 %t43, ptr %a14
  %t44 = load volatile i32, ptr @veles_stop_requested, align 4
  %t45 = icmp ne i32 %t44, 0
  br i1 %t45, label %safepoint.7, label %safepoint.on.8, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.7:
  call void @veles_gc_park()
  br label %safepoint.on.8
safepoint.on.8:
  br label %loop.cond.1
loop.end.3:
  %t46 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a47
  call void @veles_list_push(ptr %t46, ptr %a47)
  store i64 9, ptr %a47
  call void @veles_list_push(ptr %t46, ptr %a47)
  store i64 4, ptr %a47
  call void @veles_list_push(ptr %t46, ptr %a47)
  %t48 = extractvalue %str { ptr @.str.3, i64 21 }, 0
  %t49 = extractvalue %str { ptr @.str.3, i64 21 }, 1
  call void @veles_call_push(ptr %t48)
  %t50 = call { i1, i64 } @v_main.largest__i64(ptr %t46)
  call void @veles_call_pop()
  %t53 = extractvalue { i1, i64 } %t50, 0
  %t52 = xor i1 %t53, true
  br i1 %t52, label %elvis.default.9, label %elvis.some.10
elvis.some.10:
  %t54 = extractvalue { i1, i64 } %t50, 1
  store i64 %t54, ptr %a51
  br label %elvis.end.11
elvis.default.9:
  store i64 0, ptr %a51
  br label %elvis.end.11
elvis.end.11:
  %t55 = load i64, ptr %a51
  %t57 = call i64 @veles_i64_format(ptr %a56, i64 %t55)
  %t58 = insertvalue %str undef, ptr %a56, 0
  %t59 = insertvalue %str %t58, i64 %t57, 1
  %t60 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.4, i64 1 }, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  store %str { ptr @.str.5, i64 1 }, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  %t62 = extractvalue %str { ptr @.str.6, i64 21 }, 0
  %t63 = extractvalue %str { ptr @.str.6, i64 21 }, 1
  call void @veles_call_push(ptr %t62)
  %t64 = call { i1, %str } @v_main.largest__string(ptr %t60)
  call void @veles_call_pop()
  %t67 = extractvalue { i1, %str } %t64, 0
  %t66 = xor i1 %t67, true
  br i1 %t66, label %elvis.default.12, label %elvis.some.13
elvis.some.13:
  %t68 = extractvalue { i1, %str } %t64, 1
  store %str %t68, ptr %a65
  br label %elvis.end.14
elvis.default.12:
  store %str { ptr @.str.7, i64 0 }, ptr %a65
  br label %elvis.end.14
elvis.end.14:
  %t69 = load %str, ptr %a65
  %t70 = insertvalue %S.main.Point undef, i64 1, 0
  %t71 = insertvalue %S.main.Point %t70, i64 2, 1
  %t72 = call %str @show.main.Point(%S.main.Point %t71)
  %t75 = getelementptr [5 x %str], ptr %a74, i64 0, i64 0
  store %str %t59, ptr %t75
  %t76 = getelementptr [5 x %str], ptr %a74, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t76
  %t77 = getelementptr [5 x %str], ptr %a74, i64 0, i64 2
  store %str %t69, ptr %t77
  %t78 = getelementptr [5 x %str], ptr %a74, i64 0, i64 3
  store %str { ptr @.str.8, i64 1 }, ptr %t78
  %t79 = getelementptr [5 x %str], ptr %a74, i64 0, i64 4
  store %str %t72, ptr %t79
  call void @veles_string_concat_n(ptr %a73, ptr %a74, i64 5)
  %t80 = load %str, ptr %a73
  %t81 = extractvalue %str { ptr @.str.9, i64 20 }, 0
  %t82 = extractvalue %str { ptr @.str.9, i64 20 }, 1
  call void @veles_call_push(ptr %t81)
  call void @v_std.io.println(%str %t80)
  call void @veles_call_pop()
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
  ret %str { ptr @.str.10, i64 6 }
}

define %str @v_main.Display.Point.toString(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a5 = alloca [21 x i8]
  %a12 = alloca [21 x i8]
  %a16 = alloca %str
  %a17 = alloca [5 x %str]
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds %S.main.Point, ptr %t2, i32 0, i32 0
  %t4 = load i64, ptr %t3
  %t6 = call i64 @veles_i64_format(ptr %a5, i64 %t4)
  %t7 = insertvalue %str undef, ptr %a5, 0
  %t8 = insertvalue %str %t7, i64 %t6, 1
  %t9 = load ptr, ptr %a1
  %t10 = getelementptr inbounds %S.main.Point, ptr %t9, i32 0, i32 1
  %t11 = load i64, ptr %t10
  %t13 = call i64 @veles_i64_format(ptr %a12, i64 %t11)
  %t14 = insertvalue %str undef, ptr %a12, 0
  %t15 = insertvalue %str %t14, i64 %t13, 1
  %t18 = getelementptr [5 x %str], ptr %a17, i64 0, i64 0
  store %str { ptr @.str.11, i64 1 }, ptr %t18
  %t19 = getelementptr [5 x %str], ptr %a17, i64 0, i64 1
  store %str %t8, ptr %t19
  %t20 = getelementptr [5 x %str], ptr %a17, i64 0, i64 2
  store %str { ptr @.str.12, i64 2 }, ptr %t20
  %t21 = getelementptr [5 x %str], ptr %a17, i64 0, i64 3
  store %str %t15, ptr %t21
  %t22 = getelementptr [5 x %str], ptr %a17, i64 0, i64 4
  store %str { ptr @.str.13, i64 1 }, ptr %t22
  call void @veles_string_concat_n(ptr %a16, ptr %a17, i64 5)
  %t23 = load %str, ptr %a16
  ret %str %t23
}

define %str @v_main.Shape.describe_Self_main.Square_(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a6 = alloca %str
  %a8 = alloca %str
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = extractvalue %str { ptr @.str.14, i64 17 }, 0
  %t4 = extractvalue %str { ptr @.str.14, i64 17 }, 1
  call void @veles_call_push(ptr %t3)
  %t5 = call double @v_main.Shape.Square.area(ptr %t2)
  call void @veles_call_pop()
  call void @veles_f64_to_string(ptr %a6, double %t5)
  %t7 = load %str, ptr %a6
  %t9 = extractvalue %str { ptr @.str.15, i64 5 }, 0
  %t10 = extractvalue %str { ptr @.str.15, i64 5 }, 1
  %t11 = extractvalue %str %t7, 0
  %t12 = extractvalue %str %t7, 1
  call void @veles_string_concat(ptr %a8, ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  %t13 = load %str, ptr %a8
  ret %str %t13
}

define { i1, i64 } @v_main.largest__i64(ptr %p1) {
entry:
  %a1 = alloca ptr
  %a3 = alloca ptr
  store ptr %p1, ptr %a1
  %t2 = load ptr, ptr %a1
  store ptr %t2, ptr %a3
  %t4 = extractvalue %str { ptr @.str.16, i64 17 }, 0
  %t5 = extractvalue %str { ptr @.str.16, i64 17 }, 1
  call void @veles_call_push(ptr %t4)
  %t6 = call { i1, i64 } @v_std.prelude.extend.List_T.max_T_i64_(ptr %a3)
  call void @veles_call_pop()
  ret { i1, i64 } %t6
}

define { i1, %str } @v_main.largest__string(ptr %p1) {
entry:
  %a1 = alloca ptr
  %a3 = alloca ptr
  store ptr %p1, ptr %a1
  %t2 = load ptr, ptr %a1
  store ptr %t2, ptr %a3
  %t4 = extractvalue %str { ptr @.str.16, i64 17 }, 0
  %t5 = extractvalue %str { ptr @.str.16, i64 17 }, 1
  call void @veles_call_push(ptr %t4)
  %t6 = call { i1, %str } @v_std.prelude.extend.List_T.max_T_string_(ptr %a3)
  call void @veles_call_pop()
  ret { i1, %str } %t6
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
@.str.2 = private unnamed_addr constant [22 x i8] c"main.vs:33:22\00println\00"
@.str.3 = private unnamed_addr constant [22 x i8] c"main.vs:34:17\00largest\00"
@.str.4 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.5 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.6 = private unnamed_addr constant [22 x i8] c"main.vs:34:44\00largest\00"
@.str.7 = private unnamed_addr constant [1 x i8] c"\00"
@.str.8 = private unnamed_addr constant [2 x i8] c" \00"
@.str.9 = private unnamed_addr constant [21 x i8] c"main.vs:34:3\00println\00"
@.str.10 = private unnamed_addr constant [7 x i8] c"circle\00"
@.str.11 = private unnamed_addr constant [2 x i8] c"(\00"
@.str.12 = private unnamed_addr constant [3 x i8] c", \00"
@.str.13 = private unnamed_addr constant [2 x i8] c")\00"
@.str.14 = private unnamed_addr constant [18 x i8] c"main.vs:7:36\00area\00"
@.str.15 = private unnamed_addr constant [6 x i8] c"area \00"
@.str.16 = private unnamed_addr constant [18 x i8] c"main.vs:29:47\00max\00"
