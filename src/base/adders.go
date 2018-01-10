package base

import (
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func (cxt *CXProgram) AddModule (mod *CXModule) *CXProgram {
	mod.Context = cxt
	found := false
	for _, md := range cxt.Modules {
		if md.Name == mod.Name {
			cxt.CurrentModule = md
			found = true
			break
		}
	}
	if !found {
		cxt.Modules = append(cxt.Modules, mod)
		cxt.CurrentModule = mod
	}
	return cxt
}

func (mod *CXModule) AddDefinition (def *CXDefinition) *CXModule {
	def.Context = mod.Context
	def.Module = mod
	found := false
	for i, df := range mod.Definitions {
		if df.Name == def.Name {
			mod.Definitions[i] = def
			found = true
			break
		}
	}
	if !found {
		mod.Definitions = append(mod.Definitions, def)
	}
	return mod
}

func (mod *CXModule) AddFunction (fn *CXFunction) *CXModule {
	fn.Context = mod.Context
	fn.Module = mod
	fn.NumberOutputs = len(fn.Outputs)
	//mod.CurrentFunction = fn
	
	found := false
	for i, f := range mod.Functions {
		if f.Name == fn.Name {
			//mod.Functions[i] = fn
			mod.Functions[i].Name = fn.Name
			mod.Functions[i].Inputs = fn.Inputs
			mod.Functions[i].Outputs = fn.Outputs
			mod.Functions[i].Expressions = fn.Expressions
			mod.Functions[i].NumberOutputs = fn.NumberOutputs
			mod.Functions[i].CurrentExpression = fn.CurrentExpression
			mod.Functions[i].Module = fn.Module
			mod.Functions[i].Context = fn.Context
			mod.CurrentFunction = mod.Functions[i]
			found = true
			break
		}
	}
	if !found {
		mod.Functions = append(mod.Functions, fn)
		mod.CurrentFunction = fn
	}

	return mod
}

func (mod *CXModule) AddStruct (strct *CXStruct) *CXModule {
	cxt := mod.Context
	strct.Context = cxt
	strct.Module = mod
	mod.CurrentStruct = strct
	found := false
	for i, s := range mod.Structs {
		if s.Name == strct.Name {
			mod.Structs[i] = strct
			found = true
			break
		}
	}
	if !found {
		mod.Structs = append(mod.Structs, strct)
	}
	return mod
}

func (mod *CXModule) AddImport (imp *CXModule) *CXModule {
	found := false
	for _, im := range mod.Imports {
		if im.Name == imp.Name {
			found = true
			break
		}
	}
	if !found {
		mod.Imports = append(mod.Imports, imp)
	}
	
	return mod
}

func (strct *CXStruct) AddField (fld *CXField) *CXStruct {
	found := false
	for _, fl := range strct.Fields {
		if fl.Name == fld.Name {
			found = true
			break
		}
	}
	if !found {
		strct.Fields = append(strct.Fields, fld)
	}
	return strct
}

func (fn *CXFunction) AddExpression (expr *CXExpression) *CXFunction {
	expr.Context = fn.Context
	expr.Module = fn.Module
	expr.Function = fn
	expr.Line = len(fn.Expressions)
	fn.Expressions = append(fn.Expressions, expr)
	fn.CurrentExpression = expr
	return fn
}

func (fn *CXFunction) AddInput (param *CXParameter) *CXFunction {
	found := false
	for _, inp := range fn.Inputs {
		if inp.Name == param.Name {
			found = true
			break
		}
	}
	if !found {
		fn.Inputs = append(fn.Inputs, param)
	}
	
	return fn
}

func (fn *CXFunction) AddOutput (param *CXParameter) *CXFunction {
	found := false
	for _, out := range fn.Outputs {
		if out.Name == param.Name {
			found = true
			break
		}
	}
	if !found {
		fn.Outputs = append(fn.Outputs, param)
	}

	return fn
}

func (expr *CXExpression) AddArgument (arg *CXArgument) *CXExpression {
	expr.Arguments = append(expr.Arguments, arg)
	return expr
}

func (expr *CXExpression) AddOutputName (outName string) *CXExpression {
	if len(expr.Operator.Outputs) > 0 {
		nextOutIdx := len(expr.OutputNames)

		var typ string
		var ptrs string
		if expr.Operator.Name == ID_FN || expr.Operator.Name == INIT_FN {
			var tmp string
			encoder.DeserializeRaw(*expr.Arguments[0].Value, &tmp)
			
			if tmp[0] == '*' {
				for i, char := range tmp {
					if char != '*' {
						tmp = tmp[i:]
						break
					} else {
						ptrs += "*"
					}
				}
			}
			
			if expr.Operator.Name == INIT_FN {
				// then tmp is the type (e.g. initDef("i32") to initialize an i32)
				typ = ptrs + tmp
			} else {
				var err error
				// then tmp is an identifier
				if typ, err = GetIdentType(tmp, expr.FileLine, expr.FileName, expr.Context); err == nil {
					typ = ptrs + typ
				} else {
					panic(err)
				}
			}
		} else {
			typ = expr.Operator.Outputs[nextOutIdx].Typ
		}

		//print(typ + " " + outName + "\n")
		outDef := MakeDefinition(
			outName,
			MakeDefaultValue(expr.Operator.Outputs[nextOutIdx].Typ),
			//expr.Operator.Outputs[nextOutIdx].Typ)
			typ)
		
		outDef.Module = expr.Module
		outDef.Context = expr.Context
		
		expr.OutputNames = append(expr.OutputNames, outDef)
	}
	
	return expr
}

func (expr *CXExpression) AddTag (tag string) *CXExpression {
	expr.Tag = tag
	return expr
}
