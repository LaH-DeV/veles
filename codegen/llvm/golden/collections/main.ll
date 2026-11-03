%S.main.Counter = type { i64 }
@adesc.main.Counter = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 1, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a1 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a4 = alloca i64
  %a5 = alloca ptr
  %a13 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a16 = alloca ptr
  %a39 = alloca %S.main.Counter
  %a62 = alloca %S.main.Counter
  %a68 = alloca ptr
  %a76 = alloca i64
  %a92 = alloca ptr
  %a107 = alloca ptr
  %a116 = alloca i64
  %a132 = alloca { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }
  %a137 = alloca ptr
  %a142 = alloca %str
  %a143 = alloca i64
  %a148 = alloca %str
  %a149 = alloca i64
  %a154 = alloca %str
  %a157 = alloca ptr
  %a160 = alloca ptr
  %a167 = alloca ptr
  %a183 = alloca %str
  %a185 = alloca { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }
  %a190 = alloca ptr
  %a199 = alloca i64
  %a211 = alloca i64
  %a216 = alloca ptr
  %a217 = alloca i64
  %a225 = alloca { i1, i64 }
  %a226 = alloca i1
  %a248 = alloca i64
  %a253 = alloca [21 x i8]
  %a258 = alloca ptr
  %a259 = alloca i64
  %a267 = alloca { i1, i64 }
  %a268 = alloca i1
  %a290 = alloca i64
  %a295 = alloca [21 x i8]
  %a300 = alloca ptr
  %a305 = alloca ptr
  %a306 = alloca i64
  %a314 = alloca { i1, %S.main.Counter }
  %a315 = alloca i1
  %a337 = alloca { i1, %S.main.Counter }
  %a338 = alloca { i1, i64 }
  %a350 = alloca ptr
  %a351 = alloca i64
  %a359 = alloca { i1, %S.main.Counter }
  %a360 = alloca i1
  %a382 = alloca { i1, %S.main.Counter }
  %a383 = alloca { i1, i64 }
  %a394 = alloca %str
  %a395 = alloca [9 x %str]
  %a412 = alloca %str
  %a415 = alloca { i1, i64 }
  %a421 = alloca i64
  %a426 = alloca [21 x i8]
  %a434 = alloca %str
  %a437 = alloca { i1, i64 }
  %a443 = alloca i64
  %a448 = alloca [21 x i8]
  %a454 = alloca [21 x i8]
  %a460 = alloca [21 x i8]
  %a466 = alloca i64
  %a469 = alloca %str
  %a471 = alloca %str
  %a472 = alloca [9 x %str]
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a1
  %t2 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a1, i32 0, i32 3
  store i64 8, ptr %t2
  %t3 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a1, i32 0, i32 4
  store ptr @adesc.i64, ptr %t3
  call void @veles_list_reserve(ptr %a1, i64 3)
  store i64 3, ptr %a4
  call void @veles_list_push(ptr %a1, ptr %a4)
  store i64 1, ptr %a4
  call void @veles_list_push(ptr %a1, ptr %a4)
  store i64 2, ptr %a4
  call void @veles_list_push(ptr %a1, ptr %a4)
  store ptr %a1, ptr %a5
  %t6 = load ptr, ptr %a5
  %t7 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t6, i32 0, i32 6
  %t8 = load i64, ptr %t7
  %t9 = and i64 %t8, 12
  %t10 = icmp ne i64 %t9, 0
  br i1 %t10, label %cow.copy.1, label %cow.done.2, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.1:
  %t11 = call ptr @veles_list_unique(ptr %t6, ptr null)
  store ptr %t11, ptr %a5
  br label %cow.done.2
cow.done.2:
  %t12 = load ptr, ptr %a5
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a13
  %t14 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a13, i32 0, i32 3
  store i64 8, ptr %t14
  %t15 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a13, i32 0, i32 4
  store ptr @adesc.main.Counter, ptr %t15
  store ptr %a13, ptr %a16
  %t17 = load ptr, ptr %a16
  %t18 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t17, i32 0, i32 6
  %t19 = load i64, ptr %t18
  %t20 = and i64 %t19, 12
  %t21 = icmp ne i64 %t20, 0
  br i1 %t21, label %cow.copy.3, label %cow.done.4, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.3:
  %t22 = call ptr @veles_list_unique_grow(ptr %t17, ptr null)
  store ptr %t22, ptr %a16
  br label %cow.done.4
