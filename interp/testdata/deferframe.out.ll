; ModuleID = 'testdata/deferframe.ll'
source_filename = "testdata/deferframe.ll"
target datalayout = "e-m:e-p:32:32-i64:64-n32:64-S128"
target triple = "wasm32--wasi"

@result = local_unnamed_addr global i32 42

; Function Attrs: nocallback nofree nosync nounwind willreturn
declare ptr @llvm.stacksave.p0() #0

declare void @runtime.setupDeferFrame(ptr, ptr, ptr) local_unnamed_addr

declare void @runtime.destroyDeferFrame(ptr, ptr) local_unnamed_addr

declare void @externalCall() local_unnamed_addr

define void @runtime.initAll() unnamed_addr {
  call fastcc void @unsafe.init(ptr undef)
  ret void
}

define internal fastcc void @unsafe.init(ptr %context) unnamed_addr {
entry:
  %frame = alloca [32 x i8], align 4
  %sp = call ptr @llvm.stacksave.p0()
  call void @runtime.setupDeferFrame(ptr %frame, ptr %sp, ptr undef)
  call void @externalCall()
  call void @runtime.destroyDeferFrame(ptr %frame, ptr undef)
  ret void
}

attributes #0 = { nocallback nofree nosync nounwind willreturn }
