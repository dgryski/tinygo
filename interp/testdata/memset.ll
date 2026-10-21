target datalayout = "e-m:e-p:32:32-i64:64-n32:64-S128"
target triple = "wasm32--wasi"

@zeroed = global i32 0
@partial = global i32 0
@volatileResult = global i32 0

declare ptr @runtime.alloc(i32, ptr) unnamed_addr

declare void @llvm.memset.p0.i32(ptr nocapture writeonly, i8, i32, i1 immarg)

define void @runtime.initAll() unnamed_addr {
  call void @main.init()
  ret void
}

define internal void @main.init() unnamed_addr {
  %buf = call ptr @runtime.alloc(i32 8, ptr inttoptr (i32 3 to ptr))
  store i32 -1, ptr %buf
  call void @llvm.memset.p0.i32(ptr %buf, i8 0, i32 4, i1 false)
  %zeroed = load i32, ptr %buf
  store i32 %zeroed, ptr @zeroed

  %second = getelementptr inbounds i8, ptr %buf, i32 4
  store i32 -1, ptr %second
  call void @llvm.memset.p0.i32(ptr %second, i8 171, i32 3, i1 false)
  %partial = load i32, ptr %second
  store i32 %partial, ptr @partial

  ; A volatile memset must not be interpreted.
  call void @llvm.memset.p0.i32(ptr @volatileResult, i8 0, i32 4, i1 true)
  ret void
}
