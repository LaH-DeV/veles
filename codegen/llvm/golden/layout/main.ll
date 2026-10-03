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
  %a32 = alloca %S.main.Holder, align 64
  %a49 = alloca [21 x i8]
  %a56 = alloca %S.main.Data
  %a58 = alloca [21 x i8]
  %a65 = alloca [21 x i8]
  %a72 = alloca [21 x i8]
  %a76 = alloca %str
  %a77 = alloca [7 x %str]
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
  %t29 = insertvalue %S.main.Counter undef, i64 4, 0
  %t30 = insertvalue %S.main.Holder undef, i8 3, 0
  %t31 = insertvalue %S.main.Holder %t30, %S.main.Counter %t29, 2
  store %S.main.Holder %t31, ptr %a32
  %t33 = getelementptr inbounds %S.main.Holder, ptr %a32, i32 0, i32 2
  %t34 = getelementptr inbounds %S.main.Counter, ptr %t33, i32 0, i32 0
  %t35 = load i64, ptr %t34
  %t37 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t35, i64 1)
  %t38 = extractvalue { i64, i1 } %t37, 0
  %t39 = extractvalue { i64, i1 } %t37, 1
  br i1 %t39, label %overflow.3, label %arith.ok.4
overflow.3:
  %t40 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t41 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t42 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t43 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_panic_at(ptr %t40, i64 %t41, ptr %t42, i64 %t43)
  unreachable
arith.ok.4:
  %t44 = getelementptr inbounds %S.main.Holder, ptr %a32, i32 0, i32 2
  %t45 = getelementptr inbounds %S.main.Counter, ptr %t44, i32 0, i32 0
  store i64 %t38, ptr %t45
  %t46 = load %S.main.Event, ptr %a5
  %t47 = extractvalue %S.main.Event %t46, 0
  %t48 = zext i32 %t47 to i64
  %t50 = call i64 @veles_u64_format(ptr %a49, i64 %t48)
  %t51 = insertvalue %str undef, ptr %a49, 0
  %t52 = insertvalue %str %t51, i64 %t50, 1
  %t53 = load %S.main.Event, ptr %a5
  %t54 = extractvalue %S.main.Event %t53, 1
  store %S.main.Data %t54, ptr %a56
  %t55 = load i32, ptr %a56
  %t57 = sext i32 %t55 to i64
  %t59 = call i64 @veles_i64_format(ptr %a58, i64 %t57)
  %t60 = insertvalue %str undef, ptr %a58, 0
  %t61 = insertvalue %str %t60, i64 %t59, 1
  %t62 = getelementptr inbounds %S.main.Spaced, ptr %a28, i32 0, i32 2
  %t63 = load i32, ptr %t62
  %t64 = sext i32 %t63 to i64
  %t66 = call i64 @veles_i64_format(ptr %a65, i64 %t64)
  %t67 = insertvalue %str undef, ptr %a65, 0
  %t68 = insertvalue %str %t67, i64 %t66, 1
  %t69 = getelementptr inbounds %S.main.Holder, ptr %a32, i32 0, i32 2
  %t70 = getelementptr inbounds %S.main.Counter, ptr %t69, i32 0, i32 0
  %t71 = load i64, ptr %t70
  %t73 = call i64 @veles_i64_format(ptr %a72, i64 %t71)
  %t74 = insertvalue %str undef, ptr %a72, 0
  %t75 = insertvalue %str %t74, i64 %t73, 1
  %t78 = getelementptr [7 x %str], ptr %a77, i64 0, i64 0
  store %str %t52, ptr %t78
  %t79 = getelementptr [7 x %str], ptr %a77, i64 0, i64 1
  store %str { ptr @.str.4, i64 1 }, ptr %t79
  %t80 = getelementptr [7 x %str], ptr %a77, i64 0, i64 2
  store %str %t61, ptr %t80
  %t81 = getelementptr [7 x %str], ptr %a77, i64 0, i64 3
  store %str { ptr @.str.4, i64 1 }, ptr %t81
  %t82 = getelementptr [7 x %str], ptr %a77, i64 0, i64 4
  store %str %t68, ptr %t82
  %t83 = getelementptr [7 x %str], ptr %a77, i64 0, i64 5
  store %str { ptr @.str.4, i64 1 }, ptr %t83
  %t84 = getelementptr [7 x %str], ptr %a77, i64 0, i64 6
  store %str %t75, ptr %t84
  call void @veles_string_concat_n(ptr %a76, ptr %a77, i64 7)
  %t85 = load %str, ptr %a76
  %t86 = extractvalue %str { ptr @.str.5, i64 20 }, 0
  %t87 = extractvalue %str { ptr @.str.5, i64 20 }, 1
  call void @veles_call_push(ptr %t86)
  call void @v_std.io.println(%str %t85)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:39:18\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:43:14\00"
@.str.4 = private unnamed_addr constant [2 x i8] c" \00"
@.str.5 = private unnamed_addr constant [21 x i8] c"main.vs:45:3\00println\00"
