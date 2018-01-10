package base

import (
	"fmt"
	"math/rand"
	"time"
	"bytes"
	"regexp"
	"strings"
	"strconv"
	"errors"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func assignOutput (outNameNumber int, output []byte, typ string, expr *CXExpression, call *CXCall) error {
	outName := expr.OutputNames[outNameNumber].Name

	// if expr.OutputNames[outNameNumber].Typ != typ {
	// 	fmt.Println(expr.OutputNames[outNameNumber].Typ, typ, expr.Operator.Name)
	// }
	// fmt.Println(expr.OutputNames[outNameNumber].Typ, typ, expr.Operator.Name)

	///expr.OutputNames[outNameNumber].Typ = typ
	
	for _, char := range outName {
		if char == '.' {
			identParts := strings.Split(outName, ".")

			if def, err := expr.Module.GetDefinition(identParts[0]); err == nil {
				if strct, err := call.Context.GetStruct(def.Typ, expr.Module.Name); err == nil {
					_, _, offset, size := resolveStructField(identParts[1], def.Value, strct)
					firstChunk := make([]byte, offset)
					secondChunk := make([]byte, len(*def.Value) - int(offset + size))

					copy(firstChunk, (*def.Value)[:offset])
					copy(secondChunk, (*def.Value)[offset+size:])

					final := append(firstChunk, output...)
					final = append(final, secondChunk...)

					if def.Typ[0] == '*' {
						*def.Value = final
					} else {
						def.Value = &final
					}
					return nil
				}
			}

			for _, def := range call.State {
				if def.Name == identParts[0] {
					if strct, err := call.Context.GetStruct(def.Typ, expr.Module.Name); err == nil {
						byts, typ, offset, size := resolveStructField(identParts[1], def.Value, strct)

						isBasic := false
						for _, basic := range BASIC_TYPES {
							if basic == typ {
								isBasic = true
								break
							}
						}
						
						if true || typ == "str" || typ == "[]str" || typ == "[]bool" ||
							typ == "[]byte" || typ == "[]i32" ||
							typ == "[]i64" || typ == "[]f32" || typ == "[]f64" || !isBasic {

							firstChunk := make([]byte, offset)
							secondChunk := make([]byte, len(*def.Value) - int(offset + size))

							copy(firstChunk, (*def.Value)[:offset])
							copy(secondChunk, (*def.Value)[offset+size:])

							final := append(firstChunk, output...)
							final = append(final, secondChunk...)

							if def.Typ[0] == '*' {
								*def.Value = final
							} else {
								def.Value = &final
							}
							return nil
						} else {
							
							for c := 0; c < len(byts); c++ {
								byts[c] = (output)[c]
							}
						}
					} else {
						panic(err)
					}
					return nil
				}
			}
			break
		}
		
		if char == '[' {
			identParts := strings.Split(outName, "[")

			for _, def := range call.State {
				if def.Name == identParts[0] {
					idx, _ := strconv.ParseInt(identParts[1], 10, 64)
					for c := 0; c < len(output); c++ {
						if typ == "i64" || typ == "f64" {
							(*def.Value)[(int(idx)*8) + 4 + c] = (output)[c]
						} else if typ == "byte" {
							(*def.Value)[int(idx) + c] = (output)[c]
						} else {
							(*def.Value)[(int(idx)*4) + 4 + c] = (output)[c]
						}
					}
					return nil
				}
			}
			break
		}
	}

	if def, err := expr.Module.GetDefinition(outName); err == nil {
		if def.Typ[0] == '*' {
			*def.Value = output
		} else {
			def.Value = &output
		}
		return nil
	}

	for _, def := range call.State {
		if def.Name == outName {
			//fmt.Println(outName, typ, def.Typ)
			if def.Typ[0] == '*' {
				*def.Value = output
			} else {
				def.Value = &output
			}
			return nil
		}
	}
	
	if outName == "testI32" {
		fmt.Println("here", call.State[0].Name, outName)
	}

	call.State = append(call.State, MakeDefinition(outName, &output, typ))
	return nil
}

func argsToDefs (args []*CXArgument, inputs []*CXParameter, outputs []*CXParameter, mod *CXModule, cxt *CXProgram) ([]*CXDefinition, error) {
	if len(inputs) == len(args) {
		defs := make([]*CXDefinition, len(args) + len(outputs), len(args) + len(outputs) + 10)
		for i, arg := range args {
			defs[i] = &CXDefinition{
				Name: inputs[i].Name,
				Typ: arg.Typ,
				Value: arg.Value,
				Module: mod,
				Context: cxt,
			}
		}
		for i, out := range outputs {
			var zeroValue []byte
			isBasic := false
			if IsBasicType(out.Typ) {
				zeroValue = *MakeDefaultValue(out.Typ)
				isBasic = true
			}
			if !isBasic {
				var err error
				if zeroValue, err = ResolveStruct(out.Typ, cxt); err != nil {
					return nil, err
				}
			}
			defs[i+len(args)] = &CXDefinition{
				Name: out.Name,
				Typ: out.Typ,
				Value: &zeroValue,
				Module: mod,
				Context: cxt,
			}
		}
		return defs, nil
	} else {
		return nil, errors.New("Not enough definition names provided")
	}
}

func checkType (fnName string, typ string, arg *CXArgument) error {
	if arg.Typ != typ {
		return errors.New(fmt.Sprintf("%s: argument 1 is type '%s'; expected type '%s'", fnName, arg.Typ, typ))
	}
	return nil
}

func checkTwoTypes (fnName string, typ1 string, typ2 string, arg1 *CXArgument, arg2 *CXArgument) error {
	if arg1.Typ != typ1 || arg2.Typ != typ2 {
		if arg1.Typ != typ1 {
			return errors.New(fmt.Sprintf("%s: argument 1 is type '%s'; expected type '%s'", fnName, arg1.Typ, typ1))
		}
		return errors.New(fmt.Sprintf("%s: argument 2 is type '%s'; expected type '%s'", fnName, arg2.Typ, typ2))
	}
	return nil
}

func checkThreeTypes (fnName string, typ1 string, typ2 string, typ3 string, arg1 *CXArgument, arg2 *CXArgument, arg3 *CXArgument) error {
	if arg1.Typ != typ1 || arg2.Typ != typ2 || arg3.Typ != typ3 {
		if arg1.Typ != typ1 {
			return errors.New(fmt.Sprintf("%s: argument 1 is type '%s'; expected type '%s'", fnName, arg1.Typ, typ1))
		} else if arg2.Typ != typ2 {
			return errors.New(fmt.Sprintf("%s: argument 2 is type '%s'; expected type '%s'", fnName, arg2.Typ, typ2))
		}
		return errors.New(fmt.Sprintf("%s: argument 3 is type '%s'; expected type '%s'", fnName, arg3.Typ, typ3))
	}
	return nil
}

func checkFourTypes (fnName, typ1, typ2, typ3, typ4 string, arg1, arg2, arg3, arg4 *CXArgument) error {
	if arg1.Typ != typ1 || arg2.Typ != typ2 || arg3.Typ != typ3 || arg4.Typ != typ4 {
		if arg1.Typ != typ1 {
			return errors.New(fmt.Sprintf("%s: argument 1 is type '%s'; expected type '%s'", fnName, arg1.Typ, typ1))
		} else if arg2.Typ != typ2 {
			return errors.New(fmt.Sprintf("%s: argument 2 is type '%s'; expected type '%s'", fnName, arg2.Typ, typ2))
		} else if arg3.Typ != typ3 {
			return errors.New(fmt.Sprintf("%s: argument 3 is type '%s'; expected type '%s'", fnName, arg3.Typ, typ3))
		}
		return errors.New(fmt.Sprintf("%s: argument 4 is type '%s'; expected type '%s'", fnName, arg4.Typ, typ4))
	}
	return nil
}

func checkFiveTypes (fnName, typ1, typ2, typ3, typ4, typ5 string, arg1, arg2, arg3, arg4, arg5 *CXArgument) error {
	if arg1.Typ != typ1 || arg2.Typ != typ2 || arg3.Typ != typ3 || arg4.Typ != typ4 || arg5.Typ != typ5 {
		if arg1.Typ != typ1 {
			return errors.New(fmt.Sprintf("%s: argument 1 is type '%s'; expected type '%s'", fnName, arg1.Typ, typ1))
		} else if arg2.Typ != typ2 {
			return errors.New(fmt.Sprintf("%s: argument 2 is type '%s'; expected type '%s'", fnName, arg2.Typ, typ2))
		} else if arg3.Typ != typ3 {
			return errors.New(fmt.Sprintf("%s: argument 3 is type '%s'; expected type '%s'", fnName, arg3.Typ, typ3))
		} else if arg4.Typ != typ4 {
			return errors.New(fmt.Sprintf("%s: argument 4 is type '%s'; expected type '%s'", fnName, arg4.Typ, typ4))
		}
		return errors.New(fmt.Sprintf("%s: argument 5 is type '%s'; expected type '%s'", fnName, arg5.Typ, typ5))
	}
	return nil
}

func checkSixTypes (fnName, typ1, typ2, typ3, typ4, typ5, typ6 string, arg1, arg2, arg3, arg4, arg5, arg6 *CXArgument) error {
	if arg1.Typ != typ1 || arg2.Typ != typ2 || arg3.Typ != typ3 || arg4.Typ != typ4 || arg5.Typ != typ5 || arg6.Typ != typ6 {
		if arg1.Typ != typ1 {
			return errors.New(fmt.Sprintf("%s: argument 1 is type '%s'; expected type '%s'", fnName, arg1.Typ, typ1))
		} else if arg2.Typ != typ2 {
			return errors.New(fmt.Sprintf("%s: argument 2 is type '%s'; expected type '%s'", fnName, arg2.Typ, typ2))
		} else if arg3.Typ != typ3 {
			return errors.New(fmt.Sprintf("%s: argument 3 is type '%s'; expected type '%s'", fnName, arg3.Typ, typ3))
		} else if arg4.Typ != typ4 {
			return errors.New(fmt.Sprintf("%s: argument 4 is type '%s'; expected type '%s'", fnName, arg4.Typ, typ4))
		} else if arg5.Typ != typ5 {
			return errors.New(fmt.Sprintf("%s: argument 5 is type '%s'; expected type '%s'", fnName, arg5.Typ, typ5))
		}
		return errors.New(fmt.Sprintf("%s: argument 6 is type '%s'; expected type '%s'", fnName, arg6.Typ, typ6))
	}
	return nil
}

func random (min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max - min) + min
}

