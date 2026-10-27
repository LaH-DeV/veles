%V.main.Token = type { i32, [3 x i64] }
%S.main.Number = type { i64, i64 }
%S.main.Word = type { i64, %str }
%S.main.Names = type { ptr }
%S.main.Malformed = type { %str }
%U.main.Malformed_main.Overflow = type { i32, [2 x i64] }
%S._prelude_.Err_i64_main.Malformed_main.Overflow_ = type { %U.main.Malformed_main.Overflow }
%V._prelude_.Result_i64_main.Malformed_main.Overflow_ = type { i32, [3 x i64] }
%S.main.Overflow = type { i64, i64 }
%S._prelude_.Ok_i64_main.Malformed_main.Overflow_ = type { i64 }
%S.main.Failure = type { %U.main.Malformed_main.Overflow }
@adesc.main.Token = internal constant { i64, i64, i64, [2 x i64] } { i64 32, i64 1, i64 2, [2 x i64] [i64 8, i64 16] }
@adesc.main.Failure = internal constant { i64, i64, i64, [1 x i64] } { i64 24, i64 1, i64 1, [1 x i64] [i64 8] }
define i64 @v_main.weigh(%V.main.Token %p1) {
entry:
  %a1 = alloca %V.main.Token
  %a2 = alloca i64
  %a4 = alloca %V.main.Token
  %a11 = alloca i64
  %a19 = alloca %str
  %a28 = alloca ptr
  %a30 = alloca ptr
  %a36 = alloca ptr
  store %V.main.Token %p1, ptr %a1
  %t3 = load %V.main.Token, ptr %a1
  store %V.main.Token %t3, ptr %a4
  br label %when.arm.2
when.arm.2:
  %t5 = load %V.main.Token, ptr %a4
  %t6 = extractvalue %V.main.Token %t5, 0
  %t7 = icmp eq i32 %t6, 0
  br i1 %t7, label %when.bind.5, label %when.arm.3
when.bind.5:
  %t8 = getelementptr inbounds %V.main.Token, ptr %a4, i32 0, i32 1
  %t9 = getelementptr inbounds %S.main.Number, ptr %t8, i32 0, i32 0
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a11
  br label %when.body.4
when.body.4:
  %t12 = load i64, ptr %a11
  store i64 %t12, ptr %a2
  br label %when.end.1
when.arm.3:
  %t13 = load %V.main.Token, ptr %a4
  %t14 = extractvalue %V.main.Token %t13, 0
  %t15 = icmp eq i32 %t14, 1
  br i1 %t15, label %when.bind.8, label %when.arm.6
when.bind.8:
  %t16 = getelementptr inbounds %V.main.Token, ptr %a4, i32 0, i32 1
  %t17 = getelementptr inbounds %S.main.Word, ptr %t16, i32 0, i32 1
  %t18 = load %str, ptr %t17
  store %str %t18, ptr %a19
  br label %when.body.7
when.body.7:
  %t20 = load %str, ptr %a19
  %t21 = extractvalue %str %t20, 1
  store i64 %t21, ptr %a2
  br label %when.end.1
when.arm.6:
  %t22 = load %V.main.Token, ptr %a4
  %t23 = extractvalue %V.main.Token %t22, 0
  %t24 = icmp eq i32 %t23, 2
  br i1 %t24, label %when.bind.11, label %when.arm.9
when.bind.11:
  %t25 = getelementptr inbounds %V.main.Token, ptr %a4, i32 0, i32 1
  %t26 = getelementptr inbounds %S.main.Names, ptr %t25, i32 0, i32 0
  %t27 = load ptr, ptr %t26
  store ptr %t27, ptr %a28
  br label %when.body.10
when.body.10:
  %t29 = load ptr, ptr %a28
  store ptr %t29, ptr %a30
  %t31 = insertvalue { ptr, ptr } undef, ptr @v_main.weigh.lambda1.plain, 0
  %t32 = insertvalue { ptr, ptr } %t31, ptr null, 1
  %t33 = extractvalue %str { ptr @.str.1, i64 17 }, 0
  %t34 = extractvalue %str { ptr @.str.1, i64 17 }, 1
  call void @veles_call_push(ptr %t33)
  %t35 = call ptr @v_std.prelude.extend.List_T.map_T_string_i64_Never.plain(ptr %a30, { ptr, ptr } %t32)
  call void @veles_call_pop()
  store ptr %t35, ptr %a36
  %t37 = extractvalue %str { ptr @.str.2, i64 17 }, 0
  %t38 = extractvalue %str { ptr @.str.2, i64 17 }, 1
  call void @veles_call_push(ptr %t37)
  %t39 = call i64 @v_std.prelude.extend.List_i64.sum(ptr %a36)
  call void @veles_call_pop()
  store i64 %t39, ptr %a2
  br label %when.end.1
when.arm.9:
  %t40 = extractvalue %str { ptr @.str.3, i64 33 }, 0
  %t41 = extractvalue %str { ptr @.str.3, i64 33 }, 1
  %t42 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t43 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t40, i64 %t41, ptr %t42, i64 %t43)
  unreachable
