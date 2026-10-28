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
  %a14 = alloca i64
  %a15 = alloca i1
  %a17 = alloca i1
  %a28 = alloca i64
  %a42 = alloca i1
  %a44 = alloca i1
  %a59 = alloca i64
  %a65 = alloca %V.main.Token
  %a83 = alloca %V.main.Token
  %a87 = alloca [21 x i8]
  %a91 = alloca %str
  %a99 = alloca %V.main.Token
  %a117 = alloca %V.main.Token
  %a120 = alloca %str
  %a122 = alloca [21 x i8]
  %a126 = alloca %str
  %a133 = alloca %V.main.Token
  %a151 = alloca %V.main.Token
  %a152 = alloca i1
  %a164 = alloca ptr
  %a170 = alloca ptr
  %a174 = alloca [21 x i8]
  %a178 = alloca %str
  %a179 = alloca [1 x %str]
  %a184 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a187 = alloca ptr
  %a189 = alloca i64
  %a190 = alloca ptr
  %a191 = alloca i64
  %a207 = alloca i64
  %a212 = alloca %V._prelude_.Result_i64_main.Malformed_main.Overflow_
  %a213 = alloca i64
  %a220 = alloca %U.main.Malformed_main.Overflow
  %a222 = alloca %U.main.Malformed_main.Overflow
  %a227 = alloca %U.main.Malformed_main.Overflow
  %a231 = alloca [21 x i8]
  %a235 = alloca %str
  %a247 = alloca %U.main.Malformed_main.Overflow
  %a251 = alloca %str
  %a280 = alloca %S.main.Failure
  %a286 = alloca [21 x i8]
  %a290 = alloca %str
  %a305 = alloca [21 x i8]
  %a309 = alloca %str
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
  store i64 0, ptr %a14
  br label %chunk.cond.1
chunk.cond.1:
  %t16 = load i1, ptr %a13
  br i1 %t16, label %if.then.8, label %if.else.10
if.then.8:
  store i1 false, ptr %a15
  br label %if.end.9
if.else.10:
  %t18 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t19 = load i1, ptr %t18
  br i1 %t19, label %if.then.11, label %if.else.13
if.then.11:
  %t20 = load i64, ptr %a9
  %t21 = load i64, ptr %a12
  %t22 = icmp sle i64 %t20, %t21
  store i1 %t22, ptr %a17
  br label %if.end.12
if.else.13:
  %t23 = load i64, ptr %a9
  %t24 = load i64, ptr %a12
  %t25 = icmp slt i64 %t23, %t24
  store i1 %t25, ptr %a17
  br label %if.end.12
if.end.12:
  %t26 = load i1, ptr %a17
  store i1 %t26, ptr %a15
  br label %if.end.9
if.end.9:
  %t27 = load i1, ptr %a15
  br i1 %t27, label %chunk.start.7, label %chunk.end.3
chunk.start.7:
  %t29 = load i64, ptr %a9
  %t30 = icmp sgt i64 %t29, 9223372036854775543
  br i1 %t30, label %if.then.14, label %if.else.16
if.then.14:
  store i64 9223372036854775807, ptr %a28
  br label %if.end.15
if.else.16:
  %t31 = load i64, ptr %a9
  %t33 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t31, i64 264)
  %t34 = extractvalue { i64, i1 } %t33, 0
  %t35 = extractvalue { i64, i1 } %t33, 1
  br i1 %t35, label %overflow.17, label %arith.ok.18
overflow.17:
  %t36 = extractvalue %str { ptr @.str.6, i64 16 }, 0
  %t37 = extractvalue %str { ptr @.str.6, i64 16 }, 1
  %t38 = extractvalue %str { ptr @.str.7, i64 0 }, 0
  %t39 = extractvalue %str { ptr @.str.7, i64 0 }, 1
  call void @veles_panic_at(ptr %t36, i64 %t37, ptr %t38, i64 %t39)
  unreachable
arith.ok.18:
  store i64 %t34, ptr %a28
  br label %if.end.15
