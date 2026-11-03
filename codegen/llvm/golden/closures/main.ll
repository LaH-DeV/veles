define { ptr, ptr } @v_main.adder(i64 %p1) {
entry:
  %a1 = alloca ptr
  %cell1 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %cell1, ptr %a1
  store i64 %p1, ptr %cell1
  %t2 = call ptr @veles_alloc_words(i64 8)
  %t3 = load ptr, ptr %a1
  %t4 = getelementptr ptr, ptr %t2, i64 0
  store ptr %t3, ptr %t4
  %t5 = insertvalue { ptr, ptr } undef, ptr @v_main.adder.lambda1, 0
  %t6 = insertvalue { ptr, ptr } %t5, ptr %t2, 1
  ret { ptr, ptr } %t6
}

define i64 @v_main.adder.lambda1(ptr %env, i64 %p1) {
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
  %t14 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t15 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t12, i64 %t13, ptr %t14, i64 %t15)
  unreachable
arith.ok.2:
  ret i64 %t10
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
  %a1 = alloca ptr
  %a8 = alloca { ptr, ptr }
  %a18 = alloca { ptr, ptr }
  %a20 = alloca i64
  %a21 = alloca ptr
  %a27 = alloca ptr
  %a33 = alloca i64
  %a36 = alloca [21 x i8]
  %a44 = alloca [21 x i8]
  %a51 = alloca [21 x i8]
  %a55 = alloca %str
  %a56 = alloca [7 x %str]
  %t2 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %t2, ptr %a1
  store i64 0, ptr %t2
  %t3 = call ptr @veles_alloc_words(i64 8)
  %t4 = load ptr, ptr %a1
  %t5 = getelementptr ptr, ptr %t3, i64 0
  store ptr %t4, ptr %t5
  %t6 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2, 0
  %t7 = insertvalue { ptr, ptr } %t6, ptr %t3, 1
  store { ptr, ptr } %t7, ptr %a8
  %t9 = load { ptr, ptr }, ptr %a8
  %t10 = extractvalue { ptr, ptr } %t9, 0
  %t11 = extractvalue { ptr, ptr } %t9, 1
  call void %t10(ptr %t11)
  %t12 = load { ptr, ptr }, ptr %a8
  %t13 = extractvalue { ptr, ptr } %t12, 0
  %t14 = extractvalue { ptr, ptr } %t12, 1
  call void %t13(ptr %t14)
  %t15 = extractvalue %str { ptr @.str.3, i64 19 }, 0
  %t16 = extractvalue %str { ptr @.str.3, i64 19 }, 1
  call void @veles_call_push(ptr %t15)
  %t17 = call { ptr, ptr } @v_main.adder(i64 5)
  call void @veles_call_pop()
  store { ptr, ptr } %t17, ptr %a18
  %t19 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 1, ptr %a20
  call void @veles_list_push(ptr %t19, ptr %a20)
  store i64 2, ptr %a20
  call void @veles_list_push(ptr %t19, ptr %a20)
  store i64 3, ptr %a20
  call void @veles_list_push(ptr %t19, ptr %a20)
  store ptr %t19, ptr %a21
  %t22 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda3.plain, 0
  %t23 = insertvalue { ptr, ptr } %t22, ptr null, 1
  %t24 = extractvalue %str { ptr @.str.4, i64 17 }, 0
  %t25 = extractvalue %str { ptr @.str.4, i64 17 }, 1
  call void @veles_call_push(ptr %t24)
  %t26 = call ptr @v_std.prelude.extend.List_T.map_T_i64_i64_Never.plain(ptr %a21, { ptr, ptr } %t23)
  call void @veles_call_pop()
  store ptr %t26, ptr %a27
  %t28 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4.plain, 0
  %t29 = insertvalue { ptr, ptr } %t28, ptr null, 1
  %t30 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t31 = extractvalue %str { ptr @.str.5, i64 18 }, 1
  call void @veles_call_push(ptr %t30)
  %t32 = call i64 @v_std.prelude.extend.List_T.fold_T_i64_i64_Never.plain(ptr %a27, i64 0, { ptr, ptr } %t29)
  call void @veles_call_pop()
  store i64 %t32, ptr %a33
  %t34 = load ptr, ptr %a1
  %t35 = load i64, ptr %t34
  %t37 = call i64 @veles_i64_format(ptr %a36, i64 %t35)
  %t38 = insertvalue %str undef, ptr %a36, 0
  %t39 = insertvalue %str %t38, i64 %t37, 1
  %t40 = load { ptr, ptr }, ptr %a18
  %t41 = extractvalue %str { ptr @.str.6, i64 19 }, 0
  %t42 = extractvalue %str { ptr @.str.6, i64 19 }, 1
  call void @veles_call_push(ptr %t41)
  %t43 = call i64 @v_main.apply({ ptr, ptr } %t40, i64 10)
  call void @veles_call_pop()
  %t45 = call i64 @veles_i64_format(ptr %a44, i64 %t43)
  %t46 = insertvalue %str undef, ptr %a44, 0
  %t47 = insertvalue %str %t46, i64 %t45, 1
  %t48 = load ptr, ptr %a27
  %t49 = call %str @show.List_i64_(ptr %t48)
  %t50 = load i64, ptr %a33
  %t52 = call i64 @veles_i64_format(ptr %a51, i64 %t50)
  %t53 = insertvalue %str undef, ptr %a51, 0
  %t54 = insertvalue %str %t53, i64 %t52, 1
  %t57 = getelementptr [7 x %str], ptr %a56, i64 0, i64 0
  store %str %t39, ptr %t57
  %t58 = getelementptr [7 x %str], ptr %a56, i64 0, i64 1
  store %str { ptr @.str.7, i64 1 }, ptr %t58
  %t59 = getelementptr [7 x %str], ptr %a56, i64 0, i64 2
  store %str %t47, ptr %t59
  %t60 = getelementptr [7 x %str], ptr %a56, i64 0, i64 3
  store %str { ptr @.str.7, i64 1 }, ptr %t60
  %t61 = getelementptr [7 x %str], ptr %a56, i64 0, i64 4
  store %str %t49, ptr %t61
  %t62 = getelementptr [7 x %str], ptr %a56, i64 0, i64 5
  store %str { ptr @.str.7, i64 1 }, ptr %t62
  %t63 = getelementptr [7 x %str], ptr %a56, i64 0, i64 6
  store %str %t54, ptr %t63
  call void @veles_string_concat_n(ptr %a55, ptr %a56, i64 7)
  %t64 = load %str, ptr %a55
  %t65 = extractvalue %str { ptr @.str.8, i64 20 }, 0
  %t66 = extractvalue %str { ptr @.str.8, i64 20 }, 1
  call void @veles_call_push(ptr %t65)
  call void @v_std.io.println(%str %t64)
  call void @veles_call_pop()
  ret void
}