when.end.1:
  %t44 = load i64, ptr %a2
  ret i64 %t44
}

define i64 @v_main.weigh.lambda1.plain(ptr %env, %str %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca %str
  store ptr %env, ptr %a1
  store %str %p1, ptr %a2
  %t3 = load %str, ptr %a2
  %t4 = extractvalue %str %t3, 1
  ret i64 %t4
}

define %V._prelude_.Result_i64_main.Malformed_main.Overflow_ @v_main.limit(i64 %p1) {
entry:
  %a1 = alloca i64
  %a5 = alloca %U.main.Malformed_main.Overflow
  %a10 = alloca %V._prelude_.Result_i64_main.Malformed_main.Overflow_
  %a19 = alloca %U.main.Malformed_main.Overflow
  %a24 = alloca %V._prelude_.Result_i64_main.Malformed_main.Overflow_
  %a30 = alloca %V._prelude_.Result_i64_main.Malformed_main.Overflow_
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = icmp slt i64 %t2, 0
  br i1 %t3, label %if.then.1, label %if.end.2
if.then.1:
  %t4 = insertvalue %S.main.Malformed undef, %str { ptr @.str.5, i64 8 }, 0
  store %U.main.Malformed_main.Overflow zeroinitializer, ptr %a5
  %t6 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a5, i32 0, i32 0
  store i32 0, ptr %t6
  %t7 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a5, i32 0, i32 1
  store %S.main.Malformed %t4, ptr %t7
  %t8 = load %U.main.Malformed_main.Overflow, ptr %a5
  %t9 = insertvalue %S._prelude_.Err_i64_main.Malformed_main.Overflow_ undef, %U.main.Malformed_main.Overflow %t8, 0
  store %V._prelude_.Result_i64_main.Malformed_main.Overflow_ zeroinitializer, ptr %a10
  %t11 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a10, i32 0, i32 0
  store i32 1, ptr %t11
  %t12 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a10, i32 0, i32 1
  store %S._prelude_.Err_i64_main.Malformed_main.Overflow_ %t9, ptr %t12
  %t13 = load %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a10
  ret %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t13
if.end.2:
  %t14 = load i64, ptr %a1
  %t15 = icmp sgt i64 %t14, 1000
  br i1 %t15, label %if.then.3, label %if.end.4
if.then.3:
  %t16 = load i64, ptr %a1
  %t17 = insertvalue %S.main.Overflow undef, i64 1000, 0
  %t18 = insertvalue %S.main.Overflow %t17, i64 %t16, 1
  store %U.main.Malformed_main.Overflow zeroinitializer, ptr %a19
  %t20 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a19, i32 0, i32 0
  store i32 1, ptr %t20
  %t21 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a19, i32 0, i32 1
  store %S.main.Overflow %t18, ptr %t21
  %t22 = load %U.main.Malformed_main.Overflow, ptr %a19
  %t23 = insertvalue %S._prelude_.Err_i64_main.Malformed_main.Overflow_ undef, %U.main.Malformed_main.Overflow %t22, 0
  store %V._prelude_.Result_i64_main.Malformed_main.Overflow_ zeroinitializer, ptr %a24
  %t25 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a24, i32 0, i32 0
  store i32 1, ptr %t25
  %t26 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a24, i32 0, i32 1
  store %S._prelude_.Err_i64_main.Malformed_main.Overflow_ %t23, ptr %t26
  %t27 = load %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a24
  ret %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t27
if.end.4:
  %t28 = load i64, ptr %a1
  %t29 = insertvalue %S._prelude_.Ok_i64_main.Malformed_main.Overflow_ undef, i64 %t28, 0
  store %V._prelude_.Result_i64_main.Malformed_main.Overflow_ zeroinitializer, ptr %a30
  %t31 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a30, i32 0, i32 0
  store i32 0, ptr %t31
  %t32 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a30, i32 0, i32 1
  store %S._prelude_.Ok_i64_main.Malformed_main.Overflow_ %t29, ptr %t32
  %t33 = load %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a30
  ret %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t33
}

