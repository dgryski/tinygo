target datalayout = "e-m:e-p:32:32-i64:64-n32:64-S128"
target triple = "wasm32--wasi"

@result = global i32 0

declare ptr @llvm.stacksave.p0()
declare void @runtime.setupDeferFrame(ptr, ptr, ptr)
declare void @runtime.destroyDeferFrame(ptr, ptr)
declare i1 @runtime.unwindPending(ptr)
declare { ptr, ptr } @runtime._recover(i1, ptr)
declare void @externalCall()

define void @runtime.initAll() unnamed_addr {
  call void @main.init(ptr undef)
  call void @unsafe.init(ptr undef)
  ret void
}

; A defer frame that is only used when panicking.
define internal void @main.init(ptr %context) unnamed_addr {
entry:
  %frame = alloca [32 x i8], align 4
  %sp = call ptr @llvm.stacksave.p0()
  call void @runtime.setupDeferFrame(ptr %frame, ptr %sp, ptr undef)
  %setjmp = call i32 asm "", "=r,r"(ptr %frame) #0
  %setjmp.result = icmp eq i32 %setjmp, 0
  br i1 %setjmp.result, label %body, label %lpad

body:
  %recovered = call { ptr, ptr } @runtime._recover(i1 false, ptr undef)
  %type = extractvalue { ptr, ptr } %recovered, 0
  %isnil = icmp eq ptr %type, null
  br i1 %isnil, label %store, label %lpad

store:
  store i32 42, ptr @result
  %unwind = call i1 @runtime.unwindPending(ptr undef)
  br i1 %unwind, label %lpad, label %done

done:
  call void @runtime.destroyDeferFrame(ptr %frame, ptr undef)
  ret void

lpad:
  store i32 99, ptr @result
  call void @runtime.destroyDeferFrame(ptr %frame, ptr undef)
  ret void
}

; A call that can't be interpreted inside a defer frame must stay in the
; function, as it could panic and be recovered.
define internal void @unsafe.init(ptr %context) unnamed_addr {
entry:
  %frame = alloca [32 x i8], align 4
  %sp = call ptr @llvm.stacksave.p0()
  call void @runtime.setupDeferFrame(ptr %frame, ptr %sp, ptr undef)
  call void @externalCall()
  call void @runtime.destroyDeferFrame(ptr %frame, ptr undef)
  ret void
}

attributes #0 = { returns_twice }
