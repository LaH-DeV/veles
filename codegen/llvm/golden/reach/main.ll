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
  %a2 = alloca { ptr, ptr }
  %a11 = alloca ptr
  %a13 = alloca ptr
  %a14 = alloca i64
  %a33 = alloca { ptr, ptr }
  %a40 = alloca %str
  %a48 = alloca ptr
  %a55 = alloca [21 x i8]
  %a67 = alloca [21 x i8]
  %a71 = alloca %str
  %a72 = alloca [3 x %str]
  %t1 = call ptr @veles_list_new(ptr @adesc.Shape, i64 2)
  %t3 = insertvalue %S.main.Circle undef, double 0x3FF0000000000000, 0
  %t4 = call ptr @veles_gc_alloc(ptr @desc.main.Circle, i64 8)
  store %S.main.Circle %t3, ptr %t4
  %t5 = insertvalue { ptr, ptr } undef, ptr %t4, 0
  %t6 = insertvalue { ptr, ptr } %t5, ptr @vt.main.Shape.main.Circle, 1
  store { ptr, ptr } %t6, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  %t7 = insertvalue %S.main.Square undef, double 0x4000000000000000, 0
  %t8 = call ptr @veles_gc_alloc(ptr @desc.main.Square, i64 8)
  store %S.main.Square %t7, ptr %t8
  %t9 = insertvalue { ptr, ptr } undef, ptr %t8, 0
  %t10 = insertvalue { ptr, ptr } %t9, ptr @vt.main.Shape.main.Square, 1
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
  %t17 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t16, i32 0, i32 1
  %t18 = load i64, ptr %t17
  %t19 = icmp slt i64 %t15, %t18
  br i1 %t19, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t20 = load ptr, ptr %a13
  %t21 = load i64, ptr %a14
  %t22 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t20, i32 0, i32 1
  %t23 = load i64, ptr %t22
  %t24 = icmp ult i64 %t21, %t23
  br i1 %t24, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t25 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t26 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t20, i64 %t21, ptr %t25, i64 %t26)
  unreachable
idx.ok.5:
  %t27 = load ptr, ptr %t20
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t20, i32 0, i32 3
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
  %t39 = call double %t38(ptr %t35)
  call void @veles_f64_to_string(ptr %a40, double %t39)
  %t41 = load %str, ptr %a40
  %t42 = extractvalue %str { ptr @.str.4, i64 20 }, 0
  %t43 = extractvalue %str { ptr @.str.4, i64 20 }, 1
  call void @veles_call_push(ptr %t42)
  call void @v_std.io.println(%str %t41)
  call void @veles_call_pop()
  br label %loop.post.2
loop.post.2:
  %t44 = load i64, ptr %a14
  %t45 = add i64 %t44, 1
  store i64 %t45, ptr %a14
  %t46 = load volatile i32, ptr @veles_attention_line, align 64
  %t47 = icmp ne i32 %t46, 0
  br i1 %t47, label %safepoint.7, label %safepoint.on.8, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.7:
  call void @veles_backedge_plain()
  br label %safepoint.on.8
safepoint.on.8:
  br label %loop.cond.1
loop.end.3:
  %t49 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %t49, ptr %a48
  store i64 5, ptr %t49
  %t50 = insertvalue { ptr, ptr } undef, ptr @thunk.v_main.twice, 0
  %t51 = insertvalue { ptr, ptr } %t50, ptr null, 1
  %t52 = extractvalue %str { ptr @.str.5, i64 19 }, 0
  %t53 = extractvalue %str { ptr @.str.5, i64 19 }, 1
  call void @veles_call_push(ptr %t52)
  %t54 = call i64 @v_main.apply({ ptr, ptr } %t51, i64 10)
  call void @veles_call_pop()
  %t56 = call i64 @veles_i64_format(ptr %a55, i64 %t54)
  %t57 = insertvalue %str undef, ptr %a55, 0
  %t58 = insertvalue %str %t57, i64 %t56, 1
  %t59 = call ptr @veles_alloc_words(i64 8)
  %t60 = load ptr, ptr %a48
  %t61 = getelementptr ptr, ptr %t59, i64 0
  store ptr %t60, ptr %t61
  %t62 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t63 = insertvalue { ptr, ptr } %t62, ptr %t59, 1
  %t64 = extractvalue %str { ptr @.str.6, i64 19 }, 0
  %t65 = extractvalue %str { ptr @.str.6, i64 19 }, 1
  call void @veles_call_push(ptr %t64)
  %t66 = call i64 @v_main.apply({ ptr, ptr } %t63, i64 10)
  call void @veles_call_pop()
  %t68 = call i64 @veles_i64_format(ptr %a67, i64 %t66)
  %t69 = insertvalue %str undef, ptr %a67, 0
  %t70 = insertvalue %str %t69, i64 %t68, 1
  %t73 = getelementptr [3 x %str], ptr %a72, i64 0, i64 0
  store %str %t58, ptr %t73
  %t74 = getelementptr [3 x %str], ptr %a72, i64 0, i64 1
  store %str { ptr @.str.7, i64 1 }, ptr %t74
  %t75 = getelementptr [3 x %str], ptr %a72, i64 0, i64 2
  store %str %t70, ptr %t75
  call void @veles_string_concat_n(ptr %a71, ptr %a72, i64 3)
  %t76 = load %str, ptr %a71
  %t77 = extractvalue %str { ptr @.str.8, i64 20 }, 0
  %t78 = extractvalue %str { ptr @.str.8, i64 20 }, 1
  call void @veles_call_push(ptr %t77)
  call void @v_std.io.println(%str %t76)
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
