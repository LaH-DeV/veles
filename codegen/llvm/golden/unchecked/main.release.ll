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
  %t9 = load ptr, ptr %t5
  %t10 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t5, i32 0, i32 3
  %t11 = load i64, ptr %t10
  %t12 = mul i64 %t11, %t6
  %t13 = getelementptr inbounds i8, ptr %t9, i64 %t12
  %t14 = load i64, ptr %t13
  ret i64 %t14
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
  %t11 = load ptr, ptr %t7
  %t12 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t7, i32 0, i32 3
  %t13 = load i64, ptr %t12
  %t14 = mul i64 %t13, %t8
  %t15 = getelementptr inbounds i8, ptr %t11, i64 %t14
  store i64 %t6, ptr %t15
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
  %a14 = alloca [21 x i8]
  %a18 = alloca %str
  %a19 = alloca [3 x %str]
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
  %t6 = call i64 @v_main.get(ptr %t5, i64 0)
  %t7 = load ptr, ptr %a3
  %t8 = call i64 @v_main.get(ptr %t7, i64 2)
  %t9 = add i64 %t6, %t8
  call void @v_main.put(ptr %t4, i64 1, i64 %t9)
  %t10 = load ptr, ptr %a3
  %t11 = call %str @show.MutableList_i64_(ptr %t10)
  %t12 = call i8 @v_main.byte(%str { ptr @.str.1, i64 5 }, i64 1)
  %t13 = zext i8 %t12 to i64
  %t15 = call i64 @veles_u64_format(ptr %a14, i64 %t13)
  %t16 = insertvalue %str undef, ptr %a14, 0
  %t17 = insertvalue %str %t16, i64 %t15, 1
  %t20 = getelementptr [3 x %str], ptr %a19, i64 0, i64 0
  store %str %t11, ptr %t20
  %t21 = getelementptr [3 x %str], ptr %a19, i64 0, i64 1
  store %str { ptr @.str.2, i64 1 }, ptr %t21
  %t22 = getelementptr [3 x %str], ptr %a19, i64 0, i64 2
  store %str %t17, ptr %t22
  call void @veles_string_concat_n(ptr %a18, ptr %a19, i64 3)
  %t23 = load %str, ptr %a18
  call void @v_std.io.println(%str %t23)
  ret void
}

@.str.1 = private unnamed_addr constant [6 x i8] c"veles\00"
@.str.2 = private unnamed_addr constant [2 x i8] c" \00"