cow.done.4:
  %t23 = load ptr, ptr %a16
  %t24 = insertvalue %S.main.Counter undef, i64 1, 0
  %t25 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t23, i32 0, i32 1
  %t26 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t23, i32 0, i32 2
  %t27 = load i64, ptr %t25
  %t28 = load i64, ptr %t26
  %t29 = icmp slt i64 %t27, %t28
  br i1 %t29, label %push.fast.5, label %push.grow.6, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.5:
  %t30 = load ptr, ptr %t23
  %t31 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t23, i32 0, i32 3
  %t32 = load i64, ptr %t31
  %t33 = mul i64 %t32, %t27
  %t34 = getelementptr inbounds i8, ptr %t30, i64 %t33
  store %S.main.Counter %t24, ptr %t34
  %t35 = add i64 %t27, 1
  store i64 %t35, ptr %t25
  %t36 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t23, i32 0, i32 5
  %t37 = load i64, ptr %t36
  %t38 = add i64 %t37, 1
  store i64 %t38, ptr %t36
  br label %push.done.7
push.grow.6:
  store %S.main.Counter %t24, ptr %a39
  call void @veles_list_push(ptr %t23, ptr %a39)
  br label %push.done.7
push.done.7:
  %t40 = load ptr, ptr %a16
  %t41 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t40, i32 0, i32 6
  %t42 = load i64, ptr %t41
  %t43 = and i64 %t42, 12
  %t44 = icmp ne i64 %t43, 0
  br i1 %t44, label %cow.copy.8, label %cow.done.9, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.8:
  %t45 = call ptr @veles_list_unique_grow(ptr %t40, ptr null)
  store ptr %t45, ptr %a16
  br label %cow.done.9
cow.done.9:
  %t46 = load ptr, ptr %a16
  %t47 = insertvalue %S.main.Counter undef, i64 5, 0
  %t48 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t46, i32 0, i32 1
  %t49 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t46, i32 0, i32 2
  %t50 = load i64, ptr %t48
  %t51 = load i64, ptr %t49
  %t52 = icmp slt i64 %t50, %t51
  br i1 %t52, label %push.fast.10, label %push.grow.11, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.10:
  %t53 = load ptr, ptr %t46
  %t54 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t46, i32 0, i32 3
  %t55 = load i64, ptr %t54
  %t56 = mul i64 %t55, %t50
  %t57 = getelementptr inbounds i8, ptr %t53, i64 %t56
  store %S.main.Counter %t47, ptr %t57
  %t58 = add i64 %t50, 1
  store i64 %t58, ptr %t48
  %t59 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t46, i32 0, i32 5
  %t60 = load i64, ptr %t59
  %t61 = add i64 %t60, 1
  store i64 %t61, ptr %t59
  br label %push.done.12
push.grow.11:
  store %S.main.Counter %t47, ptr %a62
  call void @veles_list_push(ptr %t46, ptr %a62)
  br label %push.done.12
push.done.12:
  %t63 = load ptr, ptr %a16
  %t64 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t63, i32 0, i32 1
  %t65 = load i64, ptr %t64
  %t66 = icmp eq i64 %t65, 2
  br i1 %t66, label %if.then.13, label %if.end.14
if.then.13:
  %t67 = load ptr, ptr %a16
  store ptr %t67, ptr %a68
  %t69 = load ptr, ptr %a16
  %t70 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t69, i32 0, i32 6
  %t71 = load i64, ptr %t70
  %t72 = and i64 %t71, 12
  %t73 = icmp ne i64 %t72, 0
  br i1 %t73, label %cow.copy.15, label %cow.done.16, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.15:
  %t74 = call ptr @veles_list_unique(ptr %t69, ptr null)
  store ptr %t74, ptr %a16
  br label %cow.done.16
cow.done.16:
  %t75 = load ptr, ptr %a16
  call void @veles_list_own(ptr %t75)
  store i64 0, ptr %a76
  %t77 = load i64, ptr %a76
  %t78 = icmp slt i64 %t77, 0
  br i1 %t78, label %if.then.17, label %if.end.18
if.then.17:
  %t79 = load i64, ptr %a76
  %t80 = load ptr, ptr %a16
  %t81 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t80, i32 0, i32 1
  %t82 = load i64, ptr %t81
  %t83 = add i64 %t79, %t82
  store i64 %t83, ptr %a76
  br label %if.end.18
if.end.18:
  %t84 = load i64, ptr %a76
  %t85 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t75, i32 0, i32 1
  %t86 = load i64, ptr %t85
  %t87 = icmp ult i64 %t84, %t86
  br i1 %t87, label %idx.ok.19, label %idx.bad.20, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.20:
  %t88 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t89 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t75, i64 %t84, ptr %t88, i64 %t89)
  unreachable
idx.ok.19:
  %t90 = load ptr, ptr %t75
  %t91 = getelementptr inbounds [8 x i8], ptr %t90, i64 %t84
  store ptr %t91, ptr %a92
  %t93 = load ptr, ptr %a92
  %t94 = getelementptr inbounds %S.main.Counter, ptr %t93, i32 0, i32 0
  %t95 = load i64, ptr %t94
  %t97 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t95, i64 10)
  %t98 = extractvalue { i64, i1 } %t97, 0
  %t99 = extractvalue { i64, i1 } %t97, 1
  br i1 %t99, label %overflow.21, label %arith.ok.22
