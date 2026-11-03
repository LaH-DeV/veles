%S.main.Square = type { double }
%S.main.Circle = type { double }
@desc.main.Circle = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 0, i64 0, [0 x i64] [] }
@desc.main.Square = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 0, i64 0, [0 x i64] [] }
define i64 @v_main.twice(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t4 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t2, i64 2)
  %t5 = extractvalue { i64, i1 } %t4, 0
  %t6 = extractvalue { i64, i1 } %t4, 1
  br i1 %t6, label %overflow.1, label %arith.ok.2
overflow.1:
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t10 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
arith.ok.2:
  ret i64 %t5
}

define i64 @v_main.apply({ ptr, ptr } %p1, i64 %p2) {
entry:
  %a1 = alloca { ptr, ptr }
  %a2 = alloca i64
  store { ptr, ptr } %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load { ptr, ptr }, ptr %a1
  %t4 = extractvalue { ptr, ptr } %t3, 0
  %t5 = extractvalue { ptr, ptr } %t3, 1
  %t6 = load i64, ptr %a2
  %t7 = call i64 %t4(ptr %t5, i64 %t6)
  ret i64 %t7
}

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
  %a48 = alloca %str
  %a62 = alloca ptr
  %a69 = alloca [21 x i8]
  %a81 = alloca [21 x i8]
  %a85 = alloca %str
  %a86 = alloca [3 x %str]
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a1
  %t2 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a1, i32 0, i32 3
  store i64 16, ptr %t2
  %t3 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a1, i32 0, i32 4
  store ptr @adesc.Shape, ptr %t3
  call void @veles_list_reserve(ptr %a1, i64 2)
  %t5 = insertvalue %S.main.Circle undef, double 0x3FF0000000000000, 0
  %t6 = call ptr @veles_gc_alloc(ptr @desc.main.Circle, i64 8)
  store %S.main.Circle %t5, ptr %t6
  %t7 = insertvalue { ptr, ptr } undef, ptr %t6, 0
  %t8 = insertvalue { ptr, ptr } %t7, ptr @vt.main.Shape.main.Circle, 1
  store { ptr, ptr } %t8, ptr %a4
  call void @veles_list_push(ptr %a1, ptr %a4)
  %t9 = insertvalue %S.main.Square undef, double 0x4000000000000000, 0
  %t10 = call ptr @veles_gc_alloc(ptr @desc.main.Square, i64 8)
  store %S.main.Square %t9, ptr %t10
  %t11 = insertvalue { ptr, ptr } undef, ptr %t10, 0
  %t12 = insertvalue { ptr, ptr } %t11, ptr @vt.main.Shape.main.Square, 1
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
  %t36 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t37 = extractvalue %str { ptr @.str.3, i64 12 }, 1
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
  %t45 = getelementptr ptr, ptr %t44, i64 1
  %t46 = load ptr, ptr %t45
  %t47 = call double %t46(ptr %t43)
  call void @veles_f64_to_string(ptr %a48, double %t47)
  %t49 = load %str, ptr %a48
  %t50 = extractvalue %str { ptr @.str.4, i64 20 }, 0
  %t51 = extractvalue %str { ptr @.str.4, i64 20 }, 1
  call void @veles_call_push(ptr %t50)
  call void @v_std.io.println(%str %t49)
  call void @veles_call_pop()
  br label %loop.post.4
loop.post.4:
  %t52 = load i64, ptr %a25
  %t53 = add i64 %t52, 1
  store i64 %t53, ptr %a25
  %t54 = load volatile i32, ptr @veles_attention_line, align 64
  %t55 = icmp ne i32 %t54, 0
  br i1 %t55, label %safepoint.9, label %safepoint.on.10, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.9:
  call void @veles_backedge_plain()
  br label %safepoint.on.10
safepoint.on.10:
  br label %loop.cond.3
loop.end.5:
  %t56 = load ptr, ptr %a15
  %t57 = load i64, ptr %a24
  %t58 = icmp ne i64 %t57, 0
  br i1 %t58, label %unlend.clear.11, label %unlend.done.12
unlend.clear.11:
  %t59 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t56, i32 0, i32 6
  %t60 = load i64, ptr %t59
  %t61 = and i64 %t60, -9
  store i64 %t61, ptr %t59
  br label %unlend.done.12
