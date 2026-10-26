%S.main.Data = type { i64 }
%S.main.Event = type <{ i32, %S.main.Data }>
%S.main.Spaced = type { i8, [15 x i8], i32, [12 x i8] }
%S.main.Counter = type { i64, [56 x i8] }
%S.main.Holder = type { i8, [63 x i8], %S.main.Counter }
define void @v_main.main() {
entry:
  %a2 = alloca %S.main.Data
  %a5 = alloca %S.main.Event
  %a11 = alloca %S.main.Data
  %a23 = alloca %S.main.Data
  %a28 = alloca %S.main.Spaced, align 16
  %a29 = alloca %S.main.Holder, align 64
  %a33 = alloca %S.main.Holder, align 64
  %a50 = alloca [21 x i8]
  %a57 = alloca %S.main.Data
  %a59 = alloca [21 x i8]
  %a66 = alloca [21 x i8]
  %a73 = alloca [21 x i8]
  %a77 = alloca %str
  %a78 = alloca [7 x %str]
  store %S.main.Data zeroinitializer, ptr %a2
  store i32 7, ptr %a2
  %t1 = load %S.main.Data, ptr %a2
  %t3 = insertvalue %S.main.Event undef, i32 1, 0
  %t4 = insertvalue %S.main.Event %t3, %S.main.Data %t1, 1
  store %S.main.Event %t4, ptr %a5
  %t6 = load %S.main.Event, ptr %a5
  %t7 = insertvalue %S.main.Event %t6, i32 2, 0
  store %S.main.Event %t7, ptr %a5
  %t8 = load %S.main.Event, ptr %a5
  %t9 = extractvalue %S.main.Event %t8, 1
  store %S.main.Data %t9, ptr %a11
  %t10 = load i32, ptr %a11
  %t13 = call { i32, i1 } @llvm.sadd.with.overflow.i32(i32 %t10, i32 1)
  %t14 = extractvalue { i32, i1 } %t13, 0
  %t15 = extractvalue { i32, i1 } %t13, 1
  br i1 %t15, label %overflow.1, label %arith.ok.2
overflow.1:
  %t16 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t17 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t18 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t19 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t16, i64 %t17, ptr %t18, i64 %t19)
  unreachable
arith.ok.2:
  %t20 = load %S.main.Event, ptr %a5
  %t21 = extractvalue %S.main.Event %t20, 1
  store %S.main.Data %t21, ptr %a23
  store i32 %t14, ptr %a23
  %t22 = load %S.main.Data, ptr %a23
  %t24 = load %S.main.Event, ptr %a5
  %t25 = insertvalue %S.main.Event %t24, %S.main.Data %t22, 1
  store %S.main.Event %t25, ptr %a5
  %t26 = insertvalue %S.main.Spaced undef, i8 1, 0
  %t27 = insertvalue %S.main.Spaced %t26, i32 2, 2
  store %S.main.Spaced %t27, ptr %a28
  %t30 = getelementptr inbounds %S.main.Holder, ptr %a29, i32 0, i32 0
  store i8 3, ptr %t30
  %t31 = insertvalue %S.main.Counter undef, i64 4, 0
  %t32 = getelementptr inbounds %S.main.Holder, ptr %a29, i32 0, i32 2
  store %S.main.Counter %t31, ptr %t32
  call void @llvm.memmove.p0.p0.i64(ptr %a33, ptr %a29, i64 128, i1 false)
  %t34 = getelementptr inbounds %S.main.Holder, ptr %a33, i32 0, i32 2
  %t35 = getelementptr inbounds %S.main.Counter, ptr %t34, i32 0, i32 0
  %t36 = load i64, ptr %t35
  %t38 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t36, i64 1)
  %t39 = extractvalue { i64, i1 } %t38, 0
  %t40 = extractvalue { i64, i1 } %t38, 1
  br i1 %t40, label %overflow.3, label %arith.ok.4
overflow.3:
  %t41 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t42 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t43 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t44 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_panic_at(ptr %t41, i64 %t42, ptr %t43, i64 %t44)
  unreachable
arith.ok.4:
  %t45 = getelementptr inbounds %S.main.Holder, ptr %a33, i32 0, i32 2
  %t46 = getelementptr inbounds %S.main.Counter, ptr %t45, i32 0, i32 0
  store i64 %t39, ptr %t46
  %t47 = load %S.main.Event, ptr %a5
  %t48 = extractvalue %S.main.Event %t47, 0
  %t49 = zext i32 %t48 to i64
  %t51 = call i64 @veles_u64_format(ptr %a50, i64 %t49)
  %t52 = insertvalue %str undef, ptr %a50, 0
  %t53 = insertvalue %str %t52, i64 %t51, 1
  %t54 = load %S.main.Event, ptr %a5
  %t55 = extractvalue %S.main.Event %t54, 1
  store %S.main.Data %t55, ptr %a57
  %t56 = load i32, ptr %a57
  %t58 = sext i32 %t56 to i64
  %t60 = call i64 @veles_i64_format(ptr %a59, i64 %t58)
  %t61 = insertvalue %str undef, ptr %a59, 0
  %t62 = insertvalue %str %t61, i64 %t60, 1
  %t63 = getelementptr inbounds %S.main.Spaced, ptr %a28, i32 0, i32 2
  %t64 = load i32, ptr %t63
  %t65 = sext i32 %t64 to i64
  %t67 = call i64 @veles_i64_format(ptr %a66, i64 %t65)
  %t68 = insertvalue %str undef, ptr %a66, 0
  %t69 = insertvalue %str %t68, i64 %t67, 1
  %t70 = getelementptr inbounds %S.main.Holder, ptr %a33, i32 0, i32 2
  %t71 = getelementptr inbounds %S.main.Counter, ptr %t70, i32 0, i32 0
  %t72 = load i64, ptr %t71
  %t74 = call i64 @veles_i64_format(ptr %a73, i64 %t72)
  %t75 = insertvalue %str undef, ptr %a73, 0
  %t76 = insertvalue %str %t75, i64 %t74, 1
  %t79 = getelementptr [7 x %str], ptr %a78, i64 0, i64 0
  store %str %t53, ptr %t79
  %t80 = getelementptr [7 x %str], ptr %a78, i64 0, i64 1
  store %str { ptr @.str.4, i64 1 }, ptr %t80
  %t81 = getelementptr [7 x %str], ptr %a78, i64 0, i64 2
  store %str %t62, ptr %t81
  %t82 = getelementptr [7 x %str], ptr %a78, i64 0, i64 3
  store %str { ptr @.str.4, i64 1 }, ptr %t82
  %t83 = getelementptr [7 x %str], ptr %a78, i64 0, i64 4
  store %str %t69, ptr %t83
  %t84 = getelementptr [7 x %str], ptr %a78, i64 0, i64 5
  store %str { ptr @.str.4, i64 1 }, ptr %t84
  %t85 = getelementptr [7 x %str], ptr %a78, i64 0, i64 6
  store %str %t76, ptr %t85
  call void @veles_string_concat_n(ptr %a77, ptr %a78, i64 7)
  %t86 = load %str, ptr %a77
  %t87 = extractvalue %str { ptr @.str.5, i64 20 }, 0
  %t88 = extractvalue %str { ptr @.str.5, i64 20 }, 1
  call void @veles_call_push(ptr %t87)
  call void @v_std.io.println(%str %t86)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:39:18\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:43:14\00"
@.str.4 = private unnamed_addr constant [2 x i8] c" \00"
@.str.5 = private unnamed_addr constant [21 x i8] c"main.vs:45:3\00println\00"
