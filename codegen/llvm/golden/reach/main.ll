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
  %a30 = alloca { ptr, ptr }
  %a37 = alloca %str
  %a45 = alloca ptr
  %a52 = alloca [21 x i8]
  %a64 = alloca [21 x i8]
  %a68 = alloca %str
  %a69 = alloca [3 x %str]
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
  %t28 = getelementptr inbounds [16 x i8], ptr %t27, i64 %t21
  %t29 = load { ptr, ptr }, ptr %t28
  store { ptr, ptr } %t29, ptr %a30
  %t31 = load { ptr, ptr }, ptr %a30
  %t32 = extractvalue { ptr, ptr } %t31, 0
  %t33 = extractvalue { ptr, ptr } %t31, 1
  %t34 = getelementptr ptr, ptr %t33, i64 1
  %t35 = load ptr, ptr %t34
  %t36 = call double %t35(ptr %t32)
  call void @veles_f64_to_string(ptr %a37, double %t36)
  %t38 = load %str, ptr %a37
  %t39 = extractvalue %str { ptr @.str.4, i64 20 }, 0
  %t40 = extractvalue %str { ptr @.str.4, i64 20 }, 1
  call void @veles_call_push(ptr %t39)
  call void @v_std.io.println(%str %t38)
  call void @veles_call_pop()
  br label %loop.post.2
loop.post.2:
  %t41 = load i64, ptr %a14
  %t42 = add i64 %t41, 1
  store i64 %t42, ptr %a14
  %t43 = load volatile i32, ptr @veles_attention_line, align 64
  %t44 = icmp ne i32 %t43, 0
  br i1 %t44, label %safepoint.7, label %safepoint.on.8, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.7:
  call void @veles_backedge_plain()
  br label %safepoint.on.8
safepoint.on.8:
  br label %loop.cond.1
loop.end.3:
  %t46 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %t46, ptr %a45
  store i64 5, ptr %t46
  %t47 = insertvalue { ptr, ptr } undef, ptr @thunk.v_main.twice, 0
  %t48 = insertvalue { ptr, ptr } %t47, ptr null, 1
  %t49 = extractvalue %str { ptr @.str.5, i64 19 }, 0
  %t50 = extractvalue %str { ptr @.str.5, i64 19 }, 1
  call void @veles_call_push(ptr %t49)
  %t51 = call i64 @v_main.apply({ ptr, ptr } %t48, i64 10)
  call void @veles_call_pop()
  %t53 = call i64 @veles_i64_format(ptr %a52, i64 %t51)
  %t54 = insertvalue %str undef, ptr %a52, 0
  %t55 = insertvalue %str %t54, i64 %t53, 1
  %t56 = call ptr @veles_alloc_words(i64 8)
  %t57 = load ptr, ptr %a45
  %t58 = getelementptr ptr, ptr %t56, i64 0
  store ptr %t57, ptr %t58
  %t59 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t60 = insertvalue { ptr, ptr } %t59, ptr %t56, 1
  %t61 = extractvalue %str { ptr @.str.6, i64 19 }, 0
  %t62 = extractvalue %str { ptr @.str.6, i64 19 }, 1
  call void @veles_call_push(ptr %t61)
  %t63 = call i64 @v_main.apply({ ptr, ptr } %t60, i64 10)
  call void @veles_call_pop()
  %t65 = call i64 @veles_i64_format(ptr %a64, i64 %t63)
  %t66 = insertvalue %str undef, ptr %a64, 0
  %t67 = insertvalue %str %t66, i64 %t65, 1
  %t70 = getelementptr [3 x %str], ptr %a69, i64 0, i64 0
  store %str %t55, ptr %t70
  %t71 = getelementptr [3 x %str], ptr %a69, i64 0, i64 1
  store %str { ptr @.str.7, i64 1 }, ptr %t71
  %t72 = getelementptr [3 x %str], ptr %a69, i64 0, i64 2
  store %str %t67, ptr %t72
  call void @veles_string_concat_n(ptr %a68, ptr %a69, i64 3)
  %t73 = load %str, ptr %a68
  %t74 = extractvalue %str { ptr @.str.8, i64 20 }, 0
  %t75 = extractvalue %str { ptr @.str.8, i64 20 }, 1
  call void @veles_call_push(ptr %t74)
  call void @v_std.io.println(%str %t73)
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
