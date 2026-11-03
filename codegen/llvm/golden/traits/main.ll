%S.main.Square = type { double }
%S.main.Circle = type { double }
%S.main.Point = type { i64, i64 }
@desc.main.Square = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 0, i64 0, [0 x i64] [] }
@desc.main.Circle = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 0, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a1 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a4 = alloca { ptr, ptr }
  %a13 = alloca ptr
  %a15 = alloca ptr
  %a21 = alloca i64
  %a24 = alloca i64
  %a25 = alloca i64
  %a41 = alloca { ptr, ptr }
  %a61 = alloca i64
  %a65 = alloca i64
  %a70 = alloca [21 x i8]
  %a75 = alloca %str
  %a79 = alloca %str
  %a87 = alloca %str
  %a88 = alloca [5 x %str]
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a1
  %t2 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a1, i32 0, i32 3
  store i64 16, ptr %t2
  %t3 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a1, i32 0, i32 4
  store ptr @adesc.Shape, ptr %t3
  call void @veles_list_reserve(ptr %a1, i64 2)
  %t5 = insertvalue %S.main.Square undef, double 0x4000000000000000, 0
  %t6 = call ptr @veles_gc_alloc(ptr @desc.main.Square, i64 8)
  store %S.main.Square %t5, ptr %t6
  %t7 = insertvalue { ptr, ptr } undef, ptr %t6, 0
  %t8 = insertvalue { ptr, ptr } %t7, ptr @vt.main.Shape.main.Square, 1
  store { ptr, ptr } %t8, ptr %a4
  call void @veles_list_push(ptr %a1, ptr %a4)
  %t9 = insertvalue %S.main.Circle undef, double 0x3FF0000000000000, 0
  %t10 = call ptr @veles_gc_alloc(ptr @desc.main.Circle, i64 8)
  store %S.main.Circle %t9, ptr %t10
  %t11 = insertvalue { ptr, ptr } undef, ptr %t10, 0
  %t12 = insertvalue { ptr, ptr } %t11, ptr @vt.main.Shape.main.Circle, 1
  store { ptr, ptr } %t12, ptr %a4
  call void @veles_list_push(ptr %a1, ptr %a4)
  store ptr %a1, ptr %a13
  %t14 = load ptr, ptr %a13
  store ptr %t14, ptr %a15
  %t16 = load ptr, ptr %a15
  %t17 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t16, i32 0, i32 6
  %t18 = load i64, ptr %t17
  %t19 = and i64 %t18, 12
  %t20 = icmp eq i64 %t19, 0
  store i64 0, ptr %a21
  br i1 %t20, label %lend.set.1, label %lend.done.2
lend.set.1:
  %t22 = or i64 %t18, 8
  store i64 %t22, ptr %t17
  store i64 1, ptr %a21
  br label %lend.done.2
lend.done.2:
  %t23 = load i64, ptr %a21
  store i64 %t23, ptr %a24
  store i64 0, ptr %a25
  br label %loop.cond.3
loop.cond.3:
  %t26 = load i64, ptr %a25
  %t27 = load ptr, ptr %a15
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t27, i32 0, i32 1
  %t29 = load i64, ptr %t28
  %t30 = icmp slt i64 %t26, %t29
  br i1 %t30, label %loop.body.6, label %loop.end.5
loop.body.6:
  %t31 = load ptr, ptr %a15
  %t32 = load i64, ptr %a25
  %t33 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t31, i32 0, i32 1
  %t34 = load i64, ptr %t33
  %t35 = icmp ult i64 %t32, %t34
  br i1 %t35, label %idx.ok.7, label %idx.bad.8, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.8:
  %t36 = extractvalue %str { ptr @.str.1, i64 12 }, 0
  %t37 = extractvalue %str { ptr @.str.1, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t31, i64 %t32, ptr %t36, i64 %t37)
  unreachable
idx.ok.7:
  %t38 = load ptr, ptr %t31
  %t39 = getelementptr inbounds [16 x i8], ptr %t38, i64 %t32
  %t40 = load { ptr, ptr }, ptr %t39
  store { ptr, ptr } %t40, ptr %a41
  %t42 = load { ptr, ptr }, ptr %a41
  %t43 = extractvalue { ptr, ptr } %t42, 0
  %t44 = extractvalue { ptr, ptr } %t42, 1
  %t45 = getelementptr ptr, ptr %t44, i64 2
  %t46 = load ptr, ptr %t45
  %t47 = call %str %t46(ptr %t43)
  %t48 = extractvalue %str { ptr @.str.2, i64 21 }, 0
  %t49 = extractvalue %str { ptr @.str.2, i64 21 }, 1
  call void @veles_call_push(ptr %t48)
  call void @v_std.io.println(%str %t47)
  call void @veles_call_pop()
  br label %loop.post.4
