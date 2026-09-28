%S.main.Pair = type { i32, i64 }
define void @v_main.main() {
entry:
  %a2 = alloca ptr
  %a4 = alloca ptr
  %a8 = alloca { i64, i64, i1 }
  %a11 = alloca i64
  %a14 = alloca i64
  %a15 = alloca i1
  %a16 = alloca i1
  %a18 = alloca i1
  %a30 = alloca i64
  %a43 = alloca i1
  %a57 = alloca ptr
  %a60 = alloca [21 x i8]
  %a71 = alloca [21 x i8]
  %a78 = alloca %str
  %a83 = alloca %str
  %a85 = alloca %str
  %a86 = alloca [7 x %str]
  %a98 = alloca ptr
  %a107 = alloca [21 x i8]
  %a111 = alloca %str
  %a112 = alloca [1 x %str]
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
  br label %loop.cond.1
loop.cond.1:
  %t17 = load i1, ptr %a15
  br i1 %t17, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a16
  br label %if.end.6
if.else.7:
  %t19 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t20 = load i1, ptr %t19
  br i1 %t20, label %if.then.8, label %if.else.10
if.then.8:
  %t21 = load i64, ptr %a11
  %t22 = load i64, ptr %a14
  %t23 = icmp sle i64 %t21, %t22
  store i1 %t23, ptr %a18
  br label %if.end.9
if.else.10:
  %t24 = load i64, ptr %a11
  %t25 = load i64, ptr %a14
  %t26 = icmp slt i64 %t24, %t25
  store i1 %t26, ptr %a18
  br label %if.end.9
if.end.9:
  %t27 = load i1, ptr %a18
  store i1 %t27, ptr %a16
  br label %if.end.6
if.end.6:
  %t28 = load i1, ptr %a16
  br i1 %t28, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t29 = load i64, ptr %a11
  store i64 %t29, ptr %a30
  %t31 = load i64, ptr %a30
  %t33 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t31, i64 10)
  %t34 = extractvalue { i64, i1 } %t33, 0
  %t35 = extractvalue { i64, i1 } %t33, 1
  br i1 %t35, label %overflow.11, label %arith.ok.12
overflow.11:
  %t36 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t37 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t38 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t39 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t36, i64 %t37, ptr %t38, i64 %t39)
  unreachable
arith.ok.12:
  %t40 = load ptr, ptr %a4
  store i64 %t34, ptr %t40
  %t41 = load ptr, ptr %a4
  %t42 = getelementptr i64, ptr %t41, i64 1
  store ptr %t42, ptr %a4
  br label %loop.post.2
loop.post.2:
  %t44 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t45 = load i1, ptr %t44
  store i1 %t45, ptr %a43
  br i1 %t45, label %sc.rhs.13, label %sc.end.14
sc.rhs.13:
  %t46 = load i64, ptr %a11
  %t47 = load i64, ptr %a14
  %t48 = icmp eq i64 %t46, %t47
  store i1 %t48, ptr %a43
  br label %sc.end.14
sc.end.14:
  %t49 = load i1, ptr %a43
  br i1 %t49, label %if.then.15, label %if.else.17
if.then.15:
  store i1 true, ptr %a15
  br label %if.end.16
if.else.17:
  %t50 = load i64, ptr %a11
  %t51 = add i64 %t50, 1
  store i64 %t51, ptr %a11
  br label %if.end.16
if.end.16:
  %t52 = load volatile i32, ptr @veles_stop_requested, align 4
  %t53 = icmp ne i32 %t52, 0
  br i1 %t53, label %safepoint.18, label %safepoint.on.19, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.18:
  call void @veles_gc_park()
  br label %safepoint.on.19
safepoint.on.19:
  br label %loop.cond.1
