define i8 @v_main.add(i8 %p1, i8 %p2) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = load i8, ptr %a2
  %t6 = call { i8, i1 } @llvm.sadd.with.overflow.i8(i8 %t3, i8 %t4)
  %t7 = extractvalue { i8, i1 } %t6, 0
  %t8 = extractvalue { i8, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t12 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  ret i8 %t7
}

define i8 @v_main.sub(i8 %p1, i8 %p2) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = load i8, ptr %a2
  %t6 = call { i8, i1 } @llvm.usub.with.overflow.i8(i8 %t3, i8 %t4)
  %t7 = extractvalue { i8, i1 } %t6, 0
  %t8 = extractvalue { i8, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t12 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  ret i8 %t7
}

define i64 @v_main.mul(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t6 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t3, i64 %t4)
  %t7 = extractvalue { i64, i1 } %t6, 0
  %t8 = extractvalue { i64, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t12 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  ret i64 %t7
}

define i32 @v_main.neg(i32 %p1) {
entry:
  %a1 = alloca i32
  store i32 %p1, ptr %a1
  %t2 = load i32, ptr %a1
  %t4 = call { i32, i1 } @llvm.ssub.with.overflow.i32(i32 0, i32 %t2)
  %t5 = extractvalue { i32, i1 } %t4, 0
  %t6 = extractvalue { i32, i1 } %t4, 1
  br i1 %t6, label %overflow.1, label %arith.ok.2
overflow.1:
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.5, i64 13 }, 0
  %t10 = extractvalue %str { ptr @.str.5, i64 13 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
arith.ok.2:
  ret i32 %t5
}

define i64 @v_main.quot(i64 %p1, i64 %p2) {
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
  %t7 = extractvalue %str { ptr @.str.6, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.6, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t10 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
div.ok.2:
  %t11 = icmp eq i64 %t3, -9223372036854775808
  %t12 = icmp eq i64 %t4, -1
  %t13 = and i1 %t11, %t12
  br i1 %t13, label %divof.3, label %div.ok.4
divof.3:
  %t14 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t15 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t16 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t17 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t14, i64 %t15, ptr %t16, i64 %t17)
  unreachable
div.ok.4:
  %t5 = sdiv i64 %t3, %t4
  ret i64 %t5
}

define i16 @v_main.absolute(i16 %p1) {
entry:
  %a1 = alloca i16
  store i16 %p1, ptr %a1
  %t2 = load i16, ptr %a1
  %t3 = icmp eq i16 %t2, -32768
  br i1 %t3, label %overflow.1, label %abs.ok.2
overflow.1:
  %t4 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t5 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t6 = extractvalue %str { ptr @.str.8, i64 13 }, 0
  %t7 = extractvalue %str { ptr @.str.8, i64 13 }, 1
  call void @veles_panic_at(ptr %t4, i64 %t5, ptr %t6, i64 %t7)
  unreachable
abs.ok.2:
  %t8 = call i16 @llvm.abs.i16(i16 %t2, i1 false)
  ret i16 %t8
}

define i8 @v_main.bump(i8 %p1) {
entry:
  %a1 = alloca i8
  %a3 = alloca i8
  store i8 %p1, ptr %a1
  %t2 = load i8, ptr %a1
  store i8 %t2, ptr %a3
  %t4 = load i8, ptr %a3
  %t6 = call { i8, i1 } @llvm.sadd.with.overflow.i8(i8 %t4, i8 1)
  %t7 = extractvalue { i8, i1 } %t6, 0
  %t8 = extractvalue { i8, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.9, i64 12 }, 0
  %t12 = extractvalue %str { ptr @.str.9, i64 12 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  store i8 %t7, ptr %a3
  %t13 = load i8, ptr %a3
  ret i8 %t13
}

define i8 @v_main.wrapped(i8 %p1, i8 %p2) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = load i8, ptr %a2
  %t5 = add i8 %t3, %t4
  ret i8 %t5
}

define i8 @v_main.narrow(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = trunc i64 %t2 to i8
  ret i8 %t3
}

define i64 @v_main.sumUpTo(i8 %p1) {
entry:
  %a1 = alloca i8
  %a2 = alloca i64
  %a7 = alloca { i8, i8, i1 }
  %a10 = alloca i8
  %a13 = alloca i8
  %a14 = alloca i1
  %a15 = alloca i1
  %a17 = alloca i1
  %a29 = alloca i8
  %a41 = alloca i1
  store i8 %p1, ptr %a1
  store i64 0, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = insertvalue { i8, i8, i1 } undef, i8 250, 0
  %t5 = insertvalue { i8, i8, i1 } %t4, i8 %t3, 1
  %t6 = insertvalue { i8, i8, i1 } %t5, i1 true, 2
  store { i8, i8, i1 } %t6, ptr %a7
  %t8 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 0
  %t9 = load i8, ptr %t8
  store i8 %t9, ptr %a10
  %t11 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 1
  %t12 = load i8, ptr %t11
  store i8 %t12, ptr %a13
  store i1 false, ptr %a14
  br label %loop.cond.1
loop.cond.1:
  %t16 = load i1, ptr %a14
  br i1 %t16, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a15
  br label %if.end.6
if.else.7:
  %t18 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 2
  %t19 = load i1, ptr %t18
  br i1 %t19, label %if.then.8, label %if.else.10
if.then.8:
  %t20 = load i8, ptr %a10
  %t21 = load i8, ptr %a13
  %t22 = icmp ule i8 %t20, %t21
  store i1 %t22, ptr %a17
  br label %if.end.9
if.else.10:
  %t23 = load i8, ptr %a10
  %t24 = load i8, ptr %a13
  %t25 = icmp ult i8 %t23, %t24
  store i1 %t25, ptr %a17
  br label %if.end.9
if.end.9:
  %t26 = load i1, ptr %a17
  store i1 %t26, ptr %a15
  br label %if.end.6
if.end.6:
  %t27 = load i1, ptr %a15
  br i1 %t27, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t28 = load i8, ptr %a10
  store i8 %t28, ptr %a29
  %t30 = load i64, ptr %a2
  %t31 = load i8, ptr %a29
  %t32 = zext i8 %t31 to i64
  %t34 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t30, i64 %t32)
  %t35 = extractvalue { i64, i1 } %t34, 0
  %t36 = extractvalue { i64, i1 } %t34, 1
  br i1 %t36, label %overflow.11, label %arith.ok.12
overflow.11:
  %t37 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t38 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t39 = extractvalue %str { ptr @.str.10, i64 12 }, 0
  %t40 = extractvalue %str { ptr @.str.10, i64 12 }, 1
  call void @veles_panic_at(ptr %t37, i64 %t38, ptr %t39, i64 %t40)
  unreachable
arith.ok.12:
  store i64 %t35, ptr %a2
  br label %loop.post.2
loop.post.2:
  %t42 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 2
  %t43 = load i1, ptr %t42
  store i1 %t43, ptr %a41
  br i1 %t43, label %sc.rhs.13, label %sc.end.14
sc.rhs.13:
  %t44 = load i8, ptr %a10
  %t45 = load i8, ptr %a13
  %t46 = icmp eq i8 %t44, %t45
  store i1 %t46, ptr %a41
  br label %sc.end.14
sc.end.14:
  %t47 = load i1, ptr %a41
  br i1 %t47, label %if.then.15, label %if.else.17
if.then.15:
  store i1 true, ptr %a14
  br label %if.end.16
if.else.17:
  %t48 = load i8, ptr %a10
  %t49 = add i8 %t48, 1
  store i8 %t49, ptr %a10
  br label %if.end.16