func removeDuplicatesInt (elements []int) []int {
	// Use map to record duplicates as we find them.
	encountered := map[int]bool{}
	result := []int{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func removeDuplicates (s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func concat (strs ...string) string {
	var buffer bytes.Buffer
	
	for i := 0; i < len(strs); i++ {
		buffer.WriteString(strs[i])
	}
	
	return buffer.String()
}

func PrintValue (identName string, value *[]byte, typName string, cxt *CXProgram) string {
	var argName string
	switch typName {
	case "str":
		var val string
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("\"%s\"", val)
	case "bool":
		var val int32
		encoder.DeserializeRaw(*value, &val)
		if val == 0 {
			argName = "false"
		} else {
			fmt.Printf("true")
			argName = "true"
		}
	case "byte":
		var val []byte
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "i32":
		var val int32
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "i64":
		var val int64
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "f32":
		var val float32
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "f64":
		var val float64
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "[]byte":
		var val []byte
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "[]str":
		var val []string
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "[]i32":
		var val []int32
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "[]i64":
		var val []int64
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "[]f32":
		var val []float32
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	case "[]f64":
		var val []float64
		encoder.DeserializeRaw(*value, &val)
		argName = fmt.Sprintf("%#v", val)
	default:
		// struct or custom type
		if mod, err := cxt.GetCurrentModule(); err == nil {
			if strct, err := cxt.GetStruct(typName, mod.Name); err == nil {
				for _, fld := range strct.Fields {
					val, typ, _, _ := resolveStructField(fld.Name, value, strct)
					fmt.Printf("\t%s.%s:\t\t%s\n", identName, fld.Name, PrintValue("", &val, typ, cxt))
				}
			}
		}


		return ""
	}

	return argName
}

func rep (str string, n int) string {
	return strings.Repeat(str, n)
}

// func (cxt *CXProgram) PrintProgram (isCompressed bool) {
// 	tab := "\t"
// 	nl := "\n"
// 	if isCompressed {
// 		tab = ""
// 		nl = ""
// 	}
	
// 	fmt.Println()
// 	fmt.Printf("(Modules %s", nl)
// 	for _, mod := range cxt.Modules {
// 		if mod.Name == CORE_MODULE {
// 			continue
// 		}
// 		fmt.Printf("%s(Module %s %s", rep(tab, 1), mod.Name, nl)

// 		fmt.Printf("%s(Imports %s", rep(tab, 2), nl)
// 		fmt.Printf("%s)%s", rep(tab, 2), nl) // imports
		
// 		fmt.Printf("%s(Definitions %s", rep(tab, 2), nl)

// 		for _, def := range mod.Definitions {
// 			fmt.Printf("%s(Definition %s %s %s)%s",
// 				rep(tab, 3),
// 				def.Name,
// 				def.Typ,
// 				PrintValue(def.Value, def.Typ),
// 				nl)
// 		}
		
// 		fmt.Printf("%s)%s", rep(tab, 2), nl) // definitions

// 		fmt.Printf("%s(Structs %s", rep(tab, 2), nl)

// 		for _, strct := range mod.Structs {
// 			fmt.Printf("%s(Struct %s", rep(tab, 3), nl)

// 			for _, fld := range strct.Fields {
// 				fmt.Printf("%s%s %s%s", rep(tab, 4), fld.Name, fld.Typ, nl)
// 			}
			
// 			fmt.Printf("%s)%s", rep(tab, 3), nl) // structs
// 		}
		
// 		fmt.Printf("%s)%s", rep(tab, 2), nl) // structs

// 		fmt.Printf("%s(Functions %s", rep(tab, 2), nl)

// 		for _, fn := range mod.Functions {
// 			fmt.Printf("%s(Function %s%s", rep(tab, 3), fn.Name, nl)

// 			fmt.Printf("%s(Inputs %s", rep(tab, 4), nl)
// 			for _, inp := range fn.Inputs {
// 				fmt.Printf("%s(Input %s %s)%s", rep(tab, 5), inp.Name, inp.Typ, nl)
// 			}
// 			fmt.Printf("%s)%s", rep(tab, 4), nl) // inputs

// 			fmt.Printf("%s(Outputs %s", rep(tab, 4), nl)
// 			for _, out := range fn.Outputs {
// 				fmt.Printf("%s(Output %s %s)%s", rep(tab, 5), out.Name, out.Typ, nl)
// 			}
// 			fmt.Printf("%s)%s", rep(tab, 4), nl) // outputs

// 			fmt.Printf("%s(Expressions %s", rep(tab, 4), nl)
// 			for _, expr := range fn.Expressions {
// 				_ = expr
// 				fmt.Printf("%s(Expression %s", rep(tab, 5), nl)

// 				fmt.Printf("%s(Operator %s)%s", rep(tab, 6), expr.Operator.Name, nl)
				
// 				fmt.Printf("%s(OutputNames %s", rep(tab, 6), nl)
// 				for _, outName := range expr.OutputNames {
// 					fmt.Printf("%s(OutputName %s)%s", rep(tab, 7), outName.Name, nl)
// 				}
// 				fmt.Printf("%s)%s", rep(tab, 6), nl)
				
// 				fmt.Printf("%s(Arguments %s", rep(tab, 6), nl)
// 				for _, arg := range expr.Arguments {
// 					fmt.Printf("%s(Argument %s %s)%s", rep(tab, 7), PrintValue(arg.Value, arg.Typ), arg.Typ, nl)
// 				}
// 				fmt.Printf("%s)%s", rep(tab, 6), nl)
				
// 				fmt.Printf("%s)%s", rep(tab, 5), nl)
// 			}
// 			fmt.Printf("%s)%s", rep(tab, 4), nl) // expressions
			
// 			fmt.Printf("%s)%s", rep(tab, 3), nl) // function
// 		}
		
// 		fmt.Printf("%s)%s", rep(tab, 2), nl) // functions
		
// 		fmt.Printf("%s)%s", rep(tab, 1), nl) // modules
// 	}
// 	fmt.Printf(")")
// 	fmt.Println()
// }

func CastArgumentForArray (typ string, arg *CXArgument) *CXArgument {
	switch typ {
	case "[]bool":
		return MakeArgument(arg.Value, "bool")
	case "[]byte":
		var val int32
		encoder.DeserializeRaw(*arg.Value, &val)
		sVal := encoder.Serialize(byte(val))
		return MakeArgument(&sVal, "byte")
	case "[]str":
		return arg
	case "[]i32":
		return arg
	case "[]i64":
		var val int32
		encoder.DeserializeRaw(*arg.Value, &val)
		sVal := encoder.Serialize(int64(val))
		return MakeArgument(&sVal, "i64")
	case "[]f32":
		return arg
	case "[]f64":
		var val float32
		encoder.DeserializeRaw(*arg.Value, &val)
		sVal := encoder.Serialize(float64(val))
		return MakeArgument(&sVal, "f64")
	default:
		return arg
	}
}

func ArgToString (arg *CXArgument) string {
	switch arg.Typ {
	case "ident", "string":
		var identName string
		encoder.DeserializeRaw(*arg.Value, &identName)
		return identName
	case "f32":
		var f32 float32
		encoder.DeserializeRaw(*arg.Value, &f32)
		return fmt.Sprintf("%f", f32)
	case "i32":
		var i32 int32
		encoder.DeserializeRaw(*arg.Value, &i32)
		return fmt.Sprintf("%d", i32)
	case "i64":
		var i64 int64
		encoder.DeserializeRaw(*arg.Value, &i64)
		return fmt.Sprintf("%d", i64)
	case "f64":
		var f64 float64
		encoder.DeserializeRaw(*arg.Value, &f64)
		return fmt.Sprintf("%f", f64)
	}
	return ""
}

func IsMultiDim (typ string) bool {
	if len(typ) > 4 && typ[:4] == "[][]" {
		return true
	} else {
		return false
	}
}

func IsBasicType (typ string) bool {
	re := regexp.MustCompile("\\**(\\[\\])*(bool|str|i32|i64|f32|f64|byte)")
	if re.FindString(typ) != "" {
		return true
	} else {
		return false
	}
	// for _, basic := range BASIC_TYPES {
	// 	if basic == typ {
	// 		return true
	// 	}
	// }
	// return false
}

func IsNative (fnName string) bool {
	if _, ok := NATIVE_FUNCTIONS[fnName]; ok {
		return true
	}
	if _, ok := NATIVE_FUNCTIONS[strings.Split(fnName, ".")[1]]; ok {
		return true
	}
	return false
}
func IsArray (typ string) bool {
	if len(typ) > 2 && typ[:2] == "[]" {
		return true
	}
	return false
}
func IsStructInstance (typ string, mod *CXModule) bool {
	if _, err := mod.Context.GetStruct(typ, mod.Name); err == nil {
		return true
	} else {
		return false
	}
}
func IsLocal (identName string, call *CXCall) bool {
	for _, def := range call.State {
		if def.Name == identName {
			return true
		}
	}
	return false
}
func IsGlobal (identName string, mod *CXModule) bool {
	for _, def := range mod.Definitions {
		if def.Name == identName {
			return true
		}
	}
	for _, imp := range mod.Imports {
		for _, def := range imp.Definitions {
			if def.Name == identName {
				return true
			}
		}
	}
	return false
}

func makeArray (typ string, size *CXArgument, expr *CXExpression, call *CXCall) error {
	if err := checkType("makeArray", "i32", size); err == nil {
		var _len int32
		encoder.DeserializeRaw(*size.Value, &_len)

		switch typ {
		case "[]bool":
			arr := make([]int32, _len)
			val := encoder.Serialize(arr)

			assignOutput(0, val, typ, expr, call)
			return nil
		case "[]byte":
			arr := make([]byte, _len)
			val := encoder.Serialize(arr)

			assignOutput(0, val, typ, expr, call)
			return nil
		case "[]str":
			arr := make([]string, _len)
			val := encoder.Serialize(arr)

			assignOutput(0, val, typ, expr, call)
			return nil
		case "[]i32":
			arr := make([]int32, _len)
			val := encoder.Serialize(arr)
			
			assignOutput(0, val, typ, expr, call)
			return nil
		case "[]i64":
			arr := make([]int64, _len)
			val := encoder.Serialize(arr)

			assignOutput(0, val, typ, expr, call)
			return nil
		case "[]f32":
			arr := make([]float32, _len)
			val := encoder.Serialize(arr)

			assignOutput(0, val, typ, expr, call)
			return nil
		case "[]f64":
			arr := make([]float64, _len)
			val := encoder.Serialize(arr)

			assignOutput(0, val, typ, expr, call)
			return nil
		case "default":
			return errors.New(fmt.Sprintf("makeArray: argument 1 is type '%s'; expected type 'i32'", size.Typ))
		}
		return nil
	} else {
		return err
	}
}

func resolveStructField (fld string, val *[]byte, strct *CXStruct) ([]byte, string, int32, int32) {
	var offset int32 = 0
	for _, f := range strct.Fields {

		var fldType string
		
		isArray := false
		isBasic := false
		if f.Typ[:2] == "[]" {
			isArray = true
			for _, basic := range BASIC_TYPES {
				if basic == f.Typ[2:] {
					isBasic = true
					break
				}
			}
		} else {
			for _, basic := range BASIC_TYPES {
				if basic == f.Typ {
					isBasic = true
					break
				}
			}
		}

		if isBasic {
			fldType = f.Typ
		} else {
			if isArray {
				fldType = "[]"
			} else {
				fldType = "struct"
			}
		}
		
		if f.Name == fld {
			var size int32
			
			switch fldType {
			case "byte":
				size = 1
			case "bool", "i32", "f32":
				size = 4
			case "i64", "f64":
				size = 8
			case "[]str":
				var noElms int32
				encoder.DeserializeAtomic((*val)[offset:offset+4], &noElms)

				noSize := (*val)[offset+4:]
				
				var subOffset int32
				for c := 0; c < int(noElms); c++ {
					var strSize int32
					encoder.DeserializeRaw(noSize[subOffset:subOffset+4], &strSize)
					subOffset += strSize + 4
				}
				size = subOffset

				return (*val)[offset:offset+size + 4], f.Typ, offset, size + 4
			case "str", "[]byte":
				var arrOffset int32
				encoder.DeserializeAtomic((*val)[offset:offset+4], &arrOffset)
				size = arrOffset

				return (*val)[offset:offset+size + 4], f.Typ, offset, size + 4
			case "[]bool", "[]i32", "[]f32":
				var arrOffset int32
				encoder.DeserializeAtomic((*val)[offset:offset+4], &arrOffset)
				size = arrOffset
				
				return (*val)[offset:offset+(size * 4) + 4], f.Typ, offset, (size * 4) + 4
			case "[]i64", "[]f64":
				var arrOffset int32
				encoder.DeserializeAtomic((*val)[offset:offset+4], &arrOffset)
				size = arrOffset
				
				return (*val)[offset:offset+(size * 8) + 4], f.Typ, offset, (size * 8) + 4
			case "[]":
				if strct, err := strct.Context.GetStruct(f.Typ[2:], strct.Module.Name); err == nil {
					lastFld := strct.Fields[len(strct.Fields) - 1]
					instances := (*val)[offset+4:]

					var upperBound int32
					var size int32
					encoder.DeserializeAtomic((*val)[offset:offset + 4], &size)
					
					if size == 0 {
						return (*val)[offset:offset+4], f.Typ, offset, 4
					}

					for c := int32(0); c < size; c++ {
						subArray := instances[upperBound:]
						_, _, off, size := resolveStructField(lastFld.Name, &subArray, strct)
						
						upperBound = upperBound + off + size
					}

					return (*val)[offset:offset + upperBound + 4], f.Typ, offset, upperBound + 4
				}
			case "struct":
				if strct, err := strct.Context.GetStruct(f.Typ, strct.Module.Name); err == nil {
					lastFld := strct.Fields[len(strct.Fields) - 1]

					instances := (*val)[offset:]
					_, _, off, size := resolveStructField(lastFld.Name, &instances, strct)
					
					return (*val)[offset:offset + off + size], f.Typ, offset, off + size
				}
			}
			return (*val)[offset:offset+size], f.Typ, offset, size
		}
		
		switch fldType {
		case "byte":
			offset += 1
		case "bool", "i32", "f32":
			offset += 4
		case "i64", "f64":
			offset += 8
		case "[]str":
			var noElms int32
			encoder.DeserializeAtomic((*val)[offset:offset+4], &noElms)

			noSize := (*val)[offset+4:]

			var subOffset int32
			for c := 0; c < int(noElms); c++ {
				var strSize int32
				encoder.DeserializeRaw(noSize[subOffset:subOffset+4], &strSize)
				subOffset += strSize + 4
			}
			offset += subOffset + 4
		case "str", "[]byte":
			var arrOffset int32
			encoder.DeserializeAtomic((*val)[offset:offset+4], &arrOffset)
			offset += arrOffset + 4
		case "[]bool", "[]i32", "[]f32":
			var arrOffset int32
			encoder.DeserializeAtomic((*val)[offset:offset+4], &arrOffset)
			
			offset += (arrOffset * 4) + 4
		case "[]i64", "[]f64":
			var arrOffset int32
			encoder.DeserializeAtomic((*val)[offset:offset+4], &arrOffset)

			offset += (arrOffset * 8) + 4
		case "[]":
			if strct, err := strct.Context.GetStruct(f.Typ[2:], strct.Module.Name); err == nil {
				instances := (*val)[offset+4:]
				lastFld := strct.Fields[len(strct.Fields) - 1]
				
				var upperBound int32
				
				var size int32
				encoder.DeserializeAtomic((*val)[offset:offset+4], &size)

				// we don't need this. if size == 0, the loop won't execute
				// and we'll return lowerBound(0) + 4 = 4
				// if size == 0 {
				// 	offset += 4
				// }
				
				for c := int32(0); c < size; c++ {
					subArray := instances[upperBound:]
					_, _, off, size := resolveStructField(lastFld.Name, &subArray, strct)

					upperBound = upperBound + off + size
				}
				offset += upperBound + 4
			}
		case "struct":
			if strct, err := strct.Context.GetStruct(f.Typ, strct.Module.Name); err == nil {
				lastFld := strct.Fields[len(strct.Fields) - 1]

				instances := (*val)[offset:]
				_, _, off, size := resolveStructField(lastFld.Name, &instances, strct)

				offset += off + size
			}
		}
	}
	
	return nil, "", 0, 0
}

func resolveArrayIndex (index int, val *[]byte, typ string) ([]byte, string) {
	switch typ {
	case "[]byte":
		return (*val)[index+4:(index+1)+4], "byte"
	case "[]bool":
		return (*val)[(index+1)*4:(index+2)*4], "bool"
	case "[]i32":
		return (*val)[(index+1)*4:(index+2)*4], "i32"
	case "[]i64":
		return (*val)[((index)*8)+4:((index+1)*8)+4], "i64"
	case "[]f32":
		return (*val)[(index+1)*4:(index+2)*4], "f32"
	case "[]f64":
		return (*val)[((index)*8)+4:((index+1)*8)+4], "f64"
	}
	
	return nil, ""
}

func resolveIdent (lookingFor string, call *CXCall) (*CXArgument, error) {
	var resolvedIdent *CXDefinition
	
	isStructFld := false
	isArray := false

	identParts := strings.Split(lookingFor, ".")

	if len(identParts) > 1 {
		if mod, err := call.Context.GetModule(identParts[0]); err == nil {
			// then it's an external definition or struct
			isImported := false
			for _, imp := range call.Operator.Module.Imports {
				if imp.Name == identParts[0] {
					isImported = true
					break
				}
			}
			if isImported {
				if def, err := mod.GetDefinition(concat(identParts[1:]...)); err == nil {
					resolvedIdent = def
					// if !IsBasicType(def.Typ) {
					// 	resolvedIdent.Typ = fmt.Sprintf("%s.%s", mod.Name, def.Typ)
					// }
				}
			} else {
				return nil, errors.New(fmt.Sprintf("module '%s' was not imported or does not exist", mod.Name))
			}
		} else {
			// then it's a global struct
			mod := call.Operator.Module
			if def, err := mod.GetDefinition(identParts[0]); err == nil {
				isStructFld = true
				//resolvedIdent = def
				if strct, err := mod.Context.GetStruct(def.Typ, mod.Name); err == nil {
					byts, typ, _, _ := resolveStructField(identParts[1], def.Value, strct)
					arg := MakeArgument(&byts, typ)
					return arg, nil
					
				} else {
					return nil, err
				}
			} else {
				// then it's a local struct
				isStructFld = true

				for _, stateDef := range call.State {
					if stateDef.Name == identParts[0] {
						if strct, err := mod.Context.GetStruct(stateDef.Typ, mod.Name); err == nil {
							byts, typ, _, _ := resolveStructField(identParts[1], stateDef.Value, strct)
							arg := MakeArgument(&byts, typ)
							return arg, nil
							
						} else {
							return nil, err
						}
					}
				}
			}
		}
	} else {
		// then it's a local or global definition
		local := false
		arrayParts := strings.Split(lookingFor, "[")
		if len(arrayParts) > 1 {
			lookingFor = arrayParts[0]
		}
		for _, stateDef := range call.State {
			if stateDef.Name == arrayParts[0] {
				local = true
				resolvedIdent = stateDef
				break
			}
		}

		if len(arrayParts) > 1 && local {
			if idx, err := strconv.ParseInt(arrayParts[1], 10, 64); err == nil {
				isArray = true
				byts, typ := resolveArrayIndex(int(idx), resolvedIdent.Value, resolvedIdent.Typ)
				arg := MakeArgument(&byts, typ)
				return arg, nil
			} else {
				//excError = err
				return nil, err
			}
		}

		if !local {
			mod := call.Operator.Module
			if def, err := mod.GetDefinition(lookingFor); err == nil {
				resolvedIdent = def
			}
		}
	}

	if resolvedIdent == nil && !isStructFld && !isArray {
		return nil, errors.New(fmt.Sprintf("'%s' is undefined", lookingFor))
	}
	
	if resolvedIdent != nil && !isStructFld && !isArray {
		// if it was a struct field, we already created the argument above for efficiency reasons
		// the same goes to arrays in the form ident[index]

		
		var typ string
		if !IsBasicType(resolvedIdent.Typ) {
			if mod, err := call.Context.GetModule(identParts[0]); err == nil {
				typ = fmt.Sprintf("%s.%s", mod.Name, resolvedIdent.Typ)
			} else {
				typ = resolvedIdent.Typ
			}
		} else {
			typ = resolvedIdent.Typ
		}
		arg := MakeArgument(resolvedIdent.Value, typ)
		return arg, nil
	}
	return nil, errors.New(fmt.Sprintf("identifier '%s' could not be resolved", lookingFor))
}

func ResolveStruct (typ string, cxt *CXProgram) ([]byte, error) {
	var bs []byte

	found := false
	if mod, err := cxt.GetCurrentModule(); err == nil {
		var foundStrct *CXStruct

		if typ[:2] == "[]" {
			// empty serialized struct array
			return []byte{0, 0, 0, 0}, nil
		}
		
		for _, strct := range mod.Structs {
			if strct.Name == typ {
				found = true
				foundStrct = strct
				break
			}
		}
		if !found {
			typeParts := strings.Split(typ, ".")
			if len(typeParts) > 1 {
				for _, imp := range mod.Imports {
					if typeParts[0] == imp.Name {
						for _, strct := range imp.Structs {
							if strct.Name == typeParts[1] {
								found = true
								foundStrct = strct
								break
							}
						}
					}
				}
			}
		}
		
		if !found {
			return nil, errors.New(fmt.Sprintf("type '%s' not defined\n", typ))
		}
		
		for _, fld := range foundStrct.Fields {
			isBasic := false
			for _, basic := range BASIC_TYPES {
				if fld.Typ == basic {
					isBasic = true
					bs = append(bs, *MakeDefaultValue(basic)...)
					break
				}
			}

			if !isBasic {
				var typ string
				if fld.Typ[:2] == "[]" {
					typ = fld.Typ[2:]
				} else {
					typ = fld.Typ
				}
				if _, err := cxt.GetStruct(typ, mod.Name); err == nil {
					if byts, err := ResolveStruct(fld.Typ, cxt); err == nil {
						bs = append(bs, byts...)
					} else {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
		}
	} else {
		return nil, err
	}
	return bs, nil
}

func GetArrayFromArray (value []byte, typ string, index int32) ([]byte, error, int32, int32) {
	var arrSize int32
	encoder.DeserializeAtomic(value[:4], &arrSize)

	if index < 0 {
		return nil, errors.New(fmt.Sprintf("%s.read: negative index %d", typ, index)), 0, 0
	}

	if index >= arrSize {
		return nil, errors.New(fmt.Sprintf("%s.read: index %d exceeds array of length %d", typ, index, arrSize)), 0, 0
	}

	var typSize int
	switch typ[len(typ)-4:] {
	case "]i64", "]f64":
		typSize = 8
	case "bool", "]i32", "]f32":
		typSize = 4
	case "byte", "]str":
		typSize = 1
	}

	if typ[len(typ)-3:] == "str" {
		typ = "[]" + typ
	}
	
	var sizes []int32
	var counters []int32
	
	var finalOffset int = -1
	var finalSize int = -1
	
	var i int
	for i = 0; i < len(value); {
		if typ[:4] == "[][]" {
			var size int32
			encoder.DeserializeAtomic(value[i:i+4], &size)
			
			sizes = append(sizes, size)
			counters = append(counters, size)
			
			typ = typ[2:]
			i += 4
		}

		if typ[2] != '[' {
			var size int32
			encoder.DeserializeAtomic(value[i:i+4], &size)
			
			i += int(size) * typSize + 4
			counters[len(counters)-1]--
		}

		if len(counters) > 0 {
			for c := len(counters); c > 0; c-- {
				if counters[c-1] < 1 {
					typ = "[]" + typ
					sizes = sizes[:len(sizes)-1]
					counters = counters[:len(counters)-1]
					if len(counters) > 0 {
						counters[len(counters)-1]--
					}
				}
			}
		}

		if finalOffset < 0 {
			if index == 0 {
				finalOffset = 4
			} else if sizes[0] - counters[0] == index {
				finalOffset = i
			}
		}

		if finalSize < 0 {
			if finalOffset > 0 && (len(sizes) == 0 || index == sizes[0] - 1) {
				finalSize = len(value)
			} else if sizes[0] - counters[0] == index + 1 {
				finalSize = i
			}
		}

		if finalOffset > 0 && finalSize > 0 {
			break
		}
	}

	return value[finalOffset:finalSize], nil, int32(finalOffset), int32(finalSize - finalOffset)
}

func getStrctFromArray (arr *CXArgument, index int32, expr *CXExpression, call *CXCall) ([]byte, error, int32, int32) {
	var arrSize int32
	encoder.DeserializeAtomic((*arr.Value)[:4], &arrSize)

	if index < 0 {
		return nil, errors.New(fmt.Sprintf("%s.read: negative index %d", arr.Typ, index)), 0, 0
	}

	if index >= arrSize {
		return nil, errors.New(fmt.Sprintf("%s.read: index %d exceeds array of length %d", arr.Typ, index, arrSize)), 0, 0
	}

	if strct, err := call.Context.GetStruct(arr.Typ[2:], expr.Module.Name); err == nil {
		instances := (*arr.Value)[4:]
		lastFld := strct.Fields[len(strct.Fields) - 1]
		
		var lowerBound int32
		var upperBound int32
		
		var size int32
		encoder.DeserializeAtomic((*arr.Value)[:4], &size)

		// in here we can use <=. we can't do this in resolveStrctField
		for c := int32(0); c <= index; c++ {
			subArray := instances[upperBound:]
			_, _, off, size := resolveStructField(lastFld.Name, &subArray, strct)

			lowerBound = upperBound
			upperBound = upperBound + off + size
		}

		output := instances[lowerBound:upperBound]
		return output, nil, lowerBound + 4, upperBound - lowerBound
	} else {
		return nil, err, 0, 0
	}
}

func getValueFromArray (arr *CXArgument, index int32) ([]byte, error) {
	var arrSize int32
	encoder.DeserializeAtomic((*arr.Value)[:4], &arrSize)

	if index < 0 {
		return nil, errors.New(fmt.Sprintf("%s.read: negative index %d", arr.Typ, index))
	}

	if index >= arrSize {
		return nil, errors.New(fmt.Sprintf("%s.read: index %d exceeds array of length %d", arr.Typ, index, arrSize))
	}

	switch arr.Typ {
	case "[]byte":
		return (*arr.Value)[index + 4:index + 1 + 4], nil
	case "[]bool", "[]i32", "[]f32":
		return (*arr.Value)[index * 4 + 4:(index + 1) * 4 + 4], nil
	case "[]str":
		noSize := (*arr.Value)[4:]

		var offset int32
		for c := 0; c < int(index); c++ {
			var strSize int32
			encoder.DeserializeRaw(noSize[offset:offset+4], &strSize)
			offset += strSize + 4
		}

		sStrSize := noSize[offset:offset + 4]
		var strSize int32
		encoder.DeserializeRaw(sStrSize, &strSize)

		return noSize[offset:offset+strSize+4], nil
	case "[]i64", "f64":
		return (*arr.Value)[index * 8 + 4:(index + 1) * 8 + 4], nil
	}
	
	return nil, nil
}

func (cxt *CXProgram) PrintProgram (withAffs bool) {
	fmt.Println("Program")
	if withAffs {
		for i, aff := range cxt.GetAffordances() {
			fmt.Printf(" * %d.- %s\n", i, aff.Description)
		}
	}

	i := 0
	for _, mod := range cxt.Modules {
		cxt.SelectModule(mod.Name)
		if mod.Name == CORE_MODULE || mod.Name == "glfw" || mod.Name == "gl" || mod.Name == "gltext" {
			continue
		}

		fmt.Printf("%d.- Module: %s\n", i, mod.Name)

		if withAffs {
			for i, aff := range mod.GetAffordances() {
				fmt.Printf("\t * %d.- %s\n", i, aff.Description)
			}
		}

		if len(mod.Imports) > 0 {
			fmt.Println("\tImports")
		}

		j := 0
		for _, imp := range mod.Imports {
			fmt.Printf("\t\t%d.- Import: %s\n", j, imp.Name)
			j++
		}

		if len(mod.Definitions) > 0 {
			fmt.Println("\tDefinitions")
		}

		j = 0
		for _, v := range mod.Definitions {
			fmt.Printf("\t\t%d.- Definition: %s %s\n", j, v.Name, v.Typ)
			j++
		}

		if len(mod.Structs) > 0 {
			fmt.Println("\tStructs")
		}

		j = 0
		for _, strct := range mod.Structs {
			fmt.Printf("\t\t%d.- Struct: %s\n", j, strct.Name)

			if withAffs {
				for i, aff := range strct.GetAffordances() {
					fmt.Printf("\t\t * %d.- %s\n", i, aff.Description)
				}
			}

			for k, fld := range strct.Fields {
				fmt.Printf("\t\t\t%d.- Field: %s %s\n",
					k, fld.Name, fld.Typ)
			}
			
			j++
		}

		if len(mod.Functions) > 0 {
			fmt.Println("\tFunctions")
		}

		j = 0
		for _, fn := range mod.Functions {
			mod.SelectFunction(fn.Name)
			inOuts := make(map[string]string)
			for _, in := range fn.Inputs {
				inOuts[in.Name] = in.Typ
			}
			
			
			var inps bytes.Buffer
			for i, inp := range fn.Inputs {
				if i == len(fn.Inputs) - 1 {
					inps.WriteString(concat(inp.Name, " ", inp.Typ))
				} else {
					inps.WriteString(concat(inp.Name, " ", inp.Typ, ", "))
				}
			}

			var outs bytes.Buffer
			for i, out := range fn.Outputs {
				if i == len(fn.Outputs) - 1 {
					outs.WriteString(concat(out.Name, " ", out.Typ))
				} else {
					outs.WriteString(concat(out.Name, " ", out.Typ, ", "))
				}
			}

			fmt.Printf("\t\t%d.- Function: %s (%s) (%s)\n",
				j, fn.Name, inps.String(), outs.String())

			if withAffs {
				for i, aff := range fn.GetAffordances() {
					fmt.Printf("\t\t * %d.- %s\n", i, aff.Description)
				}
			}

			k := 0
			for _, expr := range fn.Expressions {
				//Arguments
				var args bytes.Buffer

				for i, arg := range expr.Arguments {
					typ := ""
					if arg.Typ == "ident" {
						var id string
						encoder.DeserializeRaw(*arg.Value, &id)
						var err error
						if typ, err = GetIdentType(id, expr.FileLine, expr.FileName, cxt); err != nil {
							panic(err)
						}
					} else {
						typ = arg.Typ
					}

					var argName string
					encoder.DeserializeRaw(*arg.Value, &argName)

					if arg.Typ != "ident" {
						switch typ {
						case "str":
							argName = fmt.Sprintf("%#v", argName)
						case "bool":
							var val int32
							encoder.DeserializeRaw(*arg.Value, &val)
							if val == 0 {
								argName = "false"
							} else {
								argName = "true"
							}
						case "byte":
							argName = fmt.Sprintf("%#v", *arg.Value)
						case "i32":
							var val int32
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "i64":
							var val int64
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "f32":
							var val float32
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "f64":
							var val float64
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]bool":
							var val []bool
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]byte":
							var val []byte
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]str":
							var val []string
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]i32":
							var val []int32
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]i64":
							var val []int64
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]f32":
							var val []float32
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						case "[]f64":
							var val []float64
							encoder.DeserializeRaw(*arg.Value, &val)
							argName = fmt.Sprintf("%#v", val)
						default:
							if arg.Typ[0] == '*' || arg.Typ[0] == '$' {
								var identName string
								encoder.DeserializeRaw(*arg.Value, &identName)
								argName = identName
							} else {
								argName = string(*arg.Value)
							}
						}
					}

					// if arg.Offset > -1 {
					// 	offset := arg.Offset
					// 	size := arg.Size
					// 	var val []byte
					// 	encoder.DeserializeRaw((*cxt.Heap)[offset:offset+size], &val)
					// 	arg.Value = &val
					// }

					if i == len(expr.Arguments) - 1 {
						args.WriteString(concat(argName, " ", typ))
					} else {
						args.WriteString(concat(argName, " ", typ, ", "))
					}
				}

				if len(expr.OutputNames) > 0 {
					var outNames bytes.Buffer
					for i, outName := range expr.OutputNames {
						if i == len(expr.OutputNames) - 1 {
							outNames.WriteString(outName.Name)
						} else {
							outNames.WriteString(concat(outName.Name, ", "))
						}
					}

					var exprTag string
					if expr.Tag != "" {
						exprTag = fmt.Sprintf(" :tag %s", expr.Tag)
					}

					fmt.Printf("\t\t\t%d.- Expression: %s = %s(%s)%s\n",
						k,
						outNames.String(),
						expr.Operator.Name,
						args.String(),
						exprTag)
				} else {
					var exprTag string
					if expr.Tag != "" {
						exprTag = fmt.Sprintf(" :tag %s", expr.Tag)
					}
					
					fmt.Printf("\t\t\t%d.- Expression: %s(%s)%s\n",
						k,
						expr.Operator.Name,
						args.String(),
						exprTag)
				}

				

				if withAffs {
					for i, aff := range expr.GetAffordances(nil) {
						fmt.Printf("\t\t\t * %d.- %s\n", i, aff.Description)
					}
				}
				
				k++
			}
			j++
		}
		i++
	}
}