define void @v_main.main() {
entry:
  %a2 = alloca ptr
  %a6 = alloca { i64, i64, i1 }
  %a9 = alloca i64
  %a12 = alloca i64
  %a13 = alloca i1
  %a14 = alloca i1
  %a16 = alloca i1
  %a28 = alloca i64
  %a34 = alloca %V.main.Token
  %a52 = alloca %V.main.Token
  %a56 = alloca [21 x i8]
  %a60 = alloca %str
  %a68 = alloca %V.main.Token
  %a86 = alloca %V.main.Token
  %a89 = alloca %str
  %a91 = alloca [21 x i8]
  %a95 = alloca %str
  %a102 = alloca %V.main.Token
  %a120 = alloca %V.main.Token
  %a121 = alloca i1
  %a133 = alloca ptr
  %a139 = alloca ptr
  %a143 = alloca [21 x i8]
  %a147 = alloca %str
  %a148 = alloca [1 x %str]
  %a153 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a156 = alloca ptr
  %a158 = alloca i64
  %a159 = alloca ptr
  %a160 = alloca i64
  %a176 = alloca i64
  %a181 = alloca %V._prelude_.Result_i64_main.Malformed_main.Overflow_
  %a182 = alloca i64
  %a189 = alloca %U.main.Malformed_main.Overflow
  %a191 = alloca %U.main.Malformed_main.Overflow
  %a196 = alloca %U.main.Malformed_main.Overflow
  %a200 = alloca [21 x i8]
  %a204 = alloca %str
  %a216 = alloca %U.main.Malformed_main.Overflow
  %a220 = alloca %str
  %a249 = alloca %S.main.Failure
  %a255 = alloca [21 x i8]
  %a259 = alloca %str
  %a274 = alloca [21 x i8]
  %a278 = alloca %str
  %t1 = call ptr @veles_list_new(ptr @adesc.main.Token, i64 0)
  store ptr %t1, ptr %a2
  %t3 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t4 = insertvalue { i64, i64, i1 } %t3, i64 200, 1
  %t5 = insertvalue { i64, i64, i1 } %t4, i1 true, 2
  store { i64, i64, i1 } %t5, ptr %a6
  %t7 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 0
  %t8 = load i64, ptr %t7
  store i64 %t8, ptr %a9
  %t10 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 1
  %t11 = load i64, ptr %t10
  store i64 %t11, ptr %a12
  store i1 false, ptr %a13
  br label %loop.cond.1
loop.cond.1:
  %t15 = load i1, ptr %a13
  br i1 %t15, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a14
  br label %if.end.6
if.else.7:
  %t17 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t18 = load i1, ptr %t17
  br i1 %t18, label %if.then.8, label %if.else.10
if.then.8:
  %t19 = load i64, ptr %a9
  %t20 = load i64, ptr %a12
  %t21 = icmp sle i64 %t19, %t20
  store i1 %t21, ptr %a16
  br label %if.end.9
if.else.10:
  %t22 = load i64, ptr %a9
  %t23 = load i64, ptr %a12
  %t24 = icmp slt i64 %t22, %t23
  store i1 %t24, ptr %a16
  br label %if.end.9
if.end.9:
  %t25 = load i1, ptr %a16
  store i1 %t25, ptr %a14
  br label %if.end.6
if.end.6:
  %t26 = load i1, ptr %a14
  br i1 %t26, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t27 = load i64, ptr %a9
  store i64 %t27, ptr %a28
  %t29 = load ptr, ptr %a2
  %t30 = load i64, ptr %a28
  %t31 = load i64, ptr %a28
  %t32 = insertvalue %S.main.Number undef, i64 %t30, 0
  %t33 = insertvalue %S.main.Number %t32, i64 %t31, 1
  store %V.main.Token zeroinitializer, ptr %a34
  %t35 = getelementptr inbounds %V.main.Token, ptr %a34, i32 0, i32 0
  store i32 0, ptr %t35
  %t36 = getelementptr inbounds %V.main.Token, ptr %a34, i32 0, i32 1
  store %S.main.Number %t33, ptr %t36
  %t37 = load %V.main.Token, ptr %a34
  %t38 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t29, i32 0, i32 1
  %t39 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t29, i32 0, i32 2
  %t40 = load i64, ptr %t38
  %t41 = load i64, ptr %t39
  %t42 = icmp slt i64 %t40, %t41
  br i1 %t42, label %push.fast.11, label %push.grow.12, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.11:
  %t43 = load ptr, ptr %t29
  %t44 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t29, i32 0, i32 3
  %t45 = load i64, ptr %t44
  %t46 = mul i64 %t45, %t40
  %t47 = getelementptr inbounds i8, ptr %t43, i64 %t46
  store %V.main.Token %t37, ptr %t47
  %t48 = add i64 %t40, 1
  store i64 %t48, ptr %t38
  %t49 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t29, i32 0, i32 5
  %t50 = load i64, ptr %t49
  %t51 = add i64 %t50, 1
  store i64 %t51, ptr %t49
  br label %push.done.13
push.grow.12:
  store %V.main.Token %t37, ptr %a52
  call void @veles_list_push(ptr %t29, ptr %a52)
  br label %push.done.13
push.done.13:
  %t53 = load ptr, ptr %a2
  %t54 = load i64, ptr %a28
  %t55 = load i64, ptr %a28
  %t57 = call i64 @veles_i64_format(ptr %a56, i64 %t55)
  %t58 = insertvalue %str undef, ptr %a56, 0
  %t59 = insertvalue %str %t58, i64 %t57, 1
  %t61 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t62 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  %t63 = extractvalue %str %t59, 0
  %t64 = extractvalue %str %t59, 1
  call void @veles_string_concat(ptr %a60, ptr %t61, i64 %t62, ptr %t63, i64 %t64)
  %t65 = load %str, ptr %a60
  %t66 = insertvalue %S.main.Word undef, i64 %t54, 0
  %t67 = insertvalue %S.main.Word %t66, %str %t65, 1
  store %V.main.Token zeroinitializer, ptr %a68
  %t69 = getelementptr inbounds %V.main.Token, ptr %a68, i32 0, i32 0
  store i32 1, ptr %t69
  %t70 = getelementptr inbounds %V.main.Token, ptr %a68, i32 0, i32 1
  store %S.main.Word %t67, ptr %t70
  %t71 = load %V.main.Token, ptr %a68
  %t72 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 1
  %t73 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 2
  %t74 = load i64, ptr %t72
  %t75 = load i64, ptr %t73
  %t76 = icmp slt i64 %t74, %t75
  br i1 %t76, label %push.fast.14, label %push.grow.15, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.14:
  %t77 = load ptr, ptr %t53
  %t78 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 3
  %t79 = load i64, ptr %t78
  %t80 = mul i64 %t79, %t74
  %t81 = getelementptr inbounds i8, ptr %t77, i64 %t80
  store %V.main.Token %t71, ptr %t81
  %t82 = add i64 %t74, 1
  store i64 %t82, ptr %t72
  %t83 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 5
  %t84 = load i64, ptr %t83
  %t85 = add i64 %t84, 1
  store i64 %t85, ptr %t83
  br label %push.done.16
push.grow.15:
  store %V.main.Token %t71, ptr %a86
  call void @veles_list_push(ptr %t53, ptr %a86)
  br label %push.done.16
push.done.16:
  %t87 = load ptr, ptr %a2
  %t88 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.7, i64 2 }, ptr %a89
  call void @veles_list_push(ptr %t88, ptr %a89)
  %t90 = load i64, ptr %a28
  %t92 = call i64 @veles_i64_format(ptr %a91, i64 %t90)
  %t93 = insertvalue %str undef, ptr %a91, 0
  %t94 = insertvalue %str %t93, i64 %t92, 1
  %t96 = extractvalue %str { ptr @.str.8, i64 1 }, 0
  %t97 = extractvalue %str { ptr @.str.8, i64 1 }, 1
  %t98 = extractvalue %str %t94, 0
  %t99 = extractvalue %str %t94, 1
  call void @veles_string_concat(ptr %a95, ptr %t96, i64 %t97, ptr %t98, i64 %t99)
  %t100 = load %str, ptr %a95
  store %str %t100, ptr %a89
  call void @veles_list_push(ptr %t88, ptr %a89)
  %t101 = insertvalue %S.main.Names undef, ptr %t88, 0
  store %V.main.Token zeroinitializer, ptr %a102
  %t103 = getelementptr inbounds %V.main.Token, ptr %a102, i32 0, i32 0
  store i32 2, ptr %t103
  %t104 = getelementptr inbounds %V.main.Token, ptr %a102, i32 0, i32 1
  store %S.main.Names %t101, ptr %t104
  %t105 = load %V.main.Token, ptr %a102
  %t106 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t87, i32 0, i32 1
  %t107 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t87, i32 0, i32 2
  %t108 = load i64, ptr %t106
  %t109 = load i64, ptr %t107
  %t110 = icmp slt i64 %t108, %t109
  br i1 %t110, label %push.fast.17, label %push.grow.18, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.17:
  %t111 = load ptr, ptr %t87
  %t112 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t87, i32 0, i32 3
  %t113 = load i64, ptr %t112
  %t114 = mul i64 %t113, %t108
  %t115 = getelementptr inbounds i8, ptr %t111, i64 %t114
  store %V.main.Token %t105, ptr %t115
  %t116 = add i64 %t108, 1
  store i64 %t116, ptr %t106
  %t117 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t87, i32 0, i32 5
  %t118 = load i64, ptr %t117
  %t119 = add i64 %t118, 1
  store i64 %t119, ptr %t117
  br label %push.done.19
