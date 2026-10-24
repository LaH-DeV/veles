%V.main.Expr = type { i32, [1 x i64] }
%S.main.Num = type { i64 }
%S.main.Add = type { ptr, ptr, %str }
%S.main.Upload = type { %str, i64 }
%V.main.Event = type { i32, [1 x i64] }
%S.main.Hole = type {}
%S.main.Tick = type { i64 }
@desc.main.Add = internal constant { i64, i64, i64, [3 x i64] } { i64 32, i64 0, i64 3, [3 x i64] [i64 0, i64 8, i64 16] }
@desc.main.Expr = internal constant { i64, i64, i64, [1 x i64] } { i64 16, i64 0, i64 1, [1 x i64] [i64 8] }
@desc.main.Upload = internal constant { i64, i64, i64, [1 x i64] } { i64 24, i64 0, i64 1, [1 x i64] [i64 0] }
define i64 @v_main.eval(%V.main.Expr %p1) {
entry:
  %a1 = alloca %V.main.Expr
  %a2 = alloca i64
  %a4 = alloca %V.main.Expr
  store %V.main.Expr %p1, ptr %a1
  %t3 = load %V.main.Expr, ptr %a1
  store %V.main.Expr %t3, ptr %a4
  br label %when.arm.2
when.arm.2:
  %t5 = load %V.main.Expr, ptr %a4
  %t6 = extractvalue %V.main.Expr %t5, 0
  %t7 = icmp eq i32 %t6, 0
  br i1 %t7, label %when.bind.5, label %when.arm.3
when.bind.5:
  br label %when.body.4
when.body.4:
  %t8 = getelementptr inbounds %V.main.Expr, ptr %a1, i32 0, i32 1
  %t9 = getelementptr inbounds %S.main.Num, ptr %t8, i32 0, i32 0
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a2
  br label %when.end.1
when.arm.3:
  %t11 = load %V.main.Expr, ptr %a4
  %t12 = extractvalue %V.main.Expr %t11, 0
  %t13 = icmp eq i32 %t12, 1
  br i1 %t13, label %when.bind.8, label %when.arm.6
when.bind.8:
  br label %when.body.7
when.body.7:
  %t14 = getelementptr inbounds %V.main.Expr, ptr %a1, i32 0, i32 1
  %t15 = load ptr, ptr %t14
  %t16 = getelementptr inbounds %S.main.Add, ptr %t15, i32 0, i32 0
  %t17 = load ptr, ptr %t16
  %t18 = load %V.main.Expr, ptr %t17
  %t19 = extractvalue %str { ptr @.str.1, i64 18 }, 0
  %t20 = extractvalue %str { ptr @.str.1, i64 18 }, 1
  call void @veles_call_push(ptr %t19)
  %t21 = call i64 @v_main.eval(%V.main.Expr %t18)
  call void @veles_call_pop()
  %t22 = getelementptr inbounds %V.main.Expr, ptr %a1, i32 0, i32 1
  %t23 = load ptr, ptr %t22
  %t24 = getelementptr inbounds %S.main.Add, ptr %t23, i32 0, i32 1
  %t25 = load ptr, ptr %t24
  %t26 = load %V.main.Expr, ptr %t25
  %t27 = extractvalue %str { ptr @.str.2, i64 18 }, 0
  %t28 = extractvalue %str { ptr @.str.2, i64 18 }, 1
  call void @veles_call_push(ptr %t27)
  %t29 = call i64 @v_main.eval(%V.main.Expr %t26)
  call void @veles_call_pop()
  %t31 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t21, i64 %t29)
  %t32 = extractvalue { i64, i1 } %t31, 0
  %t33 = extractvalue { i64, i1 } %t31, 1
  br i1 %t33, label %overflow.9, label %arith.ok.10
overflow.9:
  %t34 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t35 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t36 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t37 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t34, i64 %t35, ptr %t36, i64 %t37)
  unreachable
arith.ok.10:
  store i64 %t32, ptr %a2
  br label %when.end.1
when.arm.6:
  %t38 = load %V.main.Expr, ptr %a4
  %t39 = extractvalue %V.main.Expr %t38, 0
  %t40 = icmp eq i32 %t39, 2
  br i1 %t40, label %when.bind.13, label %when.arm.11