overflow.21:
  %t100 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t101 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t102 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t103 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_panic_at(ptr %t100, i64 %t101, ptr %t102, i64 %t103)
  unreachable
arith.ok.22:
  %t104 = load ptr, ptr %a92
  %t105 = getelementptr inbounds %S.main.Counter, ptr %t104, i32 0, i32 0
  store i64 %t98, ptr %t105
  br label %if.end.14
if.end.14:
  %t106 = load ptr, ptr %a16
  store ptr %t106, ptr %a107
  %t108 = insertvalue %S.main.Counter undef, i64 7, 0
  %t109 = load ptr, ptr %a16
  %t110 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t109, i32 0, i32 6
  %t111 = load i64, ptr %t110
  %t112 = and i64 %t111, 12
  %t113 = icmp ne i64 %t112, 0
  br i1 %t113, label %cow.copy.23, label %cow.done.24, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.23:
  %t114 = call ptr @veles_list_unique(ptr %t109, ptr null)
  store ptr %t114, ptr %a16
  br label %cow.done.24
cow.done.24:
  %t115 = load ptr, ptr %a16
  store i64 1, ptr %a116
  %t117 = load i64, ptr %a116
  %t118 = icmp slt i64 %t117, 0
  br i1 %t118, label %if.then.25, label %if.end.26
if.then.25:
  %t119 = load i64, ptr %a116
  %t120 = load ptr, ptr %a16
  %t121 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t120, i32 0, i32 1
  %t122 = load i64, ptr %t121
  %t123 = add i64 %t119, %t122
  store i64 %t123, ptr %a116
  br label %if.end.26
if.end.26:
  %t124 = load i64, ptr %a116
  %t125 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t115, i32 0, i32 1
  %t126 = load i64, ptr %t125
  %t127 = icmp ult i64 %t124, %t126
  br i1 %t127, label %idx.ok.27, label %idx.bad.28, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.28:
  %t128 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t129 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t115, i64 %t124, ptr %t128, i64 %t129)
  unreachable
idx.ok.27:
  %t130 = load ptr, ptr %t115
  %t131 = getelementptr inbounds [8 x i8], ptr %t130, i64 %t124
  store %S.main.Counter %t108, ptr %t131
  store { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 } zeroinitializer, ptr %a132
  %t133 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a132, i32 0, i32 3
  store ptr @adesc.string, ptr %t133
  %t134 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a132, i32 0, i32 4
  store ptr @adesc.i64, ptr %t134
  %t135 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a132, i32 0, i32 10
  store i64 16, ptr %t135
  %t136 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a132, i32 0, i32 11
  store i64 8, ptr %t136
  store ptr %a132, ptr %a137
  %t138 = load ptr, ptr %a137
  %t140 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t141 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t139 = call i64 @veles_hash_bytes(ptr %t140, i64 %t141)
  store %str { ptr @.str.4, i64 1 }, ptr %a142
  store i64 1, ptr %a143
  call i64 @veles_map_insert(ptr %t138, i64 %t139, ptr %a142, ptr %a143, ptr @eqp.string)
  %t144 = load ptr, ptr %a137
  %t146 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t147 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t145 = call i64 @veles_hash_bytes(ptr %t146, i64 %t147)
  store %str { ptr @.str.5, i64 1 }, ptr %a148
  store i64 2, ptr %a149
  call i64 @veles_map_insert(ptr %t144, i64 %t145, ptr %a148, ptr %a149, ptr @eqp.string)
  %t150 = load ptr, ptr %a137
  call void @veles_map_own(ptr %t150)
  %t152 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t153 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t151 = call i64 @veles_hash_bytes(ptr %t152, i64 %t153)
  store %str { ptr @.str.4, i64 1 }, ptr %a154
  %t155 = call i64 @veles_map_find(ptr %t150, i64 %t151, ptr %a154, ptr @eqp.string)
  %t156 = icmp sge i64 %t155, 0
  store ptr null, ptr %a157
  br i1 %t156, label %map.hit.29, label %map.end.30
map.hit.29:
  %t158 = call ptr @veles_map_val_at(ptr %t150, i64 %t155)
  store ptr %t158, ptr %a157
  br label %map.end.30
map.end.30:
  %t159 = load ptr, ptr %a157
  %t161 = icmp eq ptr %t159, null
  br i1 %t161, label %elvis.default.31, label %elvis.some.32
elvis.some.32:
  store ptr %t159, ptr %a160
  br label %elvis.end.33
