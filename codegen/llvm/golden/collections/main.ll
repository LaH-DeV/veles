%S.main.Counter = type { i64 }
@adesc.main.Counter = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 1, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a4 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a7 = alloca ptr
  %a24 = alloca %S.main.Counter
  %a41 = alloca %S.main.Counter
  %a47 = alloca ptr
  %a49 = alloca i64
  %a68 = alloca ptr
  %a83 = alloca ptr
  %a86 = alloca i64
  %a105 = alloca { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }
  %a110 = alloca ptr
  %a115 = alloca %str
  %a116 = alloca i64
  %a121 = alloca %str
  %a122 = alloca i64
  %a127 = alloca %str
  %a130 = alloca ptr
  %a133 = alloca ptr
  %a140 = alloca ptr
  %a156 = alloca %str
  %a158 = alloca { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }
  %a163 = alloca ptr
  %a166 = alloca i64
  %a172 = alloca i64
  %a177 = alloca ptr
  %a178 = alloca i64
  %a186 = alloca { i1, i64 }
  %a187 = alloca i1
  %a209 = alloca i64
  %a214 = alloca [21 x i8]
  %a219 = alloca ptr
  %a220 = alloca i64
  %a228 = alloca { i1, i64 }
  %a229 = alloca i1
  %a251 = alloca i64
  %a256 = alloca [21 x i8]
  %a261 = alloca ptr
  %a266 = alloca ptr
  %a267 = alloca i64
  %a275 = alloca { i1, %S.main.Counter }
  %a276 = alloca i1
  %a298 = alloca { i1, %S.main.Counter }
  %a299 = alloca { i1, i64 }
  %a311 = alloca ptr
  %a312 = alloca i64
  %a320 = alloca { i1, %S.main.Counter }
  %a321 = alloca i1
  %a343 = alloca { i1, %S.main.Counter }
  %a344 = alloca { i1, i64 }
  %a355 = alloca %str
  %a356 = alloca [9 x %str]
  %a373 = alloca %str
  %a376 = alloca { i1, i64 }
  %a382 = alloca i64
  %a387 = alloca [21 x i8]
  %a395 = alloca %str
  %a398 = alloca { i1, i64 }
  %a404 = alloca i64
  %a409 = alloca [21 x i8]
  %a415 = alloca [21 x i8]
  %a421 = alloca [21 x i8]
  %a427 = alloca i64
  %a430 = alloca %str
  %a432 = alloca %str
  %a433 = alloca [9 x %str]
  %t1 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 1, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 2, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store ptr %t1, ptr %a3
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a4
  %t5 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a4, i32 0, i32 3
  store i64 8, ptr %t5
  %t6 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a4, i32 0, i32 4
  store ptr @adesc.main.Counter, ptr %t6
  store ptr %a4, ptr %a7
  %t8 = load ptr, ptr %a7
  %t9 = insertvalue %S.main.Counter undef, i64 1, 0
  %t10 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 1
  %t11 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 2
  %t12 = load i64, ptr %t10
  %t13 = load i64, ptr %t11
  %t14 = icmp slt i64 %t12, %t13
  br i1 %t14, label %push.fast.1, label %push.grow.2, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.1:
  %t15 = load ptr, ptr %t8
  %t16 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 3
  %t17 = load i64, ptr %t16
  %t18 = mul i64 %t17, %t12
  %t19 = getelementptr inbounds i8, ptr %t15, i64 %t18
  store %S.main.Counter %t9, ptr %t19
  %t20 = add i64 %t12, 1
  store i64 %t20, ptr %t10
  %t21 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 5
  %t22 = load i64, ptr %t21
  %t23 = add i64 %t22, 1
  store i64 %t23, ptr %t21
  br label %push.done.3
push.grow.2:
  store %S.main.Counter %t9, ptr %a24
  call void @veles_list_push(ptr %t8, ptr %a24)
  br label %push.done.3
