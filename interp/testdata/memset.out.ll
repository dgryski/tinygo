; ModuleID = 'testdata/memset.ll'
source_filename = "testdata/memset.ll"
target datalayout = "e-m:e-p:32:32-i64:64-n32:64-S128"
target triple = "wasm32--wasi"

@zeroed = local_unnamed_addr global i32 0
@partial = local_unnamed_addr global i32 -5526613
@volatileResult = global i32 0

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: write)
declare void @llvm.memset.p0.i32(ptr nocapture writeonly, i8, i32, i1 immarg) #0

define void @runtime.initAll() unnamed_addr {
  call void @llvm.memset.p0.i32(ptr @volatileResult, i8 0, i32 4, i1 true)
  ret void
}

attributes #0 = { nocallback nofree nounwind willreturn memory(argmem: write) }