elvis.default.31:
  %t162 = extractvalue %str { ptr @.str.6, i64 15 }, 0
  %t163 = extractvalue %str { ptr @.str.6, i64 15 }, 1
  %t164 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t165 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t162, i64 %t163, ptr %t164, i64 %t165)
  unreachable
elvis.end.33:
  %t166 = load ptr, ptr %a160
  store ptr %t166, ptr %a167
  %t168 = load ptr, ptr %a167
  %t169 = load i64, ptr %t168
  %t171 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t169, i64 40)
  %t172 = extractvalue { i64, i1 } %t171, 0
  %t173 = extractvalue { i64, i1 } %t171, 1
  br i1 %t173, label %overflow.34, label %arith.ok.35
overflow.34:
  %t174 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t175 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t176 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t177 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t174, i64 %t175, ptr %t176, i64 %t177)
  unreachable
arith.ok.35:
  %t178 = load ptr, ptr %a167
  store i64 %t172, ptr %t178
  %t179 = load ptr, ptr %a137
  %t181 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t182 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t180 = call i64 @veles_hash_bytes(ptr %t181, i64 %t182)
  store %str { ptr @.str.5, i64 1 }, ptr %a183
  %t184 = call i1 @veles_map_remove(ptr %t179, i64 %t180, ptr %a183, ptr @eqp.string)
  store { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 } zeroinitializer, ptr %a185
  %t186 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a185, i32 0, i32 3
  store ptr @adesc.i64, ptr %t186
  %t187 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a185, i32 0, i32 4
  store ptr null, ptr %t187
  %t188 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a185, i32 0, i32 10
  store i64 8, ptr %t188
  %t189 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a185, i32 0, i32 11
  store i64 0, ptr %t189
  store ptr %a185, ptr %a190
  %t191 = load ptr, ptr %a190
  %t192 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %t191, i32 0, i32 13
  %t193 = load i64, ptr %t192
  %t194 = and i64 %t193, 12
  %t195 = icmp ne i64 %t194, 0
  br i1 %t195, label %cow.copy.36, label %cow.done.37, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.36:
  %t196 = call ptr @veles_map_unique(ptr %t191, ptr null, ptr null)
  store ptr %t196, ptr %a190
  br label %cow.done.37
cow.done.37:
  %t197 = load ptr, ptr %a190
  store i64 4, ptr %a199
  %t200 = call i64 @veles_map_len(ptr %t197)
  call i64 @veles_map_insert(ptr %t197, i64 4, ptr %a199, ptr null, ptr @eqp.i64)
  %t201 = call i64 @veles_map_len(ptr %t197)
  %t202 = icmp ne i64 %t200, %t201
  %t203 = load ptr, ptr %a190
  %t204 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %t203, i32 0, i32 13
  %t205 = load i64, ptr %t204
  %t206 = and i64 %t205, 12
  %t207 = icmp ne i64 %t206, 0
  br i1 %t207, label %cow.copy.38, label %cow.done.39, !prof !{!"branch_weights", i32 1, i32 2000}
cow.copy.38:
  %t208 = call ptr @veles_map_unique(ptr %t203, ptr null, ptr null)
  store ptr %t208, ptr %a190
  br label %cow.done.39
cow.done.39:
  %t209 = load ptr, ptr %a190
  store i64 4, ptr %a211
  %t212 = call i64 @veles_map_len(ptr %t209)
  call i64 @veles_map_insert(ptr %t209, i64 4, ptr %a211, ptr null, ptr @eqp.i64)
  %t213 = call i64 @veles_map_len(ptr %t209)
  %t214 = icmp ne i64 %t212, %t213
  %t215 = load ptr, ptr %a5
  store ptr %t215, ptr %a216
  store i64 1, ptr %a217
  %t218 = load i64, ptr %a217
  %t219 = icmp slt i64 %t218, 0
  br i1 %t219, label %if.then.40, label %if.end.41
if.then.40:
  %t220 = load i64, ptr %a217
  %t221 = load ptr, ptr %a216
  %t222 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t221, i32 0, i32 1
  %t223 = load i64, ptr %t222
  %t224 = add i64 %t220, %t223
  store i64 %t224, ptr %a217
  br label %if.end.41
if.end.41:
  %t227 = load i64, ptr %a217
  %t228 = icmp sge i64 %t227, 0
  store i1 %t228, ptr %a226
  br i1 %t228, label %sc.rhs.42, label %sc.end.43
sc.rhs.42:
  %t229 = load i64, ptr %a217
  %t230 = load ptr, ptr %a216
  %t231 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t230, i32 0, i32 1
  %t232 = load i64, ptr %t231
  %t233 = icmp slt i64 %t229, %t232
  store i1 %t233, ptr %a226
  br label %sc.end.43
sc.end.43:
  %t234 = load i1, ptr %a226
  br i1 %t234, label %if.then.44, label %if.else.46