push.done.3:
  %t25 = load ptr, ptr %a7
  %t26 = insertvalue %S.main.Counter undef, i64 5, 0
  %t27 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 1
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 2
  %t29 = load i64, ptr %t27
  %t30 = load i64, ptr %t28
  %t31 = icmp slt i64 %t29, %t30
  br i1 %t31, label %push.fast.4, label %push.grow.5, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.4:
  %t32 = load ptr, ptr %t25
  %t33 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 3
  %t34 = load i64, ptr %t33
  %t35 = mul i64 %t34, %t29
  %t36 = getelementptr inbounds i8, ptr %t32, i64 %t35
  store %S.main.Counter %t26, ptr %t36
  %t37 = add i64 %t29, 1
  store i64 %t37, ptr %t27
  %t38 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 5
  %t39 = load i64, ptr %t38
  %t40 = add i64 %t39, 1
  store i64 %t40, ptr %t38
  br label %push.done.6
push.grow.5:
  store %S.main.Counter %t26, ptr %a41
  call void @veles_list_push(ptr %t25, ptr %a41)
  br label %push.done.6
push.done.6:
  %t42 = load ptr, ptr %a7
  %t43 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t42, i32 0, i32 1
  %t44 = load i64, ptr %t43
  %t45 = icmp eq i64 %t44, 2
  br i1 %t45, label %if.then.7, label %if.end.8
if.then.7:
  %t46 = load ptr, ptr %a7
  store ptr %t46, ptr %a47
  %t48 = load ptr, ptr %a47
  store i64 0, ptr %a49
  %t50 = load i64, ptr %a49
  %t51 = icmp slt i64 %t50, 0
  br i1 %t51, label %if.then.9, label %if.end.10
if.then.9:
  %t52 = load i64, ptr %a49
  %t53 = load ptr, ptr %a47
  %t54 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 1
  %t55 = load i64, ptr %t54
  %t56 = add i64 %t52, %t55
  store i64 %t56, ptr %a49
  br label %if.end.10
if.end.10:
  %t57 = load i64, ptr %a49
  %t58 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t48, i32 0, i32 6
  %t59 = load i64, ptr %t58
  %t60 = icmp eq i64 %t59, 1
  br i1 %t60, label %flags.done.12, label %flags.slow.11, !prof !{!"branch_weights", i32 2000, i32 1}
flags.slow.11:
  call void @veles_list_expose(ptr %t48)
  br label %flags.done.12
flags.done.12:
  %t61 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t48, i32 0, i32 1
  %t62 = load i64, ptr %t61
  %t63 = icmp ult i64 %t57, %t62
  br i1 %t63, label %idx.ok.13, label %idx.bad.14, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.14:
  %t64 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t65 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t48, i64 %t57, ptr %t64, i64 %t65)
  unreachable
idx.ok.13:
  %t66 = load ptr, ptr %t48
  %t67 = getelementptr inbounds [8 x i8], ptr %t66, i64 %t57
  store ptr %t67, ptr %a68
  %t69 = load ptr, ptr %a68
  %t70 = getelementptr inbounds %S.main.Counter, ptr %t69, i32 0, i32 0
  %t71 = load i64, ptr %t70
  %t73 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t71, i64 10)
  %t74 = extractvalue { i64, i1 } %t73, 0
  %t75 = extractvalue { i64, i1 } %t73, 1
  br i1 %t75, label %overflow.15, label %arith.ok.16
overflow.15:
  %t76 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t77 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t78 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t79 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_panic_at(ptr %t76, i64 %t77, ptr %t78, i64 %t79)
  unreachable
arith.ok.16:
  %t80 = load ptr, ptr %a68
  %t81 = getelementptr inbounds %S.main.Counter, ptr %t80, i32 0, i32 0
  store i64 %t74, ptr %t81
  br label %if.end.8
if.end.8:
  %t82 = load ptr, ptr %a7
  store ptr %t82, ptr %a83
  %t84 = insertvalue %S.main.Counter undef, i64 7, 0
  %t85 = load ptr, ptr %a83
  store i64 1, ptr %a86
  %t87 = load i64, ptr %a86
  %t88 = icmp slt i64 %t87, 0
  br i1 %t88, label %if.then.17, label %if.end.18
if.then.17:
  %t89 = load i64, ptr %a86
  %t90 = load ptr, ptr %a83
  %t91 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t90, i32 0, i32 1
  %t92 = load i64, ptr %t91
  %t93 = add i64 %t89, %t92
  store i64 %t93, ptr %a86
  br label %if.end.18
if.end.18:
  %t94 = load i64, ptr %a86
  %t95 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t85, i32 0, i32 6
  %t96 = load i64, ptr %t95
  %t97 = icmp ult i64 %t96, 2
  br i1 %t97, label %flags.done.20, label %flags.slow.19, !prof !{!"branch_weights", i32 2000, i32 1}
