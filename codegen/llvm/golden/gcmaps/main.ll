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
  %a36 = alloca ptr
  %a42 = alloca ptr
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
  %t29 = icmp eq ptr %t27, null
  br i1 %t29, label %share.done.14, label %share.test.12
share.test.12:
  %t30 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t27, i32 0, i32 6
  %t31 = load i64, ptr %t30
  %t32 = and i64 %t31, 4
  %t33 = icmp ne i64 %t32, 0
  br i1 %t33, label %share.done.14, label %share.call.13
share.call.13:
  %t34 = call ptr @veles_list_share(ptr %t27, ptr null)
  store ptr %t34, ptr %a28
  br label %share.done.14
share.done.14:
  %t35 = load ptr, ptr %a28
  store ptr %t35, ptr %a36
  br label %when.body.10
when.body.10:
  %t37 = insertvalue { ptr, ptr } undef, ptr @v_main.weigh.lambda1.plain, 0
  %t38 = insertvalue { ptr, ptr } %t37, ptr null, 1
  %t39 = extractvalue %str { ptr @.str.1, i64 17 }, 0
  %t40 = extractvalue %str { ptr @.str.1, i64 17 }, 1
  call void @veles_call_push(ptr %t39)
  %t41 = call ptr @v_std.prelude.extend.List_T.map_T_string_i64_Never.plain(ptr %a36, { ptr, ptr } %t38)
  call void @veles_call_pop()
  store ptr %t41, ptr %a42
  %t43 = extractvalue %str { ptr @.str.2, i64 17 }, 0
  %t44 = extractvalue %str { ptr @.str.2, i64 17 }, 1
  call void @veles_call_push(ptr %t43)
  %t45 = call i64 @v_std.prelude.extend.List_i64.sum(ptr %a42)
  call void @veles_call_pop()
  store i64 %t45, ptr %a2
  br label %when.end.1
when.arm.9:
  %t46 = extractvalue %str { ptr @.str.3, i64 33 }, 0
  %t47 = extractvalue %str { ptr @.str.3, i64 33 }, 1
  %t48 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t49 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t46, i64 %t47, ptr %t48, i64 %t49)
  unreachable
when.end.1:
  %t50 = load i64, ptr %a2
  ret i64 %t50
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
  %a1 = alloca ptr
  %a5 = alloca { i64, i64, i1 }
  %a8 = alloca i64
  %a11 = alloca i64
  %a12 = alloca i1
  %a13 = alloca i64
  %a14 = alloca i1
  %a16 = alloca i1
  %a27 = alloca i64
  %a41 = alloca i1
  %a43 = alloca i1
  %a58 = alloca i64
  %a70 = alloca %V.main.Token
  %a88 = alloca %V.main.Token
  %a98 = alloca [21 x i8]
  %a102 = alloca %str
  %a110 = alloca %V.main.Token
  %a128 = alloca %V.main.Token
  %a137 = alloca %str
  %a139 = alloca [21 x i8]
  %a143 = alloca %str
  %a150 = alloca %V.main.Token
  %a168 = alloca %V.main.Token
  %a169 = alloca i1
  %a185 = alloca ptr
  %a189 = alloca [21 x i8]
  %a193 = alloca %str
  %a194 = alloca [1 x %str]
  %a199 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a202 = alloca ptr
  %a203 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a206 = alloca i64
  %a207 = alloca ptr
  %a213 = alloca i64
  %a216 = alloca i64
  %a217 = alloca i64
  %a233 = alloca i64
  %a238 = alloca %V._prelude_.Result_i64_main.Malformed_main.Overflow_
  %a239 = alloca i64
  %a246 = alloca %U.main.Malformed_main.Overflow
  %a248 = alloca %U.main.Malformed_main.Overflow
  %a253 = alloca %U.main.Malformed_main.Overflow
  %a257 = alloca [21 x i8]
  %a261 = alloca %str
  %a273 = alloca %U.main.Malformed_main.Overflow
  %a277 = alloca %str
  %a306 = alloca %S.main.Failure
  %a312 = alloca [21 x i8]
  %a316 = alloca %str
  %a337 = alloca [21 x i8]
  %a341 = alloca %str
  store ptr @const.empty.list.0, ptr %a1
  %t2 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t3 = insertvalue { i64, i64, i1 } %t2, i64 200, 1
  %t4 = insertvalue { i64, i64, i1 } %t3, i1 true, 2
  store { i64, i64, i1 } %t4, ptr %a5
  %t6 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 0
  %t7 = load i64, ptr %t6
  store i64 %t7, ptr %a8
  %t9 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 1
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a11
  store i1 false, ptr %a12
  store i64 0, ptr %a13
  br label %chunk.cond.1
chunk.cond.1:
  %t15 = load i1, ptr %a12
  br i1 %t15, label %if.then.8, label %if.else.10