if.end.16:
  %t50 = load volatile i32, ptr @veles_stop_requested, align 4
  %t51 = icmp ne i32 %t50, 0
  br i1 %t51, label %safepoint.18, label %safepoint.on.19, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.18:
  call void @veles_gc_park()
  br label %safepoint.on.19
safepoint.on.19:
  br label %loop.cond.1
loop.end.3:
  %t52 = load i64, ptr %a2
  ret i64 %t52
}

define void @v_main.edges(i8 %p1, i8 %p2, i8 %p3) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  %a3 = alloca i8
  %a17 = alloca { i8, i8, i1 }
  %a21 = alloca %S.std.prelude.RangeIter_u8_
  %a25 = alloca [21 x i8]
  %a29 = alloca %str
  %a42 = alloca { i8, i8, i1 }
  %a46 = alloca [21 x i8]
  %a57 = alloca { i8, i8, i1 }
  %a61 = alloca [21 x i8]
  %a65 = alloca %str
  %a66 = alloca [4 x %str]
  %a79 = alloca { i8, i8, i1 }
  %a83 = alloca %S.std.prelude.RangeStepIter_i8_
  %a88 = alloca %str
  %a101 = alloca { i8, i8, i1 }
  %a105 = alloca %S.std.prelude.RangeStepIter_i8_
  %a109 = alloca %S.std.prelude.RangeStepIter_i8_
  %a119 = alloca { i8, i8, i1 }
  %a123 = alloca %S.std.prelude.RangeStepIter_i8_
  %a127 = alloca %S.std.prelude.RangeStepIter_i8_
  %a132 = alloca %str
  %a133 = alloca [4 x %str]
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  store i8 %p3, ptr %a3
  %t4 = load i8, ptr %a3
  %t6 = call { i8, i1 } @llvm.usub.with.overflow.i8(i8 %t4, i8 5)
  %t7 = extractvalue { i8, i1 } %t6, 0
  %t8 = extractvalue { i8, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t12 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  %t13 = load i8, ptr %a3
  %t14 = insertvalue { i8, i8, i1 } undef, i8 %t7, 0
  %t15 = insertvalue { i8, i8, i1 } %t14, i8 %t13, 1
  %t16 = insertvalue { i8, i8, i1 } %t15, i1 true, 2
  store { i8, i8, i1 } %t16, ptr %a17
  %t18 = extractvalue %str { ptr @.str.12, i64 18 }, 0
  %t19 = extractvalue %str { ptr @.str.12, i64 18 }, 1
  call void @veles_call_push(ptr %t18)
  %t20 = call %S.std.prelude.RangeIter_u8_ @v_std.prelude.Iterable.iter_Self_Range_u8_T_u8_(ptr %a17)
  call void @veles_call_pop()
  store %S.std.prelude.RangeIter_u8_ %t20, ptr %a21
  %t22 = extractvalue %str { ptr @.str.13, i64 19 }, 0
  %t23 = extractvalue %str { ptr @.str.13, i64 19 }, 1
  call void @veles_call_push(ptr %t22)
  %t24 = call i64 @v_std.prelude.Iterator.count_Self_std.prelude.RangeIter_u8_T_u8_(ptr %a21)
  call void @veles_call_pop()
  %t26 = call i64 @veles_i64_format(ptr %a25, i64 %t24)
  %t27 = insertvalue %str undef, ptr %a25, 0
  %t28 = insertvalue %str %t27, i64 %t26, 1
  %t30 = extractvalue %str { ptr @.str.14, i64 25 }, 0
  %t31 = extractvalue %str { ptr @.str.14, i64 25 }, 1
  %t32 = extractvalue %str %t28, 0
  %t33 = extractvalue %str %t28, 1
  call void @veles_string_concat(ptr %a29, ptr %t30, i64 %t31, ptr %t32, i64 %t33)
  %t34 = load %str, ptr %a29
  %t35 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t36 = extractvalue %str { ptr @.str.15, i64 20 }, 1
  call void @veles_call_push(ptr %t35)
  call void @v_std.io.println(%str %t34)
  call void @veles_call_pop()
  %t37 = load i8, ptr %a1
  %t38 = load i8, ptr %a2
  %t39 = insertvalue { i8, i8, i1 } undef, i8 %t37, 0
  %t40 = insertvalue { i8, i8, i1 } %t39, i8 %t38, 1
  %t41 = insertvalue { i8, i8, i1 } %t40, i1 true, 2
  store { i8, i8, i1 } %t41, ptr %a42
  %t43 = extractvalue %str { ptr @.str.16, i64 17 }, 0
  %t44 = extractvalue %str { ptr @.str.16, i64 17 }, 1
  call void @veles_call_push(ptr %t43)
  %t45 = call i64 @v_std.prelude.extend.Range_T.len_T_i8_(ptr %a42)
  call void @veles_call_pop()
  %t47 = call i64 @veles_i64_format(ptr %a46, i64 %t45)
  %t48 = insertvalue %str undef, ptr %a46, 0
  %t49 = insertvalue %str %t48, i64 %t47, 1
  %t50 = load i8, ptr %a1
  %t51 = add i8 %t50, 127
  %t52 = add i8 %t51, 1
  %t53 = load i8, ptr %a1
  %t54 = insertvalue { i8, i8, i1 } undef, i8 %t52, 0
  %t55 = insertvalue { i8, i8, i1 } %t54, i8 %t53, 1
  %t56 = insertvalue { i8, i8, i1 } %t55, i1 false, 2
  store { i8, i8, i1 } %t56, ptr %a57
  %t58 = extractvalue %str { ptr @.str.17, i64 17 }, 0
  %t59 = extractvalue %str { ptr @.str.17, i64 17 }, 1
  call void @veles_call_push(ptr %t58)
  %t60 = call i64 @v_std.prelude.extend.Range_T.len_T_i8_(ptr %a57)
  call void @veles_call_pop()
  %t62 = call i64 @veles_i64_format(ptr %a61, i64 %t60)
  %t63 = insertvalue %str undef, ptr %a61, 0
  %t64 = insertvalue %str %t63, i64 %t62, 1
  %t67 = getelementptr [4 x %str], ptr %a66, i64 0, i64 0
  store %str { ptr @.str.18, i64 20 }, ptr %t67
  %t68 = getelementptr [4 x %str], ptr %a66, i64 0, i64 1
  store %str %t49, ptr %t68
  %t69 = getelementptr [4 x %str], ptr %a66, i64 0, i64 2
  store %str { ptr @.str.19, i64 21 }, ptr %t69
  %t70 = getelementptr [4 x %str], ptr %a66, i64 0, i64 3
  store %str %t64, ptr %t70
  call void @veles_string_concat_n(ptr %a65, ptr %a66, i64 4)
  %t71 = load %str, ptr %a65
  %t72 = extractvalue %str { ptr @.str.20, i64 20 }, 0
  %t73 = extractvalue %str { ptr @.str.20, i64 20 }, 1
  call void @veles_call_push(ptr %t72)
  call void @v_std.io.println(%str %t71)
  call void @veles_call_pop()
  %t74 = load i8, ptr %a1
  %t75 = load i8, ptr %a2
  %t76 = insertvalue { i8, i8, i1 } undef, i8 %t74, 0
  %t77 = insertvalue { i8, i8, i1 } %t76, i8 %t75, 1
  %t78 = insertvalue { i8, i8, i1 } %t77, i1 true, 2
  store { i8, i8, i1 } %t78, ptr %a79
  %t80 = extractvalue %str { ptr @.str.21, i64 18 }, 0
  %t81 = extractvalue %str { ptr @.str.21, i64 18 }, 1
  call void @veles_call_push(ptr %t80)
  %t82 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.step_T_i8_(ptr %a79, i8 100)
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i8_ %t82, ptr %a83
  %t84 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t85 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t84)
  %t86 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a83)
  call void @veles_call_pop()
  %t87 = call %str @show.List_i8_(ptr %t86)
  %t89 = extractvalue %str { ptr @.str.23, i64 24 }, 0
  %t90 = extractvalue %str { ptr @.str.23, i64 24 }, 1
  %t91 = extractvalue %str %t87, 0
  %t92 = extractvalue %str %t87, 1
  call void @veles_string_concat(ptr %a88, ptr %t89, i64 %t90, ptr %t91, i64 %t92)
  %t93 = load %str, ptr %a88
  %t94 = extractvalue %str { ptr @.str.24, i64 20 }, 0
  %t95 = extractvalue %str { ptr @.str.24, i64 20 }, 1
  call void @veles_call_push(ptr %t94)
  call void @v_std.io.println(%str %t93)
  call void @veles_call_pop()
  %t96 = load i8, ptr %a1
  %t97 = load i8, ptr %a2
  %t98 = insertvalue { i8, i8, i1 } undef, i8 %t96, 0
  %t99 = insertvalue { i8, i8, i1 } %t98, i8 %t97, 1
  %t100 = insertvalue { i8, i8, i1 } %t99, i1 true, 2
  store { i8, i8, i1 } %t100, ptr %a101
  %t102 = extractvalue %str { ptr @.str.25, i64 22 }, 0
  %t103 = extractvalue %str { ptr @.str.25, i64 22 }, 1
  call void @veles_call_push(ptr %t102)
  %t104 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.reversed_T_i8_(ptr %a101)
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i8_ %t104, ptr %a105
  %t106 = extractvalue %str { ptr @.str.26, i64 18 }, 0
  %t107 = extractvalue %str { ptr @.str.26, i64 18 }, 1
  call void @veles_call_push(ptr %t106)
  %t108 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.RangeStepIter.step_T_i8_(ptr %a105, i8 100)
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i8_ %t108, ptr %a109
  %t110 = extractvalue %str { ptr @.str.27, i64 20 }, 0
  %t111 = extractvalue %str { ptr @.str.27, i64 20 }, 1
  call void @veles_call_push(ptr %t110)
  %t112 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a109)
  call void @veles_call_pop()
  %t113 = call %str @show.List_i8_(ptr %t112)
  %t114 = load i8, ptr %a1
  %t115 = load i8, ptr %a2
  %t116 = insertvalue { i8, i8, i1 } undef, i8 %t114, 0
  %t117 = insertvalue { i8, i8, i1 } %t116, i8 %t115, 1
  %t118 = insertvalue { i8, i8, i1 } %t117, i1 true, 2
  store { i8, i8, i1 } %t118, ptr %a119
  %t120 = extractvalue %str { ptr @.str.28, i64 19 }, 0
  %t121 = extractvalue %str { ptr @.str.28, i64 19 }, 1
  call void @veles_call_push(ptr %t120)
  %t122 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.step_T_i8_(ptr %a119, i8 100)
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i8_ %t122, ptr %a123
  %t124 = extractvalue %str { ptr @.str.29, i64 23 }, 0
  %t125 = extractvalue %str { ptr @.str.29, i64 23 }, 1
  call void @veles_call_push(ptr %t124)
  %t126 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.RangeStepIter.reversed_T_i8_(ptr %a123)
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i8_ %t126, ptr %a127
  %t128 = extractvalue %str { ptr @.str.30, i64 21 }, 0
  %t129 = extractvalue %str { ptr @.str.30, i64 21 }, 1
  call void @veles_call_push(ptr %t128)
  %t130 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a127)
  call void @veles_call_pop()
  %t131 = call %str @show.List_i8_(ptr %t130)
  %t134 = getelementptr [4 x %str], ptr %a133, i64 0, i64 0
  store %str { ptr @.str.31, i64 21 }, ptr %t134
  %t135 = getelementptr [4 x %str], ptr %a133, i64 0, i64 1
  store %str %t113, ptr %t135
  %t136 = getelementptr [4 x %str], ptr %a133, i64 0, i64 2
  store %str { ptr @.str.32, i64 23 }, ptr %t136
  %t137 = getelementptr [4 x %str], ptr %a133, i64 0, i64 3
  store %str %t131, ptr %t137
  call void @veles_string_concat_n(ptr %a132, ptr %a133, i64 4)
  %t138 = load %str, ptr %a132
  %t139 = extractvalue %str { ptr @.str.33, i64 20 }, 0
  %t140 = extractvalue %str { ptr @.str.33, i64 20 }, 1
  call void @veles_call_push(ptr %t139)
  call void @v_std.io.println(%str %t138)
  call void @veles_call_pop()
  ret void
}