flags.slow.19:
  call void @veles_list_write_check(ptr %t85)
  br label %flags.done.20
flags.done.20:
  %t98 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t85, i32 0, i32 1
  %t99 = load i64, ptr %t98
  %t100 = icmp ult i64 %t94, %t99
  br i1 %t100, label %idx.ok.21, label %idx.bad.22, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.22:
  %t101 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t102 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t85, i64 %t94, ptr %t101, i64 %t102)
  unreachable
idx.ok.21:
  %t103 = load ptr, ptr %t85
  %t104 = getelementptr inbounds [8 x i8], ptr %t103, i64 %t94
  store %S.main.Counter %t84, ptr %t104
  store { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 } zeroinitializer, ptr %a105
  %t106 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a105, i32 0, i32 3
  store ptr @adesc.string, ptr %t106
  %t107 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a105, i32 0, i32 4
  store ptr @adesc.i64, ptr %t107
  %t108 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a105, i32 0, i32 10
  store i64 16, ptr %t108
  %t109 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a105, i32 0, i32 11
  store i64 8, ptr %t109
  store ptr %a105, ptr %a110
  %t111 = load ptr, ptr %a110
  %t113 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t114 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t112 = call i64 @veles_hash_bytes(ptr %t113, i64 %t114)
  store %str { ptr @.str.4, i64 1 }, ptr %a115
  store i64 1, ptr %a116
  call i64 @veles_map_insert(ptr %t111, i64 %t112, ptr %a115, ptr %a116, ptr @eqp.string)
  %t117 = load ptr, ptr %a110
  %t119 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t120 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t118 = call i64 @veles_hash_bytes(ptr %t119, i64 %t120)
  store %str { ptr @.str.5, i64 1 }, ptr %a121
  store i64 2, ptr %a122
  call i64 @veles_map_insert(ptr %t117, i64 %t118, ptr %a121, ptr %a122, ptr @eqp.string)
  %t123 = load ptr, ptr %a110
  call void @veles_map_expose(ptr %t123)
  %t125 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t126 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t124 = call i64 @veles_hash_bytes(ptr %t125, i64 %t126)
  store %str { ptr @.str.4, i64 1 }, ptr %a127
  %t128 = call i64 @veles_map_find(ptr %t123, i64 %t124, ptr %a127, ptr @eqp.string)
  %t129 = icmp sge i64 %t128, 0
  store ptr null, ptr %a130
  br i1 %t129, label %map.hit.23, label %map.end.24
map.hit.23:
  %t131 = call ptr @veles_map_val_at(ptr %t123, i64 %t128)
  store ptr %t131, ptr %a130
  br label %map.end.24
map.end.24:
  %t132 = load ptr, ptr %a130
  %t134 = icmp eq ptr %t132, null
  br i1 %t134, label %elvis.default.25, label %elvis.some.26
elvis.some.26:
  store ptr %t132, ptr %a133
  br label %elvis.end.27
elvis.default.25:
  %t135 = extractvalue %str { ptr @.str.6, i64 15 }, 0
  %t136 = extractvalue %str { ptr @.str.6, i64 15 }, 1
  %t137 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t138 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t135, i64 %t136, ptr %t137, i64 %t138)
  unreachable
elvis.end.27:
  %t139 = load ptr, ptr %a133
  store ptr %t139, ptr %a140
  %t141 = load ptr, ptr %a140
  %t142 = load i64, ptr %t141
  %t144 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t142, i64 40)
  %t145 = extractvalue { i64, i1 } %t144, 0
  %t146 = extractvalue { i64, i1 } %t144, 1
  br i1 %t146, label %overflow.28, label %arith.ok.29
overflow.28:
  %t147 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t148 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t149 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t150 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t147, i64 %t148, ptr %t149, i64 %t150)
  unreachable