push.grow.18:
  store %V.main.Token %t105, ptr %a120
  call void @veles_list_push(ptr %t87, ptr %a120)
  br label %push.done.19
push.done.19:
  br label %loop.post.2
loop.post.2:
  %t122 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t123 = load i1, ptr %t122
  store i1 %t123, ptr %a121
  br i1 %t123, label %sc.rhs.20, label %sc.end.21
sc.rhs.20:
  %t124 = load i64, ptr %a9
  %t125 = load i64, ptr %a12
  %t126 = icmp eq i64 %t124, %t125
  store i1 %t126, ptr %a121
  br label %sc.end.21
sc.end.21:
  %t127 = load i1, ptr %a121
  br i1 %t127, label %if.then.22, label %if.else.24
if.then.22:
  store i1 true, ptr %a13
  br label %if.end.23
if.else.24:
  %t128 = load i64, ptr %a9
  %t129 = add i64 %t128, 1
  store i64 %t129, ptr %a9
  br label %if.end.23
if.end.23:
  %t130 = load volatile i32, ptr @veles_attention_line, align 64
  %t131 = icmp ne i32 %t130, 0
  br i1 %t131, label %safepoint.25, label %safepoint.on.26, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.25:
  call void @veles_backedge_plain()
  br label %safepoint.on.26