func PrintCallStack (callStack []*CXCall) {
	for i, call := range callStack {
		tabs := strings.Repeat("___", i)
		if tabs == "" {
			//fmt.Printf("%sfn:%s ln:%d, \tlocals: ", tabs, call.Operator.Name, call.Line)
			fmt.Printf("%sfn:%s ln:%d", tabs, call.Operator.Name, call.Line)
		} else {
			//fmt.Printf("↓%sfn:%s ln:%d, \tlocals: ", tabs, call.Operator.Name, call.Line)
			fmt.Printf("↓%sfn:%s ln:%d", tabs, call.Operator.Name, call.Line)
		}

		// lenState := len(call.State)
		// idx := 0
		// for _, def := range call.State {
		// 	if def.Name == "_" || (len(def.Name) > len(NON_ASSIGN_PREFIX) && def.Name[:len(NON_ASSIGN_PREFIX)] == NON_ASSIGN_PREFIX) {
		// 		continue
		// 	}
		// 	var valI32 int32
		// 	var valI64 int64
		// 	var valF32 float32
		// 	var valF64 float64
		// 	switch def.Typ {
		// 	case "i32":
		// 		encoder.DeserializeRaw(*def.Value, &valI32)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %d", def.Name, valI32)
		// 		} else {
		// 			fmt.Printf("%s: %d, ", def.Name, valI32)
		// 		}
		// 	case "i64":
		// 		encoder.DeserializeRaw(*def.Value, &valI64)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %d", def.Name, valI64)
		// 		} else {
		// 			fmt.Printf("%s: %d, ", def.Name, valI64)
		// 		}
		// 	case "f32":
		// 		encoder.DeserializeRaw(*def.Value, &valF32)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %f", def.Name, valF32)
		// 		} else {
		// 			fmt.Printf("%s: %f, ", def.Name, valF32)
		// 		}
		// 	case "f64":
		// 		encoder.DeserializeRaw(*def.Value, &valF64)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %f", def.Name, valF64)
		// 		} else {
		// 			fmt.Printf("%s: %f, ", def.Name, valF64)
		// 		}
		// 	case "byte":
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %d", def.Name, (*def.Value)[0])
		// 		} else {
		// 			fmt.Printf("%s: %d, ", def.Name, (*def.Value)[0])
		// 		}
		// 	case "[]byte":
		// 		var val []byte
		// 		encoder.DeserializeRaw(*def.Value, &val)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %v", def.Name, val)
		// 		} else {
		// 			fmt.Printf("%s: %v, ", def.Name, val)
		// 		}
		// 	case "[]i32":
		// 		var val []int32
		// 		encoder.DeserializeRaw(*def.Value, &val)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %v", def.Name, val)
		// 		} else {
		// 			fmt.Printf("%s: %v, ", def.Name, val)
		// 		}
		// 	case "[]i64":
		// 		var val []int64
		// 		encoder.DeserializeRaw(*def.Value, &val)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %v", def.Name, val)
		// 		} else {
		// 			fmt.Printf("%s: %v, ", def.Name, val)
		// 		}
		// 	case "[]f32":
		// 		var val []float32
		// 		encoder.DeserializeRaw(*def.Value, &val)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %v", def.Name, val)
		// 		} else {
		// 			fmt.Printf("%s: %v, ", def.Name, val)
		// 		}
		// 	case "[]f64":
		// 		var val []float64
		// 		encoder.DeserializeRaw(*def.Value, &val)
		// 		if idx == lenState - 1 {
		// 			fmt.Printf("%s: %v", def.Name, val)
		// 		} else {
		// 			fmt.Printf("%s: %v, ", def.Name, val)
		// 		}
		// 	}
			
		// 	idx++
		// }
		fmt.Println()
	}
}

