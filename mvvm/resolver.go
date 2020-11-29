/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import (
	"fmt"
	"math"

	"github.com/perlin-network/life/exec"
)

const resolverSize = 1 << 8

// defaultResolver is the default version of the import resolver for MVVM.
//
// For essence, the resolver implements the ImportResolver interface, which allows the user to define imports to WASM
// modules under an MVVM.
// TODO (wangruichao@mtv.ac): (#217) consider user-defined ImportResolver here. Especially, the implementation of user-
//   defined function is in the matched .js file.
//   For now the default Resolver is applied.
type defaultResolver struct {
	arr []byte
}

// newDefaultResolver returns a *defaultResolver with the given size
func newDefaultResolver(size int) *defaultResolver {
	return &defaultResolver{arr: make([]byte, size)}
}

// ResolveFunc implements the ImportResolver interface
//
// When given the wasm import code like
// ```wast
//     (import "module" "field" (func $field (param i32 i32)))
// ```
// dr.ResolveFunc("module", "field") will be invoked, and the input parameters are stored in the currFrame.Locals.
// The defaultResolver gives the default behavior when received an import.
//
// TODO (wangruichao@mtv.ac): store emcc compiler into MultiVAC in order to offer default developing environment.
// TODO (wangruichao@mtv.ac): implement the linux system call by defaultResolver.
//   For now the necessary syscalls are implemented.
func (dr *defaultResolver) ResolveFunc(module, field string) exec.FunctionImport {
	switch module {
	case "env":
		switch field {
		case "nullFunc_ii", "nullFunc_iidiiii", "nullFunc_iii", "nullFunc_iiii", "nullFunc_iiiii", "nullFunc_iiiiid",
			"nullFunc_iiiiii", "nullFunc_iiiiiid", "nullFunc_iiiiiii", "nullFunc_iiiiiiii", "nullFunc_iiiiiiiii",
			"nullFunc_iiiiij", "nullFunc_jiji", "nullFunc_v", "nullFunc_vi", "nullFunc_vii", "nullFunc_viii",
			"nullFunc_viiii", "nullFunc_viiiii", "nullFunc_viiiiii", "nullFunc_viijii":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("%s not implemented.\n", field)
				return 0
			}
		case "___assert_fail", "abortStackOverflow":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("Abort: %v\n", field)
				return 0
			}
		case "___cxa_allocate_exception":
			return func(vm *exec.VirtualMachine) int64 {
				frame := vm.GetCurrentFrame()
				size := int(uint32(frame.Locals[0]))
				arr := make([]byte, size)
				vm.Memory = append(vm.Memory, arr...)
				return 0
			}
		case "___cxa_throw":
			return func(vm *exec.VirtualMachine) int64 {
				frame := vm.GetCurrentFrame()
				ptr := int(uint32(frame.Locals[0]))
				tp := int(uint32(frame.Locals[1]))
				destructor := int(uint32(frame.Locals[2]))
				panic(fmt.Errorf("exception: ptr %v, type %v, destructor %v", ptr, tp, destructor))
			}
		case "___cxa_uncaught_exception":
			return func(vm *exec.VirtualMachine) int64 {
				panic("uncaught exception")
			}
		case "___lock":
			return func(vm *exec.VirtualMachine) int64 { return 0 }
		case "___setErrNo":
			return func(vm *exec.VirtualMachine) int64 {
				frame := vm.GetCurrentFrame()
				errno := int64(uint32(frame.Locals[0]))
				fmt.Println("Set errno:", errno)
				return errno
			}
		case "___map_file":
			return func(vm *exec.VirtualMachine) int64 { return -1 }
		case "_emscripten_get_heap_size":
			return func(vm *exec.VirtualMachine) int64 { return int64(len(vm.Memory)) }
		case "_emscripten_resize_heap":
			return func(vm *exec.VirtualMachine) int64 {
				frame := vm.GetCurrentFrame()
				for i := 0; i < int(uint32(frame.Locals[0])); i++ {
					vm.Memory = append(vm.Memory, 0)
				}
				return 0
			}
		case "___syscall54": // ioctl
			return func(vm *exec.VirtualMachine) int64 {
				return 0
			}
		case "___syscall146": // writev
			return func(vm *exec.VirtualMachine) int64 {
				frame := vm.GetCurrentFrame()
				ptr := int(frame.Locals[1])
				iov := getMemUint32(vm.Memory, ptr+4)
				iovcnt := getMemUint32(vm.Memory, ptr+8)
				var totalWritten int64
				for i := uint32(0); i < iovcnt; i++ {
					tmpPtr := getMemUint32(vm.Memory, int(iov+i*8))
					tmpLen := getMemUint32(vm.Memory, int(iov+i*8+4))
					fmt.Print(string(vm.Memory[tmpPtr : tmpPtr+tmpLen]))
					totalWritten += int64(tmpLen)
				}
				return totalWritten
			}
		default:
			return func(vm *exec.VirtualMachine) int64 {
				currFrame := vm.GetCurrentFrame()
				switch len(currFrame.Locals) {
				case 0:
					return 0
				case 1:
					ptr := int(uint32(currFrame.Locals[0]))
					msg := []byte{vm.Memory[ptr]}
					dr.arr = msg
				case 2:
					ptr := int(uint32(currFrame.Locals[0]))
					msgLen := int(uint32(currFrame.Locals[1]))
					msg := vm.Memory[ptr : ptr+msgLen]
					dr.arr = msg
				default:
					// When an known import is invoked, the resolver will record the imported function and parameters
					// and panic.
					log.Debugf("The function is called: %s\n", field)
					log.Debugf("With parameters: %v\n", currFrame.Locals)
					panic(fmt.Errorf("don't know how to resolve for current frame: %v", currFrame))
				}
				return 0
			}
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

// ResolveGlobal implements the ImportResolver interface
//
// TODO (wangruichao@mtv.ac): implement the global data by defaultResolver.
//   For now the necessary global data (or magic data) is implemented.
func (dr *defaultResolver) ResolveGlobal(module, field string) int64 {
	switch module {
	case "env":
		switch field {
		case "__memory_base":
			return 1024
		case "__table_base":
			return 0
		case "tempDoublePtr": // "no memory initializer", volatile
			return 35856
		case "DYNAMICTOP_PTR": // volatile
			return 35840
		default:
			return 0
		}
	case "global":
		switch field {
		case "NaN":
			return int64(math.NaN())
		case "Infinity":
			return int64(math.Inf(1))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
	return -1
}