safepoint.on.26:
  br label %loop.cond.1
loop.end.3:
  %t132 = load ptr, ptr %a2
  store ptr %t132, ptr %a133
  %t134 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2.plain, 0
  %t135 = insertvalue { ptr, ptr } %t134, ptr null, 1
  %t136 = extractvalue %str { ptr @.str.9, i64 17 }, 0
  %t137 = extractvalue %str { ptr @.str.9, i64 17 }, 1
  call void @veles_call_push(ptr %t136)
  %t138 = call ptr @v_std.prelude.extend.List_T.map_T_main.Token_i64_Never.plain(ptr %a133, { ptr, ptr } %t135)
  call void @veles_call_pop()
  store ptr %t138, ptr %a139
  %t140 = extractvalue %str { ptr @.str.10, i64 17 }, 0
  %t141 = extractvalue %str { ptr @.str.10, i64 17 }, 1
  call void @veles_call_push(ptr %t140)
  %t142 = call i64 @v_std.prelude.extend.List_i64.sum(ptr %a139)
  call void @veles_call_pop()
  %t144 = call i64 @veles_i64_format(ptr %a143, i64 %t142)
  %t145 = insertvalue %str undef, ptr %a143, 0
  %t146 = insertvalue %str %t145, i64 %t144, 1
  %t149 = getelementptr [1 x %str], ptr %a148, i64 0, i64 0
  store %str %t146, ptr %t149
  call void @veles_string_concat_n(ptr %a147, ptr %a148, i64 1)
  %t150 = load %str, ptr %a147
  %t151 = extractvalue %str { ptr @.str.11, i64 20 }, 0
  %t152 = extractvalue %str { ptr @.str.11, i64 20 }, 1
  call void @veles_call_push(ptr %t151)
  call void @v_std.io.println(%str %t150)
  call void @veles_call_pop()
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a153
  %t154 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a153, i32 0, i32 3
  store i64 24, ptr %t154
  %t155 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a153, i32 0, i32 4
  store ptr @adesc.main.Failure, ptr %t155
  store ptr %a153, ptr %a156
  %t157 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 5, ptr %a158
  call void @veles_list_push(ptr %t157, ptr %a158)
  store i64 -1, ptr %a158
  call void @veles_list_push(ptr %t157, ptr %a158)
  store i64 5000, ptr %a158
  call void @veles_list_push(ptr %t157, ptr %a158)
  store ptr %t157, ptr %a159
  store i64 0, ptr %a160
  br label %loop.cond.27
loop.cond.27:
  %t161 = load i64, ptr %a160
  %t162 = load ptr, ptr %a159
  %t163 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t162, i32 0, i32 1
  %t164 = load i64, ptr %t163
  %t165 = icmp slt i64 %t161, %t164
  br i1 %t165, label %loop.body.30, label %loop.end.29
loop.body.30:
  %t166 = load ptr, ptr %a159
  %t167 = load i64, ptr %a160
  %t168 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t166, i32 0, i32 1
  %t169 = load i64, ptr %t168
  %t170 = icmp ult i64 %t167, %t169
  br i1 %t170, label %idx.ok.31, label %idx.bad.32, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.32:
  %t171 = extractvalue %str { ptr @.str.12, i64 12 }, 0
  %t172 = extractvalue %str { ptr @.str.12, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t166, i64 %t167, ptr %t171, i64 %t172)
  unreachable
idx.ok.31:
  %t173 = load ptr, ptr %t166
  %t174 = getelementptr inbounds [8 x i8], ptr %t173, i64 %t167
  %t175 = load i64, ptr %t174
  store i64 %t175, ptr %a176
  %t177 = load i64, ptr %a176
  %t178 = extractvalue %str { ptr @.str.13, i64 18 }, 0
  %t179 = extractvalue %str { ptr @.str.13, i64 18 }, 1
  call void @veles_call_push(ptr %t178)
  %t180 = call %V._prelude_.Result_i64_main.Malformed_main.Overflow_ @v_main.limit(i64 %t177)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t180, ptr %a181
  %t183 = load %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a181
  %t184 = extractvalue %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t183, 0
  %t185 = icmp eq i32 %t184, 1
  br i1 %t185, label %if.then.33, label %if.else.35
if.then.33:
  %t186 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a181, i32 0, i32 1
  %t187 = getelementptr inbounds %S._prelude_.Err_i64_main.Malformed_main.Overflow_, ptr %t186, i32 0, i32 0
  %t188 = load %U.main.Malformed_main.Overflow, ptr %t187
  store %U.main.Malformed_main.Overflow %t188, ptr %a189
  %t190 = load %U.main.Malformed_main.Overflow, ptr %a189
  store %U.main.Malformed_main.Overflow %t190, ptr %a191
  br label %when.arm.37
