%S.main.Pair = type { i32, i64 }
define void @v_main.main() {
entry:
  %a2 = alloca ptr
  %a4 = alloca ptr
  %a8 = alloca { i64, i64, i1 }
  %a11 = alloca i64
  %a14 = alloca i64
  %a15 = alloca i1
  %a16 = alloca i64
  %a17 = alloca i1
  %a19 = alloca i1
  %a30 = alloca i64
  %a44 = alloca i1
  %a46 = alloca i1
  %a61 = alloca i64
  %a74 = alloca i1
  %a88 = alloca ptr
  %a91 = alloca [21 x i8]
  %a102 = alloca [21 x i8]
  %a109 = alloca %str
  %a114 = alloca %str
  %a116 = alloca %str
  %a117 = alloca [7 x %str]
  %a129 = alloca ptr
  %a138 = alloca [21 x i8]
  %a142 = alloca %str
  %a143 = alloca [1 x %str]
  call void @veles_blocking_enter()
  %t1 = call ptr @malloc(i64 32)
  call void @veles_blocking_leave()
  store ptr %t1, ptr %a2
  %t3 = load ptr, ptr %a2
  store ptr %t3, ptr %a4
  %t5 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t6 = insertvalue { i64, i64, i1 } %t5, i64 4, 1
  %t7 = insertvalue { i64, i64, i1 } %t6, i1 false, 2
  store { i64, i64, i1 } %t7, ptr %a8
  %t9 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 0
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a11
  %t12 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 1
  %t13 = load i64, ptr %t12
  store i64 %t13, ptr %a14
  store i1 false, ptr %a15
  store i64 0, ptr %a16
  br label %chunk.cond.1
chunk.cond.1:
  %t18 = load i1, ptr %a15
  br i1 %t18, label %if.then.8, label %if.else.10
if.then.8:
  store i1 false, ptr %a17
  br label %if.end.9
if.else.10:
  %t20 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t21 = load i1, ptr %t20
  br i1 %t21, label %if.then.11, label %if.else.13
if.then.11:
  %t22 = load i64, ptr %a11
  %t23 = load i64, ptr %a14
  %t24 = icmp sle i64 %t22, %t23
  store i1 %t24, ptr %a19
  br label %if.end.12
if.else.13:
  %t25 = load i64, ptr %a11
  %t26 = load i64, ptr %a14
  %t27 = icmp slt i64 %t25, %t26
  store i1 %t27, ptr %a19
  br label %if.end.12
if.end.12:
  %t28 = load i1, ptr %a19
  store i1 %t28, ptr %a17
  br label %if.end.9
if.end.9:
  %t29 = load i1, ptr %a17
  br i1 %t29, label %chunk.start.7, label %chunk.end.3
chunk.start.7:
  %t31 = load i64, ptr %a11
  %t32 = icmp sgt i64 %t31, 9223372036854775443
  br i1 %t32, label %if.then.14, label %if.else.16
if.then.14:
  store i64 9223372036854775807, ptr %a30
  br label %if.end.15
if.else.16:
  %t33 = load i64, ptr %a11
  %t35 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t33, i64 364)
  %t36 = extractvalue { i64, i1 } %t35, 0
  %t37 = extractvalue { i64, i1 } %t35, 1
  br i1 %t37, label %overflow.17, label %arith.ok.18
overflow.17:
  %t38 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t39 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t40 = extractvalue %str { ptr @.str.2, i64 0 }, 0
  %t41 = extractvalue %str { ptr @.str.2, i64 0 }, 1
  call void @veles_panic_at(ptr %t38, i64 %t39, ptr %t40, i64 %t41)
  unreachable
arith.ok.18:
  store i64 %t36, ptr %a30
  br label %if.end.15
if.end.15:
  %t42 = load i64, ptr %a30
  store i64 %t42, ptr %a16
  br label %chunk.body.5
chunk.inner.4:
  %t45 = load i1, ptr %a15
  br i1 %t45, label %if.then.19, label %if.else.21