func oneI32oneI32 (fn func(int32)int32, arg1 *CXArgument) []byte {
	var num1 int32
	encoder.DeserializeAtomic(*arg1.Value, &num1)
	return encoder.SerializeAtomic(int32(fn(num1)))
}

func twoI32oneI32 (fn func(int32, int32)int32, arg1, arg2 *CXArgument) []byte {
	var num1 int32
	var num2 int32
	encoder.DeserializeAtomic(*arg1.Value, &num1)
	encoder.DeserializeAtomic(*arg2.Value, &num2)
	return encoder.SerializeAtomic(int32(fn(num1, num2)))
}

func GetIdentType (lookingFor string, line int, fileName string, cxt *CXProgram) (string, error) {
	identParts := strings.Split(lookingFor, ".")

	mod, err := cxt.GetCurrentModule();
	if err != nil {
		return "", err
	}
	
	if len(identParts) > 1 {
		if extMod, err := cxt.GetModule(identParts[0]); err == nil {
			// then it's an external definition or struct
			isImported := false
			for _, imp := range mod.Imports {
				if imp.Name == identParts[0] {
					isImported = true
					break
				}
			}
			if isImported {
				if def, err := extMod.GetDefinition(concat(identParts[1:]...)); err == nil {
					return def.Typ, nil
				}
			} else {
				return "", errors.New(fmt.Sprintf("%s: %d: module '%s' was not imported or does not exist", fileName, line, extMod.Name))
			}
		} else {
			// local struct instance
			if fn, err := cxt.GetCurrentFunction(); err == nil {
				for _, inp := range fn.Inputs {
					if inp.Name == identParts[0] {
						if strct, err := cxt.GetStruct(inp.Typ, mod.Name); err == nil {
							for _, fld := range strct.Fields {
								if fld.Name == identParts[1] {
									return fld.Typ, nil
								}
							}
						}
					}
				}
				for _, out := range fn.Outputs {
					if out.Name == identParts[0] {
						if strct, err := cxt.GetStruct(out.Typ, mod.Name); err == nil {
							for _, fld := range strct.Fields {
								if fld.Name == identParts[1] {
									return fld.Typ, nil
								}
							}
						}
					}
				}
				for _, expr := range fn.Expressions {
					if expr.Operator.Name == "initDef" && expr.OutputNames[0].Name == identParts[0] {
						var typ string
						encoder.DeserializeRaw(*expr.Arguments[0].Value, &typ)
						
						if strct, err := cxt.GetStruct(typ, mod.Name); err == nil {
							for _, fld := range strct.Fields {
								if fld.Name == identParts[1] {
									return fld.Typ, nil
								}
							}
						}
					}
					for _, out := range expr.OutputNames {
						if out.Name == lookingFor {
							return out.Typ, nil
						}
						if out.Name == identParts[0] {
							if strct, err := cxt.GetStruct(out.Typ, mod.Name); err == nil {
								for _, fld := range strct.Fields {
									if fld.Name == identParts[1] {
										return fld.Typ, nil
									}
								}
							}
						}
					}
				}
			} else {
				return "", err
			}

			// global struct instance
			if def, err := mod.GetDefinition(identParts[0]); err == nil {
				if strct, err := cxt.GetStruct(def.Typ, mod.Name); err == nil {
					for _, fld := range strct.Fields {
						if fld.Name == identParts[1] {
							return fld.Typ, nil
						}
					}
				}
			} else {
				// then it's a local struct
				
			}
		}
	} else {
		// then it's a local or global definition
		arrayParts := strings.Split(lookingFor, "[")
		if len(arrayParts) > 1 {
			lookingFor = arrayParts[0]
		}

		if fn, err := cxt.GetCurrentFunction(); err == nil {
			for _, inp := range fn.Inputs {
				if inp.Name == arrayParts[0] {
					if len(arrayParts) > 1 {
						return inp.Typ[2:], nil
					} else {
						return inp.Typ, nil
					}
				}
			}
			for _, out := range fn.Outputs {
				if out.Name == arrayParts[0] {
					if len(arrayParts) > 1 {
						return out.Typ[2:], nil
					} else {
						return out.Typ, nil
					}
				}
			}
			for _, expr := range fn.Expressions {
				if expr.Operator.Name == "initDef" && expr.OutputNames[0].Name == identParts[0] {
					var typ string
					encoder.DeserializeRaw(*expr.Arguments[0].Value, &typ)

					return typ, nil
				}
				for _, out := range expr.OutputNames {
					if out.Name == arrayParts[0] {
						//fmt.Println("here", out.Name, out.Typ)

						// if expr.Operator.Name == "identity" {
						// 	return fn.Expressions[i-1].OutputNames[0].Typ, nil
						// }
						
						if len(arrayParts) > 1 {
							return out.Typ[2:], nil
						} else {
							return out.Typ, nil
						}
					}
				}
			}
		} else {
			return "", err
		}

		// then it's a global definition
		if def, err := mod.GetDefinition(lookingFor); err == nil {
			return def.Typ, nil
		}
	}

	return "", errors.New(fmt.Sprintf("%s: %d: identifier '%s' could not be resolved", fileName, line, lookingFor))
}