if.then.44:
  %t235 = load ptr, ptr %a216
  %t236 = load i64, ptr %a217
  %t237 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t235, i32 0, i32 1
  %t238 = load i64, ptr %t237
  %t239 = icmp ult i64 %t236, %t238
  br i1 %t239, label %idx.ok.47, label %idx.bad.48, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.48:
  %t240 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t241 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t235, i64 %t236, ptr %t240, i64 %t241)
  unreachable
idx.ok.47:
  %t242 = load ptr, ptr %t235
  %t243 = getelementptr inbounds [8 x i8], ptr %t242, i64 %t236
  %t244 = load i64, ptr %t243
  %t245 = insertvalue { i1, i64 } undef, i1 true, 0
  %t246 = insertvalue { i1, i64 } %t245, i64 %t244, 1
  store { i1, i64 } %t246, ptr %a225
  br label %if.end.45
if.else.46:
  store { i1, i64 } zeroinitializer, ptr %a225
  br label %if.end.45
if.end.45:
  %t247 = load { i1, i64 }, ptr %a225
  %t250 = extractvalue { i1, i64 } %t247, 0
  %t249 = xor i1 %t250, true
  br i1 %t249, label %elvis.default.49, label %elvis.some.50
elvis.some.50:
  %t251 = extractvalue { i1, i64 } %t247, 1
  store i64 %t251, ptr %a248
  br label %elvis.end.51
elvis.default.49:
  store i64 -1, ptr %a248
  br label %elvis.end.51
elvis.end.51:
  %t252 = load i64, ptr %a248
  %t254 = call i64 @veles_i64_format(ptr %a253, i64 %t252)
  %t255 = insertvalue %str undef, ptr %a253, 0
  %t256 = insertvalue %str %t255, i64 %t254, 1
  %t257 = load ptr, ptr %a5
  store ptr %t257, ptr %a258
  store i64 9, ptr %a259
  %t260 = load i64, ptr %a259
  %t261 = icmp slt i64 %t260, 0
  br i1 %t261, label %if.then.52, label %if.end.53
if.then.52:
  %t262 = load i64, ptr %a259
  %t263 = load ptr, ptr %a258
  %t264 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t263, i32 0, i32 1
  %t265 = load i64, ptr %t264
  %t266 = add i64 %t262, %t265
  store i64 %t266, ptr %a259
  br label %if.end.53
if.end.53:
  %t269 = load i64, ptr %a259
  %t270 = icmp sge i64 %t269, 0
  store i1 %t270, ptr %a268
  br i1 %t270, label %sc.rhs.54, label %sc.end.55
sc.rhs.54:
  %t271 = load i64, ptr %a259
  %t272 = load ptr, ptr %a258
  %t273 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t272, i32 0, i32 1
  %t274 = load i64, ptr %t273
  %t275 = icmp slt i64 %t271, %t274
  store i1 %t275, ptr %a268
  br label %sc.end.55
sc.end.55:
  %t276 = load i1, ptr %a268
  br i1 %t276, label %if.then.56, label %if.else.58
if.then.56:
  %t277 = load ptr, ptr %a258
  %t278 = load i64, ptr %a259
  %t279 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t277, i32 0, i32 1
  %t280 = load i64, ptr %t279
  %t281 = icmp ult i64 %t278, %t280
  br i1 %t281, label %idx.ok.59, label %idx.bad.60, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.60:
  %t282 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t283 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t277, i64 %t278, ptr %t282, i64 %t283)
  unreachable
idx.ok.59:
  %t284 = load ptr, ptr %t277
  %t285 = getelementptr inbounds [8 x i8], ptr %t284, i64 %t278
  %t286 = load i64, ptr %t285
  %t287 = insertvalue { i1, i64 } undef, i1 true, 0
  %t288 = insertvalue { i1, i64 } %t287, i64 %t286, 1
  store { i1, i64 } %t288, ptr %a267
  br label %if.end.57
if.else.58:
  store { i1, i64 } zeroinitializer, ptr %a267
  br label %if.end.57
if.end.57:
  %t289 = load { i1, i64 }, ptr %a267
  %t292 = extractvalue { i1, i64 } %t289, 0
  %t291 = xor i1 %t292, true
  br i1 %t291, label %elvis.default.61, label %elvis.some.62
elvis.some.62:
  %t293 = extractvalue { i1, i64 } %t289, 1
  store i64 %t293, ptr %a290
  br label %elvis.end.63
elvis.default.61:
  store i64 -1, ptr %a290
  br label %elvis.end.63