arith.ok.29:
  %t151 = load ptr, ptr %a140
  store i64 %t145, ptr %t151
  %t152 = load ptr, ptr %a110
  %t154 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t155 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t153 = call i64 @veles_hash_bytes(ptr %t154, i64 %t155)
  store %str { ptr @.str.5, i64 1 }, ptr %a156
  %t157 = call i1 @veles_map_remove(ptr %t152, i64 %t153, ptr %a156, ptr @eqp.string)
  store { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 } zeroinitializer, ptr %a158
  %t159 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a158, i32 0, i32 3
  store ptr @adesc.i64, ptr %t159
  %t160 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a158, i32 0, i32 4
  store ptr null, ptr %t160
  %t161 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a158, i32 0, i32 10
  store i64 8, ptr %t161
  %t162 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a158, i32 0, i32 11
  store i64 0, ptr %t162
  store ptr %a158, ptr %a163
  %t164 = load ptr, ptr %a163
  store i64 4, ptr %a166
  %t167 = call i64 @veles_map_len(ptr %t164)
  call i64 @veles_map_insert(ptr %t164, i64 4, ptr %a166, ptr null, ptr @eqp.i64)
  %t168 = call i64 @veles_map_len(ptr %t164)
  %t169 = icmp ne i64 %t167, %t168
  %t170 = load ptr, ptr %a163
  store i64 4, ptr %a172
  %t173 = call i64 @veles_map_len(ptr %t170)
  call i64 @veles_map_insert(ptr %t170, i64 4, ptr %a172, ptr null, ptr @eqp.i64)
  %t174 = call i64 @veles_map_len(ptr %t170)
  %t175 = icmp ne i64 %t173, %t174
  %t176 = load ptr, ptr %a3
  store ptr %t176, ptr %a177
  store i64 1, ptr %a178
  %t179 = load i64, ptr %a178
  %t180 = icmp slt i64 %t179, 0
  br i1 %t180, label %if.then.30, label %if.end.31
if.then.30:
  %t181 = load i64, ptr %a178
  %t182 = load ptr, ptr %a177
  %t183 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t182, i32 0, i32 1
  %t184 = load i64, ptr %t183
  %t185 = add i64 %t181, %t184
  store i64 %t185, ptr %a178
  br label %if.end.31
if.end.31:
  %t188 = load i64, ptr %a178
  %t189 = icmp sge i64 %t188, 0
  store i1 %t189, ptr %a187
  br i1 %t189, label %sc.rhs.32, label %sc.end.33
sc.rhs.32:
  %t190 = load i64, ptr %a178
  %t191 = load ptr, ptr %a177
  %t192 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t191, i32 0, i32 1
  %t193 = load i64, ptr %t192
  %t194 = icmp slt i64 %t190, %t193
  store i1 %t194, ptr %a187
  br label %sc.end.33
sc.end.33:
  %t195 = load i1, ptr %a187
  br i1 %t195, label %if.then.34, label %if.else.36
if.then.34:
  %t196 = load ptr, ptr %a177
  %t197 = load i64, ptr %a178
  %t198 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t196, i32 0, i32 1
  %t199 = load i64, ptr %t198
  %t200 = icmp ult i64 %t197, %t199
  br i1 %t200, label %idx.ok.37, label %idx.bad.38, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.38:
  %t201 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t202 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t196, i64 %t197, ptr %t201, i64 %t202)
  unreachable
idx.ok.37:
  %t203 = load ptr, ptr %t196
  %t204 = getelementptr inbounds [8 x i8], ptr %t203, i64 %t197
  %t205 = load i64, ptr %t204
  %t206 = insertvalue { i1, i64 } undef, i1 true, 0
  %t207 = insertvalue { i1, i64 } %t206, i64 %t205, 1
  store { i1, i64 } %t207, ptr %a186
  br label %if.end.35
if.else.36:
  store { i1, i64 } zeroinitializer, ptr %a186
  br label %if.end.35
if.end.35:
  %t208 = load { i1, i64 }, ptr %a186
  %t211 = extractvalue { i1, i64 } %t208, 0
  %t210 = xor i1 %t211, true
  br i1 %t210, label %elvis.default.39, label %elvis.some.40
elvis.some.40:
  %t212 = extractvalue { i1, i64 } %t208, 1
  store i64 %t212, ptr %a209
  br label %elvis.end.41
elvis.default.39:
  store i64 -1, ptr %a209
  br label %elvis.end.41
elvis.end.41:
  %t213 = load i64, ptr %a209
  %t215 = call i64 @veles_i64_format(ptr %a214, i64 %t213)
  %t216 = insertvalue %str undef, ptr %a214, 0
  %t217 = insertvalue %str %t216, i64 %t215, 1
  %t218 = load ptr, ptr %a3
  store ptr %t218, ptr %a219
  store i64 9, ptr %a220
  %t221 = load i64, ptr %a220
  %t222 = icmp slt i64 %t221, 0
  br i1 %t222, label %if.then.42, label %if.end.43