if.end.15:
  %t40 = load i64, ptr %a28
  store i64 %t40, ptr %a14
  br label %chunk.body.5
chunk.inner.4:
  %t43 = load i1, ptr %a13
  br i1 %t43, label %if.then.19, label %if.else.21
if.then.19:
  store i1 false, ptr %a42
  br label %if.end.20
if.else.21:
  %t45 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t46 = load i1, ptr %t45
  br i1 %t46, label %if.then.22, label %if.else.24
if.then.22:
  %t47 = load i64, ptr %a9
  %t48 = load i64, ptr %a12
  %t49 = icmp sle i64 %t47, %t48
  store i1 %t49, ptr %a44
  br label %if.end.23
if.else.24:
  %t50 = load i64, ptr %a9
  %t51 = load i64, ptr %a12
  %t52 = icmp slt i64 %t50, %t51
  store i1 %t52, ptr %a44
  br label %if.end.23
if.end.23:
  %t53 = load i1, ptr %a44
  store i1 %t53, ptr %a42
  br label %if.end.20
if.end.20:
  %t54 = load i1, ptr %a42
  %t55 = load i64, ptr %a9
  %t56 = load i64, ptr %a14
  %t57 = icmp slt i64 %t55, %t56
  %t41 = and i1 %t54, %t57
  br i1 %t41, label %chunk.body.5, label %chunk.looked.6
chunk.body.5:
  %t58 = load i64, ptr %a9
  store i64 %t58, ptr %a59
  %t60 = load ptr, ptr %a2
  %t61 = load i64, ptr %a59
  %t62 = load i64, ptr %a59
  %t63 = insertvalue %S.main.Number undef, i64 %t61, 0
  %t64 = insertvalue %S.main.Number %t63, i64 %t62, 1
  store %V.main.Token zeroinitializer, ptr %a65
  %t66 = getelementptr inbounds %V.main.Token, ptr %a65, i32 0, i32 0
  store i32 0, ptr %t66
  %t67 = getelementptr inbounds %V.main.Token, ptr %a65, i32 0, i32 1
  store %S.main.Number %t64, ptr %t67
  %t68 = load %V.main.Token, ptr %a65
  %t69 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t60, i32 0, i32 1
  %t70 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t60, i32 0, i32 2
  %t71 = load i64, ptr %t69
  %t72 = load i64, ptr %t70
  %t73 = icmp slt i64 %t71, %t72
  br i1 %t73, label %push.fast.25, label %push.grow.26, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.25:
  %t74 = load ptr, ptr %t60
  %t75 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t60, i32 0, i32 3
  %t76 = load i64, ptr %t75
  %t77 = mul i64 %t76, %t71
  %t78 = getelementptr inbounds i8, ptr %t74, i64 %t77
  store %V.main.Token %t68, ptr %t78
  %t79 = add i64 %t71, 1
  store i64 %t79, ptr %t69
  %t80 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t60, i32 0, i32 5
  %t81 = load i64, ptr %t80
  %t82 = add i64 %t81, 1
  store i64 %t82, ptr %t80
  br label %push.done.27
push.grow.26:
  store %V.main.Token %t68, ptr %a83
  call void @veles_list_push(ptr %t60, ptr %a83)
  br label %push.done.27