elvis.end.63:
  %t294 = load i64, ptr %a290
  %t296 = call i64 @veles_i64_format(ptr %a295, i64 %t294)
  %t297 = insertvalue %str undef, ptr %a295, 0
  %t298 = insertvalue %str %t297, i64 %t296, 1
  %t299 = load ptr, ptr %a5
  store ptr %t299, ptr %a300
  %t301 = load ptr, ptr %a300
  %t302 = call ptr @veles_list_copy(ptr %t301)
  call void @veles_list_sort_native(ptr %t302, i32 4)
  %t303 = call %str @show.List_i64_(ptr %t302)
  %t304 = load ptr, ptr %a16
  store ptr %t304, ptr %a305
  store i64 0, ptr %a306
  %t307 = load i64, ptr %a306
  %t308 = icmp slt i64 %t307, 0
  br i1 %t308, label %if.then.64, label %if.end.65
if.then.64:
  %t309 = load i64, ptr %a306
  %t310 = load ptr, ptr %a305
  %t311 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t310, i32 0, i32 1
  %t312 = load i64, ptr %t311
  %t313 = add i64 %t309, %t312
  store i64 %t313, ptr %a306
  br label %if.end.65
if.end.65:
  %t316 = load i64, ptr %a306
  %t317 = icmp sge i64 %t316, 0
  store i1 %t317, ptr %a315
  br i1 %t317, label %sc.rhs.66, label %sc.end.67
sc.rhs.66:
  %t318 = load i64, ptr %a306
  %t319 = load ptr, ptr %a305
  %t320 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t319, i32 0, i32 1
  %t321 = load i64, ptr %t320
  %t322 = icmp slt i64 %t318, %t321
  store i1 %t322, ptr %a315
  br label %sc.end.67
sc.end.67:
  %t323 = load i1, ptr %a315
  br i1 %t323, label %if.then.68, label %if.else.70
if.then.68:
  %t324 = load ptr, ptr %a305
  %t325 = load i64, ptr %a306
  %t326 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t324, i32 0, i32 1
  %t327 = load i64, ptr %t326
  %t328 = icmp ult i64 %t325, %t327
  br i1 %t328, label %idx.ok.71, label %idx.bad.72, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.72:
  %t329 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t330 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t324, i64 %t325, ptr %t329, i64 %t330)
  unreachable
idx.ok.71:
  %t331 = load ptr, ptr %t324
  %t332 = getelementptr inbounds [8 x i8], ptr %t331, i64 %t325
  %t333 = load %S.main.Counter, ptr %t332
  %t334 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t335 = insertvalue { i1, %S.main.Counter } %t334, %S.main.Counter %t333, 1
  store { i1, %S.main.Counter } %t335, ptr %a314
  br label %if.end.69
if.else.70:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a314
  br label %if.end.69
if.end.69:
  %t336 = load { i1, %S.main.Counter }, ptr %a314
  store { i1, %S.main.Counter } %t336, ptr %a337
  %t339 = load { i1, %S.main.Counter }, ptr %a337
  %t341 = extractvalue { i1, %S.main.Counter } %t339, 0
  %t340 = xor i1 %t341, true
  br i1 %t340, label %if.then.73, label %if.else.75
if.then.73:
  store { i1, i64 } zeroinitializer, ptr %a338
  br label %if.end.74
if.else.75:
  %t342 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a337, i32 0, i32 1
  %t343 = getelementptr inbounds %S.main.Counter, ptr %t342, i32 0, i32 0
  %t344 = load i64, ptr %t343
  %t345 = insertvalue { i1, i64 } undef, i1 true, 0
  %t346 = insertvalue { i1, i64 } %t345, i64 %t344, 1
  store { i1, i64 } %t346, ptr %a338
  br label %if.end.74
if.end.74:
  %t347 = load { i1, i64 }, ptr %a338
  %t348 = call %str @show.T_i64_N({ i1, i64 } %t347)
  %t349 = load ptr, ptr %a16
  store ptr %t349, ptr %a350
  store i64 1, ptr %a351
  %t352 = load i64, ptr %a351
  %t353 = icmp slt i64 %t352, 0
  br i1 %t353, label %if.then.76, label %if.end.77
if.then.76:
  %t354 = load i64, ptr %a351
  %t355 = load ptr, ptr %a350
  %t356 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t355, i32 0, i32 1
  %t357 = load i64, ptr %t356
  %t358 = add i64 %t354, %t357
  store i64 %t358, ptr %a351
  br label %if.end.77
if.end.77:
  %t361 = load i64, ptr %a351
  %t362 = icmp sge i64 %t361, 0
  store i1 %t362, ptr %a360
  br i1 %t362, label %sc.rhs.78, label %sc.end.79
sc.rhs.78:
  %t363 = load i64, ptr %a351
  %t364 = load ptr, ptr %a350
  %t365 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t364, i32 0, i32 1
  %t366 = load i64, ptr %t365
  %t367 = icmp slt i64 %t363, %t366
  store i1 %t367, ptr %a360
  br label %sc.end.79