define void @v_main.main.lambda2(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr ptr, ptr %t2, i64 0
  %t4 = load ptr, ptr %t3
  %t5 = load i64, ptr %t4
  %t7 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t5, i64 1)
  %t8 = extractvalue { i64, i1 } %t7, 0
  %t9 = extractvalue { i64, i1 } %t7, 1
  br i1 %t9, label %overflow.1, label %arith.ok.2
overflow.1:
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.9, i64 12 }, 0
  %t13 = extractvalue %str { ptr @.str.9, i64 12 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
arith.ok.2:
  %t14 = load ptr, ptr %a1
  %t15 = getelementptr ptr, ptr %t14, i64 0
  %t16 = load ptr, ptr %t15
  store i64 %t8, ptr %t16
  ret void
}

define i64 @v_main.main.lambda3.plain(ptr %env, i64 %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  %t3 = load i64, ptr %a2
  %t5 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t3, i64 2)
  %t6 = extractvalue { i64, i1 } %t5, 0
  %t7 = extractvalue { i64, i1 } %t5, 1
  br i1 %t7, label %overflow.1, label %arith.ok.2
overflow.1:
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t10 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t11 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_panic_at(ptr %t8, i64 %t9, ptr %t10, i64 %t11)
  unreachable
arith.ok.2:
  ret i64 %t6
}

define i64 @v_main.main.lambda4.plain(ptr %env, i64 %p1, i64 %p2) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a3 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  store i64 %p2, ptr %a3
  %t4 = load i64, ptr %a2
  %t5 = load i64, ptr %a3
  %t7 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t4, i64 %t5)
  %t8 = extractvalue { i64, i1 } %t7, 0
  %t9 = extractvalue { i64, i1 } %t7, 1
  br i1 %t9, label %overflow.1, label %arith.ok.2
overflow.1:
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t13 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
arith.ok.2:
  ret i64 %t8
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:4:43\00"
@.str.3 = private unnamed_addr constant [20 x i8] c"main.vs:15:14\00adder\00"
@.str.4 = private unnamed_addr constant [18 x i8] c"main.vs:16:17\00map\00"
@.str.5 = private unnamed_addr constant [19 x i8] c"main.vs:17:15\00fold\00"
@.str.6 = private unnamed_addr constant [20 x i8] c"main.vs:18:24\00apply\00"
@.str.7 = private unnamed_addr constant [2 x i8] c" \00"
@.str.8 = private unnamed_addr constant [21 x i8] c"main.vs:18:3\00println\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:11:5\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:16:36\00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:17:41\00"