when.bind.13:
  br label %when.body.12
when.body.12:
  store i64 0, ptr %a2
  br label %when.end.1
when.arm.11:
  %t41 = extractvalue %str { ptr @.str.5, i64 33 }, 0
  %t42 = extractvalue %str { ptr @.str.5, i64 33 }, 1
  %t43 = extractvalue %str { ptr @.str.6, i64 13 }, 0
  %t44 = extractvalue %str { ptr @.str.6, i64 13 }, 1
  call void @veles_panic_at(ptr %t41, i64 %t42, ptr %t43, i64 %t44)
  unreachable
when.end.1:
  %t45 = load i64, ptr %a2
  ret i64 %t45
}

define %V.main.Expr @v_main.relabel(%V.main.Expr %p1) {
entry:
  %a1 = alloca %V.main.Expr
  %a3 = alloca %V.main.Expr
  store %V.main.Expr %p1, ptr %a1
  %t2 = load %V.main.Expr, ptr %a1
  store %V.main.Expr %t2, ptr %a3
  %t4 = load %V.main.Expr, ptr %a3
  %t5 = extractvalue %V.main.Expr %t4, 0
  %t6 = icmp eq i32 %t5, 1
  br i1 %t6, label %if.then.1, label %if.end.2
if.then.1:
  %t7 = getelementptr inbounds %V.main.Expr, ptr %a3, i32 0, i32 1
  %t8 = load ptr, ptr %t7
  %t9 = call ptr @veles_gc_alloc(ptr @desc.main.Add, i64 32)
  call void @llvm.memcpy.p0.p0.i64(ptr %t9, ptr %t8, i64 32, i1 false)
  store ptr %t9, ptr %t7
  %t10 = getelementptr inbounds %S.main.Add, ptr %t9, i32 0, i32 2
  store %str { ptr @.str.7, i64 1 }, ptr %t10
  br label %if.end.2
if.end.2:
  %t11 = load %V.main.Expr, ptr %a3
  ret %V.main.Expr %t11
}