if.then.42:
  %t223 = load i64, ptr %a220
  %t224 = load ptr, ptr %a219
  %t225 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t224, i32 0, i32 1
  %t226 = load i64, ptr %t225
  %t227 = add i64 %t223, %t226
  store i64 %t227, ptr %a220
  br label %if.end.43
if.end.43:
  %t230 = load i64, ptr %a220
  %t231 = icmp sge i64 %t230, 0
  store i1 %t231, ptr %a229
  br i1 %t231, label %sc.rhs.44, label %sc.end.45
sc.rhs.44:
  %t232 = load i64, ptr %a220
  %t233 = load ptr, ptr %a219
  %t234 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t233, i32 0, i32 1
  %t235 = load i64, ptr %t234
  %t236 = icmp slt i64 %t232, %t235
  store i1 %t236, ptr %a229
  br label %sc.end.45
sc.end.45:
  %t237 = load i1, ptr %a229
  br i1 %t237, label %if.then.46, label %if.else.48
if.then.46:
  %t238 = load ptr, ptr %a219
  %t239 = load i64, ptr %a220
  %t240 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t238, i32 0, i32 1
  %t241 = load i64, ptr %t240
  %t242 = icmp ult i64 %t239, %t241
  br i1 %t242, label %idx.ok.49, label %idx.bad.50, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.50:
  %t243 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t244 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t238, i64 %t239, ptr %t243, i64 %t244)
  unreachable
idx.ok.49:
  %t245 = load ptr, ptr %t238
  %t246 = getelementptr inbounds [8 x i8], ptr %t245, i64 %t239
  %t247 = load i64, ptr %t246
  %t248 = insertvalue { i1, i64 } undef, i1 true, 0
  %t249 = insertvalue { i1, i64 } %t248, i64 %t247, 1
  store { i1, i64 } %t249, ptr %a228
  br label %if.end.47
if.else.48:
  store { i1, i64 } zeroinitializer, ptr %a228
  br label %if.end.47
if.end.47:
  %t250 = load { i1, i64 }, ptr %a228
  %t253 = extractvalue { i1, i64 } %t250, 0
  %t252 = xor i1 %t253, true
  br i1 %t252, label %elvis.default.51, label %elvis.some.52
elvis.some.52:
  %t254 = extractvalue { i1, i64 } %t250, 1
  store i64 %t254, ptr %a251
  br label %elvis.end.53
elvis.default.51:
  store i64 -1, ptr %a251
  br label %elvis.end.53
elvis.end.53:
  %t255 = load i64, ptr %a251
  %t257 = call i64 @veles_i64_format(ptr %a256, i64 %t255)
  %t258 = insertvalue %str undef, ptr %a256, 0
  %t259 = insertvalue %str %t258, i64 %t257, 1
  %t260 = load ptr, ptr %a3
  store ptr %t260, ptr %a261
  %t262 = load ptr, ptr %a261
  %t263 = call ptr @veles_list_copy(ptr %t262)
  call void @veles_list_sort_native(ptr %t263, i32 4)
  %t264 = call %str @show.List_i64_(ptr %t263)
  %t265 = load ptr, ptr %a7
  store ptr %t265, ptr %a266
  store i64 0, ptr %a267
  %t268 = load i64, ptr %a267
  %t269 = icmp slt i64 %t268, 0
  br i1 %t269, label %if.then.54, label %if.end.55
if.then.54:
  %t270 = load i64, ptr %a267
  %t271 = load ptr, ptr %a266
  %t272 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t271, i32 0, i32 1
  %t273 = load i64, ptr %t272
  %t274 = add i64 %t270, %t273
  store i64 %t274, ptr %a267
  br label %if.end.55
if.end.55:
  %t277 = load i64, ptr %a267
  %t278 = icmp sge i64 %t277, 0
  store i1 %t278, ptr %a276
  br i1 %t278, label %sc.rhs.56, label %sc.end.57
sc.rhs.56:
  %t279 = load i64, ptr %a267
  %t280 = load ptr, ptr %a266
  %t281 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t280, i32 0, i32 1
  %t282 = load i64, ptr %t281
  %t283 = icmp slt i64 %t279, %t282
  store i1 %t283, ptr %a276
  br label %sc.end.57
