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
  %t7 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t5, i32 0, i32 1
  %t8 = load i64, ptr %t7
  %t9 = icmp ult i64 %t6, %t8
  br i1 %t9, label %uidx.ok.1, label %uidx.bad.2, !prof !{!"branch_weights", i32 2000, i32 1}
uidx.bad.2:
  %t10 = extractvalue %str { ptr @.str.1, i64 12 }, 0
  %t11 = extractvalue %str { ptr @.str.1, i64 12 }, 1
  call void @veles_unchecked_index_panic(i64 %t6, i64 %t8, i64 0, ptr %t10, i64 %t11)
  unreachable
uidx.ok.1:
  %t12 = load ptr, ptr %t5
  %t13 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t5, i32 0, i32 3
  %t14 = load i64, ptr %t13
  %t15 = mul i64 %t14, %t6
  %t16 = getelementptr inbounds i8, ptr %t12, i64 %t15
  %t17 = load i64, ptr %t16
  ret i64 %t17
}

define void @v_main.put(ptr %p1, i64 %p2, i64 %p3) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a3 = alloca i64
  %a5 = alloca ptr
  store ptr %p1, ptr %a1
  store i64 %p2, ptr %a2
  store i64 %p3, ptr %a3
  %t4 = load ptr, ptr %a1
  store ptr %t4, ptr %a5
  %t6 = load i64, ptr %a3
  %t7 = load ptr, ptr %a5
  %t8 = load i64, ptr %a2
  %t9 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t7, i32 0, i32 1
  %t10 = load i64, ptr %t9
  %t11 = icmp ult i64 %t8, %t10
  br i1 %t11, label %uidx.ok.1, label %uidx.bad.2, !prof !{!"branch_weights", i32 2000, i32 1}
uidx.bad.2:
  %t12 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t13 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_unchecked_index_panic(i64 %t8, i64 %t10, i64 0, ptr %t12, i64 %t13)
  unreachable
uidx.ok.1:
  %t14 = load ptr, ptr %t7
  %t15 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t7, i32 0, i32 3
  %t16 = load i64, ptr %t15
  %t17 = mul i64 %t16, %t8
  %t18 = getelementptr inbounds i8, ptr %t14, i64 %t17
  store i64 %t6, ptr %t18
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
  %t8 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t9 = extractvalue %str { ptr @.str.3, i64 13 }, 1
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
  %a29 = alloca [21 x i8]
  %a33 = alloca %str
  %a34 = alloca [3 x %str]
  %t1 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 5, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 7, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store ptr %t1, ptr %a3
  %t4 = load ptr, ptr %a3
  %t5 = load ptr, ptr %a3
  %t6 = extractvalue %str { ptr @.str.4, i64 17 }, 0
  %t7 = extractvalue %str { ptr @.str.4, i64 17 }, 1
  call void @veles_call_push(ptr %t6)
  %t8 = call i64 @v_main.get(ptr %t5, i64 0)
  call void @veles_call_pop()
  %t9 = load ptr, ptr %a3
  %t10 = extractvalue %str { ptr @.str.5, i64 17 }, 0
  %t11 = extractvalue %str { ptr @.str.5, i64 17 }, 1
  call void @veles_call_push(ptr %t10)
  %t12 = call i64 @v_main.get(ptr %t9, i64 2)
  call void @veles_call_pop()
  %t14 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t8, i64 %t12)
  %t15 = extractvalue { i64, i1 } %t14, 0
  %t16 = extractvalue { i64, i1 } %t14, 1
  br i1 %t16, label %overflow.1, label %arith.ok.2
overflow.1:
  %t17 = extractvalue %str { ptr @.str.6, i64 16 }, 0
  %t18 = extractvalue %str { ptr @.str.6, i64 16 }, 1
  %t19 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t20 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t17, i64 %t18, ptr %t19, i64 %t20)
  unreachable
arith.ok.2:
  %t21 = extractvalue %str { ptr @.str.8, i64 16 }, 0
  %t22 = extractvalue %str { ptr @.str.8, i64 16 }, 1
  call void @veles_call_push(ptr %t21)
  call void @v_main.put(ptr %t4, i64 1, i64 %t15)
  call void @veles_call_pop()
  %t23 = load ptr, ptr %a3
  %t24 = call %str @show.MutableList_i64_(ptr %t23)
  %t25 = extractvalue %str { ptr @.str.9, i64 18 }, 0
  %t26 = extractvalue %str { ptr @.str.9, i64 18 }, 1
  call void @veles_call_push(ptr %t25)
  %t27 = call i8 @v_main.byte(%str { ptr @.str.10, i64 5 }, i64 1)
  call void @veles_call_pop()
  %t28 = zext i8 %t27 to i64
  %t30 = call i64 @veles_u64_format(ptr %a29, i64 %t28)
  %t31 = insertvalue %str undef, ptr %a29, 0
  %t32 = insertvalue %str %t31, i64 %t30, 1
  %t35 = getelementptr [3 x %str], ptr %a34, i64 0, i64 0
  store %str %t24, ptr %t35
  %t36 = getelementptr [3 x %str], ptr %a34, i64 0, i64 1
  store %str { ptr @.str.11, i64 1 }, ptr %t36
  %t37 = getelementptr [3 x %str], ptr %a34, i64 0, i64 2
  store %str %t32, ptr %t37
  call void @veles_string_concat_n(ptr %a33, ptr %a34, i64 3)
  %t38 = load %str, ptr %a33
  %t39 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t40 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t39)
  call void @v_std.io.println(%str %t38)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [13 x i8] c"main.vs:8:12\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:13:12\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:18:12\00"
@.str.4 = private unnamed_addr constant [18 x i8] c"main.vs:23:14\00get\00"
@.str.5 = private unnamed_addr constant [18 x i8] c"main.vs:23:27\00get\00"
@.str.6 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:23:14\00"
@.str.8 = private unnamed_addr constant [17 x i8] c"main.vs:23:3\00put\00"
@.str.9 = private unnamed_addr constant [19 x i8] c"main.vs:24:23\00byte\00"
@.str.10 = private unnamed_addr constant [6 x i8] c"veles\00"
@.str.11 = private unnamed_addr constant [2 x i8] c" \00"
@.str.12 = private unnamed_addr constant [21 x i8] c"main.vs:24:3\00println\00"