define void @v_main.main() {
entry:
  %a2 = alloca %V.main.Expr
  %a8 = alloca %V.main.Expr
  %a17 = alloca %V.main.Expr
  %a21 = alloca %V.main.Expr
  %a25 = alloca %V.main.Event
  %a29 = alloca %V.main.Event
  %a34 = alloca [21 x i8]
  %a47 = alloca %V.main.Expr
  %a52 = alloca %str
  %a53 = alloca [9 x %str]
  %t1 = insertvalue %S.main.Num undef, i64 2, 0
  store %V.main.Expr zeroinitializer, ptr %a2
  %t3 = getelementptr inbounds %V.main.Expr, ptr %a2, i32 0, i32 0
  store i32 0, ptr %t3
  %t4 = getelementptr inbounds %V.main.Expr, ptr %a2, i32 0, i32 1
  store %S.main.Num %t1, ptr %t4
  %t5 = load %V.main.Expr, ptr %a2
  %t6 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 16)
  store %V.main.Expr %t5, ptr %t6
  %t7 = insertvalue %S.main.Num undef, i64 3, 0
  store %V.main.Expr zeroinitializer, ptr %a8
  %t9 = getelementptr inbounds %V.main.Expr, ptr %a8, i32 0, i32 0
  store i32 0, ptr %t9
  %t10 = getelementptr inbounds %V.main.Expr, ptr %a8, i32 0, i32 1
  store %S.main.Num %t7, ptr %t10
  %t11 = load %V.main.Expr, ptr %a8
  %t12 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 16)
  store %V.main.Expr %t11, ptr %t12
  %t13 = insertvalue %S.main.Add undef, ptr %t6, 0
  %t14 = insertvalue %S.main.Add %t13, ptr %t12, 1
  %t15 = insertvalue %S.main.Add %t14, %str { ptr @.str.8, i64 1 }, 2
  %t16 = call ptr @veles_gc_alloc(ptr @desc.main.Add, i64 32)
  store %S.main.Add %t15, ptr %t16
  store %V.main.Expr zeroinitializer, ptr %a17
  %t18 = getelementptr inbounds %V.main.Expr, ptr %a17, i32 0, i32 0
  store i32 1, ptr %t18
  %t19 = getelementptr inbounds %V.main.Expr, ptr %a17, i32 0, i32 1
  store ptr %t16, ptr %t19
  %t20 = load %V.main.Expr, ptr %a17
  store %V.main.Expr %t20, ptr %a21
  %t22 = insertvalue %S.main.Upload undef, %str { ptr @.str.8, i64 1 }, 0
  %t23 = insertvalue %S.main.Upload %t22, i64 4, 1
  %t24 = call ptr @veles_gc_alloc(ptr @desc.main.Upload, i64 24)
  store %S.main.Upload %t23, ptr %t24
  store %V.main.Event zeroinitializer, ptr %a25
  %t26 = getelementptr inbounds %V.main.Event, ptr %a25, i32 0, i32 0
  store i32 1, ptr %t26
  %t27 = getelementptr inbounds %V.main.Event, ptr %a25, i32 0, i32 1
  store ptr %t24, ptr %t27
  %t28 = load %V.main.Event, ptr %a25
  store %V.main.Event %t28, ptr %a29
  %t30 = load %V.main.Expr, ptr %a21
  %t31 = extractvalue %str { ptr @.str.9, i64 18 }, 0
  %t32 = extractvalue %str { ptr @.str.9, i64 18 }, 1
  call void @veles_call_push(ptr %t31)
  %t33 = call i64 @v_main.eval(%V.main.Expr %t30)
  call void @veles_call_pop()
  %t35 = call i64 @veles_i64_format(ptr %a34, i64 %t33)
  %t36 = insertvalue %str undef, ptr %a34, 0
  %t37 = insertvalue %str %t36, i64 %t35, 1
  %t38 = load %V.main.Expr, ptr %a21
  %t39 = extractvalue %str { ptr @.str.10, i64 21 }, 0
  %t40 = extractvalue %str { ptr @.str.10, i64 21 }, 1
  call void @veles_call_push(ptr %t39)
  %t41 = call %V.main.Expr @v_main.relabel(%V.main.Expr %t38)
  call void @veles_call_pop()
  %t42 = call %str @show.main.Expr(%V.main.Expr %t41)
  %t43 = load %V.main.Expr, ptr %a21
  %t44 = call %str @show.main.Expr(%V.main.Expr %t43)
  %t45 = load %V.main.Event, ptr %a29
  %t46 = call %str @show.main.Event(%V.main.Event %t45)
  store %V.main.Expr zeroinitializer, ptr %a47
  %t48 = getelementptr inbounds %V.main.Expr, ptr %a47, i32 0, i32 0
  store i32 2, ptr %t48
  %t49 = getelementptr inbounds %V.main.Expr, ptr %a47, i32 0, i32 1
  store %S.main.Hole zeroinitializer, ptr %t49
  %t50 = load %V.main.Expr, ptr %a47
  %t51 = call %str @show.main.Expr(%V.main.Expr %t50)
  %t54 = getelementptr [9 x %str], ptr %a53, i64 0, i64 0
  store %str %t37, ptr %t54
  %t55 = getelementptr [9 x %str], ptr %a53, i64 0, i64 1
  store %str { ptr @.str.11, i64 1 }, ptr %t55
  %t56 = getelementptr [9 x %str], ptr %a53, i64 0, i64 2
  store %str %t42, ptr %t56
  %t57 = getelementptr [9 x %str], ptr %a53, i64 0, i64 3
  store %str { ptr @.str.11, i64 1 }, ptr %t57
  %t58 = getelementptr [9 x %str], ptr %a53, i64 0, i64 4
  store %str %t44, ptr %t58
  %t59 = getelementptr [9 x %str], ptr %a53, i64 0, i64 5
  store %str { ptr @.str.11, i64 1 }, ptr %t59
  %t60 = getelementptr [9 x %str], ptr %a53, i64 0, i64 6
  store %str %t46, ptr %t60
  %t61 = getelementptr [9 x %str], ptr %a53, i64 0, i64 7
  store %str { ptr @.str.11, i64 1 }, ptr %t61
  %t62 = getelementptr [9 x %str], ptr %a53, i64 0, i64 8
  store %str %t51, ptr %t62
  call void @veles_string_concat_n(ptr %a52, ptr %a53, i64 9)
  %t63 = load %str, ptr %a52
  %t64 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t65 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t64)
  call void @v_std.io.println(%str %t63)
  call void @veles_call_pop()
  ret void
}

