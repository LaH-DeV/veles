@const.main.PRIMES.data = private unnamed_addr constant [3 x i64] [i64 2, i64 3, i64 5]
@const.main.PRIMES = private unnamed_addr constant { ptr, i64, i64, i64, ptr, i64 } { ptr @const.main.PRIMES.data, i64 3, i64 3, i64 8, ptr @adesc.i64, i64 0 }
@const.main.NESTED.0.data = private unnamed_addr constant [1 x i8] [i8 1]
@const.main.NESTED.0 = private unnamed_addr constant { ptr, i64, i64, i64, ptr, i64 } { ptr @const.main.NESTED.0.data, i64 1, i64 1, i64 1, ptr @adesc.u8, i64 0 }
@const.main.NESTED.1.data = private unnamed_addr constant [0 x i8] zeroinitializer
@const.main.NESTED.1 = private unnamed_addr constant { ptr, i64, i64, i64, ptr, i64 } { ptr @const.main.NESTED.1.data, i64 0, i64 0, i64 1, ptr @adesc.u8, i64 0 }
@const.main.NESTED.data = private unnamed_addr constant [2 x ptr] [ptr @const.main.NESTED.0, ptr @const.main.NESTED.1]
@const.main.NESTED = private unnamed_addr constant { ptr, i64, i64, i64, ptr, i64 } { ptr @const.main.NESTED.data, i64 2, i64 2, i64 8, ptr @adesc.List_u8_, i64 0 }
@const.main.WORDS.keys = private unnamed_addr constant [2 x %str] [%str { ptr @.str.1, i64 3 }, %str { ptr @.str.2, i64 3 }]
@const.main.WORDS.vals = private unnamed_addr constant [2 x i64] [i64 1, i64 2]
@const.main.WORDS.meta = private unnamed_addr constant [2 x { i64, i8 }] [{ i64, i8 } { i64 4044999173930542849, i8 1 }, { i64, i8 } { i64 -7621125959628391285, i8 1 }]
@const.main.WORDS.index = private unnamed_addr constant [16 x i64] [i64 0, i64 1, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 2, i64 0, i64 0, i64 0, i64 0]
@const.main.WORDS = private unnamed_addr constant { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64 } { ptr @const.main.WORDS.keys, ptr @const.main.WORDS.vals, ptr @const.main.WORDS.meta, ptr @adesc.string, ptr @adesc.i64, ptr @const.main.WORDS.index, i64 16, i64 2, i64 2, i64 2, i64 16, i64 8, i64 0 }
define i64 @v_main.size() {
entry:
  %t2 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 1048576, i64 3)
  %t3 = extractvalue { i64, i1 } %t2, 0
  %t4 = extractvalue { i64, i1 } %t2, 1
  br i1 %t4, label %overflow.1, label %arith.ok.2
overflow.1:
  %t5 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t6 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t7 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t8 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t5, i64 %t6, ptr %t7, i64 %t8)
  unreachable
arith.ok.2:
  ret i64 %t3
}

