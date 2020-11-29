/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mvvm

import "fmt"

// Defination of the symbol of data type for function parameter.
//
// Here are the defaule data types in Clang. For user-defined data type, add length before the type name.
// For example, "typedef struct {} jK" will be translated into "2jK".
const (
	integerParam = "i"
	// charParam             = "c"
	// longLongParam         = "x"
	// longParam             = "l"
	// signedCharParam       = "a"
	// shortParam            = "s"
	// unsignedIntegerParam  = "j"
	// unsignedCharParam     = "h"
	// unsignedLongLongParam = "y"
	// unsignedLongParam     = "m"
	// unsignedShortParam    = "t"

	//doubleParam     = "d"
	//floatParam      = "f"
	//longDoubleParam = "e"

	//pointerParam = "P"
	//voidParam = "v"
)

// getWasmFuncName converts the given function name f into the function name the .wast uses.
// For example, if developer defines a function whose name is "whatever" with an integer parameter, the function name
// in .wasm/.wast after compiling will be "_Z8whateveri", not "whatever" itself.
//
// Note that the conversion rule is tested after the WasmExplorer (https://mbebenita.github.io/WasmExplorer/) and the
// original Golang interpreter.
func getWasmFuncName(f string, params string) string {
	return fmt.Sprintf("_Z%d%s%s", len(f), f, params)
}