loop.post.4:
  %t50 = load i64, ptr %a25
  %t51 = add i64 %t50, 1
  store i64 %t51, ptr %a25
  %t52 = load volatile i32, ptr @veles_attention_line, align 64
  %t53 = icmp ne i32 %t52, 0
  br i1 %t53, label %safepoint.9, label %safepoint.on.10, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.9:
  call void @veles_backedge_plain()
  br label %safepoint.on.10
safepoint.on.10:
  br label %loop.cond.3
loop.end.5:
  %t54 = load ptr, ptr %a15
  %t55 = load i64, ptr %a24
  %t56 = icmp ne i64 %t55, 0
  br i1 %t56, label %unlend.clear.11, label %unlend.done.12
unlend.clear.11:
  %t57 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t54, i32 0, i32 6
  %t58 = load i64, ptr %t57
  %t59 = and i64 %t58, -9
  store i64 %t59, ptr %t57
  br label %unlend.done.12
unlend.done.12:
  %t60 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  store i64 9, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  store i64 4, ptr %a61
  call void @veles_list_push(ptr %t60, ptr %a61)
  %t62 = extractvalue %str { ptr @.str.3, i64 21 }, 0
  %t63 = extractvalue %str { ptr @.str.3, i64 21 }, 1
  call void @veles_call_push(ptr %t62)
  %t64 = call { i1, i64 } @v_main.largest__i64(ptr %t60)
  call void @veles_call_pop()
  %t67 = extractvalue { i1, i64 } %t64, 0
  %t66 = xor i1 %t67, true
  br i1 %t66, label %elvis.default.13, label %elvis.some.14
elvis.some.14:
  %t68 = extractvalue { i1, i64 } %t64, 1
  store i64 %t68, ptr %a65
  br label %elvis.end.15
elvis.default.13:
  store i64 0, ptr %a65
  br label %elvis.end.15
elvis.end.15:
  %t69 = load i64, ptr %a65
  %t71 = call i64 @veles_i64_format(ptr %a70, i64 %t69)
  %t72 = insertvalue %str undef, ptr %a70, 0
  %t73 = insertvalue %str %t72, i64 %t71, 1
  %t74 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.4, i64 1 }, ptr %a75
  call void @veles_list_push(ptr %t74, ptr %a75)
  store %str { ptr @.str.5, i64 1 }, ptr %a75
  call void @veles_list_push(ptr %t74, ptr %a75)
  %t76 = extractvalue %str { ptr @.str.6, i64 21 }, 0
  %t77 = extractvalue %str { ptr @.str.6, i64 21 }, 1
  call void @veles_call_push(ptr %t76)
  %t78 = call { i1, %str } @v_main.largest__string(ptr %t74)
  call void @veles_call_pop()
  %t81 = extractvalue { i1, %str } %t78, 0
  %t80 = xor i1 %t81, true
  br i1 %t80, label %elvis.default.16, label %elvis.some.17
elvis.some.17:
  %t82 = extractvalue { i1, %str } %t78, 1
  store %str %t82, ptr %a79
  br label %elvis.end.18
elvis.default.16:
  store %str { ptr @.str.7, i64 0 }, ptr %a79
  br label %elvis.end.18
elvis.end.18:
  %t83 = load %str, ptr %a79
  %t84 = insertvalue %S.main.Point undef, i64 1, 0
  %t85 = insertvalue %S.main.Point %t84, i64 2, 1
  %t86 = call %str @show.main.Point(%S.main.Point %t85)
  %t89 = getelementptr [5 x %str], ptr %a88, i64 0, i64 0
  store %str %t73, ptr %t89
  %t90 = getelementptr [5 x %str], ptr %a88, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t90
  %t91 = getelementptr [5 x %str], ptr %a88, i64 0, i64 2
  store %str %t83, ptr %t91
  %t92 = getelementptr [5 x %str], ptr %a88, i64 0, i64 3
  store %str { ptr @.str.8, i64 1 }, ptr %t92
  %t93 = getelementptr [5 x %str], ptr %a88, i64 0, i64 4
  store %str %t86, ptr %t93
  call void @veles_string_concat_n(ptr %a87, ptr %a88, i64 5)
  %t94 = load %str, ptr %a87
  %t95 = extractvalue %str { ptr @.str.9, i64 20 }, 0
  %t96 = extractvalue %str { ptr @.str.9, i64 20 }, 1
  call void @veles_call_push(ptr %t95)
  call void @v_std.io.println(%str %t94)
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