push.done.27:
  %t84 = load ptr, ptr %a2
  %t85 = load i64, ptr %a59
  %t86 = load i64, ptr %a59
  %t88 = call i64 @veles_i64_format(ptr %a87, i64 %t86)
  %t89 = insertvalue %str undef, ptr %a87, 0
  %t90 = insertvalue %str %t89, i64 %t88, 1
  %t92 = extractvalue %str { ptr @.str.8, i64 1 }, 0
  %t93 = extractvalue %str { ptr @.str.8, i64 1 }, 1
  %t94 = extractvalue %str %t90, 0
  %t95 = extractvalue %str %t90, 1
  call void @veles_string_concat(ptr %a91, ptr %t92, i64 %t93, ptr %t94, i64 %t95)
  %t96 = load %str, ptr %a91
  %t97 = insertvalue %S.main.Word undef, i64 %t85, 0
  %t98 = insertvalue %S.main.Word %t97, %str %t96, 1
  store %V.main.Token zeroinitializer, ptr %a99
  %t100 = getelementptr inbounds %V.main.Token, ptr %a99, i32 0, i32 0
  store i32 1, ptr %t100
  %t101 = getelementptr inbounds %V.main.Token, ptr %a99, i32 0, i32 1
  store %S.main.Word %t98, ptr %t101
  %t102 = load %V.main.Token, ptr %a99
  %t103 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t84, i32 0, i32 1
  %t104 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t84, i32 0, i32 2
  %t105 = load i64, ptr %t103
  %t106 = load i64, ptr %t104
  %t107 = icmp slt i64 %t105, %t106
  br i1 %t107, label %push.fast.28, label %push.grow.29, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.28:
  %t108 = load ptr, ptr %t84
  %t109 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t84, i32 0, i32 3
  %t110 = load i64, ptr %t109
  %t111 = mul i64 %t110, %t105
  %t112 = getelementptr inbounds i8, ptr %t108, i64 %t111
  store %V.main.Token %t102, ptr %t112
  %t113 = add i64 %t105, 1
  store i64 %t113, ptr %t103
  %t114 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t84, i32 0, i32 5
  %t115 = load i64, ptr %t114
  %t116 = add i64 %t115, 1
  store i64 %t116, ptr %t114
  br label %push.done.30
push.grow.29:
  store %V.main.Token %t102, ptr %a117
  call void @veles_list_push(ptr %t84, ptr %a117)
  br label %push.done.30
push.done.30:
  %t118 = load ptr, ptr %a2
  %t119 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.9, i64 2 }, ptr %a120
  call void @veles_list_push(ptr %t119, ptr %a120)
  %t121 = load i64, ptr %a59
  %t123 = call i64 @veles_i64_format(ptr %a122, i64 %t121)
  %t124 = insertvalue %str undef, ptr %a122, 0
  %t125 = insertvalue %str %t124, i64 %t123, 1
  %t127 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t128 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  %t129 = extractvalue %str %t125, 0
  %t130 = extractvalue %str %t125, 1
  call void @veles_string_concat(ptr %a126, ptr %t127, i64 %t128, ptr %t129, i64 %t130)
  %t131 = load %str, ptr %a126
  store %str %t131, ptr %a120
  call void @veles_list_push(ptr %t119, ptr %a120)
  %t132 = insertvalue %S.main.Names undef, ptr %t119, 0
  store %V.main.Token zeroinitializer, ptr %a133
  %t134 = getelementptr inbounds %V.main.Token, ptr %a133, i32 0, i32 0
  store i32 2, ptr %t134
  %t135 = getelementptr inbounds %V.main.Token, ptr %a133, i32 0, i32 1
  store %S.main.Names %t132, ptr %t135
  %t136 = load %V.main.Token, ptr %a133
  %t137 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t118, i32 0, i32 1
  %t138 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t118, i32 0, i32 2
  %t139 = load i64, ptr %t137
  %t140 = load i64, ptr %t138
  %t141 = icmp slt i64 %t139, %t140
  br i1 %t141, label %push.fast.31, label %push.grow.32, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.31:
  %t142 = load ptr, ptr %t118
  %t143 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t118, i32 0, i32 3
  %t144 = load i64, ptr %t143
  %t145 = mul i64 %t144, %t139
  %t146 = getelementptr inbounds i8, ptr %t142, i64 %t145
  store %V.main.Token %t136, ptr %t146
  %t147 = add i64 %t139, 1
  store i64 %t147, ptr %t137
  %t148 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t118, i32 0, i32 5
  %t149 = load i64, ptr %t148
  %t150 = add i64 %t149, 1
  store i64 %t150, ptr %t148
  br label %push.done.33