sc.end.57:
  %t284 = load i1, ptr %a276
  br i1 %t284, label %if.then.58, label %if.else.60
if.then.58:
  %t285 = load ptr, ptr %a266
  %t286 = load i64, ptr %a267
  %t287 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t285, i32 0, i32 1
  %t288 = load i64, ptr %t287
  %t289 = icmp ult i64 %t286, %t288
  br i1 %t289, label %idx.ok.61, label %idx.bad.62, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.62:
  %t290 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t291 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t285, i64 %t286, ptr %t290, i64 %t291)
  unreachable
idx.ok.61:
  %t292 = load ptr, ptr %t285
  %t293 = getelementptr inbounds [8 x i8], ptr %t292, i64 %t286
  %t294 = load %S.main.Counter, ptr %t293
  %t295 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t296 = insertvalue { i1, %S.main.Counter } %t295, %S.main.Counter %t294, 1
  store { i1, %S.main.Counter } %t296, ptr %a275
  br label %if.end.59
if.else.60:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a275
  br label %if.end.59
if.end.59:
  %t297 = load { i1, %S.main.Counter }, ptr %a275
  store { i1, %S.main.Counter } %t297, ptr %a298
  %t300 = load { i1, %S.main.Counter }, ptr %a298
  %t302 = extractvalue { i1, %S.main.Counter } %t300, 0
  %t301 = xor i1 %t302, true
  br i1 %t301, label %if.then.63, label %if.else.65
if.then.63:
  store { i1, i64 } zeroinitializer, ptr %a299
  br label %if.end.64
if.else.65:
  %t303 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a298, i32 0, i32 1
  %t304 = getelementptr inbounds %S.main.Counter, ptr %t303, i32 0, i32 0
  %t305 = load i64, ptr %t304
  %t306 = insertvalue { i1, i64 } undef, i1 true, 0
  %t307 = insertvalue { i1, i64 } %t306, i64 %t305, 1
  store { i1, i64 } %t307, ptr %a299
  br label %if.end.64
if.end.64:
  %t308 = load { i1, i64 }, ptr %a299
  %t309 = call %str @show.T_i64_N({ i1, i64 } %t308)
  %t310 = load ptr, ptr %a7
  store ptr %t310, ptr %a311
  store i64 1, ptr %a312
  %t313 = load i64, ptr %a312
  %t314 = icmp slt i64 %t313, 0
  br i1 %t314, label %if.then.66, label %if.end.67
if.then.66:
  %t315 = load i64, ptr %a312
  %t316 = load ptr, ptr %a311
  %t317 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t316, i32 0, i32 1
  %t318 = load i64, ptr %t317
  %t319 = add i64 %t315, %t318
  store i64 %t319, ptr %a312
  br label %if.end.67
if.end.67:
  %t322 = load i64, ptr %a312
  %t323 = icmp sge i64 %t322, 0
  store i1 %t323, ptr %a321
  br i1 %t323, label %sc.rhs.68, label %sc.end.69
sc.rhs.68:
  %t324 = load i64, ptr %a312
  %t325 = load ptr, ptr %a311
  %t326 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t325, i32 0, i32 1
  %t327 = load i64, ptr %t326
  %t328 = icmp slt i64 %t324, %t327
  store i1 %t328, ptr %a321
  br label %sc.end.69
sc.end.69:
  %t329 = load i1, ptr %a321
  br i1 %t329, label %if.then.70, label %if.else.72
if.then.70:
  %t330 = load ptr, ptr %a311
  %t331 = load i64, ptr %a312
  %t332 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t330, i32 0, i32 1
  %t333 = load i64, ptr %t332
  %t334 = icmp ult i64 %t331, %t333
  br i1 %t334, label %idx.ok.73, label %idx.bad.74, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.74:
  %t335 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t336 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t330, i64 %t331, ptr %t335, i64 %t336)
  unreachable
idx.ok.73:
  %t337 = load ptr, ptr %t330
  %t338 = getelementptr inbounds [8 x i8], ptr %t337, i64 %t331
  %t339 = load %S.main.Counter, ptr %t338
  %t340 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t341 = insertvalue { i1, %S.main.Counter } %t340, %S.main.Counter %t339, 1
  store { i1, %S.main.Counter } %t341, ptr %a320
  br label %if.end.71
