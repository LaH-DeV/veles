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
  %a52 = alloca [21 x i8]
  %a57 = alloca %str
  %a59 = alloca %str
  %a67 = alloca %str
  %a68 = alloca [5 x %str]
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
  %t53 = call i64 @veles_i64_format(ptr %a52, i64 %t51)
  %t54 = insertvalue %str undef, ptr %a52, 0
  %t55 = insertvalue %str %t54, i64 %t53, 1
  %t56 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.2, i64 1 }, ptr %a57
  call void @veles_list_push(ptr %t56, ptr %a57)
  store %str { ptr @.str.3, i64 1 }, ptr %a57
  call void @veles_list_push(ptr %t56, ptr %a57)
  %t58 = call { i1, %str } @v_main.largest__string(ptr %t56)
  %t61 = extractvalue { i1, %str } %t58, 0
  %t60 = xor i1 %t61, true
  br i1 %t60, label %elvis.default.12, label %elvis.some.13
elvis.some.13:
  %t62 = extractvalue { i1, %str } %t58, 1
  store %str %t62, ptr %a59
  br label %elvis.end.14
elvis.default.12:
  store %str { ptr @.str.4, i64 0 }, ptr %a59
  br label %elvis.end.14
elvis.end.14:
  %t63 = load %str, ptr %a59
  %t64 = insertvalue %S.main.Point undef, i64 1, 0
  %t65 = insertvalue %S.main.Point %t64, i64 2, 1
  %t66 = call %str @show.main.Point(%S.main.Point %t65)
  %t69 = getelementptr [5 x %str], ptr %a68, i64 0, i64 0
  store %str %t55, ptr %t69
  %t70 = getelementptr [5 x %str], ptr %a68, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t70
  %t71 = getelementptr [5 x %str], ptr %a68, i64 0, i64 2
  store %str %t63, ptr %t71
  %t72 = getelementptr [5 x %str], ptr %a68, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t72
  %t73 = getelementptr [5 x %str], ptr %a68, i64 0, i64 4
  store %str %t66, ptr %t73
  call void @veles_string_concat_n(ptr %a67, ptr %a68, i64 5)
  %t74 = load %str, ptr %a67
  call void @v_std.io.println(%str %t74)
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
  store %str { ptr @.str.7, i64 1 }, ptr %t18
  %t19 = getelementptr [5 x %str], ptr %a17, i64 0, i64 1
  store %str %t8, ptr %t19
  %t20 = getelementptr [5 x %str], ptr %a17, i64 0, i64 2
  store %str { ptr @.str.8, i64 2 }, ptr %t20
  %t21 = getelementptr [5 x %str], ptr %a17, i64 0, i64 3
  store %str %t15, ptr %t21
  %t22 = getelementptr [5 x %str], ptr %a17, i64 0, i64 4
  store %str { ptr @.str.9, i64 1 }, ptr %t22
  call void @veles_string_concat_n(ptr %a16, ptr %a17, i64 5)
  %t23 = load %str, ptr %a16
  ret %str %t23
}

define %str @v_main.Shape.describe_Self_main.Square_(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a4 = alloca %str
  %a6 = alloca %str
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = call double @v_main.Shape.Square.area(ptr %t2)
  call void @veles_f64_to_string(ptr %a4, double %t3)
  %t5 = load %str, ptr %a4
  %t7 = extractvalue %str { ptr @.str.10, i64 5 }, 0
  %t8 = extractvalue %str { ptr @.str.10, i64 5 }, 1
  %t9 = extractvalue %str %t5, 0
  %t10 = extractvalue %str %t5, 1
  call void @veles_string_concat(ptr %a6, ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  %t11 = load %str, ptr %a6
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
@.str.2 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.3 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.4 = private unnamed_addr constant [1 x i8] c"\00"
@.str.5 = private unnamed_addr constant [2 x i8] c" \00"
@.str.6 = private unnamed_addr constant [7 x i8] c"circle\00"
@.str.7 = private unnamed_addr constant [2 x i8] c"(\00"
@.str.8 = private unnamed_addr constant [3 x i8] c", \00"
@.str.9 = private unnamed_addr constant [2 x i8] c")\00"
@.str.10 = private unnamed_addr constant [6 x i8] c"area \00"