if.then.8:
  store i1 false, ptr %a14
  br label %if.end.9
if.else.10:
  %t17 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t18 = load i1, ptr %t17
  br i1 %t18, label %if.then.11, label %if.else.13
if.then.11:
  %t19 = load i64, ptr %a8
  %t20 = load i64, ptr %a11
  %t21 = icmp sle i64 %t19, %t20
  store i1 %t21, ptr %a16
  br label %if.end.12
if.else.13:
  %t22 = load i64, ptr %a8
  %t23 = load i64, ptr %a11
  %t24 = icmp slt i64 %t22, %t23
  store i1 %t24, ptr %a16
  br label %if.end.12
if.end.12:
  %t25 = load i1, ptr %a16
  store i1 %t25, ptr %a14
  br label %if.end.9
if.end.9:
  %t26 = load i1, ptr %a14
  br i1 %t26, label %chunk.start.7, label %chunk.end.3
chunk.start.7:
  %t28 = load i64, ptr %a8
  %t29 = icmp sgt i64 %t28, 9223372036854775543
  br i1 %t29, label %if.then.14, label %if.else.16
if.then.14:
  store i64 9223372036854775807, ptr %a27
  br label %if.end.15
if.else.16:
  %t30 = load i64, ptr %a8
  %t32 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t30, i64 264)
  %t33 = extractvalue { i64, i1 } %t32, 0
  %t34 = extractvalue { i64, i1 } %t32, 1
  br i1 %t34, label %overflow.17, label %arith.ok.18
overflow.17:
  %t35 = extractvalue %str { ptr @.str.6, i64 16 }, 0
  %t36 = extractvalue %str { ptr @.str.6, i64 16 }, 1
  %t37 = extractvalue %str { ptr @.str.7, i64 0 }, 0
  %t38 = extractvalue %str { ptr @.str.7, i64 0 }, 1
  call void @veles_panic_at(ptr %t35, i64 %t36, ptr %t37, i64 %t38)
  unreachable
arith.ok.18:
  store i64 %t33, ptr %a27
  br label %if.end.15
if.end.15:
  %t39 = load i64, ptr %a27
  store i64 %t39, ptr %a13
  br label %chunk.body.5
chunk.inner.4:
  %t42 = load i1, ptr %a12
  br i1 %t42, label %if.then.19, label %if.else.21
if.then.19:
  store i1 false, ptr %a41
  br label %if.end.20
if.else.21:
  %t44 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t45 = load i1, ptr %t44
  br i1 %t45, label %if.then.22, label %if.else.24
if.then.22:
  %t46 = load i64, ptr %a8
  %t47 = load i64, ptr %a11
  %t48 = icmp sle i64 %t46, %t47
  store i1 %t48, ptr %a43
  br label %if.end.23
if.else.24:
  %t49 = load i64, ptr %a8
  %t50 = load i64, ptr %a11
  %t51 = icmp slt i64 %t49, %t50
  store i1 %t51, ptr %a43
  br label %if.end.23
if.end.23:
  %t52 = load i1, ptr %a43
  store i1 %t52, ptr %a41
  br label %if.end.20
if.end.20:
  %t53 = load i1, ptr %a41
  %t54 = load i64, ptr %a8
  %t55 = load i64, ptr %a13
  %t56 = icmp slt i64 %t54, %t55
  %t40 = and i1 %t53, %t56
  br i1 %t40, label %chunk.body.5, label %chunk.looked.6
chunk.body.5:
  %t57 = load i64, ptr %a8
  store i64 %t57, ptr %a58
  %t59 = load ptr, ptr %a1
  %t60 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t59, i32 0, i32 6
  %t61 = load i64, ptr %t60
  %t62 = and i64 %t61, 12
  %t63 = icmp ne i64 %t62, 0
  br i1 %t63, label %cow.copy.25, label %cow.done.26, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.25:
  %t64 = call ptr @veles_list_unique_grow(ptr %t59, ptr @share.main.Token)
  store ptr %t64, ptr %a1
  br label %cow.done.26