if.else.72:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a320
  br label %if.end.71
if.end.71:
  %t342 = load { i1, %S.main.Counter }, ptr %a320
  store { i1, %S.main.Counter } %t342, ptr %a343
  %t345 = load { i1, %S.main.Counter }, ptr %a343
  %t347 = extractvalue { i1, %S.main.Counter } %t345, 0
  %t346 = xor i1 %t347, true
  br i1 %t346, label %if.then.75, label %if.else.77
if.then.75:
  store { i1, i64 } zeroinitializer, ptr %a344
  br label %if.end.76
if.else.77:
  %t348 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a343, i32 0, i32 1
  %t349 = getelementptr inbounds %S.main.Counter, ptr %t348, i32 0, i32 0
  %t350 = load i64, ptr %t349
  %t351 = insertvalue { i1, i64 } undef, i1 true, 0
  %t352 = insertvalue { i1, i64 } %t351, i64 %t350, 1
  store { i1, i64 } %t352, ptr %a344
  br label %if.end.76
if.end.76:
  %t353 = load { i1, i64 }, ptr %a344
  %t354 = call %str @show.T_i64_N({ i1, i64 } %t353)
  %t357 = getelementptr [9 x %str], ptr %a356, i64 0, i64 0
  store %str %t217, ptr %t357
  %t358 = getelementptr [9 x %str], ptr %a356, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t358
  %t359 = getelementptr [9 x %str], ptr %a356, i64 0, i64 2
  store %str %t259, ptr %t359
  %t360 = getelementptr [9 x %str], ptr %a356, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t360
  %t361 = getelementptr [9 x %str], ptr %a356, i64 0, i64 4
  store %str %t264, ptr %t361
  %t362 = getelementptr [9 x %str], ptr %a356, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t362
  %t363 = getelementptr [9 x %str], ptr %a356, i64 0, i64 6
  store %str %t309, ptr %t363
  %t364 = getelementptr [9 x %str], ptr %a356, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t364
  %t365 = getelementptr [9 x %str], ptr %a356, i64 0, i64 8
  store %str %t354, ptr %t365
  call void @veles_string_concat_n(ptr %a355, ptr %a356, i64 9)
  %t366 = load %str, ptr %a355
  %t367 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t368 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t367)
  call void @v_std.io.println(%str %t366)
  call void @veles_call_pop()
  %t369 = load ptr, ptr %a110
  %t371 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t372 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t370 = call i64 @veles_hash_bytes(ptr %t371, i64 %t372)
  store %str { ptr @.str.4, i64 1 }, ptr %a373
  %t374 = call i64 @veles_map_find(ptr %t369, i64 %t370, ptr %a373, ptr @eqp.string)
  %t375 = icmp sge i64 %t374, 0
  store { i1, i64 } zeroinitializer, ptr %a376
  br i1 %t375, label %map.hit.78, label %map.end.79
map.hit.78:
  %t377 = call ptr @veles_map_val_at(ptr %t369, i64 %t374)
  %t378 = load i64, ptr %t377
  %t379 = insertvalue { i1, i64 } undef, i1 true, 0
  %t380 = insertvalue { i1, i64 } %t379, i64 %t378, 1
  store { i1, i64 } %t380, ptr %a376
  br label %map.end.79
map.end.79:
  %t381 = load { i1, i64 }, ptr %a376
  %t384 = extractvalue { i1, i64 } %t381, 0
  %t383 = xor i1 %t384, true
  br i1 %t383, label %elvis.default.80, label %elvis.some.81
elvis.some.81:
  %t385 = extractvalue { i1, i64 } %t381, 1
  store i64 %t385, ptr %a382
  br label %elvis.end.82
elvis.default.80:
  store i64 0, ptr %a382
  br label %elvis.end.82
elvis.end.82:
  %t386 = load i64, ptr %a382
  %t388 = call i64 @veles_i64_format(ptr %a387, i64 %t386)
  %t389 = insertvalue %str undef, ptr %a387, 0
  %t390 = insertvalue %str %t389, i64 %t388, 1
  %t391 = load ptr, ptr %a110
  %t393 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t394 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t392 = call i64 @veles_hash_bytes(ptr %t393, i64 %t394)
  store %str { ptr @.str.5, i64 1 }, ptr %a395
  %t396 = call i64 @veles_map_find(ptr %t391, i64 %t392, ptr %a395, ptr @eqp.string)
  %t397 = icmp sge i64 %t396, 0
  store { i1, i64 } zeroinitializer, ptr %a398
  br i1 %t397, label %map.hit.83, label %map.end.84