define i64 @v_main.call({ ptr, ptr } %p1) {
entry:
  %a1 = alloca { ptr, ptr }
  store { ptr, ptr } %p1, ptr %a1
  %t2 = load { ptr, ptr }, ptr %a1
  %t3 = extractvalue { ptr, ptr } %t2, 0
  %t4 = extractvalue { ptr, ptr } %t2, 1
  %t5 = call i64 %t3(ptr %t4)
  ret i64 %t5
}

define ptr @v_main.attempt(ptr %task, %str %p1, { ptr, ptr } %p2) presplitcoroutine {
entry:
  %a1 = alloca %str
  %a2 = alloca { ptr, ptr }
  %a4 = alloca ptr
  %a5 = alloca { ptr, ptr, ptr }
  %a11 = alloca ptr
  %a24 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a27 = alloca i64
  %a32 = alloca i64
  %a40 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a46 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a52 = alloca { %V._prelude_.Result_i64_std.prelude.Panic_ }
  %a55 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a62 = alloca i64
  %a65 = alloca [21 x i8]
  %a69 = alloca %str
  %a70 = alloca [3 x %str]
  %a83 = alloca %S.std.prelude.Panic
  %a87 = alloca %str
  %a88 = alloca [3 x %str]
  store %str %p1, ptr %a1
  store { ptr, ptr } %p2, ptr %a2
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t3 = call ptr @veles_scope_begin(ptr %task, i64 0)
  store ptr %t3, ptr %a4
  call void @veles_cleanup_push(ptr %a5, ptr @scope.cancel.thunk, ptr %a4)
  %t6 = load ptr, ptr %a4
  %t7 = call ptr @veles_task_launch(ptr %t6, i64 0)
  %t8 = load { ptr, ptr }, ptr %a2
  %t9 = call ptr @veles_alloc_words(i64 24)
  %t10 = getelementptr inbounds { { ptr, ptr } }, ptr %t9, i32 0, i32 0
  store { ptr, ptr } %t8, ptr %t10
  call void @veles_task_spawn(ptr %t7, ptr @entry.ramp.v_main.call, ptr %t9)
  store ptr %t7, ptr %a11
  br label %scope.wait.1
scope.wait.1:
  %t12 = load ptr, ptr %a4
  %t13 = call i64 @veles_scope_wait(ptr %task, ptr %t12)
  %t14 = icmp ne i64 %t13, 0
  br i1 %t14, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  %t15 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t15, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t16 = call i64 @veles_task_cancelled(ptr %task)
  %t17 = icmp ne i64 %t16, 0
  br i1 %t17, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_cleanup_pop(ptr %a5)
  %t18 = load ptr, ptr %a4
  call void @veles_scope_cancel(ptr %t18)
  br label %abandon.wait.7
abandon.wait.7:
  %t19 = call i64 @veles_scope_wait(ptr %task, ptr %t18)
  %t20 = icmp ne i64 %t19, 0
  br i1 %t20, label %abandon.done.8, label %abandon.susp.9
abandon.susp.9:
  %t21 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t21, label %coro.suspend [ i8 0, label %resume.10 i8 1, label %coro.cleanup ]
resume.10:
  br label %abandon.wait.7
abandon.done.8:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.6:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop(ptr %a5)
  %t22 = load ptr, ptr %a11
  %t23 = call ptr @veles_task_result(ptr %t22)
  %t25 = call i64 @veles_task_panicked(ptr %t22)
  %t26 = icmp ne i64 %t25, 0
  br i1 %t26, label %gather.panic.11, label %gather.value.12
gather.panic.11:
  %t28 = call ptr @veles_task_panic_msg(ptr %t22, ptr %a27)
  %t29 = load i64, ptr %a27
  %t30 = insertvalue %str undef, ptr %t28, 0
  %t31 = insertvalue %str %t30, i64 %t29, 1
  %t33 = call ptr @veles_task_panic_loc(ptr %t22, ptr %a32)
  %t34 = load i64, ptr %a32
  %t35 = insertvalue %str undef, ptr %t33, 0
  %t36 = insertvalue %str %t35, i64 %t34, 1
  %t37 = insertvalue %S.std.prelude.Panic undef, %str %t31, 0
  %t38 = insertvalue %S.std.prelude.Panic %t37, %str %t36, 1
  %t39 = insertvalue %S._prelude_.Err_i64_std.prelude.Panic_ undef, %S.std.prelude.Panic %t38, 0
  store %V._prelude_.Result_i64_std.prelude.Panic_ zeroinitializer, ptr %a40
  %t41 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a40, i32 0, i32 0
  store i32 1, ptr %t41
  %t42 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a40, i32 0, i32 1
  store %S._prelude_.Err_i64_std.prelude.Panic_ %t39, ptr %t42
  %t43 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a40
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t43, ptr %a24
  br label %gather.join.13
gather.value.12:
  %t44 = load i64, ptr %t23
  %t45 = insertvalue %S._prelude_.Ok_i64_std.prelude.Panic_ undef, i64 %t44, 0
  store %V._prelude_.Result_i64_std.prelude.Panic_ zeroinitializer, ptr %a46
  %t47 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a46, i32 0, i32 0
  store i32 0, ptr %t47
  %t48 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a46, i32 0, i32 1
  store %S._prelude_.Ok_i64_std.prelude.Panic_ %t45, ptr %t48
  %t49 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a46
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t49, ptr %a24
  br label %gather.join.13
gather.join.13:
  %t50 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a24
  %t51 = insertvalue { %V._prelude_.Result_i64_std.prelude.Panic_ } undef, %V._prelude_.Result_i64_std.prelude.Panic_ %t50, 0
  store { %V._prelude_.Result_i64_std.prelude.Panic_ } %t51, ptr %a52
  %t53 = load { %V._prelude_.Result_i64_std.prelude.Panic_ }, ptr %a52
  %t54 = extractvalue { %V._prelude_.Result_i64_std.prelude.Panic_ } %t53, 0
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t54, ptr %a55
  br label %when.arm.15
when.arm.15:
  %t56 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55
  %t57 = extractvalue %V._prelude_.Result_i64_std.prelude.Panic_ %t56, 0
  %t58 = icmp eq i32 %t57, 0
  br i1 %t58, label %when.bind.18, label %when.arm.16
when.bind.18:
  %t59 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55, i32 0, i32 1
  %t60 = getelementptr inbounds %S._prelude_.Ok_i64_std.prelude.Panic_, ptr %t59, i32 0, i32 0
  %t61 = load i64, ptr %t60
  store i64 %t61, ptr %a62
  br label %when.body.17
when.body.17:
  %t63 = load %str, ptr %a1
  %t64 = load i64, ptr %a62
  %t66 = call i64 @veles_i64_format(ptr %a65, i64 %t64)
  %t67 = insertvalue %str undef, ptr %a65, 0
  %t68 = insertvalue %str %t67, i64 %t66, 1
  %t71 = getelementptr [3 x %str], ptr %a70, i64 0, i64 0
  store %str %t63, ptr %t71
  %t72 = getelementptr [3 x %str], ptr %a70, i64 0, i64 1
  store %str { ptr @.str.34, i64 3 }, ptr %t72
  %t73 = getelementptr [3 x %str], ptr %a70, i64 0, i64 2
  store %str %t68, ptr %t73
  call void @veles_string_concat_n(ptr %a69, ptr %a70, i64 3)
  %t74 = load %str, ptr %a69
  %t75 = extractvalue %str { ptr @.str.35, i64 21 }, 0
  %t76 = extractvalue %str { ptr @.str.35, i64 21 }, 1
  call void @veles_call_push(ptr %t75)
  call void @v_std.io.println(%str %t74)
  call void @veles_call_pop()
  br label %when.end.14
when.arm.16:
  %t77 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55
  %t78 = extractvalue %V._prelude_.Result_i64_std.prelude.Panic_ %t77, 0
  %t79 = icmp eq i32 %t78, 1
  br i1 %t79, label %when.bind.21, label %when.arm.19
when.bind.21:
  %t80 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55, i32 0, i32 1
  %t81 = getelementptr inbounds %S._prelude_.Err_i64_std.prelude.Panic_, ptr %t80, i32 0, i32 0
  %t82 = load %S.std.prelude.Panic, ptr %t81
  store %S.std.prelude.Panic %t82, ptr %a83
  br label %when.body.20
when.body.20:
  %t84 = load %str, ptr %a1
  %t85 = getelementptr inbounds %S.std.prelude.Panic, ptr %a83, i32 0, i32 0
  %t86 = load %str, ptr %t85
  %t89 = getelementptr [3 x %str], ptr %a88, i64 0, i64 0
  store %str %t84, ptr %t89
  %t90 = getelementptr [3 x %str], ptr %a88, i64 0, i64 1
  store %str { ptr @.str.36, i64 2 }, ptr %t90
  %t91 = getelementptr [3 x %str], ptr %a88, i64 0, i64 2
  store %str %t86, ptr %t91
  call void @veles_string_concat_n(ptr %a87, ptr %a88, i64 3)
  %t92 = load %str, ptr %a87
  %t93 = extractvalue %str { ptr @.str.37, i64 21 }, 0
  %t94 = extractvalue %str { ptr @.str.37, i64 21 }, 1
  call void @veles_call_push(ptr %t93)
  call void @v_std.io.println(%str %t92)
  call void @veles_call_pop()
  br label %when.end.14
when.arm.19:
  %t95 = extractvalue %str { ptr @.str.38, i64 33 }, 0
  %t96 = extractvalue %str { ptr @.str.38, i64 33 }, 1
  %t97 = extractvalue %str { ptr @.str.39, i64 12 }, 0
  %t98 = extractvalue %str { ptr @.str.39, i64 12 }, 1
  call void @veles_panic_at(ptr %t95, i64 %t96, ptr %t97, i64 %t98)
  unreachable
when.end.14:
  call void @veles_task_finish(ptr %task, ptr null, i64 0, i64 0)
  br label %coro.final
coro.final:
  %coro.fs = call i8 @llvm.coro.suspend(token none, i1 true)
  switch i8 %coro.fs, label %coro.suspend [ i8 0, label %coro.trap i8 1, label %coro.cleanup ]
coro.trap:
  unreachable
coro.cleanup:
  %coro.freed = call ptr @llvm.coro.free(token %coro.id, ptr %coro.hdl)
  br label %coro.suspend
coro.suspend:
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false, token none)
  ret ptr %coro.hdl
}