unlend.done.12:
  %t63 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %t63, ptr %a62
  store i64 5, ptr %t63
  %t64 = insertvalue { ptr, ptr } undef, ptr @thunk.v_main.twice, 0
  %t65 = insertvalue { ptr, ptr } %t64, ptr null, 1
  %t66 = extractvalue %str { ptr @.str.5, i64 19 }, 0
  %t67 = extractvalue %str { ptr @.str.5, i64 19 }, 1
  call void @veles_call_push(ptr %t66)
  %t68 = call i64 @v_main.apply({ ptr, ptr } %t65, i64 10)
  call void @veles_call_pop()
  %t70 = call i64 @veles_i64_format(ptr %a69, i64 %t68)
  %t71 = insertvalue %str undef, ptr %a69, 0
  %t72 = insertvalue %str %t71, i64 %t70, 1
  %t73 = call ptr @veles_alloc_words(i64 8)
  %t74 = load ptr, ptr %a62
  %t75 = getelementptr ptr, ptr %t73, i64 0
  store ptr %t74, ptr %t75
  %t76 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t77 = insertvalue { ptr, ptr } %t76, ptr %t73, 1
  %t78 = extractvalue %str { ptr @.str.6, i64 19 }, 0
  %t79 = extractvalue %str { ptr @.str.6, i64 19 }, 1
  call void @veles_call_push(ptr %t78)
  %t80 = call i64 @v_main.apply({ ptr, ptr } %t77, i64 10)
  call void @veles_call_pop()
  %t82 = call i64 @veles_i64_format(ptr %a81, i64 %t80)
  %t83 = insertvalue %str undef, ptr %a81, 0
  %t84 = insertvalue %str %t83, i64 %t82, 1
  %t87 = getelementptr [3 x %str], ptr %a86, i64 0, i64 0
  store %str %t72, ptr %t87
  %t88 = getelementptr [3 x %str], ptr %a86, i64 0, i64 1
  store %str { ptr @.str.7, i64 1 }, ptr %t88
  %t89 = getelementptr [3 x %str], ptr %a86, i64 0, i64 2
  store %str %t84, ptr %t89
  call void @veles_string_concat_n(ptr %a85, ptr %a86, i64 3)
  %t90 = load %str, ptr %a85
  %t91 = extractvalue %str { ptr @.str.8, i64 20 }, 0
  %t92 = extractvalue %str { ptr @.str.8, i64 20 }, 1
  call void @veles_call_push(ptr %t91)
  call void @v_std.io.println(%str %t90)
  call void @veles_call_pop()
  ret void
}

define i64 @v_main.main.lambda1(ptr %env, i64 %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  %t3 = load i64, ptr %a2
  %t4 = load ptr, ptr %a1
  %t5 = getelementptr ptr, ptr %t4, i64 0
  %t6 = load ptr, ptr %t5
  %t7 = load i64, ptr %t6
  %t9 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t3, i64 %t7)
  %t10 = extractvalue { i64, i1 } %t9, 0
  %t11 = extractvalue { i64, i1 } %t9, 1
  br i1 %t11, label %overflow.1, label %arith.ok.2
overflow.1:
  %t12 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t13 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t14 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t15 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_panic_at(ptr %t12, i64 %t13, ptr %t14, i64 %t15)
  unreachable
arith.ok.2:
  ret i64 %t10
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

define internal double @vt.main.Shape.main.Circle.0(ptr %self) {
entry:
  %t1 = call double @v_main.Shape.Circle.area(ptr %self)
  ret double %t1
}

@vt.main.Shape.main.Circle = internal constant [2 x ptr] [ptr @ti.main.Circle, ptr @vt.main.Shape.main.Circle.0]
define internal double @vt.main.Shape.main.Square.0(ptr %self) {
entry:
  %t1 = call double @v_main.Shape.Square.area(ptr %self)
  ret double %t1
}

@vt.main.Shape.main.Square = internal constant [2 x ptr] [ptr @ti.main.Square, ptr @vt.main.Shape.main.Square.0]
define internal i64 @thunk.v_main.twice(ptr %env, i64 %p0) {
entry:
  %t1 = call i64 @v_main.twice(i64 %p0)
  ret i64 %t1
}

@ti.main.Circle = internal constant i64 0
@ti.main.Square = internal constant i64 1

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:30:27\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:36:3\00"
@.str.4 = private unnamed_addr constant [21 x i8] c"main.vs:37:5\00println\00"
@.str.5 = private unnamed_addr constant [20 x i8] c"main.vs:40:17\00apply\00"
@.str.6 = private unnamed_addr constant [20 x i8] c"main.vs:40:37\00apply\00"
@.str.7 = private unnamed_addr constant [2 x i8] c" \00"
@.str.8 = private unnamed_addr constant [21 x i8] c"main.vs:40:3\00println\00"
@.str.9 = private unnamed_addr constant [14 x i8] c"main.vs:40:48\00"