when.arm.37:
  %t192 = load %U.main.Malformed_main.Overflow, ptr %a191
  %t193 = extractvalue %U.main.Malformed_main.Overflow %t192, 0
  %t194 = icmp eq i32 %t193, 1
  br i1 %t194, label %when.bind.40, label %when.arm.38
when.bind.40:
  br label %when.body.39
when.body.39:
  %t195 = load %U.main.Malformed_main.Overflow, ptr %a189
  store %U.main.Malformed_main.Overflow %t195, ptr %a196
  %t197 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a196, i32 0, i32 1
  %t198 = load %S.main.Overflow, ptr %t197
  %t199 = extractvalue %S.main.Overflow %t198, 1
  %t201 = call i64 @veles_i64_format(ptr %a200, i64 %t199)
  %t202 = insertvalue %str undef, ptr %a200, 0
  %t203 = insertvalue %str %t202, i64 %t201, 1
  %t205 = extractvalue %str { ptr @.str.14, i64 9 }, 0
  %t206 = extractvalue %str { ptr @.str.14, i64 9 }, 1
  %t207 = extractvalue %str %t203, 0
  %t208 = extractvalue %str %t203, 1
  call void @veles_string_concat(ptr %a204, ptr %t205, i64 %t206, ptr %t207, i64 %t208)
  %t209 = load %str, ptr %a204
  %t210 = extractvalue %str { ptr @.str.15, i64 21 }, 0
  %t211 = extractvalue %str { ptr @.str.15, i64 21 }, 1
  call void @veles_call_push(ptr %t210)
  call void @v_std.io.println(%str %t209)
  call void @veles_call_pop()
  br label %when.end.36
when.arm.38:
  %t212 = load %U.main.Malformed_main.Overflow, ptr %a191
  %t213 = extractvalue %U.main.Malformed_main.Overflow %t212, 0
  %t214 = icmp eq i32 %t213, 0
  br i1 %t214, label %when.bind.43, label %when.arm.41
when.bind.43:
  br label %when.body.42
when.body.42:
  %t215 = load %U.main.Malformed_main.Overflow, ptr %a189
  store %U.main.Malformed_main.Overflow %t215, ptr %a216
  %t217 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a216, i32 0, i32 1
  %t218 = load %S.main.Malformed, ptr %t217
  %t219 = extractvalue %S.main.Malformed %t218, 0
  %t221 = extractvalue %str { ptr @.str.16, i64 10 }, 0
  %t222 = extractvalue %str { ptr @.str.16, i64 10 }, 1
  %t223 = extractvalue %str %t219, 0
  %t224 = extractvalue %str %t219, 1
  call void @veles_string_concat_shared(ptr %a220, ptr %t221, i64 %t222, ptr %t223, i64 %t224)
  %t225 = load %str, ptr %a220
  %t226 = extractvalue %str { ptr @.str.17, i64 21 }, 0
  %t227 = extractvalue %str { ptr @.str.17, i64 21 }, 1
  call void @veles_call_push(ptr %t226)
  call void @v_std.io.println(%str %t225)
  call void @veles_call_pop()
  br label %when.end.36
when.arm.41:
  %t228 = extractvalue %str { ptr @.str.3, i64 33 }, 0
  %t229 = extractvalue %str { ptr @.str.3, i64 33 }, 1
  %t230 = extractvalue %str { ptr @.str.18, i64 12 }, 0
  %t231 = extractvalue %str { ptr @.str.18, i64 12 }, 1
  call void @veles_panic_at(ptr %t228, i64 %t229, ptr %t230, i64 %t231)
  unreachable
when.end.36:
  %t232 = load ptr, ptr %a156
  %t233 = load %U.main.Malformed_main.Overflow, ptr %a189
  %t234 = insertvalue %S.main.Failure undef, %U.main.Malformed_main.Overflow %t233, 0
  %t235 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t232, i32 0, i32 1
  %t236 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t232, i32 0, i32 2
  %t237 = load i64, ptr %t235
  %t238 = load i64, ptr %t236
  %t239 = icmp slt i64 %t237, %t238
  br i1 %t239, label %push.fast.44, label %push.grow.45, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.44:
  %t240 = load ptr, ptr %t232
  %t241 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t232, i32 0, i32 3
  %t242 = load i64, ptr %t241
  %t243 = mul i64 %t242, %t237
  %t244 = getelementptr inbounds i8, ptr %t240, i64 %t243
  store %S.main.Failure %t234, ptr %t244
  %t245 = add i64 %t237, 1
  store i64 %t245, ptr %t235
  %t246 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t232, i32 0, i32 5
  %t247 = load i64, ptr %t246
  %t248 = add i64 %t247, 1
  store i64 %t248, ptr %t246
  br label %push.done.46