define %str @v_main.Shape.describe_This_main.Square_(ptr %p0) {
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
  call void @veles_string_concat_shared(ptr %a8, ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  %t13 = load %str, ptr %a8
  ret %str %t13
}

define { i1, i64 } @v_main.largest__i64(ptr %p1) {
entry:
  %a1 = alloca ptr
  store ptr %p1, ptr %a1
  %t2 = extractvalue %str { ptr @.str.16, i64 17 }, 0
  %t3 = extractvalue %str { ptr @.str.16, i64 17 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call { i1, i64 } @v_std.prelude.extend.List_T.max_T_i64_(ptr %a1)
  call void @veles_call_pop()
  ret { i1, i64 } %t4
}

define { i1, %str } @v_main.largest__string(ptr %p1) {
entry:
  %a1 = alloca ptr
  store ptr %p1, ptr %a1
  %t2 = extractvalue %str { ptr @.str.16, i64 17 }, 0
  %t3 = extractvalue %str { ptr @.str.16, i64 17 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call { i1, %str } @v_std.prelude.extend.List_T.max_T_string_(ptr %a1)
  call void @veles_call_pop()
  ret { i1, %str } %t4
}

define internal double @vt.main.Shape.main.Square.0(ptr %self) {
entry:
  %t1 = call double @v_main.Shape.Square.area(ptr %self)
  ret double %t1
}

define internal %str @vt.main.Shape.main.Square.1(ptr %self) {
entry:
  %t1 = call %str @v_main.Shape.describe_This_main.Square_(ptr %self)
  ret %str %t1
}

@vt.main.Shape.main.Square = internal constant [3 x ptr] [ptr @ti.main.Square, ptr @vt.main.Shape.main.Square.0, ptr @vt.main.Shape.main.Square.1]
define internal double @vt.main.Shape.main.Circle.0(ptr %self) {
entry:
  %t1 = call double @v_main.Shape.Circle.area(ptr %self)
  ret double %t1
}

define internal %str @vt.main.Shape.main.Circle.1(ptr %self) {
entry:
  %t1 = call %str @v_main.Shape.Circle.describe(ptr %self)
  ret %str %t1
}

@vt.main.Shape.main.Circle = internal constant [3 x ptr] [ptr @ti.main.Circle, ptr @vt.main.Shape.main.Circle.0, ptr @vt.main.Shape.main.Circle.1]
define internal %str @show.main.Point(%S.main.Point %v) {
entry:
  %a2 = alloca %S.main.Point
  store %S.main.Point %v, ptr %a2
  %t1 = call %str @v_main.Display.Point.toString(ptr %a2)
  ret %str %t1
}

@ti.main.Square = internal constant i64 0
@ti.main.Circle = internal constant i64 1

@.str.1 = private unnamed_addr constant [13 x i8] c"main.vs:37:3\00"
@.str.2 = private unnamed_addr constant [22 x i8] c"main.vs:37:22\00println\00"
@.str.3 = private unnamed_addr constant [22 x i8] c"main.vs:38:17\00largest\00"
@.str.4 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.5 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.6 = private unnamed_addr constant [22 x i8] c"main.vs:38:44\00largest\00"
@.str.7 = private unnamed_addr constant [1 x i8] c"\00"
@.str.8 = private unnamed_addr constant [2 x i8] c" \00"
@.str.9 = private unnamed_addr constant [21 x i8] c"main.vs:38:3\00println\00"
@.str.10 = private unnamed_addr constant [7 x i8] c"circle\00"
@.str.11 = private unnamed_addr constant [2 x i8] c"(\00"
@.str.12 = private unnamed_addr constant [3 x i8] c", \00"
@.str.13 = private unnamed_addr constant [2 x i8] c")\00"
@.str.14 = private unnamed_addr constant [18 x i8] c"main.vs:7:37\00area\00"
@.str.15 = private unnamed_addr constant [6 x i8] c"area \00"
@.str.16 = private unnamed_addr constant [18 x i8] c"main.vs:33:48\00max\00"