cow.done.26:
  %t65 = load ptr, ptr %a1
  %t66 = load i64, ptr %a58
  %t67 = load i64, ptr %a58
  %t68 = insertvalue %S.main.Number undef, i64 %t66, 0
  %t69 = insertvalue %S.main.Number %t68, i64 %t67, 1
  store %V.main.Token zeroinitializer, ptr %a70
  %t71 = getelementptr inbounds %V.main.Token, ptr %a70, i32 0, i32 0
  store i32 0, ptr %t71
  %t72 = getelementptr inbounds %V.main.Token, ptr %a70, i32 0, i32 1
  store %S.main.Number %t69, ptr %t72
  %t73 = load %V.main.Token, ptr %a70
  %t74 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t65, i32 0, i32 1
  %t75 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t65, i32 0, i32 2
  %t76 = load i64, ptr %t74
  %t77 = load i64, ptr %t75
  %t78 = icmp slt i64 %t76, %t77
  br i1 %t78, label %push.fast.27, label %push.grow.28, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.27:
  %t79 = load ptr, ptr %t65
  %t80 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t65, i32 0, i32 3
  %t81 = load i64, ptr %t80
  %t82 = mul i64 %t81, %t76
  %t83 = getelementptr inbounds i8, ptr %t79, i64 %t82
  store %V.main.Token %t73, ptr %t83
  %t84 = add i64 %t76, 1
  store i64 %t84, ptr %t74
  %t85 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t65, i32 0, i32 5
  %t86 = load i64, ptr %t85
  %t87 = add i64 %t86, 1
  store i64 %t87, ptr %t85
  br label %push.done.29
push.grow.28:
  store %V.main.Token %t73, ptr %a88
  call void @veles_list_push(ptr %t65, ptr %a88)
  br label %push.done.29
push.done.29:
  %t89 = load ptr, ptr %a1
  %t90 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t89, i32 0, i32 6
  %t91 = load i64, ptr %t90
  %t92 = and i64 %t91, 12
  %t93 = icmp ne i64 %t92, 0
  br i1 %t93, label %cow.copy.30, label %cow.done.31, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.30:
  %t94 = call ptr @veles_list_unique_grow(ptr %t89, ptr @share.main.Token)
  store ptr %t94, ptr %a1
  br label %cow.done.31
cow.done.31:
  %t95 = load ptr, ptr %a1
  %t96 = load i64, ptr %a58
  %t97 = load i64, ptr %a58
  %t99 = call i64 @veles_i64_format(ptr %a98, i64 %t97)
  %t100 = insertvalue %str undef, ptr %a98, 0
  %t101 = insertvalue %str %t100, i64 %t99, 1
  %t103 = extractvalue %str { ptr @.str.8, i64 1 }, 0
  %t104 = extractvalue %str { ptr @.str.8, i64 1 }, 1
  %t105 = extractvalue %str %t101, 0
  %t106 = extractvalue %str %t101, 1
  call void @veles_string_concat(ptr %a102, ptr %t103, i64 %t104, ptr %t105, i64 %t106)
  %t107 = load %str, ptr %a102
  %t108 = insertvalue %S.main.Word undef, i64 %t96, 0
  %t109 = insertvalue %S.main.Word %t108, %str %t107, 1
  store %V.main.Token zeroinitializer, ptr %a110
  %t111 = getelementptr inbounds %V.main.Token, ptr %a110, i32 0, i32 0
  store i32 1, ptr %t111
  %t112 = getelementptr inbounds %V.main.Token, ptr %a110, i32 0, i32 1
  store %S.main.Word %t109, ptr %t112
  %t113 = load %V.main.Token, ptr %a110
  %t114 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t95, i32 0, i32 1
  %t115 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t95, i32 0, i32 2
  %t116 = load i64, ptr %t114
  %t117 = load i64, ptr %t115
  %t118 = icmp slt i64 %t116, %t117
  br i1 %t118, label %push.fast.32, label %push.grow.33, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.32:
  %t119 = load ptr, ptr %t95
  %t120 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t95, i32 0, i32 3
  %t121 = load i64, ptr %t120
  %t122 = mul i64 %t121, %t116
  %t123 = getelementptr inbounds i8, ptr %t119, i64 %t122
  store %V.main.Token %t113, ptr %t123
  %t124 = add i64 %t116, 1
  store i64 %t124, ptr %t114
  %t125 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t95, i32 0, i32 5
  %t126 = load i64, ptr %t125
  %t127 = add i64 %t126, 1
  store i64 %t127, ptr %t125
  br label %push.done.34
push.grow.33:
  store %V.main.Token %t113, ptr %a128
  call void @veles_list_push(ptr %t95, ptr %a128)
  br label %push.done.34
push.done.34:
  %t129 = load ptr, ptr %a1
  %t130 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t129, i32 0, i32 6
  %t131 = load i64, ptr %t130
  %t132 = and i64 %t131, 12
  %t133 = icmp ne i64 %t132, 0
  br i1 %t133, label %cow.copy.35, label %cow.done.36, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.35:
  %t134 = call ptr @veles_list_unique_grow(ptr %t129, ptr @share.main.Token)
  store ptr %t134, ptr %a1
  br label %cow.done.36