define ptr @v_main.main(ptr %task) presplitcoroutine {
entry:
  %a110 = alloca [21 x i8]
  %a114 = alloca %str
  %a126 = alloca [21 x i8]
  %a134 = alloca [21 x i8]
  %a138 = alloca %str
  %a139 = alloca [4 x %str]
  %a150 = alloca [21 x i8]
  %a154 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t2 = insertvalue { ptr, ptr } %t1, ptr null, 1
  %t3 = extractvalue %str { ptr @.str.40, i64 20 }, 0
  %t4 = extractvalue %str { ptr @.str.40, i64 20 }, 1
  call void @veles_call_push(ptr %t3)
  %t5 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t5)
  %t6 = call ptr @veles_alloc_words(i64 40)
  %t7 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t6, i32 0, i32 0
  store %str { ptr @.str.41, i64 12 }, ptr %t7
  %t8 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t6, i32 0, i32 1
  store { ptr, ptr } %t2, ptr %t8
  call void @veles_task_start(ptr %t5, ptr @entry.v_main.attempt, ptr %t6)
  br label %await.1
await.1:
  %t9 = call i64 @veles_task_await(ptr %task, ptr %t5)
  %t10 = icmp ne i64 %t9, 0
  br i1 %t10, label %await.got.2, label %await.susp.3