push.grow.45:
  store %S.main.Failure %t234, ptr %a249
  call void @veles_list_push(ptr %t232, ptr %a249)
  br label %push.done.46
push.done.46:
  br label %loop.post.28
if.else.35:
  %t250 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a181, i32 0, i32 1
  %t251 = getelementptr inbounds %S._prelude_.Ok_i64_main.Malformed_main.Overflow_, ptr %t250, i32 0, i32 0
  %t252 = load i64, ptr %t251
  store i64 %t252, ptr %a182
  br label %if.end.34
if.end.34:
  %t253 = load i64, ptr %a182
  %t254 = load i64, ptr %a176
  %t256 = call i64 @veles_i64_format(ptr %a255, i64 %t254)
  %t257 = insertvalue %str undef, ptr %a255, 0
  %t258 = insertvalue %str %t257, i64 %t256, 1
  %t260 = extractvalue %str { ptr @.str.19, i64 5 }, 0
  %t261 = extractvalue %str { ptr @.str.19, i64 5 }, 1
  %t262 = extractvalue %str %t258, 0
  %t263 = extractvalue %str %t258, 1
  call void @veles_string_concat(ptr %a259, ptr %t260, i64 %t261, ptr %t262, i64 %t263)
  %t264 = load %str, ptr %a259
  %t265 = extractvalue %str { ptr @.str.20, i64 20 }, 0
  %t266 = extractvalue %str { ptr @.str.20, i64 20 }, 1
  call void @veles_call_push(ptr %t265)
  call void @v_std.io.println(%str %t264)
  call void @veles_call_pop()
  br label %loop.post.28
loop.post.28:
  %t267 = load i64, ptr %a160
  %t268 = add i64 %t267, 1
  store i64 %t268, ptr %a160
  %t269 = load volatile i32, ptr @veles_attention_line, align 64
  %t270 = icmp ne i32 %t269, 0
  br i1 %t270, label %safepoint.47, label %safepoint.on.48, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.47:
  call void @veles_backedge_plain()
  br label %safepoint.on.48
safepoint.on.48:
  br label %loop.cond.27
loop.end.29:
  %t271 = load ptr, ptr %a156
  %t272 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t271, i32 0, i32 1
  %t273 = load i64, ptr %t272
  %t275 = call i64 @veles_i64_format(ptr %a274, i64 %t273)
  %t276 = insertvalue %str undef, ptr %a274, 0
  %t277 = insertvalue %str %t276, i64 %t275, 1
  %t279 = extractvalue %str %t277, 0
  %t280 = extractvalue %str %t277, 1
  %t281 = extractvalue %str { ptr @.str.21, i64 7 }, 0
  %t282 = extractvalue %str { ptr @.str.21, i64 7 }, 1
  call void @veles_string_concat(ptr %a278, ptr %t279, i64 %t280, ptr %t281, i64 %t282)
  %t283 = load %str, ptr %a278
  %t284 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t285 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t284)
  call void @v_std.io.println(%str %t283)
  call void @veles_call_pop()
  ret void
}