cow.done.36:
  %t135 = load ptr, ptr %a1
  %t136 = call ptr @veles_list_new(ptr @adesc.string, i64 2)
  store %str { ptr @.str.9, i64 2 }, ptr %a137
  call void @veles_list_push(ptr %t136, ptr %a137)
  %t138 = load i64, ptr %a58
  %t140 = call i64 @veles_i64_format(ptr %a139, i64 %t138)
  %t141 = insertvalue %str undef, ptr %a139, 0
  %t142 = insertvalue %str %t141, i64 %t140, 1
  %t144 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t145 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  %t146 = extractvalue %str %t142, 0
  %t147 = extractvalue %str %t142, 1
  call void @veles_string_concat(ptr %a143, ptr %t144, i64 %t145, ptr %t146, i64 %t147)
  %t148 = load %str, ptr %a143
  store %str %t148, ptr %a137
  call void @veles_list_push(ptr %t136, ptr %a137)
  %t149 = insertvalue %S.main.Names undef, ptr %t136, 0
  store %V.main.Token zeroinitializer, ptr %a150
  %t151 = getelementptr inbounds %V.main.Token, ptr %a150, i32 0, i32 0
  store i32 2, ptr %t151
  %t152 = getelementptr inbounds %V.main.Token, ptr %a150, i32 0, i32 1
  store %S.main.Names %t149, ptr %t152
  %t153 = load %V.main.Token, ptr %a150
  %t154 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t135, i32 0, i32 1
  %t155 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t135, i32 0, i32 2
  %t156 = load i64, ptr %t154
  %t157 = load i64, ptr %t155
  %t158 = icmp slt i64 %t156, %t157
  br i1 %t158, label %push.fast.37, label %push.grow.38, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.37:
  %t159 = load ptr, ptr %t135
  %t160 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t135, i32 0, i32 3
  %t161 = load i64, ptr %t160
  %t162 = mul i64 %t161, %t156
  %t163 = getelementptr inbounds i8, ptr %t159, i64 %t162
  store %V.main.Token %t153, ptr %t163
  %t164 = add i64 %t156, 1
  store i64 %t164, ptr %t154
  %t165 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t135, i32 0, i32 5
  %t166 = load i64, ptr %t165
  %t167 = add i64 %t166, 1
  store i64 %t167, ptr %t165
  br label %push.done.39
push.grow.38:
  store %V.main.Token %t153, ptr %a168
  call void @veles_list_push(ptr %t135, ptr %a168)
  br label %push.done.39
push.done.39:
  br label %chunk.post.2
chunk.post.2:
  %t170 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t171 = load i1, ptr %t170
  store i1 %t171, ptr %a169
  br i1 %t171, label %sc.rhs.40, label %sc.end.41
sc.rhs.40:
  %t172 = load i64, ptr %a8
  %t173 = load i64, ptr %a11
  %t174 = icmp eq i64 %t172, %t173
  store i1 %t174, ptr %a169
  br label %sc.end.41
sc.end.41:
  %t175 = load i1, ptr %a169
  br i1 %t175, label %if.then.42, label %if.else.44
if.then.42:
  store i1 true, ptr %a12
  br label %if.end.43
if.else.44:
  %t176 = load i64, ptr %a8
  %t177 = add i64 %t176, 1
  store i64 %t177, ptr %a8
  br label %if.end.43
if.end.43:
  br label %chunk.inner.4
chunk.looked.6:
  %t178 = load volatile i32, ptr @veles_attention_line, align 64
  %t179 = icmp ne i32 %t178, 0
  br i1 %t179, label %safepoint.45, label %safepoint.on.46, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.45:
  call void @veles_backedge_plain()
  br label %safepoint.on.46
safepoint.on.46:
  br label %chunk.cond.1