push.grow.32:
  store %V.main.Token %t136, ptr %a151
  call void @veles_list_push(ptr %t118, ptr %a151)
  br label %push.done.33
push.done.33:
  br label %chunk.post.2
chunk.post.2:
  %t153 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t154 = load i1, ptr %t153
  store i1 %t154, ptr %a152
  br i1 %t154, label %sc.rhs.34, label %sc.end.35
sc.rhs.34:
  %t155 = load i64, ptr %a9
  %t156 = load i64, ptr %a12
  %t157 = icmp eq i64 %t155, %t156
  store i1 %t157, ptr %a152
  br label %sc.end.35
sc.end.35:
  %t158 = load i1, ptr %a152
  br i1 %t158, label %if.then.36, label %if.else.38
if.then.36:
  store i1 true, ptr %a13
  br label %if.end.37
if.else.38:
  %t159 = load i64, ptr %a9
  %t160 = add i64 %t159, 1
  store i64 %t160, ptr %a9
  br label %if.end.37
if.end.37:
  br label %chunk.inner.4
chunk.looked.6:
  %t161 = load volatile i32, ptr @veles_attention_line, align 64
  %t162 = icmp ne i32 %t161, 0
  br i1 %t162, label %safepoint.39, label %safepoint.on.40, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.39:
  call void @veles_backedge_plain()
  br label %safepoint.on.40
safepoint.on.40:
  br label %chunk.cond.1
chunk.end.3:
  %t163 = load ptr, ptr %a2
  store ptr %t163, ptr %a164
  %t165 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2.plain, 0
  %t166 = insertvalue { ptr, ptr } %t165, ptr null, 1
  %t167 = extractvalue %str { ptr @.str.11, i64 17 }, 0
  %t168 = extractvalue %str { ptr @.str.11, i64 17 }, 1
  call void @veles_call_push(ptr %t167)
  %t169 = call ptr @v_std.prelude.extend.List_T.map_T_main.Token_i64_Never.plain(ptr %a164, { ptr, ptr } %t166)
  call void @veles_call_pop()
  store ptr %t169, ptr %a170
  %t171 = extractvalue %str { ptr @.str.12, i64 17 }, 0
  %t172 = extractvalue %str { ptr @.str.12, i64 17 }, 1
  call void @veles_call_push(ptr %t171)
  %t173 = call i64 @v_std.prelude.extend.List_i64.sum(ptr %a170)
  call void @veles_call_pop()
  %t175 = call i64 @veles_i64_format(ptr %a174, i64 %t173)
  %t176 = insertvalue %str undef, ptr %a174, 0
  %t177 = insertvalue %str %t176, i64 %t175, 1
  %t180 = getelementptr [1 x %str], ptr %a179, i64 0, i64 0
  store %str %t177, ptr %t180
  call void @veles_string_concat_n(ptr %a178, ptr %a179, i64 1)
  %t181 = load %str, ptr %a178
  %t182 = extractvalue %str { ptr @.str.13, i64 20 }, 0
  %t183 = extractvalue %str { ptr @.str.13, i64 20 }, 1
  call void @veles_call_push(ptr %t182)
  call void @v_std.io.println(%str %t181)
  call void @veles_call_pop()
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a184
  %t185 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a184, i32 0, i32 3
  store i64 24, ptr %t185
  %t186 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a184, i32 0, i32 4
  store ptr @adesc.main.Failure, ptr %t186
  store ptr %a184, ptr %a187
  %t188 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 5, ptr %a189
  call void @veles_list_push(ptr %t188, ptr %a189)
  store i64 -1, ptr %a189
  call void @veles_list_push(ptr %t188, ptr %a189)
  store i64 5000, ptr %a189
  call void @veles_list_push(ptr %t188, ptr %a189)
  store ptr %t188, ptr %a190
  store i64 0, ptr %a191
  br label %loop.cond.41
loop.cond.41:
  %t192 = load i64, ptr %a191
  %t193 = load ptr, ptr %a190
  %t194 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t193, i32 0, i32 1
  %t195 = load i64, ptr %t194
  %t196 = icmp slt i64 %t192, %t195
  br i1 %t196, label %loop.body.44, label %loop.end.43