sc.end.79:
  %t368 = load i1, ptr %a360
  br i1 %t368, label %if.then.80, label %if.else.82
if.then.80:
  %t369 = load ptr, ptr %a350
  %t370 = load i64, ptr %a351
  %t371 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t369, i32 0, i32 1
  %t372 = load i64, ptr %t371
  %t373 = icmp ult i64 %t370, %t372
  br i1 %t373, label %idx.ok.83, label %idx.bad.84, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.84:
  %t374 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t375 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t369, i64 %t370, ptr %t374, i64 %t375)
  unreachable
idx.ok.83:
  %t376 = load ptr, ptr %t369
  %t377 = getelementptr inbounds [8 x i8], ptr %t376, i64 %t370
  %t378 = load %S.main.Counter, ptr %t377
  %t379 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t380 = insertvalue { i1, %S.main.Counter } %t379, %S.main.Counter %t378, 1
  store { i1, %S.main.Counter } %t380, ptr %a359
  br label %if.end.81
if.else.82:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a359
  br label %if.end.81
if.end.81:
  %t381 = load { i1, %S.main.Counter }, ptr %a359
  store { i1, %S.main.Counter } %t381, ptr %a382
  %t384 = load { i1, %S.main.Counter }, ptr %a382
  %t386 = extractvalue { i1, %S.main.Counter } %t384, 0
  %t385 = xor i1 %t386, true
  br i1 %t385, label %if.then.85, label %if.else.87
if.then.85:
  store { i1, i64 } zeroinitializer, ptr %a383
  br label %if.end.86
if.else.87:
  %t387 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a382, i32 0, i32 1
  %t388 = getelementptr inbounds %S.main.Counter, ptr %t387, i32 0, i32 0
  %t389 = load i64, ptr %t388
  %t390 = insertvalue { i1, i64 } undef, i1 true, 0
  %t391 = insertvalue { i1, i64 } %t390, i64 %t389, 1
  store { i1, i64 } %t391, ptr %a383
  br label %if.end.86
if.end.86:
  %t392 = load { i1, i64 }, ptr %a383
  %t393 = call %str @show.T_i64_N({ i1, i64 } %t392)
  %t396 = getelementptr [9 x %str], ptr %a395, i64 0, i64 0
  store %str %t256, ptr %t396
  %t397 = getelementptr [9 x %str], ptr %a395, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t397
  %t398 = getelementptr [9 x %str], ptr %a395, i64 0, i64 2
  store %str %t298, ptr %t398
  %t399 = getelementptr [9 x %str], ptr %a395, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t399
  %t400 = getelementptr [9 x %str], ptr %a395, i64 0, i64 4
  store %str %t303, ptr %t400
  %t401 = getelementptr [9 x %str], ptr %a395, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t401
  %t402 = getelementptr [9 x %str], ptr %a395, i64 0, i64 6
  store %str %t348, ptr %t402
  %t403 = getelementptr [9 x %str], ptr %a395, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t403
  %t404 = getelementptr [9 x %str], ptr %a395, i64 0, i64 8
  store %str %t393, ptr %t404
  call void @veles_string_concat_n(ptr %a394, ptr %a395, i64 9)
  %t405 = load %str, ptr %a394
  %t406 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t407 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t406)
  call void @v_std.io.println(%str %t405)
  call void @veles_call_pop()
  %t408 = load ptr, ptr %a137
  %t410 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t411 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t409 = call i64 @veles_hash_bytes(ptr %t410, i64 %t411)
  store %str { ptr @.str.4, i64 1 }, ptr %a412
  %t413 = call i64 @veles_map_find(ptr %t408, i64 %t409, ptr %a412, ptr @eqp.string)
  %t414 = icmp sge i64 %t413, 0
  store { i1, i64 } zeroinitializer, ptr %a415
  br i1 %t414, label %map.hit.88, label %map.end.89
map.hit.88:
  %t416 = call ptr @veles_map_val_at(ptr %t408, i64 %t413)
  %t417 = load i64, ptr %t416
  %t418 = insertvalue { i1, i64 } undef, i1 true, 0
  %t419 = insertvalue { i1, i64 } %t418, i64 %t417, 1
  store { i1, i64 } %t419, ptr %a415
  br label %map.end.89
map.end.89:
  %t420 = load { i1, i64 }, ptr %a415
  %t423 = extractvalue { i1, i64 } %t420, 0
  %t422 = xor i1 %t423, true
  br i1 %t422, label %elvis.default.90, label %elvis.some.91
elvis.some.91:
  %t424 = extractvalue { i1, i64 } %t420, 1
  store i64 %t424, ptr %a421
  br label %elvis.end.92
elvis.default.90:
  store i64 0, ptr %a421
  br label %elvis.end.92