define internal %str @show.main.Expr(%V.main.Expr %v) {
entry:
  %a1 = alloca %str
  %a3 = alloca %V.main.Expr
  %a7 = alloca %V.main.Expr
  %a12 = alloca %V.main.Expr
  store %str { ptr @.str.13, i64 1 }, ptr %a1
  %t2 = extractvalue %V.main.Expr %v, 0
  switch i32 %t2, label %show.end.1 [ i32 0, label %show.case.2 i32 1, label %show.case.3 i32 2, label %show.case.4 ]
show.case.2:
  store %V.main.Expr %v, ptr %a3
  %t4 = getelementptr inbounds %V.main.Expr, ptr %a3, i32 0, i32 1
  %t5 = load %S.main.Num, ptr %t4
  %t6 = call %str @show.main.Num(%S.main.Num %t5)
  store %str %t6, ptr %a1
  br label %show.end.1
show.case.3:
  store %V.main.Expr %v, ptr %a7
  %t8 = getelementptr inbounds %V.main.Expr, ptr %a7, i32 0, i32 1
  %t9 = load ptr, ptr %t8
  %t10 = load %S.main.Add, ptr %t9
  %t11 = call %str @show.main.Add(%S.main.Add %t10)
  store %str %t11, ptr %a1
  br label %show.end.1
show.case.4:
  store %V.main.Expr %v, ptr %a12
  %t13 = getelementptr inbounds %V.main.Expr, ptr %a12, i32 0, i32 1
  %t14 = load %S.main.Hole, ptr %t13
  %t15 = call %str @show.main.Hole(%S.main.Hole %t14)
  store %str %t15, ptr %a1
  br label %show.end.1
show.end.1:
  %t16 = load %str, ptr %a1
  ret %str %t16
}

define internal %str @show.main.Event(%V.main.Event %v) {
entry:
  %a1 = alloca %str
  %a3 = alloca %V.main.Event
  %a7 = alloca %V.main.Event
  store %str { ptr @.str.13, i64 1 }, ptr %a1
  %t2 = extractvalue %V.main.Event %v, 0
  switch i32 %t2, label %show.end.1 [ i32 0, label %show.case.2 i32 1, label %show.case.3 ]
show.case.2:
  store %V.main.Event %v, ptr %a3
  %t4 = getelementptr inbounds %V.main.Event, ptr %a3, i32 0, i32 1
  %t5 = load %S.main.Tick, ptr %t4
  %t6 = call %str @show.main.Tick(%S.main.Tick %t5)
  store %str %t6, ptr %a1
  br label %show.end.1
show.case.3:
  store %V.main.Event %v, ptr %a7
  %t8 = getelementptr inbounds %V.main.Event, ptr %a7, i32 0, i32 1
  %t9 = load ptr, ptr %t8
  %t10 = load %S.main.Upload, ptr %t9
  %t11 = call %str @show.main.Upload(%S.main.Upload %t10)
  store %str %t11, ptr %a1
  br label %show.end.1
show.end.1:
  %t12 = load %str, ptr %a1
  ret %str %t12
}