loop.body.44:
  %t197 = load ptr, ptr %a190
  %t198 = load i64, ptr %a191
  %t199 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t197, i32 0, i32 1
  %t200 = load i64, ptr %t199
  %t201 = icmp ult i64 %t198, %t200
  br i1 %t201, label %idx.ok.45, label %idx.bad.46, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.46:
  %t202 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t203 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t197, i64 %t198, ptr %t202, i64 %t203)
  unreachable
idx.ok.45:
  %t204 = load ptr, ptr %t197
  %t205 = getelementptr inbounds [8 x i8], ptr %t204, i64 %t198
  %t206 = load i64, ptr %t205
  store i64 %t206, ptr %a207
  %t208 = load i64, ptr %a207
  %t209 = extractvalue %str { ptr @.str.15, i64 18 }, 0
  %t210 = extractvalue %str { ptr @.str.15, i64 18 }, 1
  call void @veles_call_push(ptr %t209)
  %t211 = call %V._prelude_.Result_i64_main.Malformed_main.Overflow_ @v_main.limit(i64 %t208)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t211, ptr %a212
  %t214 = load %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a212
  %t215 = extractvalue %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t214, 0
  %t216 = icmp eq i32 %t215, 1
  br i1 %t216, label %if.then.47, label %if.else.49
if.then.47:
  %t217 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a212, i32 0, i32 1
  %t218 = getelementptr inbounds %S._prelude_.Err_i64_main.Malformed_main.Overflow_, ptr %t217, i32 0, i32 0
  %t219 = load %U.main.Malformed_main.Overflow, ptr %t218
  store %U.main.Malformed_main.Overflow %t219, ptr %a220
  %t221 = load %U.main.Malformed_main.Overflow, ptr %a220
  store %U.main.Malformed_main.Overflow %t221, ptr %a222
  br label %when.arm.51
when.arm.51:
  %t223 = load %U.main.Malformed_main.Overflow, ptr %a222
  %t224 = extractvalue %U.main.Malformed_main.Overflow %t223, 0
  %t225 = icmp eq i32 %t224, 1
  br i1 %t225, label %when.bind.54, label %when.arm.52
when.bind.54:
  br label %when.body.53
when.body.53:
  %t226 = load %U.main.Malformed_main.Overflow, ptr %a220
  store %U.main.Malformed_main.Overflow %t226, ptr %a227
  %t228 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a227, i32 0, i32 1
  %t229 = load %S.main.Overflow, ptr %t228
  %t230 = extractvalue %S.main.Overflow %t229, 1
  %t232 = call i64 @veles_i64_format(ptr %a231, i64 %t230)
  %t233 = insertvalue %str undef, ptr %a231, 0
  %t234 = insertvalue %str %t233, i64 %t232, 1
  %t236 = extractvalue %str { ptr @.str.16, i64 9 }, 0
  %t237 = extractvalue %str { ptr @.str.16, i64 9 }, 1
  %t238 = extractvalue %str %t234, 0
  %t239 = extractvalue %str %t234, 1
  call void @veles_string_concat(ptr %a235, ptr %t236, i64 %t237, ptr %t238, i64 %t239)
  %t240 = load %str, ptr %a235
  %t241 = extractvalue %str { ptr @.str.17, i64 21 }, 0
  %t242 = extractvalue %str { ptr @.str.17, i64 21 }, 1
  call void @veles_call_push(ptr %t241)
  call void @v_std.io.println(%str %t240)
  call void @veles_call_pop()
  br label %when.end.50
when.arm.52:
  %t243 = load %U.main.Malformed_main.Overflow, ptr %a222
  %t244 = extractvalue %U.main.Malformed_main.Overflow %t243, 0
  %t245 = icmp eq i32 %t244, 0
  br i1 %t245, label %when.bind.57, label %when.arm.55
when.bind.57:
  br label %when.body.56