loop.end.3:
  %t54 = load ptr, ptr %a4
  %t55 = sub i64 0, 1
  %t56 = getelementptr i64, ptr %t54, i64 %t55
  store ptr %t56, ptr %a57
  %t58 = load ptr, ptr %a57
  %t59 = load i64, ptr %t58
  %t61 = call i64 @veles_i64_format(ptr %a60, i64 %t59)
  %t62 = insertvalue %str undef, ptr %a60, 0
  %t63 = insertvalue %str %t62, i64 %t61, 1
  %t64 = load ptr, ptr %a4
  %t65 = load ptr, ptr %a2
  %t66 = ptrtoint ptr %t64 to i64
  %t67 = ptrtoint ptr %t65 to i64
  %t68 = sub i64 %t66, %t67
  %t69 = ptrtoint ptr getelementptr (i64, ptr null, i64 1) to i64
  %t70 = sdiv i64 %t68, %t69
  %t72 = call i64 @veles_i64_format(ptr %a71, i64 %t70)
  %t73 = insertvalue %str undef, ptr %a71, 0
  %t74 = insertvalue %str %t73, i64 %t72, 1
  %t75 = load ptr, ptr %a2
  %t76 = load ptr, ptr %a4
  %t77 = icmp ult ptr %t75, %t76
  call void @veles_bool_to_string(ptr %a78, i1 zeroext %t77)
  %t79 = load %str, ptr %a78
  %t80 = load ptr, ptr %a57
  %t81 = load ptr, ptr %a4
  %t82 = icmp uge ptr %t80, %t81
  call void @veles_bool_to_string(ptr %a83, i1 zeroext %t82)
  %t84 = load %str, ptr %a83
  %t87 = getelementptr [7 x %str], ptr %a86, i64 0, i64 0
  store %str %t63, ptr %t87
  %t88 = getelementptr [7 x %str], ptr %a86, i64 0, i64 1
  store %str { ptr @.str.3, i64 1 }, ptr %t88
  %t89 = getelementptr [7 x %str], ptr %a86, i64 0, i64 2
  store %str %t74, ptr %t89
  %t90 = getelementptr [7 x %str], ptr %a86, i64 0, i64 3
  store %str { ptr @.str.3, i64 1 }, ptr %t90
  %t91 = getelementptr [7 x %str], ptr %a86, i64 0, i64 4
  store %str %t79, ptr %t91
  %t92 = getelementptr [7 x %str], ptr %a86, i64 0, i64 5
  store %str { ptr @.str.3, i64 1 }, ptr %t92
  %t93 = getelementptr [7 x %str], ptr %a86, i64 0, i64 6
  store %str %t84, ptr %t93
  call void @veles_string_concat_n(ptr %a85, ptr %a86, i64 7)
  %t94 = load %str, ptr %a85
  %t95 = extractvalue %str { ptr @.str.4, i64 20 }, 0
  %t96 = extractvalue %str { ptr @.str.4, i64 20 }, 1
  call void @veles_call_push(ptr %t95)
  call void @v_std.io.println(%str %t94)
  call void @veles_call_pop()
  %t97 = load ptr, ptr %a2
  store ptr %t97, ptr %a98
  %t99 = load ptr, ptr %a98
  %t100 = getelementptr %S.main.Pair, ptr %t99, i64 1
  %t101 = load ptr, ptr %a2
  %t102 = ptrtoint ptr %t100 to i64
  %t103 = ptrtoint ptr %t101 to i64
  %t104 = sub i64 %t102, %t103
  %t105 = ptrtoint ptr getelementptr (i8, ptr null, i64 1) to i64
  %t106 = sdiv i64 %t104, %t105
  %t108 = call i64 @veles_i64_format(ptr %a107, i64 %t106)
  %t109 = insertvalue %str undef, ptr %a107, 0
  %t110 = insertvalue %str %t109, i64 %t108, 1
  %t113 = getelementptr [1 x %str], ptr %a112, i64 0, i64 0
  store %str %t110, ptr %t113
  call void @veles_string_concat_n(ptr %a111, ptr %a112, i64 1)
  %t114 = load %str, ptr %a111
  %t115 = extractvalue %str { ptr @.str.5, i64 20 }, 0
  %t116 = extractvalue %str { ptr @.str.5, i64 20 }, 1
  call void @veles_call_push(ptr %t115)
  call void @v_std.io.println(%str %t114)
  call void @veles_call_pop()
  %t117 = load ptr, ptr %a2
  call void @veles_blocking_enter()
  call void @free(ptr %t117)
  call void @veles_blocking_leave()
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:21:12\00"
@.str.3 = private unnamed_addr constant [2 x i8] c" \00"
@.str.4 = private unnamed_addr constant [21 x i8] c"main.vs:25:5\00println\00"
@.str.5 = private unnamed_addr constant [21 x i8] c"main.vs:27:5\00println\00"
