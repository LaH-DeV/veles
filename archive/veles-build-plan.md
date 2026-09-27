# Veles — v0.1 Build Plan

Against `veles-spec.md` v0.20. This is a sequencing document, not a schedule.

---

## 0. The honest framing

Veles as designed is a large language. It has traits with associated types and default bodies, generics by GC-shape stenciling, sealed traits with inline tagged-union layout, two inferred effects, error-set union inference, a stackless coroutine transform, DWARF unwinding bridged to a coroutine-frame chain, a non-moving collector, compile-time data-race freedom, a package manager, and a C FFI.

Each of those is independently a serious piece of work. Several have no single existing implementation to copy, because the *combination* is novel even where the parts aren't. This is a multi-year project at one person's pace, and the plan below is organised around finding out early which parts don't work rather than around shipping features in a pleasing order.

---

## 1. Stage 0 — Spike the two novel things first

**Before writing a lexer.** Two decisions in this design have no proven reference implementation in combination, and both are load-bearing enough that failure would change the language rather than the compiler.

**Spike A — inferred suspension and the CPS transform.** Hand-write LLVM IR for a function that suspends, heap-allocates its frame, and resumes. No parser, no type checker. The question is whether the transform is tractable at all and what it costs in code size.

**Spike B — unwinding across the frame boundary.** A panic raised inside a coroutine frame, walking the coroutine chain to the resume point, converting to a native DWARF unwind, and landing in the executor (D49). This is the single most novel mechanism in the design.

If Spike A fails, D2 collapses and the concurrency model needs rethinking — probably toward green threads, which would also change the GC's root-scanning story. If Spike B fails, D49 gives way to the hidden error-return path, which was the rejected alternative and is a smaller change. Either way, finding out here costs weeks; finding out after the front-end costs a year.

**Everything else in the design is well-trodden.** Stencilled generics, sealed-type layout, mark-sweep collection, MVS resolution — these have working references. They are work, not risk.

---

## 2. Stage 1 — Hello world end to end

Lexer, parser, a minimal type checker, LLVM text emission, `llc` invocation, linking. `fun`, `val`/`var`, integers, strings, `if`, `loop`, `io.println`.

**Deliberate cut: no garbage collector. Leak everything.** A compiler is a batch process that exits; leaking is fine and defers the single largest runtime component past the point where the front-end is proving itself. Plenty of bootstrap compilers have done this.

The goal is an end-to-end pipeline, however thin. Everything afterwards is filling it in.

---

## 3. Stage 2 — Enough language to write a compiler in

Driven by I4: the first real Veles program is the Veles compiler, so the stdlib priority order (stdlib §4) is fixed and not negotiable.

Needed: structs and inherent methods; sealed traits with `when` and patterns; generics with shape stenciling — unavoidable, since `Option`, `Result`, `List` and `Map` are all in the prelude; `throws`/`try` with union inference; traits with associated types and default bodies; modules as directories; `fs`, `io`, `path`, `strings`, `os.spawn`.

`os.spawn` is not optional. I2 has the compiler emitting textual IR and invoking `llc`, so a self-hosted compiler that can't spawn a process can't build anything.

Still absent: concurrency, GC, FFI, package manager.

---

## 4. Stage 3 — The collector

Non-moving mark-sweep, size-segregated spans with per-span metadata, conservative stack scanning, precise heap scanning from object headers (I3).

Worth restating why this is smaller than it would be in most languages: D10's interior pointers forced a non-moving collector, and non-moving means **no stack maps, no pinning, no LLVM `gc.statepoint`** — the weakest corner of LLVM, avoided entirely. That was a cost when it was decided and it's a discount here.

---

## 5. Stage 4 — Concurrency

The CPS transform proven in Spike A, an executor, channels, `scope`, `gather`, `race`, `Mutex`, and `Sendable` checking.

**Ship the executor single-threaded first.** D35's isolation rules are enforced from day one — they're a type-system feature and retrofitting them later would break every program written in between — but the executor itself can run on one thread until the rest is stable. That defers work-stealing and the thread-state protocol without any language consequence.

`scope` first, then `gather` and `race`. `gather` needs tuples and the panic-capture path (D52); `race` needs atomic multi-channel registration and is the fiddliest of the three.

---

## 6. Stage 5 — Package manager, FFI, self-hosting

MVS resolution, `veles.toml`, `veles.sum`, module interface materialization. The interface format (spec §5) is load-bearing for effect and error inference across module boundaries, so it cannot be improvised late.

FFI per `veles-ffi.md`. Then self-hosting, which is where the Go compiler stops being the product and starts being a legacy artifact you have to keep alive through the cut-over.

**Plan the cut-over explicitly.** Both compilers must build the same language for as long as the transition takes, and every language change during that window is implemented twice. The window should be short and it should be scheduled, not drifted into.

---

## 7. Cut from v0.1

- **WebAssembly** — already deferred, and D49 means the native unwinding half needs replacing wholesale when it arrives.
- **Multi-threaded executor** — checking from day one, parallelism later (§5).
- **Specialization** — D15 made this permanently deferrable by keeping it semantically invisible. Take that.
- **Variance** — D14 deferred it, and widening an invariant parameter later is backward compatible.
- **Incremental compilation, LSP, formatter, debugger integration.**
- **Attributes beyond `@test`** — the mechanism (D51) is small; the set can grow.
- **A C header importer** — `extern` blocks are hand-written for now (FFI §2).

---

## 8. Risk register

Ordered by how much of the design fails with them.

1. **Inferred suspension and the CPS transform (D2)** — no coloring-free stackless language exists to copy. Spike A.
2. **DWARF bridged to a coroutine chain (D49)** — two unwinders meeting at a resume boundary, plus two platform ABIs. Spike B.
3. **`Sendable` without ownership (D35)** — Swift 6 proves it works and that the rollout hurts. Veles starts this way, which is easier, but the stdlib must be designed `Sendable`-clean from the first commit (D54).
4. **Effect and error-union inference across module interfaces (D40, D45)** — the interface format has to carry both, and D45's unions grow along call chains.
5. **Stenciling plus associated types plus default bodies** — Go has stenciling, Rust has the trait system; the combination has no single reference.
6. **Five breaking-change classes** (D17, D26, D28, D45, D53) — none is a compiler risk, all are ecosystem risks, and MVS (M7) trusts authors to honour every one.

---

## 9. What "v0.1" should mean

A self-hosted compiler that builds itself, targeting native code on one platform, with concurrency, a working collector, a package manager, and enough stdlib to write real programs. No wasm, no parallelism, no tooling beyond a build command and a test runner.

That is a defensible first release and it is still a very large amount of work.