when.body.56:
  %t246 = load %U.main.Malformed_main.Overflow, ptr %a220
  store %U.main.Malformed_main.Overflow %t246, ptr %a247
  %t248 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a247, i32 0, i32 1
  %t249 = load %S.main.Malformed, ptr %t248
  %t250 = extractvalue %S.main.Malformed %t249, 0
  %t252 = extractvalue %str { ptr @.str.18, i64 10 }, 0
  %t253 = extractvalue %str { ptr @.str.18, i64 10 }, 1
  %t254 = extractvalue %str %t250, 0
  %t255 = extractvalue %str %t250, 1
  call void @veles_string_concat_shared(ptr %a251, ptr %t252, i64 %t253, ptr %t254, i64 %t255)
  %t256 = load %str, ptr %a251
  %t257 = extractvalue %str { ptr @.str.19, i64 21 }, 0
  %t258 = extractvalue %str { ptr @.str.19, i64 21 }, 1
  call void @veles_call_push(ptr %t257)
  call void @v_std.io.println(%str %t256)
  call void @veles_call_pop()
  br label %when.end.50
when.arm.55:
  %t259 = extractvalue %str { ptr @.str.3, i64 33 }, 0
  %t260 = extractvalue %str { ptr @.str.3, i64 33 }, 1
  %t261 = extractvalue %str { ptr @.str.20, i64 12 }, 0
  %t262 = extractvalue %str { ptr @.str.20, i64 12 }, 1
  call void @veles_panic_at(ptr %t259, i64 %t260, ptr %t261, i64 %t262)
  unreachable
when.end.50:
  %t263 = load ptr, ptr %a187
  %t264 = load %U.main.Malformed_main.Overflow, ptr %a220
  %t265 = insertvalue %S.main.Failure undef, %U.main.Malformed_main.Overflow %t264, 0
  %t266 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t263, i32 0, i32 1
  %t267 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t263, i32 0, i32 2
  %t268 = load i64, ptr %t266
  %t269 = load i64, ptr %t267
  %t270 = icmp slt i64 %t268, %t269
  br i1 %t270, label %push.fast.58, label %push.grow.59, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.58:
  %t271 = load ptr, ptr %t263
  %t272 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t263, i32 0, i32 3
  %t273 = load i64, ptr %t272
  %t274 = mul i64 %t273, %t268
  %t275 = getelementptr inbounds i8, ptr %t271, i64 %t274
  store %S.main.Failure %t265, ptr %t275
  %t276 = add i64 %t268, 1
  store i64 %t276, ptr %t266
  %t277 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t263, i32 0, i32 5
  %t278 = load i64, ptr %t277
  %t279 = add i64 %t278, 1
  store i64 %t279, ptr %t277
  br label %push.done.60
push.grow.59:
  store %S.main.Failure %t265, ptr %a280
  call void @veles_list_push(ptr %t263, ptr %a280)
  br label %push.done.60
push.done.60:
  br label %loop.post.42
if.else.49:
  %t281 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a212, i32 0, i32 1
  %t282 = getelementptr inbounds %S._prelude_.Ok_i64_main.Malformed_main.Overflow_, ptr %t281, i32 0, i32 0
  %t283 = load i64, ptr %t282
  store i64 %t283, ptr %a213
  br label %if.end.48
if.end.48:
  %t284 = load i64, ptr %a213
  %t285 = load i64, ptr %a207
  %t287 = call i64 @veles_i64_format(ptr %a286, i64 %t285)
  %t288 = insertvalue %str undef, ptr %a286, 0
  %t289 = insertvalue %str %t288, i64 %t287, 1
  %t291 = extractvalue %str { ptr @.str.21, i64 5 }, 0
  %t292 = extractvalue %str { ptr @.str.21, i64 5 }, 1
  %t293 = extractvalue %str %t289, 0
  %t294 = extractvalue %str %t289, 1
  call void @veles_string_concat(ptr %a290, ptr %t291, i64 %t292, ptr %t293, i64 %t294)
  %t295 = load %str, ptr %a290
  %t296 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t297 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t296)
  call void @v_std.io.println(%str %t295)
  call void @veles_call_pop()
  br label %loop.post.42