elvis.end.92:
  %t425 = load i64, ptr %a421
  %t427 = call i64 @veles_i64_format(ptr %a426, i64 %t425)
  %t428 = insertvalue %str undef, ptr %a426, 0
  %t429 = insertvalue %str %t428, i64 %t427, 1
  %t430 = load ptr, ptr %a137
  %t432 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t433 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t431 = call i64 @veles_hash_bytes(ptr %t432, i64 %t433)
  store %str { ptr @.str.5, i64 1 }, ptr %a434
  %t435 = call i64 @veles_map_find(ptr %t430, i64 %t431, ptr %a434, ptr @eqp.string)
  %t436 = icmp sge i64 %t435, 0
  store { i1, i64 } zeroinitializer, ptr %a437
  br i1 %t436, label %map.hit.93, label %map.end.94
map.hit.93:
  %t438 = call ptr @veles_map_val_at(ptr %t430, i64 %t435)
  %t439 = load i64, ptr %t438
  %t440 = insertvalue { i1, i64 } undef, i1 true, 0
  %t441 = insertvalue { i1, i64 } %t440, i64 %t439, 1
  store { i1, i64 } %t441, ptr %a437
  br label %map.end.94
map.end.94:
  %t442 = load { i1, i64 }, ptr %a437
  %t445 = extractvalue { i1, i64 } %t442, 0
  %t444 = xor i1 %t445, true
  br i1 %t444, label %elvis.default.95, label %elvis.some.96
elvis.some.96:
  %t446 = extractvalue { i1, i64 } %t442, 1
  store i64 %t446, ptr %a443
  br label %elvis.end.97
elvis.default.95:
  store i64 0, ptr %a443
  br label %elvis.end.97
elvis.end.97:
  %t447 = load i64, ptr %a443
  %t449 = call i64 @veles_i64_format(ptr %a448, i64 %t447)
  %t450 = insertvalue %str undef, ptr %a448, 0
  %t451 = insertvalue %str %t450, i64 %t449, 1
  %t452 = load ptr, ptr %a137
  %t453 = call i64 @veles_map_len(ptr %t452)
  %t455 = call i64 @veles_i64_format(ptr %a454, i64 %t453)
  %t456 = insertvalue %str undef, ptr %a454, 0
  %t457 = insertvalue %str %t456, i64 %t455, 1
  %t458 = load ptr, ptr %a190
  %t459 = call i64 @veles_map_len(ptr %t458)
  %t461 = call i64 @veles_i64_format(ptr %a460, i64 %t459)
  %t462 = insertvalue %str undef, ptr %a460, 0
  %t463 = insertvalue %str %t462, i64 %t461, 1
  %t464 = load ptr, ptr %a190
  store i64 4, ptr %a466
  %t467 = call i64 @veles_map_find(ptr %t464, i64 4, ptr %a466, ptr @eqp.i64)
  %t468 = icmp sge i64 %t467, 0
  call void @veles_bool_to_string(ptr %a469, i1 zeroext %t468)
  %t470 = load %str, ptr %a469
  %t473 = getelementptr [9 x %str], ptr %a472, i64 0, i64 0
  store %str %t429, ptr %t473
  %t474 = getelementptr [9 x %str], ptr %a472, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t474
  %t475 = getelementptr [9 x %str], ptr %a472, i64 0, i64 2
  store %str %t451, ptr %t475
  %t476 = getelementptr [9 x %str], ptr %a472, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t476
  %t477 = getelementptr [9 x %str], ptr %a472, i64 0, i64 4
  store %str %t457, ptr %t477
  %t478 = getelementptr [9 x %str], ptr %a472, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t478
  %t479 = getelementptr [9 x %str], ptr %a472, i64 0, i64 6
  store %str %t463, ptr %t479
  %t480 = getelementptr [9 x %str], ptr %a472, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t480
  %t481 = getelementptr [9 x %str], ptr %a472, i64 0, i64 8
  store %str %t470, ptr %t481
  call void @veles_string_concat_n(ptr %a471, ptr %a472, i64 9)
  %t482 = load %str, ptr %a471
  %t483 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t484 = extractvalue %str { ptr @.str.15, i64 20 }, 1
  call void @veles_call_push(ptr %t483)
  call void @v_std.io.println(%str %t482)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [14 x i8] c"main.vs:13:22\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:14:3\00"
@.str.4 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.5 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.6 = private unnamed_addr constant [16 x i8] c"a was set above\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:18:18\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:18:3\00"
@.str.9 = private unnamed_addr constant [14 x i8] c"main.vs:23:17\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:23:36\00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:23:70\00"
@.str.12 = private unnamed_addr constant [14 x i8] c"main.vs:23:86\00"
@.str.13 = private unnamed_addr constant [2 x i8] c" \00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:23:3\00println\00"
@.str.15 = private unnamed_addr constant [21 x i8] c"main.vs:24:3\00println\00"
