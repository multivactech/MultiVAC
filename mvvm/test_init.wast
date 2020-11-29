;;; // origin code (C++): 
;;;
;;; char* init(int n) {
;;;     char *arr  = new char[26 + 1];
;;;     arr = "abcdefghijklmnopqrstuvwxyz";
;;;     return arr+n;
;;; }
;;;

(module
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 16) "abcdefghijklmnopqrstuvwxyz\00")
 (export "memory" (memory $0))
 (export "_Z4initi" (func $_Z4initi))
 (func $_Z4initi (; 0 ;) (param $0 i32) (result i32)
  (i32.add
   (get_local $0)
   (i32.const 16)
  )
 )
)
