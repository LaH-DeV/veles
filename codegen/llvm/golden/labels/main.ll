define { i64, i64 } @v_main.divmod(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t6 = icmp eq i64 %t4, 0
  br i1 %t6, label %divzero.1, label %div.ok.2
divzero.1:
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t10 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
div.ok.2:
  %t11 = icmp eq i64 %t3, -9223372036854775808
  %t12 = icmp eq i64 %t4, -1
  %t13 = and i1 %t11, %t12
  br i1 %t13, label %divof.3, label %div.ok.4
divof.3:
  %t14 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t15 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t16 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t17 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t14, i64 %t15, ptr %t16, i64 %t17)
  unreachable
div.ok.4:
  %t5 = sdiv i64 %t3, %t4
  %t18 = insertvalue { i64, i64 } undef, i64 %t5, 0
  %t19 = load i64, ptr %a1
  %t20 = load i64, ptr %a2
  %t22 = icmp eq i64 %t20, 0
  br i1 %t22, label %divzero.5, label %div.ok.6
divzero.5:
  %t23 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t24 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t25 = extractvalue %str { ptr @.str.4, i64 12 }, 0
  %t26 = extractvalue %str { ptr @.str.4, i64 12 }, 1
  call void @veles_panic_at(ptr %t23, i64 %t24, ptr %t25, i64 %t26)
  unreachable
div.ok.6:
  %t27 = icmp eq i64 %t19, -9223372036854775808
  %t28 = icmp eq i64 %t20, -1
  %t29 = and i1 %t27, %t28
  br i1 %t29, label %divof.7, label %div.ok.8
divof.7:
  %t30 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t31 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t32 = extractvalue %str { ptr @.str.4, i64 12 }, 0
  %t33 = extractvalue %str { ptr @.str.4, i64 12 }, 1
  call void @veles_panic_at(ptr %t30, i64 %t31, ptr %t32, i64 %t33)
  unreachable
div.ok.8:
  %t21 = srem i64 %t19, %t20
  %t34 = insertvalue { i64, i64 } %t18, i64 %t21, 1
  ret { i64, i64 } %t34
}

define { i64, i64 } @v_main.divmodLabelled(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t6 = icmp eq i64 %t4, 0
  br i1 %t6, label %divzero.1, label %div.ok.2
divzero.1:
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.5, i64 12 }, 0
  %t10 = extractvalue %str { ptr @.str.5, i64 12 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
div.ok.2:
  %t11 = icmp eq i64 %t3, -9223372036854775808
  %t12 = icmp eq i64 %t4, -1
  %t13 = and i1 %t11, %t12
  br i1 %t13, label %divof.3, label %div.ok.4
divof.3:
  %t14 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t15 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t16 = extractvalue %str { ptr @.str.5, i64 12 }, 0
  %t17 = extractvalue %str { ptr @.str.5, i64 12 }, 1
  call void @veles_panic_at(ptr %t14, i64 %t15, ptr %t16, i64 %t17)
  unreachable
div.ok.4:
  %t5 = sdiv i64 %t3, %t4
  %t18 = insertvalue { i64, i64 } undef, i64 %t5, 0
  %t19 = load i64, ptr %a1
  %t20 = load i64, ptr %a2
  %t22 = icmp eq i64 %t20, 0
  br i1 %t22, label %divzero.5, label %div.ok.6
divzero.5:
  %t23 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t24 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t25 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t26 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t23, i64 %t24, ptr %t25, i64 %t26)
  unreachable
div.ok.6:
  %t27 = icmp eq i64 %t19, -9223372036854775808
  %t28 = icmp eq i64 %t20, -1
  %t29 = and i1 %t27, %t28
  br i1 %t29, label %divof.7, label %div.ok.8
divof.7:
  %t30 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t31 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t32 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t33 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t30, i64 %t31, ptr %t32, i64 %t33)
  unreachable
div.ok.8:
  %t21 = srem i64 %t19, %t20
  %t34 = insertvalue { i64, i64 } %t18, i64 %t21, 1
  ret { i64, i64 } %t34
}

define void @v_main.main() {
entry:
  %a4 = alloca { i64, i64 }
  %a8 = alloca { i64, i64 }
  %a13 = alloca { i64, i64 }
  %a30 = alloca [21 x i8]
  %a45 = alloca %str
  %a47 = alloca %str
  %a48 = alloca [11 x %str]
  %t1 = extractvalue %str { ptr @.str.7, i64 20 }, 0
  %t2 = extractvalue %str { ptr @.str.7, i64 20 }, 1
  call void @veles_call_push(ptr %t1)
  %t3 = call { i64, i64 } @v_main.divmod(i64 7, i64 2)
  call void @veles_call_pop()
  store { i64, i64 } %t3, ptr %a4
  %t5 = extractvalue %str { ptr @.str.8, i64 28 }, 0
  %t6 = extractvalue %str { ptr @.str.8, i64 28 }, 1
  call void @veles_call_push(ptr %t5)
  %t7 = call { i64, i64 } @v_main.divmodLabelled(i64 7, i64 2)
  call void @veles_call_pop()
  store { i64, i64 } %t7, ptr %a8
  %t9 = load { i64, i64 }, ptr %a8
  %t10 = extractvalue %str { ptr @.str.9, i64 18 }, 0
  %t11 = extractvalue %str { ptr @.str.9, i64 18 }, 1
  call void @veles_call_push(ptr %t10)
  %t12 = call { i64, i64 } @v_main.swap__i64_i64({ i64, i64 } %t9)
  call void @veles_call_pop()
  store { i64, i64 } %t12, ptr %a13
  %t14 = load { i64, i64 }, ptr %a4
  %t15 = call %str @show.T_i64_i64_({ i64, i64 } %t14)
  %t16 = load { i64, i64 }, ptr %a8
  %t17 = call %str @show.T_i64_i64_.labelled1({ i64, i64 } %t16)
  %t18 = load { i64, i64 }, ptr %a8
  %t19 = extractvalue { i64, i64 } %t18, 0
  %t20 = load { i64, i64 }, ptr %a4
  %t21 = extractvalue { i64, i64 } %t20, 1
  %t23 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t19, i64 %t21)
  %t24 = extractvalue { i64, i1 } %t23, 0
  %t25 = extractvalue { i64, i1 } %t23, 1
  br i1 %t25, label %overflow.1, label %arith.ok.2