define internal %str @show.main.Num(%S.main.Num %v) {
entry:
  %a5 = alloca %str
  %a8 = alloca %str
  %a14 = alloca %str
  %a20 = alloca %str
  %t1 = extractvalue %str { ptr @.str.14, i64 4 }, 0
  %t2 = extractvalue %str { ptr @.str.14, i64 4 }, 1
  %t3 = extractvalue %str { ptr @.str.15, i64 7 }, 0
  %t4 = extractvalue %str { ptr @.str.15, i64 7 }, 1
  call void @veles_string_concat(ptr %a5, ptr %t1, i64 %t2, ptr %t3, i64 %t4)
  %t6 = load %str, ptr %a5
  %t7 = extractvalue %S.main.Num %v, 0
  call void @veles_i64_to_string(ptr %a8, i64 %t7)
  %t9 = load %str, ptr %a8
  %t10 = extractvalue %str %t6, 0
  %t11 = extractvalue %str %t6, 1
  %t12 = extractvalue %str %t9, 0
  %t13 = extractvalue %str %t9, 1
  call void @veles_string_concat(ptr %a14, ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  %t15 = load %str, ptr %a14
  %t16 = extractvalue %str %t15, 0
  %t17 = extractvalue %str %t15, 1
  %t18 = extractvalue %str { ptr @.str.16, i64 1 }, 0
  %t19 = extractvalue %str { ptr @.str.16, i64 1 }, 1
  call void @veles_string_concat(ptr %a20, ptr %t16, i64 %t17, ptr %t18, i64 %t19)
  %t21 = load %str, ptr %a20
  ret %str %t21
}

define internal %str @show.main.Add(%S.main.Add %v) {
entry:
  %a5 = alloca %str
  %a13 = alloca %str
  %a19 = alloca %str
  %a25 = alloca %str
  %a33 = alloca %str
  %a39 = alloca %str
  %a45 = alloca %str
  %a52 = alloca %str
  %a58 = alloca %str
  %t1 = extractvalue %str { ptr @.str.17, i64 4 }, 0
  %t2 = extractvalue %str { ptr @.str.17, i64 4 }, 1
  %t3 = extractvalue %str { ptr @.str.18, i64 6 }, 0
  %t4 = extractvalue %str { ptr @.str.18, i64 6 }, 1
  call void @veles_string_concat(ptr %a5, ptr %t1, i64 %t2, ptr %t3, i64 %t4)
  %t6 = load %str, ptr %a5
  %t7 = extractvalue %S.main.Add %v, 0
  %t8 = call %str @show.Pmain.Expr(ptr %t7)
  %t9 = extractvalue %str %t6, 0
  %t10 = extractvalue %str %t6, 1
  %t11 = extractvalue %str %t8, 0
  %t12 = extractvalue %str %t8, 1
  call void @veles_string_concat(ptr %a13, ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  %t14 = load %str, ptr %a13
  %t15 = extractvalue %str %t14, 0
  %t16 = extractvalue %str %t14, 1
  %t17 = extractvalue %str { ptr @.str.19, i64 2 }, 0
  %t18 = extractvalue %str { ptr @.str.19, i64 2 }, 1
  call void @veles_string_concat(ptr %a19, ptr %t15, i64 %t16, ptr %t17, i64 %t18)
  %t20 = load %str, ptr %a19
  %t21 = extractvalue %str %t20, 0
  %t22 = extractvalue %str %t20, 1
  %t23 = extractvalue %str { ptr @.str.20, i64 7 }, 0
  %t24 = extractvalue %str { ptr @.str.20, i64 7 }, 1
  call void @veles_string_concat(ptr %a25, ptr %t21, i64 %t22, ptr %t23, i64 %t24)
  %t26 = load %str, ptr %a25
  %t27 = extractvalue %S.main.Add %v, 1
  %t28 = call %str @show.Pmain.Expr(ptr %t27)
  %t29 = extractvalue %str %t26, 0
  %t30 = extractvalue %str %t26, 1
  %t31 = extractvalue %str %t28, 0
  %t32 = extractvalue %str %t28, 1
  call void @veles_string_concat(ptr %a33, ptr %t29, i64 %t30, ptr %t31, i64 %t32)
  %t34 = load %str, ptr %a33
  %t35 = extractvalue %str %t34, 0
  %t36 = extractvalue %str %t34, 1
  %t37 = extractvalue %str { ptr @.str.19, i64 2 }, 0
  %t38 = extractvalue %str { ptr @.str.19, i64 2 }, 1
  call void @veles_string_concat(ptr %a39, ptr %t35, i64 %t36, ptr %t37, i64 %t38)
  %t40 = load %str, ptr %a39
  %t41 = extractvalue %str %t40, 0
  %t42 = extractvalue %str %t40, 1
  %t43 = extractvalue %str { ptr @.str.21, i64 7 }, 0
  %t44 = extractvalue %str { ptr @.str.21, i64 7 }, 1
  call void @veles_string_concat(ptr %a45, ptr %t41, i64 %t42, ptr %t43, i64 %t44)
  %t46 = load %str, ptr %a45
  %t47 = extractvalue %S.main.Add %v, 2
  %t48 = extractvalue %str %t46, 0
  %t49 = extractvalue %str %t46, 1
  %t50 = extractvalue %str %t47, 0
  %t51 = extractvalue %str %t47, 1
  call void @veles_string_concat(ptr %a52, ptr %t48, i64 %t49, ptr %t50, i64 %t51)
  %t53 = load %str, ptr %a52
  %t54 = extractvalue %str %t53, 0
  %t55 = extractvalue %str %t53, 1
  %t56 = extractvalue %str { ptr @.str.16, i64 1 }, 0
  %t57 = extractvalue %str { ptr @.str.16, i64 1 }, 1
  call void @veles_string_concat(ptr %a58, ptr %t54, i64 %t55, ptr %t56, i64 %t57)
  %t59 = load %str, ptr %a58
  ret %str %t59
}

define internal %str @show.main.Hole(%S.main.Hole %v) {
entry:
  ret %str { ptr @.str.22, i64 4 }
}

define internal %str @show.main.Tick(%S.main.Tick %v) {
entry:
  %a5 = alloca %str
  %a8 = alloca %str
  %a14 = alloca %str
  %a20 = alloca %str
  %t1 = extractvalue %str { ptr @.str.23, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.23, i64 5 }, 1
  %t3 = extractvalue %str { ptr @.str.24, i64 4 }, 0
  %t4 = extractvalue %str { ptr @.str.24, i64 4 }, 1
  call void @veles_string_concat(ptr %a5, ptr %t1, i64 %t2, ptr %t3, i64 %t4)
  %t6 = load %str, ptr %a5
  %t7 = extractvalue %S.main.Tick %v, 0
  call void @veles_i64_to_string(ptr %a8, i64 %t7)
  %t9 = load %str, ptr %a8
  %t10 = extractvalue %str %t6, 0
  %t11 = extractvalue %str %t6, 1
  %t12 = extractvalue %str %t9, 0
  %t13 = extractvalue %str %t9, 1
  call void @veles_string_concat(ptr %a14, ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  %t15 = load %str, ptr %a14
  %t16 = extractvalue %str %t15, 0
  %t17 = extractvalue %str %t15, 1
  %t18 = extractvalue %str { ptr @.str.16, i64 1 }, 0
  %t19 = extractvalue %str { ptr @.str.16, i64 1 }, 1
  call void @veles_string_concat(ptr %a20, ptr %t16, i64 %t17, ptr %t18, i64 %t19)
  %t21 = load %str, ptr %a20
  ret %str %t21
}

define internal %str @show.main.Upload(%S.main.Upload %v) {
entry:
  %a5 = alloca %str
  %a12 = alloca %str
  %a18 = alloca %str
  %a24 = alloca %str
  %a27 = alloca %str
  %a33 = alloca %str
  %a39 = alloca %str
  %t1 = extractvalue %str { ptr @.str.25, i64 7 }, 0
  %t2 = extractvalue %str { ptr @.str.25, i64 7 }, 1
  %t3 = extractvalue %str { ptr @.str.26, i64 6 }, 0
  %t4 = extractvalue %str { ptr @.str.26, i64 6 }, 1
  call void @veles_string_concat(ptr %a5, ptr %t1, i64 %t2, ptr %t3, i64 %t4)
  %t6 = load %str, ptr %a5
  %t7 = extractvalue %S.main.Upload %v, 0
  %t8 = extractvalue %str %t6, 0
  %t9 = extractvalue %str %t6, 1
  %t10 = extractvalue %str %t7, 0
  %t11 = extractvalue %str %t7, 1
  call void @veles_string_concat(ptr %a12, ptr %t8, i64 %t9, ptr %t10, i64 %t11)
  %t13 = load %str, ptr %a12
  %t14 = extractvalue %str %t13, 0
  %t15 = extractvalue %str %t13, 1
  %t16 = extractvalue %str { ptr @.str.19, i64 2 }, 0
  %t17 = extractvalue %str { ptr @.str.19, i64 2 }, 1
  call void @veles_string_concat(ptr %a18, ptr %t14, i64 %t15, ptr %t16, i64 %t17)
  %t19 = load %str, ptr %a18
  %t20 = extractvalue %str %t19, 0
  %t21 = extractvalue %str %t19, 1
  %t22 = extractvalue %str { ptr @.str.27, i64 6 }, 0
  %t23 = extractvalue %str { ptr @.str.27, i64 6 }, 1
  call void @veles_string_concat(ptr %a24, ptr %t20, i64 %t21, ptr %t22, i64 %t23)
  %t25 = load %str, ptr %a24
  %t26 = extractvalue %S.main.Upload %v, 1
  call void @veles_i64_to_string(ptr %a27, i64 %t26)
  %t28 = load %str, ptr %a27
  %t29 = extractvalue %str %t25, 0
  %t30 = extractvalue %str %t25, 1
  %t31 = extractvalue %str %t28, 0
  %t32 = extractvalue %str %t28, 1
  call void @veles_string_concat(ptr %a33, ptr %t29, i64 %t30, ptr %t31, i64 %t32)
  %t34 = load %str, ptr %a33
  %t35 = extractvalue %str %t34, 0
  %t36 = extractvalue %str %t34, 1
  %t37 = extractvalue %str { ptr @.str.16, i64 1 }, 0
  %t38 = extractvalue %str { ptr @.str.16, i64 1 }, 1
  call void @veles_string_concat(ptr %a39, ptr %t35, i64 %t36, ptr %t37, i64 %t38)
  %t40 = load %str, ptr %a39
  ret %str %t40
}

define internal %str @show.Pmain.Expr(ptr %v) {
entry:
  %a7 = alloca %str
  %t1 = load %V.main.Expr, ptr %v
  %t2 = call %str @show.main.Expr(%V.main.Expr %t1)
  %t3 = extractvalue %str { ptr @.str.28, i64 1 }, 0
  %t4 = extractvalue %str { ptr @.str.28, i64 1 }, 1
  %t5 = extractvalue %str %t2, 0
  %t6 = extractvalue %str %t2, 1
  call void @veles_string_concat(ptr %a7, ptr %t3, i64 %t4, ptr %t5, i64 %t6)
  %t8 = load %str, ptr %a7
  ret %str %t8
}

@.str.1 = private unnamed_addr constant [19 x i8] c"main.vs:23:14\00eval\00"
@.str.2 = private unnamed_addr constant [19 x i8] c"main.vs:23:30\00eval\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:23:14\00"
@.str.5 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.6 = private unnamed_addr constant [14 x i8] c"main.vs:21:27\00"
@.str.7 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.8 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.9 = private unnamed_addr constant [19 x i8] c"main.vs:46:17\00eval\00"
@.str.10 = private unnamed_addr constant [22 x i8] c"main.vs:46:30\00relabel\00"
@.str.11 = private unnamed_addr constant [2 x i8] c" \00"
@.str.12 = private unnamed_addr constant [21 x i8] c"main.vs:46:3\00println\00"
@.str.13 = private unnamed_addr constant [2 x i8] c"?\00"
@.str.14 = private unnamed_addr constant [5 x i8] c"Num(\00"
@.str.15 = private unnamed_addr constant [8 x i8] c"value: \00"
@.str.16 = private unnamed_addr constant [2 x i8] c")\00"
@.str.17 = private unnamed_addr constant [5 x i8] c"Add(\00"
@.str.18 = private unnamed_addr constant [7 x i8] c"left: \00"
@.str.19 = private unnamed_addr constant [3 x i8] c", \00"
@.str.20 = private unnamed_addr constant [8 x i8] c"right: \00"
@.str.21 = private unnamed_addr constant [8 x i8] c"label: \00"
@.str.22 = private unnamed_addr constant [5 x i8] c"Hole\00"
@.str.23 = private unnamed_addr constant [6 x i8] c"Tick(\00"
@.str.24 = private unnamed_addr constant [5 x i8] c"at: \00"
@.str.25 = private unnamed_addr constant [8 x i8] c"Upload(\00"
@.str.26 = private unnamed_addr constant [7 x i8] c"name: \00"
@.str.27 = private unnamed_addr constant [7 x i8] c"size: \00"
@.str.28 = private unnamed_addr constant [2 x i8] c"&\00"