if.then.19:
  store i1 false, ptr %a44
  br label %if.end.20
if.else.21:
  %t47 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t48 = load i1, ptr %t47
  br i1 %t48, label %if.then.22, label %if.else.24
if.then.22:
  %t49 = load i64, ptr %a11
  %t50 = load i64, ptr %a14
  %t51 = icmp sle i64 %t49, %t50
  store i1 %t51, ptr %a46
  br label %if.end.23
if.else.24:
  %t52 = load i64, ptr %a11
  %t53 = load i64, ptr %a14
  %t54 = icmp slt i64 %t52, %t53
  store i1 %t54, ptr %a46
  br label %if.end.23
if.end.23:
  %t55 = load i1, ptr %a46
  store i1 %t55, ptr %a44
  br label %if.end.20
if.end.20:
  %t56 = load i1, ptr %a44
  %t57 = load i64, ptr %a11
  %t58 = load i64, ptr %a16
  %t59 = icmp slt i64 %t57, %t58
  %t43 = and i1 %t56, %t59
  br i1 %t43, label %chunk.body.5, label %chunk.looked.6
chunk.body.5:
  %t60 = load i64, ptr %a11
  store i64 %t60, ptr %a61
  %t62 = load i64, ptr %a61
  %t64 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t62, i64 10)
  %t65 = extractvalue { i64, i1 } %t64, 0
  %t66 = extractvalue { i64, i1 } %t64, 1
  br i1 %t66, label %overflow.25, label %arith.ok.26
overflow.25:
  %t67 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t68 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t69 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t70 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_panic_at(ptr %t67, i64 %t68, ptr %t69, i64 %t70)
  unreachable
arith.ok.26:
  %t71 = load ptr, ptr %a4
  store i64 %t65, ptr %t71
  %t72 = load ptr, ptr %a4
  %t73 = getelementptr i64, ptr %t72, i64 1
  store ptr %t73, ptr %a4
  br label %chunk.post.2
chunk.post.2:
  %t75 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t76 = load i1, ptr %t75
  store i1 %t76, ptr %a74
  br i1 %t76, label %sc.rhs.27, label %sc.end.28
sc.rhs.27:
  %t77 = load i64, ptr %a11
  %t78 = load i64, ptr %a14
  %t79 = icmp eq i64 %t77, %t78
  store i1 %t79, ptr %a74
  br label %sc.end.28
sc.end.28:
  %t80 = load i1, ptr %a74
  br i1 %t80, label %if.then.29, label %if.else.31
if.then.29:
  store i1 true, ptr %a15
  br label %if.end.30
if.else.31:
  %t81 = load i64, ptr %a11
  %t82 = add i64 %t81, 1
  store i64 %t82, ptr %a11
  br label %if.end.30
if.end.30:
  br label %chunk.inner.4
chunk.looked.6:
  %t83 = load volatile i32, ptr @veles_attention_line, align 64
  %t84 = icmp ne i32 %t83, 0
  br i1 %t84, label %safepoint.32, label %safepoint.on.33, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.32:
  call void @veles_backedge_plain()
  br label %safepoint.on.33
safepoint.on.33:
  br label %chunk.cond.1