loop.post.42:
  %t298 = load i64, ptr %a191
  %t299 = add i64 %t298, 1
  store i64 %t299, ptr %a191
  %t300 = load volatile i32, ptr @veles_attention_line, align 64
  %t301 = icmp ne i32 %t300, 0
  br i1 %t301, label %safepoint.61, label %safepoint.on.62, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.61:
  call void @veles_backedge_plain()
  br label %safepoint.on.62
safepoint.on.62:
  br label %loop.cond.41
loop.end.43:
  %t302 = load ptr, ptr %a187
  %t303 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t302, i32 0, i32 1
  %t304 = load i64, ptr %t303
  %t306 = call i64 @veles_i64_format(ptr %a305, i64 %t304)
  %t307 = insertvalue %str undef, ptr %a305, 0
  %t308 = insertvalue %str %t307, i64 %t306, 1
  %t310 = extractvalue %str %t308, 0
  %t311 = extractvalue %str %t308, 1
  %t312 = extractvalue %str { ptr @.str.23, i64 7 }, 0
  %t313 = extractvalue %str { ptr @.str.23, i64 7 }, 1
  call void @veles_string_concat(ptr %a309, ptr %t310, i64 %t311, ptr %t312, i64 %t313)
  %t314 = load %str, ptr %a309
  %t315 = extractvalue %str { ptr @.str.24, i64 20 }, 0
  %t316 = extractvalue %str { ptr @.str.24, i64 20 }, 1
  call void @veles_call_push(ptr %t315)
  call void @v_std.io.println(%str %t314)
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
  %t24 = extractvalue %str { ptr @.str.25, i64 28 }, 0
  %t25 = extractvalue %str { ptr @.str.25, i64 28 }, 1
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
  %t4 = extractvalue %str { ptr @.str.26, i64 19 }, 0
  %t5 = extractvalue %str { ptr @.str.26, i64 19 }, 1
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
@.str.6 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.7 = private unnamed_addr constant [1 x i8] c"\00"
@.str.8 = private unnamed_addr constant [2 x i8] c"w\00"
@.str.9 = private unnamed_addr constant [3 x i8] c"ab\00"
@.str.10 = private unnamed_addr constant [2 x i8] c"c\00"
@.str.11 = private unnamed_addr constant [18 x i8] c"main.vs:50:17\00map\00"
@.str.12 = private unnamed_addr constant [18 x i8] c"main.vs:50:17\00sum\00"
@.str.13 = private unnamed_addr constant [21 x i8] c"main.vs:50:3\00println\00"
@.str.14 = private unnamed_addr constant [13 x i8] c"main.vs:52:3\00"
@.str.15 = private unnamed_addr constant [19 x i8] c"main.vs:53:5\00limit\00"
@.str.16 = private unnamed_addr constant [10 x i8] c"overflow \00"
@.str.17 = private unnamed_addr constant [22 x i8] c"main.vs:55:25\00println\00"
@.str.18 = private unnamed_addr constant [11 x i8] c"malformed \00"
@.str.19 = private unnamed_addr constant [22 x i8] c"main.vs:56:25\00println\00"
@.str.20 = private unnamed_addr constant [13 x i8] c"main.vs:54:7\00"
@.str.21 = private unnamed_addr constant [6 x i8] c"fine \00"
@.str.22 = private unnamed_addr constant [21 x i8] c"main.vs:61:5\00println\00"
@.str.23 = private unnamed_addr constant [8 x i8] c" failed\00"
@.str.24 = private unnamed_addr constant [21 x i8] c"main.vs:63:3\00println\00"
@.str.25 = private unnamed_addr constant [29 x i8] c"std/prelude/adapters.vs:15:5\00"
@.str.26 = private unnamed_addr constant [20 x i8] c"main.vs:50:37\00weigh\00"