overflow.1:
  %t26 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t27 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t28 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t29 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_panic_at(ptr %t26, i64 %t27, ptr %t28, i64 %t29)
  unreachable
arith.ok.2:
  %t31 = call i64 @veles_i64_format(ptr %a30, i64 %t24)
  %t32 = insertvalue %str undef, ptr %a30, 0
  %t33 = insertvalue %str %t32, i64 %t31, 1
  %t34 = load { i64, i64 }, ptr %a4
  %t35 = extractvalue %str { ptr @.str.11, i64 18 }, 0
  %t36 = extractvalue %str { ptr @.str.11, i64 18 }, 1
  call void @veles_call_push(ptr %t35)
  %t37 = call { i64, i64 } @v_main.swap__i64_i64({ i64, i64 } %t34)
  call void @veles_call_pop()
  %t38 = call %str @show.T_i64_i64_({ i64, i64 } %t37)
  %t39 = load { i64, i64 }, ptr %a13
  %t40 = call %str @show.T_i64_i64_({ i64, i64 } %t39)
  %t41 = load { i64, i64 }, ptr %a4
  %t42 = load { i64, i64 }, ptr %a8
  %t44 = call i1 @eq.T_i64_i64_({ i64, i64 } %t41, { i64, i64 } %t42)
  call void @veles_bool_to_string(ptr %a45, i1 zeroext %t44)
  %t46 = load %str, ptr %a45
  %t49 = getelementptr [11 x %str], ptr %a48, i64 0, i64 0
  store %str %t15, ptr %t49
  %t50 = getelementptr [11 x %str], ptr %a48, i64 0, i64 1
  store %str { ptr @.str.12, i64 1 }, ptr %t50
  %t51 = getelementptr [11 x %str], ptr %a48, i64 0, i64 2
  store %str %t17, ptr %t51
  %t52 = getelementptr [11 x %str], ptr %a48, i64 0, i64 3
  store %str { ptr @.str.12, i64 1 }, ptr %t52
  %t53 = getelementptr [11 x %str], ptr %a48, i64 0, i64 4
  store %str %t33, ptr %t53
  %t54 = getelementptr [11 x %str], ptr %a48, i64 0, i64 5
  store %str { ptr @.str.12, i64 1 }, ptr %t54
  %t55 = getelementptr [11 x %str], ptr %a48, i64 0, i64 6
  store %str %t38, ptr %t55
  %t56 = getelementptr [11 x %str], ptr %a48, i64 0, i64 7
  store %str { ptr @.str.12, i64 1 }, ptr %t56
  %t57 = getelementptr [11 x %str], ptr %a48, i64 0, i64 8
  store %str %t40, ptr %t57
  %t58 = getelementptr [11 x %str], ptr %a48, i64 0, i64 9
  store %str { ptr @.str.12, i64 1 }, ptr %t58
  %t59 = getelementptr [11 x %str], ptr %a48, i64 0, i64 10
  store %str %t46, ptr %t59
  call void @veles_string_concat_n(ptr %a47, ptr %a48, i64 11)
  %t60 = load %str, ptr %a47
  %t61 = extractvalue %str { ptr @.str.13, i64 20 }, 0
  %t62 = extractvalue %str { ptr @.str.13, i64 20 }, 1
  call void @veles_call_push(ptr %t61)
  call void @v_std.io.println(%str %t60)
  call void @veles_call_pop()
  ret void
}

define { i64, i64 } @v_main.swap__i64_i64({ i64, i64 } %p1) {
entry:
  %a1 = alloca { i64, i64 }
  store { i64, i64 } %p1, ptr %a1
  %t2 = load { i64, i64 }, ptr %a1
  %t3 = extractvalue { i64, i64 } %t2, 1
  %t4 = insertvalue { i64, i64 } undef, i64 %t3, 0
  %t5 = load { i64, i64 }, ptr %a1
  %t6 = extractvalue { i64, i64 } %t5, 0
  %t7 = insertvalue { i64, i64 } %t4, i64 %t6, 1
  ret { i64, i64 } %t7
}

@.str.1 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:7:44\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.4 = private unnamed_addr constant [13 x i8] c"main.vs:7:51\00"
@.str.5 = private unnamed_addr constant [13 x i8] c"main.vs:9:73\00"
@.str.6 = private unnamed_addr constant [13 x i8] c"main.vs:9:80\00"
@.str.7 = private unnamed_addr constant [21 x i8] c"main.vs:14:15\00divmod\00"
@.str.8 = private unnamed_addr constant [29 x i8] c"main.vs:15:15\00divmodLabelled\00"
@.str.9 = private unnamed_addr constant [19 x i8] c"main.vs:16:14\00swap\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:17:31\00"
@.str.11 = private unnamed_addr constant [19 x i8] c"main.vs:17:59\00swap\00"
@.str.12 = private unnamed_addr constant [2 x i8] c" \00"
@.str.13 = private unnamed_addr constant [21 x i8] c"main.vs:17:3\00println\00"