chunk.end.3:
  %t85 = load ptr, ptr %a4
  %t86 = sub i64 0, 1
  %t87 = getelementptr i64, ptr %t85, i64 %t86
  store ptr %t87, ptr %a88
  %t89 = load ptr, ptr %a88
  %t90 = load i64, ptr %t89
  %t92 = call i64 @veles_i64_format(ptr %a91, i64 %t90)
  %t93 = insertvalue %str undef, ptr %a91, 0
  %t94 = insertvalue %str %t93, i64 %t92, 1
  %t95 = load ptr, ptr %a4
  %t96 = load ptr, ptr %a2
  %t97 = ptrtoint ptr %t95 to i64
  %t98 = ptrtoint ptr %t96 to i64
  %t99 = sub i64 %t97, %t98
  %t100 = ptrtoint ptr getelementptr (i64, ptr null, i64 1) to i64
  %t101 = sdiv i64 %t99, %t100
  %t103 = call i64 @veles_i64_format(ptr %a102, i64 %t101)
  %t104 = insertvalue %str undef, ptr %a102, 0
  %t105 = insertvalue %str %t104, i64 %t103, 1
  %t106 = load ptr, ptr %a2
  %t107 = load ptr, ptr %a4
  %t108 = icmp ult ptr %t106, %t107
  call void @veles_bool_to_string(ptr %a109, i1 zeroext %t108)
  %t110 = load %str, ptr %a109
  %t111 = load ptr, ptr %a88
  %t112 = load ptr, ptr %a4
  %t113 = icmp uge ptr %t111, %t112
  call void @veles_bool_to_string(ptr %a114, i1 zeroext %t113)
  %t115 = load %str, ptr %a114
  %t118 = getelementptr [7 x %str], ptr %a117, i64 0, i64 0
  store %str %t94, ptr %t118
  %t119 = getelementptr [7 x %str], ptr %a117, i64 0, i64 1
  store %str { ptr @.str.4, i64 1 }, ptr %t119
  %t120 = getelementptr [7 x %str], ptr %a117, i64 0, i64 2
  store %str %t105, ptr %t120
  %t121 = getelementptr [7 x %str], ptr %a117, i64 0, i64 3
  store %str { ptr @.str.4, i64 1 }, ptr %t121
  %t122 = getelementptr [7 x %str], ptr %a117, i64 0, i64 4
  store %str %t110, ptr %t122
  %t123 = getelementptr [7 x %str], ptr %a117, i64 0, i64 5
  store %str { ptr @.str.4, i64 1 }, ptr %t123
  %t124 = getelementptr [7 x %str], ptr %a117, i64 0, i64 6
  store %str %t115, ptr %t124
  call void @veles_string_concat_n(ptr %a116, ptr %a117, i64 7)
  %t125 = load %str, ptr %a116
  %t126 = extractvalue %str { ptr @.str.5, i64 20 }, 0
  %t127 = extractvalue %str { ptr @.str.5, i64 20 }, 1
  call void @veles_call_push(ptr %t126)
  call void @v_std.io.println(%str %t125)
  call void @veles_call_pop()
  %t128 = load ptr, ptr %a2
  store ptr %t128, ptr %a129
  %t130 = load ptr, ptr %a129
  %t131 = getelementptr %S.main.Pair, ptr %t130, i64 1
  %t132 = load ptr, ptr %a2
  %t133 = ptrtoint ptr %t131 to i64
  %t134 = ptrtoint ptr %t132 to i64
  %t135 = sub i64 %t133, %t134
  %t136 = ptrtoint ptr getelementptr (i8, ptr null, i64 1) to i64
  %t137 = sdiv i64 %t135, %t136
  %t139 = call i64 @veles_i64_format(ptr %a138, i64 %t137)
  %t140 = insertvalue %str undef, ptr %a138, 0
  %t141 = insertvalue %str %t140, i64 %t139, 1
  %t144 = getelementptr [1 x %str], ptr %a143, i64 0, i64 0
  store %str %t141, ptr %t144
  call void @veles_string_concat_n(ptr %a142, ptr %a143, i64 1)
  %t145 = load %str, ptr %a142
  %t146 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t147 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t146)
  call void @v_std.io.println(%str %t145)
  call void @veles_call_pop()
  %t148 = load ptr, ptr %a2
  call void @veles_blocking_enter()
  call void @free(ptr %t148)
  call void @veles_blocking_leave()
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [1 x i8] c"\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:21:12\00"
@.str.4 = private unnamed_addr constant [2 x i8] c" \00"
@.str.5 = private unnamed_addr constant [21 x i8] c"main.vs:25:5\00println\00"
@.str.6 = private unnamed_addr constant [21 x i8] c"main.vs:27:5\00println\00"
