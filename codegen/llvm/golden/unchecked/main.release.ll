define i64 @v_main.get(ptr %p1, i64 %p2) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a4 = alloca ptr
  store ptr %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load ptr, ptr %a1
  store ptr %t3, ptr %a4
  %t5 = load ptr, ptr %a4
  %t6 = load i64, ptr %a2
  %t7 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t5, i32 0, i32 1
  %t8 = load i64, ptr %t7
  %t9 = load ptr, ptr %t5
  %t10 = getelementptr inbounds [8 x i8], ptr %t9, i64 %t6
  %t11 = load i64, ptr %t10
  ret i64 %t11
}

define void @v_main.put(ptr %p1, i64 %p2, i64 %p3) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a3 = alloca i64
  %a6 = alloca ptr
  store ptr %p1, ptr %a1
  store i64 %p2, ptr %a2
  store i64 %p3, ptr %a3
  %t4 = load ptr, ptr %a1
  %t5 = load ptr, ptr %t4
  store ptr %t5, ptr %a6
  %t7 = load i64, ptr %a3
  %t8 = load ptr, ptr %a1
  %t9 = load ptr, ptr %t8
  %t10 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t9, i32 0, i32 6
  %t11 = load i64, ptr %t10
  %t12 = and i64 %t11, 12
  %t13 = icmp ne i64 %t12, 0
  br i1 %t13, label %cow.copy.1, label %cow.done.2, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.1:
  %t14 = call ptr @veles_list_unique(ptr %t9, ptr null)
  store ptr %t14, ptr %t8
  br label %cow.done.2
cow.done.2:
  %t15 = load ptr, ptr %t8
  %t16 = load i64, ptr %a2
  %t17 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t15, i32 0, i32 1
  %t18 = load i64, ptr %t17
  %t19 = load ptr, ptr %t15
  %t20 = getelementptr inbounds [8 x i8], ptr %t19, i64 %t16
  store i64 %t7, ptr %t20
  ret void
}

define i8 @v_main.byte(%str %p1, i64 %p2) {
entry:
  %a1 = alloca %str
  %a2 = alloca i64
  store %str %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load %str, ptr %a1
  %t4 = load i64, ptr %a2
  %t5 = extractvalue %str %t3, 0
  %t6 = extractvalue %str %t3, 1
  %t7 = getelementptr inbounds i8, ptr %t5, i64 %t4
  %t8 = load i8, ptr %t7
  ret i8 %t8
}

define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a5 = alloca ptr
  %a15 = alloca ptr
  %a36 = alloca [21 x i8]
  %a40 = alloca %str
  %a41 = alloca [3 x %str]
  %t1 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 5, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 7, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store ptr %t1, ptr %a3
  %t4 = load ptr, ptr %a3
  store ptr %t4, ptr %a5
  %t6 = icmp eq ptr %t4, null
  br i1 %t6, label %share.done.3, label %share.test.1
share.test.1:
  %t7 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t4, i32 0, i32 6
  %t8 = load i64, ptr %t7
  %t9 = and i64 %t8, 4
  %t10 = icmp ne i64 %t9, 0
  br i1 %t10, label %share.done.3, label %share.call.2
share.call.2:
  %t11 = call ptr @veles_list_share(ptr %t4, ptr null)
  store ptr %t11, ptr %a5
  br label %share.done.3
share.done.3:
  %t12 = load ptr, ptr %a5
  %t13 = call i64 @v_main.get(ptr %t12, i64 0)
  %t14 = load ptr, ptr %a3
  store ptr %t14, ptr %a15
  %t16 = icmp eq ptr %t14, null
  br i1 %t16, label %share.done.6, label %share.test.4
share.test.4:
  %t17 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t14, i32 0, i32 6
  %t18 = load i64, ptr %t17
  %t19 = and i64 %t18, 4
  %t20 = icmp ne i64 %t19, 0
  br i1 %t20, label %share.done.6, label %share.call.5
share.call.5:
  %t21 = call ptr @veles_list_share(ptr %t14, ptr null)
  store ptr %t21, ptr %a15
  br label %share.done.6
share.done.6:
  %t22 = load ptr, ptr %a15
  %t23 = call i64 @v_main.get(ptr %t22, i64 2)
  %t25 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t13, i64 %t23)
  %t26 = extractvalue { i64, i1 } %t25, 0
  %t27 = extractvalue { i64, i1 } %t25, 1
  br i1 %t27, label %overflow.7, label %arith.ok.8
overflow.7:
  %t28 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t29 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t30 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t31 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t28, i64 %t29, ptr %t30, i64 %t31)
  unreachable
arith.ok.8:
  call void @v_main.put(ptr %a3, i64 1, i64 %t26)
  %t32 = load ptr, ptr %a3
  %t33 = call %str @show.List_i64_(ptr %t32)
  %t34 = call i8 @v_main.byte(%str { ptr @.str.3, i64 5 }, i64 1)
  %t35 = zext i8 %t34 to i64
  %t37 = call i64 @veles_u64_format(ptr %a36, i64 %t35)
  %t38 = insertvalue %str undef, ptr %a36, 0
  %t39 = insertvalue %str %t38, i64 %t37, 1
  %t42 = getelementptr [3 x %str], ptr %a41, i64 0, i64 0
  store %str %t33, ptr %t42
  %t43 = getelementptr [3 x %str], ptr %a41, i64 0, i64 1
  store %str { ptr @.str.4, i64 1 }, ptr %t43
  %t44 = getelementptr [3 x %str], ptr %a41, i64 0, i64 2
  store %str %t39, ptr %t44
  call void @veles_string_concat_n(ptr %a40, ptr %a41, i64 3)
  %t45 = load %str, ptr %a40
  call void @v_std.io.println(%str %t45)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:29:15\00"
@.str.3 = private unnamed_addr constant [6 x i8] c"veles\00"
@.str.4 = private unnamed_addr constant [2 x i8] c" \00"