define ptr @v_std.prelude.extend.List_T.map_T_main.Token_i64_Never.plain(ptr %p0, { ptr, ptr } %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca { ptr, ptr }
  %a4 = alloca ptr
  %a12 = alloca ptr
  %a13 = alloca i64
  %a29 = alloca %V.main.Token
  %a50 = alloca i64
  store ptr %p0, ptr %a1
  store { ptr, ptr } %p1, ptr %a2
  %t3 = call ptr @veles_list_new(ptr @adesc.i64, i64 0)
  store ptr %t3, ptr %a4
  %t5 = load ptr, ptr %a4
  %t6 = load ptr, ptr %a1
  %t7 = load ptr, ptr %t6
  %t8 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t7, i32 0, i32 1
  %t9 = load i64, ptr %t8
  call void @veles_list_reserve(ptr %t5, i64 %t9)
  %t10 = load ptr, ptr %a1
  %t11 = load ptr, ptr %t10
  store ptr %t11, ptr %a12
  store i64 0, ptr %a13
  br label %loop.cond.1
loop.cond.1:
  %t14 = load i64, ptr %a13
  %t15 = load ptr, ptr %a12
  %t16 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t15, i32 0, i32 1
  %t17 = load i64, ptr %t16
  %t18 = icmp slt i64 %t14, %t17
  br i1 %t18, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t19 = load ptr, ptr %a12
  %t20 = load i64, ptr %a13
  %t21 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t19, i32 0, i32 1
  %t22 = load i64, ptr %t21
  %t23 = icmp ult i64 %t20, %t22
  br i1 %t23, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t24 = extractvalue %str { ptr @.str.23, i64 28 }, 0
  %t25 = extractvalue %str { ptr @.str.23, i64 28 }, 1
  call void @veles_list_index_panic(ptr %t19, i64 %t20, ptr %t24, i64 %t25)
  unreachable
idx.ok.5:
  %t26 = load ptr, ptr %t19
  %t27 = getelementptr inbounds [32 x i8], ptr %t26, i64 %t20
  %t28 = load %V.main.Token, ptr %t27
  store %V.main.Token %t28, ptr %a29
  %t30 = load ptr, ptr %a4
  %t31 = load { ptr, ptr }, ptr %a2
  %t32 = extractvalue { ptr, ptr } %t31, 0
  %t33 = extractvalue { ptr, ptr } %t31, 1
  %t34 = load %V.main.Token, ptr %a29
  %t35 = call i64 %t32(ptr %t33, %V.main.Token %t34)
  %t36 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t30, i32 0, i32 1
  %t37 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t30, i32 0, i32 2
  %t38 = load i64, ptr %t36
  %t39 = load i64, ptr %t37
  %t40 = icmp slt i64 %t38, %t39
  br i1 %t40, label %push.fast.7, label %push.grow.8, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.7:
  %t41 = load ptr, ptr %t30
  %t42 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t30, i32 0, i32 3
  %t43 = load i64, ptr %t42
  %t44 = mul i64 %t43, %t38
  %t45 = getelementptr inbounds i8, ptr %t41, i64 %t44
  store i64 %t35, ptr %t45
  %t46 = add i64 %t38, 1
  store i64 %t46, ptr %t36
  %t47 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t30, i32 0, i32 5
  %t48 = load i64, ptr %t47
  %t49 = add i64 %t48, 1
  store i64 %t49, ptr %t47
  br label %push.done.9
push.grow.8:
  store i64 %t35, ptr %a50
  call void @veles_list_push(ptr %t30, ptr %a50)
  br label %push.done.9
push.done.9:
  br label %loop.post.2
loop.post.2:
  %t51 = load i64, ptr %a13
  %t52 = add i64 %t51, 1
  store i64 %t52, ptr %a13
  %t53 = load volatile i32, ptr @veles_attention_line, align 64
  %t54 = icmp ne i32 %t53, 0
  br i1 %t54, label %safepoint.10, label %safepoint.on.11, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.10:
  call void @veles_backedge_plain()
  br label %safepoint.on.11
safepoint.on.11:
  br label %loop.cond.1
loop.end.3:
  %t55 = load ptr, ptr %a4
  ret ptr %t55
}

define i64 @v_main.main.lambda2.plain(ptr %env, %V.main.Token %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca %V.main.Token
  store ptr %env, ptr %a1
  store %V.main.Token %p1, ptr %a2
  %t3 = load %V.main.Token, ptr %a2
  %t4 = extractvalue %str { ptr @.str.24, i64 19 }, 0
  %t5 = extractvalue %str { ptr @.str.24, i64 19 }, 1
  call void @veles_call_push(ptr %t4)
  %t6 = call i64 @v_main.weigh(%V.main.Token %t3)
  call void @veles_call_pop()
  ret i64 %t6
}

@.str.1 = private unnamed_addr constant [18 x i8] c"main.vs:34:23\00map\00"
@.str.2 = private unnamed_addr constant [18 x i8] c"main.vs:34:23\00sum\00"
@.str.3 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:31:29\00"
@.str.5 = private unnamed_addr constant [9 x i8] c"negative\00"
@.str.6 = private unnamed_addr constant [2 x i8] c"w\00"
@.str.7 = private unnamed_addr constant [3 x i8] c"ab\00"
@.str.8 = private unnamed_addr constant [2 x i8] c"c\00"
@.str.9 = private unnamed_addr constant [18 x i8] c"main.vs:50:17\00map\00"
@.str.10 = private unnamed_addr constant [18 x i8] c"main.vs:50:17\00sum\00"
@.str.11 = private unnamed_addr constant [21 x i8] c"main.vs:50:3\00println\00"
@.str.12 = private unnamed_addr constant [13 x i8] c"main.vs:52:3\00"
@.str.13 = private unnamed_addr constant [19 x i8] c"main.vs:53:5\00limit\00"
@.str.14 = private unnamed_addr constant [10 x i8] c"overflow \00"
@.str.15 = private unnamed_addr constant [22 x i8] c"main.vs:55:25\00println\00"
@.str.16 = private unnamed_addr constant [11 x i8] c"malformed \00"
@.str.17 = private unnamed_addr constant [22 x i8] c"main.vs:56:25\00println\00"
@.str.18 = private unnamed_addr constant [13 x i8] c"main.vs:54:7\00"
@.str.19 = private unnamed_addr constant [6 x i8] c"fine \00"
@.str.20 = private unnamed_addr constant [21 x i8] c"main.vs:61:5\00println\00"
@.str.21 = private unnamed_addr constant [8 x i8] c" failed\00"
@.str.22 = private unnamed_addr constant [21 x i8] c"main.vs:63:3\00println\00"
@.str.23 = private unnamed_addr constant [29 x i8] c"std/prelude/adapters.vs:15:5\00"
@.str.24 = private unnamed_addr constant [20 x i8] c"main.vs:50:37\00weigh\00"