await.susp.3:
  %t11 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t11, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t12 = call i64 @veles_task_cancelled(ptr %task)
  %t13 = icmp ne i64 %t12, 0
  br i1 %t13, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.6:
  br label %await.1
await.got.2:
  %t14 = call i64 @veles_task_panicked(ptr %t5)
  %t15 = icmp ne i64 %t14, 0
  br i1 %t15, label %await.repanic.7, label %await.fine.8
await.repanic.7:
  call void @veles_task_repanic(ptr %t5)
  unreachable
await.fine.8:
  call void @veles_call_pop()
  %t16 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2, 0
  %t17 = insertvalue { ptr, ptr } %t16, ptr null, 1
  %t18 = extractvalue %str { ptr @.str.42, i64 20 }, 0
  %t19 = extractvalue %str { ptr @.str.42, i64 20 }, 1
  call void @veles_call_push(ptr %t18)
  %t20 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t20)
  %t21 = call ptr @veles_alloc_words(i64 40)
  %t22 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t21, i32 0, i32 0
  store %str { ptr @.str.43, i64 10 }, ptr %t22
  %t23 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t21, i32 0, i32 1
  store { ptr, ptr } %t17, ptr %t23
  call void @veles_task_start(ptr %t20, ptr @entry.v_main.attempt, ptr %t21)
  br label %await.9
await.9:
  %t24 = call i64 @veles_task_await(ptr %task, ptr %t20)
  %t25 = icmp ne i64 %t24, 0
  br i1 %t25, label %await.got.10, label %await.susp.11
await.susp.11:
  %t26 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t26, label %coro.suspend [ i8 0, label %resume.12 i8 1, label %coro.cleanup ]
resume.12:
  %t27 = call i64 @veles_task_cancelled(ptr %task)
  %t28 = icmp ne i64 %t27, 0
  br i1 %t28, label %cancelled.13, label %cont.14
cancelled.13:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.14:
  br label %await.9
await.got.10:
  %t29 = call i64 @veles_task_panicked(ptr %t20)
  %t30 = icmp ne i64 %t29, 0
  br i1 %t30, label %await.repanic.15, label %await.fine.16
await.repanic.15:
  call void @veles_task_repanic(ptr %t20)
  unreachable
await.fine.16:
  call void @veles_call_pop()
  %t31 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda3, 0
  %t32 = insertvalue { ptr, ptr } %t31, ptr null, 1
  %t33 = extractvalue %str { ptr @.str.44, i64 20 }, 0
  %t34 = extractvalue %str { ptr @.str.44, i64 20 }, 1
  call void @veles_call_push(ptr %t33)
  %t35 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t35)
  %t36 = call ptr @veles_alloc_words(i64 40)
  %t37 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t36, i32 0, i32 0
  store %str { ptr @.str.45, i64 8 }, ptr %t37
  %t38 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t36, i32 0, i32 1
  store { ptr, ptr } %t32, ptr %t38
  call void @veles_task_start(ptr %t35, ptr @entry.v_main.attempt, ptr %t36)
  br label %await.17
await.17:
  %t39 = call i64 @veles_task_await(ptr %task, ptr %t35)
  %t40 = icmp ne i64 %t39, 0
  br i1 %t40, label %await.got.18, label %await.susp.19
await.susp.19:
  %t41 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t41, label %coro.suspend [ i8 0, label %resume.20 i8 1, label %coro.cleanup ]
resume.20:
  %t42 = call i64 @veles_task_cancelled(ptr %task)
  %t43 = icmp ne i64 %t42, 0
  br i1 %t43, label %cancelled.21, label %cont.22
cancelled.21:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.22:
  br label %await.17
await.got.18:
  %t44 = call i64 @veles_task_panicked(ptr %t35)
  %t45 = icmp ne i64 %t44, 0
  br i1 %t45, label %await.repanic.23, label %await.fine.24
await.repanic.23:
  call void @veles_task_repanic(ptr %t35)
  unreachable
await.fine.24:
  call void @veles_call_pop()
  %t46 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t47 = insertvalue { ptr, ptr } %t46, ptr null, 1
  %t48 = extractvalue %str { ptr @.str.46, i64 20 }, 0
  %t49 = extractvalue %str { ptr @.str.46, i64 20 }, 1
  call void @veles_call_push(ptr %t48)
  %t50 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t50)
  %t51 = call ptr @veles_alloc_words(i64 40)
  %t52 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t51, i32 0, i32 0
  store %str { ptr @.str.47, i64 10 }, ptr %t52
  %t53 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t51, i32 0, i32 1
  store { ptr, ptr } %t47, ptr %t53
  call void @veles_task_start(ptr %t50, ptr @entry.v_main.attempt, ptr %t51)
  br label %await.25
await.25:
  %t54 = call i64 @veles_task_await(ptr %task, ptr %t50)
  %t55 = icmp ne i64 %t54, 0
  br i1 %t55, label %await.got.26, label %await.susp.27
await.susp.27:
  %t56 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t56, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  %t57 = call i64 @veles_task_cancelled(ptr %task)
  %t58 = icmp ne i64 %t57, 0
  br i1 %t58, label %cancelled.29, label %cont.30
cancelled.29:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.30:
  br label %await.25
await.got.26:
  %t59 = call i64 @veles_task_panicked(ptr %t50)
  %t60 = icmp ne i64 %t59, 0
  br i1 %t60, label %await.repanic.31, label %await.fine.32
await.repanic.31:
  call void @veles_task_repanic(ptr %t50)
  unreachable
await.fine.32:
  call void @veles_call_pop()
  %t61 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda5, 0
  %t62 = insertvalue { ptr, ptr } %t61, ptr null, 1
  %t63 = extractvalue %str { ptr @.str.48, i64 20 }, 0
  %t64 = extractvalue %str { ptr @.str.48, i64 20 }, 1
  call void @veles_call_push(ptr %t63)
  %t65 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t65)
  %t66 = call ptr @veles_alloc_words(i64 40)
  %t67 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t66, i32 0, i32 0
  store %str { ptr @.str.49, i64 8 }, ptr %t67
  %t68 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t66, i32 0, i32 1
  store { ptr, ptr } %t62, ptr %t68
  call void @veles_task_start(ptr %t65, ptr @entry.v_main.attempt, ptr %t66)
  br label %await.33
await.33:
  %t69 = call i64 @veles_task_await(ptr %task, ptr %t65)
  %t70 = icmp ne i64 %t69, 0
  br i1 %t70, label %await.got.34, label %await.susp.35