define void @v_main.main() {
entry:
  %a4 = alloca [21 x i8]
  %a10 = alloca [21 x i8]
  %a19 = alloca %str
  %a22 = alloca { i1, i64 }
  %a28 = alloca i64
  %a33 = alloca [21 x i8]
  %a37 = alloca %str
  %a38 = alloca [11 x %str]
  %t1 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t2 = extractvalue %str { ptr @.str.5, i64 18 }, 1
  call void @veles_call_push(ptr %t1)
  %t3 = call i64 @v_main.size()
  call void @veles_call_pop()
  %t5 = call i64 @veles_i64_format(ptr %a4, i64 %t3)
  %t6 = insertvalue %str undef, ptr %a4, 0
  %t7 = insertvalue %str %t6, i64 %t5, 1
  %t8 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr @const.main.PRIMES, i32 0, i32 1
  %t9 = load i64, ptr %t8
  %t11 = call i64 @veles_i64_format(ptr %a10, i64 %t9)
  %t12 = insertvalue %str undef, ptr %a10, 0
  %t13 = insertvalue %str %t12, i64 %t11, 1
  %t14 = call %str @show.List_i64_(ptr @const.main.PRIMES)
  %t15 = call %str @show.List_List_u8__(ptr @const.main.NESTED)
  %t17 = extractvalue %str { ptr @.str.2, i64 3 }, 0
  %t18 = extractvalue %str { ptr @.str.2, i64 3 }, 1
  %t16 = call i64 @veles_hash_bytes(ptr %t17, i64 %t18)
  store %str { ptr @.str.2, i64 3 }, ptr %a19
  %t20 = call i64 @veles_map_find(ptr @const.main.WORDS, i64 %t16, ptr %a19, ptr @eqp.string)
  %t21 = icmp sge i64 %t20, 0
  store { i1, i64 } zeroinitializer, ptr %a22
  br i1 %t21, label %map.hit.1, label %map.end.2
map.hit.1:
  %t23 = call ptr @veles_map_val_at(ptr @const.main.WORDS, i64 %t20)
  %t24 = load i64, ptr %t23
  %t25 = insertvalue { i1, i64 } undef, i1 true, 0
  %t26 = insertvalue { i1, i64 } %t25, i64 %t24, 1
  store { i1, i64 } %t26, ptr %a22
  br label %map.end.2
map.end.2:
  %t27 = load { i1, i64 }, ptr %a22
  %t30 = extractvalue { i1, i64 } %t27, 0
  %t29 = xor i1 %t30, true
  br i1 %t29, label %elvis.default.3, label %elvis.some.4
elvis.some.4:
  %t31 = extractvalue { i1, i64 } %t27, 1
  store i64 %t31, ptr %a28
  br label %elvis.end.5
elvis.default.3:
  store i64 0, ptr %a28
  br label %elvis.end.5
elvis.end.5:
  %t32 = load i64, ptr %a28
  %t34 = call i64 @veles_i64_format(ptr %a33, i64 %t32)
  %t35 = insertvalue %str undef, ptr %a33, 0
  %t36 = insertvalue %str %t35, i64 %t34, 1
  %t39 = getelementptr [11 x %str], ptr %a38, i64 0, i64 0
  store %str %t7, ptr %t39
  %t40 = getelementptr [11 x %str], ptr %a38, i64 0, i64 1
  store %str { ptr @.str.6, i64 1 }, ptr %t40
  %t41 = getelementptr [11 x %str], ptr %a38, i64 0, i64 2
  store %str { ptr @.str.7, i64 12 }, ptr %t41
  %t42 = getelementptr [11 x %str], ptr %a38, i64 0, i64 3
  store %str { ptr @.str.6, i64 1 }, ptr %t42
  %t43 = getelementptr [11 x %str], ptr %a38, i64 0, i64 4
  store %str %t13, ptr %t43
  %t44 = getelementptr [11 x %str], ptr %a38, i64 0, i64 5
  store %str { ptr @.str.6, i64 1 }, ptr %t44
  %t45 = getelementptr [11 x %str], ptr %a38, i64 0, i64 6
  store %str %t14, ptr %t45
  %t46 = getelementptr [11 x %str], ptr %a38, i64 0, i64 7
  store %str { ptr @.str.6, i64 1 }, ptr %t46
  %t47 = getelementptr [11 x %str], ptr %a38, i64 0, i64 8
  store %str %t15, ptr %t47
  %t48 = getelementptr [11 x %str], ptr %a38, i64 0, i64 9
  store %str { ptr @.str.6, i64 1 }, ptr %t48
  %t49 = getelementptr [11 x %str], ptr %a38, i64 0, i64 10
  store %str %t36, ptr %t49
  call void @veles_string_concat_n(ptr %a37, ptr %a38, i64 11)
  %t50 = load %str, ptr %a37
  %t51 = extractvalue %str { ptr @.str.8, i64 20 }, 0
  %t52 = extractvalue %str { ptr @.str.8, i64 20 }, 1
  call void @veles_call_push(ptr %t51)
  call void @v_std.io.println(%str %t50)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [4 x i8] c"one\00"
@.str.2 = private unnamed_addr constant [4 x i8] c"two\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:13:20\00"
@.str.5 = private unnamed_addr constant [19 x i8] c"main.vs:16:17\00size\00"
@.str.6 = private unnamed_addr constant [2 x i8] c" \00"
@.str.7 = private unnamed_addr constant [13 x i8] c"size 1048576\00"
@.str.8 = private unnamed_addr constant [21 x i8] c"main.vs:16:3\00println\00"