chunk.end.3:
  %t180 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2.plain, 0
  %t181 = insertvalue { ptr, ptr } %t180, ptr null, 1
  %t182 = extractvalue %str { ptr @.str.11, i64 17 }, 0
  %t183 = extractvalue %str { ptr @.str.11, i64 17 }, 1
  call void @veles_call_push(ptr %t182)
  %t184 = call ptr @v_std.prelude.extend.List_T.map_T_main.Token_i64_Never.plain(ptr %a1, { ptr, ptr } %t181)
  call void @veles_call_pop()
  store ptr %t184, ptr %a185
  %t186 = extractvalue %str { ptr @.str.12, i64 17 }, 0
  %t187 = extractvalue %str { ptr @.str.12, i64 17 }, 1
  call void @veles_call_push(ptr %t186)
  %t188 = call i64 @v_std.prelude.extend.List_i64.sum(ptr %a185)
  call void @veles_call_pop()
  %t190 = call i64 @veles_i64_format(ptr %a189, i64 %t188)
  %t191 = insertvalue %str undef, ptr %a189, 0
  %t192 = insertvalue %str %t191, i64 %t190, 1
  %t195 = getelementptr [1 x %str], ptr %a194, i64 0, i64 0
  store %str %t192, ptr %t195
  call void @veles_string_concat_n(ptr %a193, ptr %a194, i64 1)
  %t196 = load %str, ptr %a193
  %t197 = extractvalue %str { ptr @.str.13, i64 20 }, 0
  %t198 = extractvalue %str { ptr @.str.13, i64 20 }, 1
  call void @veles_call_push(ptr %t197)
  call void @v_std.io.println(%str %t196)
  call void @veles_call_pop()
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a199
  %t200 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a199, i32 0, i32 3
  store i64 24, ptr %t200
  %t201 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a199, i32 0, i32 4
  store ptr @adesc.main.Failure, ptr %t201
  store ptr %a199, ptr %a202
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a203
  %t204 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a203, i32 0, i32 3
  store i64 8, ptr %t204
  %t205 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a203, i32 0, i32 4
  store ptr @adesc.i64, ptr %t205
  call void @veles_list_reserve(ptr %a203, i64 3)
  store i64 5, ptr %a206
  call void @veles_list_push(ptr %a203, ptr %a206)
  store i64 -1, ptr %a206
  call void @veles_list_push(ptr %a203, ptr %a206)
  store i64 5000, ptr %a206
  call void @veles_list_push(ptr %a203, ptr %a206)
  store ptr %a203, ptr %a207
  %t208 = load ptr, ptr %a207
  %t209 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t208, i32 0, i32 6
  %t210 = load i64, ptr %t209
  %t211 = and i64 %t210, 12
  %t212 = icmp eq i64 %t211, 0
  store i64 0, ptr %a213
  br i1 %t212, label %lend.set.47, label %lend.done.48
lend.set.47:
  %t214 = or i64 %t210, 8
  store i64 %t214, ptr %t209
  store i64 1, ptr %a213
  br label %lend.done.48
lend.done.48:
  %t215 = load i64, ptr %a213
  store i64 %t215, ptr %a216
  store i64 0, ptr %a217
  br label %loop.cond.49
loop.cond.49:
  %t218 = load i64, ptr %a217
  %t219 = load ptr, ptr %a207
  %t220 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t219, i32 0, i32 1
  %t221 = load i64, ptr %t220
  %t222 = icmp slt i64 %t218, %t221
  br i1 %t222, label %loop.body.52, label %loop.end.51
loop.body.52:
  %t223 = load ptr, ptr %a207
  %t224 = load i64, ptr %a217
  %t225 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t223, i32 0, i32 1
  %t226 = load i64, ptr %t225
  %t227 = icmp ult i64 %t224, %t226
  br i1 %t227, label %idx.ok.53, label %idx.bad.54, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.54:
  %t228 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t229 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t223, i64 %t224, ptr %t228, i64 %t229)
  unreachable
idx.ok.53:
  %t230 = load ptr, ptr %t223
  %t231 = getelementptr inbounds [8 x i8], ptr %t230, i64 %t224
  %t232 = load i64, ptr %t231
  store i64 %t232, ptr %a233
  %t234 = load i64, ptr %a233
  %t235 = extractvalue %str { ptr @.str.15, i64 18 }, 0
  %t236 = extractvalue %str { ptr @.str.15, i64 18 }, 1
  call void @veles_call_push(ptr %t235)
  %t237 = call %V._prelude_.Result_i64_main.Malformed_main.Overflow_ @v_main.limit(i64 %t234)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t237, ptr %a238
  %t240 = load %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a238
  %t241 = extractvalue %V._prelude_.Result_i64_main.Malformed_main.Overflow_ %t240, 0
  %t242 = icmp eq i32 %t241, 1
  br i1 %t242, label %if.then.55, label %if.else.57
if.then.55:
  %t243 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a238, i32 0, i32 1
  %t244 = getelementptr inbounds %S._prelude_.Err_i64_main.Malformed_main.Overflow_, ptr %t243, i32 0, i32 0
  %t245 = load %U.main.Malformed_main.Overflow, ptr %t244
  store %U.main.Malformed_main.Overflow %t245, ptr %a246
  %t247 = load %U.main.Malformed_main.Overflow, ptr %a246
  store %U.main.Malformed_main.Overflow %t247, ptr %a248
  br label %when.arm.59
when.arm.59:
  %t249 = load %U.main.Malformed_main.Overflow, ptr %a248
  %t250 = extractvalue %U.main.Malformed_main.Overflow %t249, 0
  %t251 = icmp eq i32 %t250, 1
  br i1 %t251, label %when.bind.62, label %when.arm.60
when.bind.62:
  br label %when.body.61
