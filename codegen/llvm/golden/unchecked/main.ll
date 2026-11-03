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
  %t9 = icmp ult i64 %t6, %t8
  br i1 %t9, label %uidx.ok.1, label %uidx.bad.2, !prof !{!"branch_weights", i32 2000, i32 1}
uidx.bad.2:
  %t10 = extractvalue %str { ptr @.str.1, i64 11 }, 0
  %t11 = extractvalue %str { ptr @.str.1, i64 11 }, 1
  call void @veles_unchecked_index_panic(i64 %t6, i64 %t8, i64 0, ptr %t10, i64 %t11)
  unreachable
uidx.ok.1:
  %t12 = load ptr, ptr %t5
  %t13 = getelementptr inbounds [8 x i8], ptr %t12, i64 %t6
  %t14 = load i64, ptr %t13
  ret i64 %t14
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
  %t19 = icmp ult i64 %t16, %t18
  br i1 %t19, label %uidx.ok.3, label %uidx.bad.4, !prof !{!"branch_weights", i32 2000, i32 1}
uidx.bad.4:
  %t20 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t21 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_unchecked_index_panic(i64 %t16, i64 %t18, i64 0, ptr %t20, i64 %t21)
  unreachable
uidx.ok.3:
  %t22 = load ptr, ptr %t15
  %t23 = getelementptr inbounds [8 x i8], ptr %t22, i64 %t16
  store i64 %t7, ptr %t23
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
  %t7 = icmp ult i64 %t4, %t6
  br i1 %t7, label %uidx.ok.1, label %uidx.bad.2, !prof !{!"branch_weights", i32 2000, i32 1}
uidx.bad.2:
  %t8 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t9 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_unchecked_index_panic(i64 %t4, i64 %t6, i64 1, ptr %t8, i64 %t9)
  unreachable
uidx.ok.1:
  %t10 = getelementptr inbounds i8, ptr %t5, i64 %t4
  %t11 = load i8, ptr %t10
  ret i8 %t11
}

define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a5 = alloca ptr
  %a17 = alloca ptr
  %a44 = alloca [21 x i8]
  %a48 = alloca %str
  %a49 = alloca [3 x %str]
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
  %t13 = extractvalue %str { ptr @.str.4, i64 17 }, 0
  %t14 = extractvalue %str { ptr @.str.4, i64 17 }, 1
  call void @veles_call_push(ptr %t13)
  %t15 = call i64 @v_main.get(ptr %t12, i64 0)
  call void @veles_call_pop()
  %t16 = load ptr, ptr %a3
  store ptr %t16, ptr %a17
  %t18 = icmp eq ptr %t16, null
  br i1 %t18, label %share.done.6, label %share.test.4
share.test.4:
  %t19 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t16, i32 0, i32 6
  %t20 = load i64, ptr %t19
  %t21 = and i64 %t20, 4
  %t22 = icmp ne i64 %t21, 0
  br i1 %t22, label %share.done.6, label %share.call.5
share.call.5:
  %t23 = call ptr @veles_list_share(ptr %t16, ptr null)
  store ptr %t23, ptr %a17
  br label %share.done.6
share.done.6:
  %t24 = load ptr, ptr %a17
  %t25 = extractvalue %str { ptr @.str.5, i64 17 }, 0
  %t26 = extractvalue %str { ptr @.str.5, i64 17 }, 1
  call void @veles_call_push(ptr %t25)
  %t27 = call i64 @v_main.get(ptr %t24, i64 2)
  call void @veles_call_pop()
  %t29 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t15, i64 %t27)
  %t30 = extractvalue { i64, i1 } %t29, 0
  %t31 = extractvalue { i64, i1 } %t29, 1
  br i1 %t31, label %overflow.7, label %arith.ok.8
overflow.7:
  %t32 = extractvalue %str { ptr @.str.6, i64 16 }, 0
  %t33 = extractvalue %str { ptr @.str.6, i64 16 }, 1
  %t34 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t35 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t32, i64 %t33, ptr %t34, i64 %t35)
  unreachable
arith.ok.8:
  %t36 = extractvalue %str { ptr @.str.8, i64 16 }, 0
  %t37 = extractvalue %str { ptr @.str.8, i64 16 }, 1
  call void @veles_call_push(ptr %t36)
  call void @v_main.put(ptr %a3, i64 1, i64 %t30)
  call void @veles_call_pop()
  %t38 = load ptr, ptr %a3
  %t39 = call %str @show.List_i64_(ptr %t38)
  %t40 = extractvalue %str { ptr @.str.9, i64 18 }, 0
  %t41 = extractvalue %str { ptr @.str.9, i64 18 }, 1
  call void @veles_call_push(ptr %t40)
  %t42 = call i8 @v_main.byte(%str { ptr @.str.10, i64 5 }, i64 1)
  call void @veles_call_pop()
  %t43 = zext i8 %t42 to i64
  %t45 = call i64 @veles_u64_format(ptr %a44, i64 %t43)
  %t46 = insertvalue %str undef, ptr %a44, 0
  %t47 = insertvalue %str %t46, i64 %t45, 1
  %t50 = getelementptr [3 x %str], ptr %a49, i64 0, i64 0
  store %str %t39, ptr %t50
  %t51 = getelementptr [3 x %str], ptr %a49, i64 0, i64 1
  store %str { ptr @.str.11, i64 1 }, ptr %t51
  %t52 = getelementptr [3 x %str], ptr %a49, i64 0, i64 2
  store %str %t47, ptr %t52
  call void @veles_string_concat_n(ptr %a48, ptr %a49, i64 3)
  %t53 = load %str, ptr %a48
  %t54 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t55 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t54)
  call void @v_std.io.println(%str %t53)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [12 x i8] c"main.vs:9:5\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:16:5\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:23:5\00"
@.str.4 = private unnamed_addr constant [18 x i8] c"main.vs:29:15\00get\00"
@.str.5 = private unnamed_addr constant [18 x i8] c"main.vs:29:28\00get\00"
@.str.6 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:29:15\00"
@.str.8 = private unnamed_addr constant [17 x i8] c"main.vs:29:3\00put\00"
@.str.9 = private unnamed_addr constant [19 x i8] c"main.vs:30:23\00byte\00"
@.str.10 = private unnamed_addr constant [6 x i8] c"veles\00"
@.str.11 = private unnamed_addr constant [2 x i8] c" \00"
@.str.12 = private unnamed_addr constant [21 x i8] c"main.vs:30:3\00println\00"