await.susp.35:
  %t71 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t71, label %coro.suspend [ i8 0, label %resume.36 i8 1, label %coro.cleanup ]
resume.36:
  %t72 = call i64 @veles_task_cancelled(ptr %task)
  %t73 = icmp ne i64 %t72, 0
  br i1 %t73, label %cancelled.37, label %cont.38
cancelled.37:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.38:
  br label %await.33
await.got.34:
  %t74 = call i64 @veles_task_panicked(ptr %t65)
  %t75 = icmp ne i64 %t74, 0
  br i1 %t75, label %await.repanic.39, label %await.fine.40
await.repanic.39:
  call void @veles_task_repanic(ptr %t65)
  unreachable
await.fine.40:
  call void @veles_call_pop()
  %t76 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda6, 0
  %t77 = insertvalue { ptr, ptr } %t76, ptr null, 1
  %t78 = extractvalue %str { ptr @.str.50, i64 20 }, 0
  %t79 = extractvalue %str { ptr @.str.50, i64 20 }, 1
  call void @veles_call_push(ptr %t78)
  %t80 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t80)
  %t81 = call ptr @veles_alloc_words(i64 40)
  %t82 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t81, i32 0, i32 0
  store %str { ptr @.str.51, i64 15 }, ptr %t82
  %t83 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t81, i32 0, i32 1
  store { ptr, ptr } %t77, ptr %t83
  call void @veles_task_start(ptr %t80, ptr @entry.v_main.attempt, ptr %t81)
  br label %await.41
await.41:
  %t84 = call i64 @veles_task_await(ptr %task, ptr %t80)
  %t85 = icmp ne i64 %t84, 0
  br i1 %t85, label %await.got.42, label %await.susp.43
await.susp.43:
  %t86 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t86, label %coro.suspend [ i8 0, label %resume.44 i8 1, label %coro.cleanup ]
resume.44:
  %t87 = call i64 @veles_task_cancelled(ptr %task)
  %t88 = icmp ne i64 %t87, 0
  br i1 %t88, label %cancelled.45, label %cont.46
cancelled.45:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.46:
  br label %await.41
await.got.42:
  %t89 = call i64 @veles_task_panicked(ptr %t80)
  %t90 = icmp ne i64 %t89, 0
  br i1 %t90, label %await.repanic.47, label %await.fine.48
await.repanic.47:
  call void @veles_task_repanic(ptr %t80)
  unreachable
await.fine.48:
  call void @veles_call_pop()
  %t91 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda7, 0
  %t92 = insertvalue { ptr, ptr } %t91, ptr null, 1
  %t93 = extractvalue %str { ptr @.str.52, i64 20 }, 0
  %t94 = extractvalue %str { ptr @.str.52, i64 20 }, 1
  call void @veles_call_push(ptr %t93)
  %t95 = call ptr @veles_task_new()
  call void @veles_call_link(ptr %t95)
  %t96 = call ptr @veles_alloc_words(i64 40)
  %t97 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t96, i32 0, i32 0
  store %str { ptr @.str.53, i64 18 }, ptr %t97
  %t98 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t96, i32 0, i32 1
  store { ptr, ptr } %t92, ptr %t98
  call void @veles_task_start(ptr %t95, ptr @entry.v_main.attempt, ptr %t96)
  br label %await.49
await.49:
  %t99 = call i64 @veles_task_await(ptr %task, ptr %t95)
  %t100 = icmp ne i64 %t99, 0
  br i1 %t100, label %await.got.50, label %await.susp.51
await.susp.51:
  %t101 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t101, label %coro.suspend [ i8 0, label %resume.52 i8 1, label %coro.cleanup ]
resume.52:
  %t102 = call i64 @veles_task_cancelled(ptr %task)
  %t103 = icmp ne i64 %t102, 0
  br i1 %t103, label %cancelled.53, label %cont.54
cancelled.53:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.54:
  br label %await.49
await.got.50:
  %t104 = call i64 @veles_task_panicked(ptr %t95)
  %t105 = icmp ne i64 %t104, 0
  br i1 %t105, label %await.repanic.55, label %await.fine.56
await.repanic.55:
  call void @veles_task_repanic(ptr %t95)
  unreachable
await.fine.56:
  call void @veles_call_pop()
  %t106 = extractvalue %str { ptr @.str.54, i64 21 }, 0
  %t107 = extractvalue %str { ptr @.str.54, i64 21 }, 1
  call void @veles_call_push(ptr %t106)
  %t108 = call i8 @v_main.wrapped(i8 127, i8 1)
  call void @veles_call_pop()
  %t109 = sext i8 %t108 to i64
  %t111 = call i64 @veles_i64_format(ptr %a110, i64 %t109)
  %t112 = insertvalue %str undef, ptr %a110, 0
  %t113 = insertvalue %str %t112, i64 %t111, 1
  %t115 = extractvalue %str { ptr @.str.55, i64 16 }, 0
  %t116 = extractvalue %str { ptr @.str.55, i64 16 }, 1
  %t117 = extractvalue %str %t113, 0
  %t118 = extractvalue %str %t113, 1
  call void @veles_string_concat(ptr %a114, ptr %t115, i64 %t116, ptr %t117, i64 %t118)
  %t119 = load %str, ptr %a114
  %t120 = extractvalue %str { ptr @.str.56, i64 20 }, 0
  %t121 = extractvalue %str { ptr @.str.56, i64 20 }, 1
  call void @veles_call_push(ptr %t120)
  call void @v_std.io.println(%str %t119)
  call void @veles_call_pop()
  %t122 = extractvalue %str { ptr @.str.57, i64 20 }, 0
  %t123 = extractvalue %str { ptr @.str.57, i64 20 }, 1
  call void @veles_call_push(ptr %t122)
  %t124 = call i8 @v_main.narrow(i64 300)
  call void @veles_call_pop()
  %t125 = zext i8 %t124 to i64
  %t127 = call i64 @veles_u64_format(ptr %a126, i64 %t125)
  %t128 = insertvalue %str undef, ptr %a126, 0
  %t129 = insertvalue %str %t128, i64 %t127, 1
  %t130 = extractvalue %str { ptr @.str.58, i64 20 }, 0
  %t131 = extractvalue %str { ptr @.str.58, i64 20 }, 1
  call void @veles_call_push(ptr %t130)
  %t132 = call i8 @v_main.narrow(i64 -1)
  call void @veles_call_pop()
  %t133 = zext i8 %t132 to i64
  %t135 = call i64 @veles_u64_format(ptr %a134, i64 %t133)
  %t136 = insertvalue %str undef, ptr %a134, 0
  %t137 = insertvalue %str %t136, i64 %t135, 1
  %t140 = getelementptr [4 x %str], ptr %a139, i64 0, i64 0
  store %str { ptr @.str.59, i64 12 }, ptr %t140
  %t141 = getelementptr [4 x %str], ptr %a139, i64 0, i64 1
  store %str %t129, ptr %t141
  %t142 = getelementptr [4 x %str], ptr %a139, i64 0, i64 2
  store %str { ptr @.str.60, i64 13 }, ptr %t142
  %t143 = getelementptr [4 x %str], ptr %a139, i64 0, i64 3
  store %str %t137, ptr %t143
  call void @veles_string_concat_n(ptr %a138, ptr %a139, i64 4)
  %t144 = load %str, ptr %a138
  %t145 = extractvalue %str { ptr @.str.61, i64 20 }, 0
  %t146 = extractvalue %str { ptr @.str.61, i64 20 }, 1
  call void @veles_call_push(ptr %t145)
  call void @v_std.io.println(%str %t144)
  call void @veles_call_pop()
  %t147 = extractvalue %str { ptr @.str.62, i64 21 }, 0
  %t148 = extractvalue %str { ptr @.str.62, i64 21 }, 1
  call void @veles_call_push(ptr %t147)
  %t149 = call i64 @v_main.sumUpTo(i8 255)
  call void @veles_call_pop()
  %t151 = call i64 @veles_i64_format(ptr %a150, i64 %t149)
  %t152 = insertvalue %str undef, ptr %a150, 0
  %t153 = insertvalue %str %t152, i64 %t151, 1
  %t155 = extractvalue %str { ptr @.str.63, i64 23 }, 0
  %t156 = extractvalue %str { ptr @.str.63, i64 23 }, 1
  %t157 = extractvalue %str %t153, 0
  %t158 = extractvalue %str %t153, 1
  call void @veles_string_concat(ptr %a154, ptr %t155, i64 %t156, ptr %t157, i64 %t158)
  %t159 = load %str, ptr %a154
  %t160 = extractvalue %str { ptr @.str.64, i64 20 }, 0
  %t161 = extractvalue %str { ptr @.str.64, i64 20 }, 1
  call void @veles_call_push(ptr %t160)
  call void @v_std.io.println(%str %t159)
  call void @veles_call_pop()
  %t162 = extractvalue %str { ptr @.str.65, i64 18 }, 0
  %t163 = extractvalue %str { ptr @.str.65, i64 18 }, 1
  call void @veles_call_push(ptr %t162)
  call void @v_main.edges(i8 -128, i8 127, i8 255)
  call void @veles_call_pop()
  call void @veles_task_finish(ptr %task, ptr null, i64 0, i64 0)
  br label %coro.final