when.body.61:
  %t252 = load %U.main.Malformed_main.Overflow, ptr %a246
  store %U.main.Malformed_main.Overflow %t252, ptr %a253
  %t254 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a253, i32 0, i32 1
  %t255 = load %S.main.Overflow, ptr %t254
  %t256 = extractvalue %S.main.Overflow %t255, 1
  %t258 = call i64 @veles_i64_format(ptr %a257, i64 %t256)
  %t259 = insertvalue %str undef, ptr %a257, 0
  %t260 = insertvalue %str %t259, i64 %t258, 1
  %t262 = extractvalue %str { ptr @.str.16, i64 9 }, 0
  %t263 = extractvalue %str { ptr @.str.16, i64 9 }, 1
  %t264 = extractvalue %str %t260, 0
  %t265 = extractvalue %str %t260, 1
  call void @veles_string_concat(ptr %a261, ptr %t262, i64 %t263, ptr %t264, i64 %t265)
  %t266 = load %str, ptr %a261
  %t267 = extractvalue %str { ptr @.str.17, i64 21 }, 0
  %t268 = extractvalue %str { ptr @.str.17, i64 21 }, 1
  call void @veles_call_push(ptr %t267)
  call void @v_std.io.println(%str %t266)
  call void @veles_call_pop()
  br label %when.end.58
when.arm.60:
  %t269 = load %U.main.Malformed_main.Overflow, ptr %a248
  %t270 = extractvalue %U.main.Malformed_main.Overflow %t269, 0
  %t271 = icmp eq i32 %t270, 0
  br i1 %t271, label %when.bind.65, label %when.arm.63
when.bind.65:
  br label %when.body.64
when.body.64:
  %t272 = load %U.main.Malformed_main.Overflow, ptr %a246
  store %U.main.Malformed_main.Overflow %t272, ptr %a273
  %t274 = getelementptr inbounds %U.main.Malformed_main.Overflow, ptr %a273, i32 0, i32 1
  %t275 = load %S.main.Malformed, ptr %t274
  %t276 = extractvalue %S.main.Malformed %t275, 0
  %t278 = extractvalue %str { ptr @.str.18, i64 10 }, 0
  %t279 = extractvalue %str { ptr @.str.18, i64 10 }, 1
  %t280 = extractvalue %str %t276, 0
  %t281 = extractvalue %str %t276, 1
  call void @veles_string_concat_shared(ptr %a277, ptr %t278, i64 %t279, ptr %t280, i64 %t281)
  %t282 = load %str, ptr %a277
  %t283 = extractvalue %str { ptr @.str.19, i64 21 }, 0
  %t284 = extractvalue %str { ptr @.str.19, i64 21 }, 1
  call void @veles_call_push(ptr %t283)
  call void @v_std.io.println(%str %t282)
  call void @veles_call_pop()
  br label %when.end.58
when.arm.63:
  %t285 = extractvalue %str { ptr @.str.3, i64 33 }, 0
  %t286 = extractvalue %str { ptr @.str.3, i64 33 }, 1
  %t287 = extractvalue %str { ptr @.str.20, i64 12 }, 0
  %t288 = extractvalue %str { ptr @.str.20, i64 12 }, 1
  call void @veles_panic_at(ptr %t285, i64 %t286, ptr %t287, i64 %t288)
  unreachable
when.end.58:
  %t289 = load ptr, ptr %a202
  %t290 = load %U.main.Malformed_main.Overflow, ptr %a246
  %t291 = insertvalue %S.main.Failure undef, %U.main.Malformed_main.Overflow %t290, 0
  %t292 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t289, i32 0, i32 1
  %t293 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t289, i32 0, i32 2
  %t294 = load i64, ptr %t292
  %t295 = load i64, ptr %t293
  %t296 = icmp slt i64 %t294, %t295
  br i1 %t296, label %push.fast.66, label %push.grow.67, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.66:
  %t297 = load ptr, ptr %t289
  %t298 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t289, i32 0, i32 3
  %t299 = load i64, ptr %t298
  %t300 = mul i64 %t299, %t294
  %t301 = getelementptr inbounds i8, ptr %t297, i64 %t300
  store %S.main.Failure %t291, ptr %t301
  %t302 = add i64 %t294, 1
  store i64 %t302, ptr %t292
  %t303 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t289, i32 0, i32 5
  %t304 = load i64, ptr %t303
  %t305 = add i64 %t304, 1
  store i64 %t305, ptr %t303
  br label %push.done.68
push.grow.67:
  store %S.main.Failure %t291, ptr %a306
  call void @veles_list_push(ptr %t289, ptr %a306)
  br label %push.done.68
push.done.68:
  br label %loop.post.50
if.else.57:
  %t307 = getelementptr inbounds %V._prelude_.Result_i64_main.Malformed_main.Overflow_, ptr %a238, i32 0, i32 1
  %t308 = getelementptr inbounds %S._prelude_.Ok_i64_main.Malformed_main.Overflow_, ptr %t307, i32 0, i32 0
  %t309 = load i64, ptr %t308
  store i64 %t309, ptr %a239
  br label %if.end.56