map.hit.83:
  %t399 = call ptr @veles_map_val_at(ptr %t391, i64 %t396)
  %t400 = load i64, ptr %t399
  %t401 = insertvalue { i1, i64 } undef, i1 true, 0
  %t402 = insertvalue { i1, i64 } %t401, i64 %t400, 1
  store { i1, i64 } %t402, ptr %a398
  br label %map.end.84
map.end.84:
  %t403 = load { i1, i64 }, ptr %a398
  %t406 = extractvalue { i1, i64 } %t403, 0
  %t405 = xor i1 %t406, true
  br i1 %t405, label %elvis.default.85, label %elvis.some.86
elvis.some.86:
  %t407 = extractvalue { i1, i64 } %t403, 1
  store i64 %t407, ptr %a404
  br label %elvis.end.87
elvis.default.85:
  store i64 0, ptr %a404
  br label %elvis.end.87
elvis.end.87:
  %t408 = load i64, ptr %a404
  %t410 = call i64 @veles_i64_format(ptr %a409, i64 %t408)
  %t411 = insertvalue %str undef, ptr %a409, 0
  %t412 = insertvalue %str %t411, i64 %t410, 1
  %t413 = load ptr, ptr %a110
  %t414 = call i64 @veles_map_len(ptr %t413)
  %t416 = call i64 @veles_i64_format(ptr %a415, i64 %t414)
  %t417 = insertvalue %str undef, ptr %a415, 0
  %t418 = insertvalue %str %t417, i64 %t416, 1
  %t419 = load ptr, ptr %a163
  %t420 = call i64 @veles_map_len(ptr %t419)
  %t422 = call i64 @veles_i64_format(ptr %a421, i64 %t420)
  %t423 = insertvalue %str undef, ptr %a421, 0
  %t424 = insertvalue %str %t423, i64 %t422, 1
  %t425 = load ptr, ptr %a163
  store i64 4, ptr %a427
  %t428 = call i64 @veles_map_find(ptr %t425, i64 4, ptr %a427, ptr @eqp.i64)
  %t429 = icmp sge i64 %t428, 0
  call void @veles_bool_to_string(ptr %a430, i1 zeroext %t429)
  %t431 = load %str, ptr %a430
  %t434 = getelementptr [9 x %str], ptr %a433, i64 0, i64 0
  store %str %t390, ptr %t434
  %t435 = getelementptr [9 x %str], ptr %a433, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t435
  %t436 = getelementptr [9 x %str], ptr %a433, i64 0, i64 2
  store %str %t412, ptr %t436
  %t437 = getelementptr [9 x %str], ptr %a433, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t437
  %t438 = getelementptr [9 x %str], ptr %a433, i64 0, i64 4
  store %str %t418, ptr %t438
  %t439 = getelementptr [9 x %str], ptr %a433, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t439
  %t440 = getelementptr [9 x %str], ptr %a433, i64 0, i64 6
  store %str %t424, ptr %t440
  %t441 = getelementptr [9 x %str], ptr %a433, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t441
  %t442 = getelementptr [9 x %str], ptr %a433, i64 0, i64 8
  store %str %t431, ptr %t442
  call void @veles_string_concat_n(ptr %a432, ptr %a433, i64 9)
  %t443 = load %str, ptr %a432
  %t444 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t445 = extractvalue %str { ptr @.str.15, i64 20 }, 1
  call void @veles_call_push(ptr %t444)
  call void @v_std.io.println(%str %t443)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [14 x i8] c"main.vs:13:22\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:14:3\00"
@.str.4 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.5 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.6 = private unnamed_addr constant [16 x i8] c"a was set above\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:18:19\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:18:3\00"
@.str.9 = private unnamed_addr constant [14 x i8] c"main.vs:23:17\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:23:35\00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:23:68\00"
@.str.12 = private unnamed_addr constant [14 x i8] c"main.vs:23:83\00"
@.str.13 = private unnamed_addr constant [2 x i8] c" \00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:23:3\00println\00"
@.str.15 = private unnamed_addr constant [21 x i8] c"main.vs:24:3\00println\00"