coro.final:
  %coro.fs = call i8 @llvm.coro.suspend(token none, i1 true)
  switch i8 %coro.fs, label %coro.suspend [ i8 0, label %coro.trap i8 1, label %coro.cleanup ]
coro.trap:
  unreachable
coro.cleanup:
  %coro.freed = call ptr @llvm.coro.free(token %coro.id, ptr %coro.hdl)
  br label %coro.suspend
coro.suspend:
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false, token none)
  ret ptr %coro.hdl
}

define i64 @v_main.main.lambda1(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = extractvalue %str { ptr @.str.66, i64 17 }, 0
  %t3 = extractvalue %str { ptr @.str.66, i64 17 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call i8 @v_main.add(i8 127, i8 1)
  call void @veles_call_pop()
  %t5 = sext i8 %t4 to i64
  ret i64 %t5
}

define i64 @v_main.main.lambda2(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = extractvalue %str { ptr @.str.67, i64 17 }, 0
  %t3 = extractvalue %str { ptr @.str.67, i64 17 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call i8 @v_main.sub(i8 0, i8 1)
  call void @veles_call_pop()
  %t5 = zext i8 %t4 to i64
  ret i64 %t5
}

define i64 @v_main.main.lambda3(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t3 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 -9223372036854775807, i64 1)
  %t4 = extractvalue { i64, i1 } %t3, 0
  %t5 = extractvalue { i64, i1 } %t3, 1
  br i1 %t5, label %overflow.1, label %arith.ok.2
overflow.1:
  %t6 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t8 = extractvalue %str { ptr @.str.68, i64 13 }, 0
  %t9 = extractvalue %str { ptr @.str.68, i64 13 }, 1
  call void @veles_panic_at(ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  unreachable
arith.ok.2:
  %t10 = extractvalue %str { ptr @.str.69, i64 17 }, 0
  %t11 = extractvalue %str { ptr @.str.69, i64 17 }, 1
  call void @veles_call_push(ptr %t10)
  %t12 = call i64 @v_main.mul(i64 %t4, i64 -1)
  call void @veles_call_pop()
  ret i64 %t12
}

define i64 @v_main.main.lambda4(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t3 = call { i32, i1 } @llvm.ssub.with.overflow.i32(i32 -2147483647, i32 1)
  %t4 = extractvalue { i32, i1 } %t3, 0
  %t5 = extractvalue { i32, i1 } %t3, 1
  br i1 %t5, label %overflow.1, label %arith.ok.2
overflow.1:
  %t6 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t8 = extractvalue %str { ptr @.str.70, i64 13 }, 0
  %t9 = extractvalue %str { ptr @.str.70, i64 13 }, 1
  call void @veles_panic_at(ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  unreachable
arith.ok.2:
  %t10 = extractvalue %str { ptr @.str.71, i64 17 }, 0
  %t11 = extractvalue %str { ptr @.str.71, i64 17 }, 1
  call void @veles_call_push(ptr %t10)
  %t12 = call i32 @v_main.neg(i32 %t4)
  call void @veles_call_pop()
  %t13 = sext i32 %t12 to i64
  ret i64 %t13
}

define i64 @v_main.main.lambda5(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t3 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 -9223372036854775807, i64 1)
  %t4 = extractvalue { i64, i1 } %t3, 0
  %t5 = extractvalue { i64, i1 } %t3, 1
  br i1 %t5, label %overflow.1, label %arith.ok.2
overflow.1:
  %t6 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t8 = extractvalue %str { ptr @.str.72, i64 13 }, 0
  %t9 = extractvalue %str { ptr @.str.72, i64 13 }, 1
  call void @veles_panic_at(ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  unreachable
arith.ok.2:
  %t10 = extractvalue %str { ptr @.str.73, i64 18 }, 0
  %t11 = extractvalue %str { ptr @.str.73, i64 18 }, 1
  call void @veles_call_push(ptr %t10)
  %t12 = call i64 @v_main.quot(i64 %t4, i64 -1)
  call void @veles_call_pop()
  ret i64 %t12
}

define i64 @v_main.main.lambda6(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = extractvalue %str { ptr @.str.74, i64 22 }, 0
  %t3 = extractvalue %str { ptr @.str.74, i64 22 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call i16 @v_main.absolute(i16 -32768)
  call void @veles_call_pop()
  %t5 = sext i16 %t4 to i64
  ret i64 %t5
}

define i64 @v_main.main.lambda7(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = extractvalue %str { ptr @.str.75, i64 18 }, 0
  %t3 = extractvalue %str { ptr @.str.75, i64 18 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call i8 @v_main.bump(i8 127)
  call void @veles_call_pop()
  %t5 = sext i8 %t4 to i64
  ret i64 %t5
}

define internal ptr @ramp.v_main.call(ptr %task, { ptr, ptr } %p0) presplitcoroutine {
entry:
  %a2 = alloca i64
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = call i64 @v_main.call({ ptr, ptr } %p0)
  store i64 %t1, ptr %a2
  call void @veles_task_finish(ptr %task, ptr %a2, i64 8, i64 0)
  br label %coro.final
coro.final:
  %coro.fs = call i8 @llvm.coro.suspend(token none, i1 true)
  switch i8 %coro.fs, label %coro.suspend [ i8 0, label %coro.trap i8 1, label %coro.cleanup ]
coro.trap:
  unreachable
coro.cleanup:
  %coro.freed = call ptr @llvm.coro.free(token %coro.id, ptr %coro.hdl)
  br label %coro.suspend
coro.suspend:
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false, token none)
  ret ptr %coro.hdl
}

define internal void @entry.ramp.v_main.call(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { { ptr, ptr } }, ptr %args, i32 0, i32 0
  %t2 = load { ptr, ptr }, ptr %t1
  %t3 = extractvalue %str { ptr @.str.76, i64 5 }, 0
  %t4 = extractvalue %str { ptr @.str.76, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t3)
  %t5 = call ptr @ramp.v_main.call(ptr %task, { ptr, ptr } %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.attempt(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %args, i32 0, i32 0
  %t2 = load %str, ptr %t1
  %t3 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %args, i32 0, i32 1
  %t4 = load { ptr, ptr }, ptr %t3
  %t5 = extractvalue %str { ptr @.str.77, i64 8 }, 0
  %t6 = extractvalue %str { ptr @.str.77, i64 8 }, 1
  call void @veles_call_base(ptr %task, ptr %t5)
  %t7 = call ptr @v_main.attempt(ptr %task, %str %t2, { ptr, ptr } %t4)
  call void @veles_task_started(ptr %task, ptr %t7)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.78, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.78, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:8:29\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:9:29\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:10:32\00"
@.str.5 = private unnamed_addr constant [14 x i8] c"main.vs:11:24\00"
@.str.6 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:12:33\00"
@.str.8 = private unnamed_addr constant [14 x i8] c"main.vs:13:29\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:16:3\00"
@.str.10 = private unnamed_addr constant [13 x i8] c"main.vs:25:5\00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:33:44\00"
@.str.12 = private unnamed_addr constant [19 x i8] c"main.vs:33:42\00iter\00"
@.str.13 = private unnamed_addr constant [20 x i8] c"main.vs:33:42\00count\00"
@.str.14 = private unnamed_addr constant [26 x i8] c"(250..255).iter() yields \00"
@.str.15 = private unnamed_addr constant [21 x i8] c"main.vs:33:3\00println\00"
@.str.16 = private unnamed_addr constant [18 x i8] c"main.vs:34:37\00len\00"
@.str.17 = private unnamed_addr constant [18 x i8] c"main.vs:34:75\00len\00"
@.str.18 = private unnamed_addr constant [21 x i8] c"(-128..127).len() = \00"
@.str.19 = private unnamed_addr constant [22 x i8] c", (0..<-128).len() = \00"
@.str.20 = private unnamed_addr constant [21 x i8] c"main.vs:34:3\00println\00"
@.str.21 = private unnamed_addr constant [19 x i8] c"main.vs:35:41\00step\00"
@.str.22 = private unnamed_addr constant [21 x i8] c"main.vs:35:41\00toList\00"
@.str.23 = private unnamed_addr constant [25 x i8] c"(-128..127).step(100) = \00"
@.str.24 = private unnamed_addr constant [21 x i8] c"main.vs:35:3\00println\00"
@.str.25 = private unnamed_addr constant [23 x i8] c"main.vs:36:38\00reversed\00"
@.str.26 = private unnamed_addr constant [19 x i8] c"main.vs:36:38\00step\00"
@.str.27 = private unnamed_addr constant [21 x i8] c"main.vs:36:38\00toList\00"
@.str.28 = private unnamed_addr constant [20 x i8] c"main.vs:36:102\00step\00"
@.str.29 = private unnamed_addr constant [24 x i8] c"main.vs:36:102\00reversed\00"
@.str.30 = private unnamed_addr constant [22 x i8] c"main.vs:36:102\00toList\00"
@.str.31 = private unnamed_addr constant [22 x i8] c"reversed then step = \00"
@.str.32 = private unnamed_addr constant [24 x i8] c", step then reversed = \00"
@.str.33 = private unnamed_addr constant [21 x i8] c"main.vs:36:3\00println\00"
@.str.34 = private unnamed_addr constant [4 x i8] c" = \00"
@.str.35 = private unnamed_addr constant [22 x i8] c"main.vs:46:18\00println\00"
@.str.36 = private unnamed_addr constant [3 x i8] c": \00"
@.str.37 = private unnamed_addr constant [22 x i8] c"main.vs:47:18\00println\00"
@.str.38 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.39 = private unnamed_addr constant [13 x i8] c"main.vs:45:3\00"
@.str.40 = private unnamed_addr constant [21 x i8] c"main.vs:52:3\00attempt\00"
@.str.41 = private unnamed_addr constant [13 x i8] c"127 + 1 (i8)\00"
@.str.42 = private unnamed_addr constant [21 x i8] c"main.vs:53:3\00attempt\00"
@.str.43 = private unnamed_addr constant [11 x i8] c"0 - 1 (u8)\00"
@.str.44 = private unnamed_addr constant [21 x i8] c"main.vs:54:3\00attempt\00"
@.str.45 = private unnamed_addr constant [9 x i8] c"MIN * -1\00"
@.str.46 = private unnamed_addr constant [21 x i8] c"main.vs:55:3\00attempt\00"
@.str.47 = private unnamed_addr constant [11 x i8] c"-MIN (i32)\00"
@.str.48 = private unnamed_addr constant [21 x i8] c"main.vs:56:3\00attempt\00"
@.str.49 = private unnamed_addr constant [9 x i8] c"MIN / -1\00"
@.str.50 = private unnamed_addr constant [21 x i8] c"main.vs:57:3\00attempt\00"
@.str.51 = private unnamed_addr constant [16 x i8] c"MIN.abs() (i16)\00"
@.str.52 = private unnamed_addr constant [21 x i8] c"main.vs:58:3\00attempt\00"
@.str.53 = private unnamed_addr constant [19 x i8] c"x += 1 at 127 (i8)\00"
@.str.54 = private unnamed_addr constant [22 x i8] c"main.vs:59:33\00wrapped\00"
@.str.55 = private unnamed_addr constant [17 x i8] c"127 +% 1 (i8) = \00"
@.str.56 = private unnamed_addr constant [21 x i8] c"main.vs:59:3\00println\00"
@.str.57 = private unnamed_addr constant [21 x i8] c"main.vs:60:29\00narrow\00"
@.str.58 = private unnamed_addr constant [21 x i8] c"main.vs:60:56\00narrow\00"
@.str.59 = private unnamed_addr constant [13 x i8] c"300 as u8 = \00"
@.str.60 = private unnamed_addr constant [14 x i8] c", -1 as u8 = \00"
@.str.61 = private unnamed_addr constant [21 x i8] c"main.vs:60:3\00println\00"
@.str.62 = private unnamed_addr constant [22 x i8] c"main.vs:61:40\00sumUpTo\00"
@.str.63 = private unnamed_addr constant [24 x i8] c"sum of 250..255 (u8) = \00"
@.str.64 = private unnamed_addr constant [21 x i8] c"main.vs:61:3\00println\00"
@.str.65 = private unnamed_addr constant [19 x i8] c"main.vs:62:3\00edges\00"
@.str.66 = private unnamed_addr constant [18 x i8] c"main.vs:52:34\00add\00"
@.str.67 = private unnamed_addr constant [18 x i8] c"main.vs:53:32\00sub\00"
@.str.68 = private unnamed_addr constant [14 x i8] c"main.vs:54:33\00"
@.str.69 = private unnamed_addr constant [18 x i8] c"main.vs:54:29\00mul\00"
@.str.70 = private unnamed_addr constant [14 x i8] c"main.vs:55:36\00"
@.str.71 = private unnamed_addr constant [18 x i8] c"main.vs:55:32\00neg\00"
@.str.72 = private unnamed_addr constant [14 x i8] c"main.vs:56:34\00"
@.str.73 = private unnamed_addr constant [19 x i8] c"main.vs:56:29\00quot\00"
@.str.74 = private unnamed_addr constant [23 x i8] c"main.vs:57:37\00absolute\00"
@.str.75 = private unnamed_addr constant [19 x i8] c"main.vs:58:39\00bump\00"
@.str.76 = private unnamed_addr constant [6 x i8] c"\00call\00"
@.str.77 = private unnamed_addr constant [9 x i8] c"\00attempt\00"
@.str.78 = private unnamed_addr constant [6 x i8] c"\00main\00"