if.end.56:
  %t310 = load i64, ptr %a239
  %t311 = load i64, ptr %a233
  %t313 = call i64 @veles_i64_format(ptr %a312, i64 %t311)
  %t314 = insertvalue %str undef, ptr %a312, 0
  %t315 = insertvalue %str %t314, i64 %t313, 1
  %t317 = extractvalue %str { ptr @.str.21, i64 5 }, 0
  %t318 = extractvalue %str { ptr @.str.21, i64 5 }, 1
  %t319 = extractvalue %str %t315, 0
  %t320 = extractvalue %str %t315, 1
  call void @veles_string_concat(ptr %a316, ptr %t317, i64 %t318, ptr %t319, i64 %t320)
  %t321 = load %str, ptr %a316
  %t322 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t323 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t322)
  call void @v_std.io.println(%str %t321)
  call void @veles_call_pop()
  br label %loop.post.50
loop.post.50:
  %t324 = load i64, ptr %a217
  %t325 = add i64 %t324, 1
  store i64 %t325, ptr %a217
  %t326 = load volatile i32, ptr @veles_attention_line, align 64
  %t327 = icmp ne i32 %t326, 0
  br i1 %t327, label %safepoint.69, label %safepoint.on.70, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.69:
  call void @veles_backedge_plain()
  br label %safepoint.on.70
safepoint.on.70:
  br label %loop.cond.49
loop.end.51:
  %t328 = load ptr, ptr %a207
  %t329 = load i64, ptr %a216
  %t330 = icmp ne i64 %t329, 0
  br i1 %t330, label %unlend.clear.71, label %unlend.done.72
unlend.clear.71:
  %t331 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t328, i32 0, i32 6
  %t332 = load i64, ptr %t331
  %t333 = and i64 %t332, -9
  store i64 %t333, ptr %t331
  br label %unlend.done.72
unlend.done.72:
  %t334 = load ptr, ptr %a202
  %t335 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t334, i32 0, i32 1
  %t336 = load i64, ptr %t335
  %t338 = call i64 @veles_i64_format(ptr %a337, i64 %t336)
  %t339 = insertvalue %str undef, ptr %a337, 0
  %t340 = insertvalue %str %t339, i64 %t338, 1
  %t342 = extractvalue %str %t340, 0
  %t343 = extractvalue %str %t340, 1
  %t344 = extractvalue %str { ptr @.str.23, i64 7 }, 0
  %t345 = extractvalue %str { ptr @.str.23, i64 7 }, 1
  call void @veles_string_concat(ptr %a341, ptr %t342, i64 %t343, ptr %t344, i64 %t345)
  %t346 = load %str, ptr %a341
  %t347 = extractvalue %str { ptr @.str.24, i64 20 }, 0
  %t348 = extractvalue %str { ptr @.str.24, i64 20 }, 1
  call void @veles_call_push(ptr %t347)
  call void @v_std.io.println(%str %t346)
  call void @veles_call_pop()
  ret void
}

define ptr @v_std.prelude.extend.List_T.map_T_main.Token_i64_Never.plain(ptr %p0, { ptr, ptr } %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca { ptr, ptr }
  %a4 = alloca ptr
  %a12 = alloca ptr
  %a18 = alloca i64
  %a21 = alloca i64
  %a22 = alloca i64
  %a38 = alloca %V.main.Token
  %a40 = alloca %V.main.Token
  %a46 = alloca %V.main.Token
  %a63 = alloca i64
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
  %t13 = load ptr, ptr %a12
  %t14 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t13, i32 0, i32 6
  %t15 = load i64, ptr %t14
  %t16 = and i64 %t15, 12
  %t17 = icmp eq i64 %t16, 0
  store i64 0, ptr %a18
  br i1 %t17, label %lend.set.1, label %lend.done.2
lend.set.1:
  %t19 = or i64 %t15, 8
  store i64 %t19, ptr %t14
  store i64 1, ptr %a18
  br label %lend.done.2
lend.done.2:
  %t20 = load i64, ptr %a18
  store i64 %t20, ptr %a21
  store i64 0, ptr %a22
  br label %loop.cond.3
loop.cond.3:
  %t23 = load i64, ptr %a22
  %t24 = load ptr, ptr %a12
  %t25 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t24, i32 0, i32 1
  %t26 = load i64, ptr %t25
  %t27 = icmp slt i64 %t23, %t26
  br i1 %t27, label %loop.body.6, label %loop.end.5
loop.body.6:
  %t28 = load ptr, ptr %a12
  %t29 = load i64, ptr %a22
  %t30 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t28, i32 0, i32 1
  %t31 = load i64, ptr %t30
  %t32 = icmp ult i64 %t29, %t31
  br i1 %t32, label %idx.ok.7, label %idx.bad.8, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.8:
  %t33 = extractvalue %str { ptr @.str.25, i64 28 }, 0
  %t34 = extractvalue %str { ptr @.str.25, i64 28 }, 1
  call void @veles_list_index_panic(ptr %t28, i64 %t29, ptr %t33, i64 %t34)
  unreachable
idx.ok.7:
  %t35 = load ptr, ptr %t28
  %t36 = getelementptr inbounds [32 x i8], ptr %t35, i64 %t29
  %t37 = load %V.main.Token, ptr %t36
  store %V.main.Token %t37, ptr %a38
  call void @share.main.Token(ptr %a38)
  %t39 = load %V.main.Token, ptr %a38
  store %V.main.Token %t39, ptr %a40
  %t41 = load ptr, ptr %a4
  %t42 = load { ptr, ptr }, ptr %a2
  %t43 = extractvalue { ptr, ptr } %t42, 0
  %t44 = extractvalue { ptr, ptr } %t42, 1
  %t45 = load %V.main.Token, ptr %a40
  store %V.main.Token %t45, ptr %a46
  call void @share.main.Token(ptr %a46)
  %t47 = load %V.main.Token, ptr %a46
  %t48 = call i64 %t43(ptr %t44, %V.main.Token %t47)
  %t49 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t41, i32 0, i32 1
  %t50 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t41, i32 0, i32 2
  %t51 = load i64, ptr %t49
  %t52 = load i64, ptr %t50
  %t53 = icmp slt i64 %t51, %t52
  br i1 %t53, label %push.fast.9, label %push.grow.10, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.9:
  %t54 = load ptr, ptr %t41
  %t55 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t41, i32 0, i32 3
  %t56 = load i64, ptr %t55
  %t57 = mul i64 %t56, %t51
  %t58 = getelementptr inbounds i8, ptr %t54, i64 %t57
  store i64 %t48, ptr %t58
  %t59 = add i64 %t51, 1
  store i64 %t59, ptr %t49
  %t60 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t41, i32 0, i32 5
  %t61 = load i64, ptr %t60
  %t62 = add i64 %t61, 1
  store i64 %t62, ptr %t60
  br label %push.done.11
push.grow.10:
  store i64 %t48, ptr %a63
  call void @veles_list_push(ptr %t41, ptr %a63)
  br label %push.done.11
push.done.11:
  br label %loop.post.4
loop.post.4:
  %t64 = load i64, ptr %a22
  %t65 = add i64 %t64, 1
  store i64 %t65, ptr %a22
  %t66 = load volatile i32, ptr @veles_attention_line, align 64
  %t67 = icmp ne i32 %t66, 0
  br i1 %t67, label %safepoint.12, label %safepoint.on.13, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.12:
  call void @veles_backedge_plain()
  br label %safepoint.on.13
safepoint.on.13:
  br label %loop.cond.3
loop.end.5:
  %t68 = load ptr, ptr %a12
  %t69 = load i64, ptr %a21
  %t70 = icmp ne i64 %t69, 0
  br i1 %t70, label %unlend.clear.14, label %unlend.done.15
unlend.clear.14:
  %t71 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t68, i32 0, i32 6
  %t72 = load i64, ptr %t71
  %t73 = and i64 %t72, -9
  store i64 %t73, ptr %t71
  br label %unlend.done.15
unlend.done.15:
  %t74 = load ptr, ptr %a4
  ret ptr %t74
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

define internal void @share.main.Token(ptr %p) {
entry:
  %a7 = alloca ptr
  %t2 = getelementptr inbounds %V.main.Token, ptr %p, i32 0, i32 0
  %t1 = load i32, ptr %t2
  %t3 = getelementptr inbounds %V.main.Token, ptr %p, i32 0, i32 1
  %t4 = icmp eq i32 %t1, 2
  br i1 %t4, label %share.variant.2, label %share.next.3
share.variant.2:
  %t5 = getelementptr inbounds %S.main.Names, ptr %t3, i32 0, i32 0
  %t6 = load ptr, ptr %t5
  store ptr %t6, ptr %a7
  %t8 = icmp eq ptr %t6, null
  br i1 %t8, label %share.done.6, label %share.test.4
share.test.4:
  %t9 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t6, i32 0, i32 6
  %t10 = load i64, ptr %t9
  %t11 = and i64 %t10, 4
  %t12 = icmp ne i64 %t11, 0
  br i1 %t12, label %share.done.6, label %share.call.5
share.call.5:
  %t13 = call ptr @veles_list_share(ptr %t6, ptr null)
  store ptr %t13, ptr %a7
  br label %share.done.6
share.done.6:
  %t14 = load ptr, ptr %a7
  store ptr %t14, ptr %t5
  br label %share.done.1
share.next.3:
  br label %share.done.1
share.done.1:
  ret void
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
