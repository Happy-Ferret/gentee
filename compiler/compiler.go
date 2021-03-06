// Copyright 2018 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"

	"github.com/gentee/gentee/core"
)

// Compiler contains information of the compilation process
type compiler struct {
	vm       *core.VirtualMachine
	unit     *core.Unit
	owners   []core.ICmd
	exp      []core.ICmd
	expbuf   []ExpBuf
	lexems   []int // stack of lexeme
	runID    int
	pos      int // current position
	newPos   int // new position
	states   *[]StateStack
	curType  *core.TypeObject // the current type of parameters or variables
	curConst string
	curFunc  int       // index of the latest func
	expConst core.ICmd // expression for constants
	curIota  int64     // current iota
	inits    int       // initilization level mode
	endColon int
	isImport bool // import or include mode
	next     *cmState
	dynamic  *cmState
	goStack  []goStack
}

// StateStack is used for storing a sequence of states
type StateStack struct {
	Origin *cmState
	Pos    int
	State  int
}

// Priority is a structure for operations in expressions
type Priority struct {
	Priority  int
	RightLeft bool
	Name      string
}

// ExpBuf is a structure for buffer of expression operations
type ExpBuf struct {
	Oper   int
	Pos    int
	LenExp int
}

var (
	priority = map[int]Priority{
		tkRange: {3, true, `NewRange`},
		//		tkColon:        {4, true, `NewKeyValue`},
		tkAssign:                 {5, true, `Assign`},
		tkAddEq:                  {5, true, `AssignAdd`},
		tkSubEq:                  {5, true, `AssignSub`},
		tkMulEq:                  {5, true, `AssignMul`},
		tkDivEq:                  {5, true, `AssignDiv`},
		tkModEq:                  {5, true, `AssignMod`},
		tkLShiftEq:               {5, true, `AssignLShift`},
		tkRShiftEq:               {5, true, `AssignRShift`},
		tkBitAndEq:               {5, true, `AssignBitAnd`},
		tkBitOrEq:                {5, true, `AssignBitOr`},
		tkBitXorEq:               {5, true, `AssignBitXor`},
		tkCtxEq:                  {6, false, `CtxSet`},
		tkAnd:                    {7, false, ``},
		tkOr:                     {8, false, ``},
		tkEqual:                  {10, false, `Equal`},
		tkNotEqual:               {10, false, `Equal`},
		tkLess:                   {10, false, `Less`},
		tkLessEqual:              {10, false, `Greater`},
		tkGreater:                {10, false, `Greater`},
		tkGreaterEqual:           {10, false, `Less`},
		tkBitOr:                  {11, false, `BitOr`},
		tkBitXor:                 {12, false, `BitXor`},
		tkBitAnd:                 {13, false, `BitAnd`},
		tkLShift:                 {14, false, `LShift`},
		tkRShift:                 {14, false, `RShift`},
		tkAdd:                    {15, false, `Add`},
		tkSub:                    {15, false, `Sub`},
		tkDiv:                    {20, false, `Div`},
		tkMod:                    {20, false, `Mod`},
		tkMul:                    {20, false, `Mul`},
		tkInc | tkUnary | tkPost: {29, false, ``},
		tkDec | tkUnary | tkPost: {29, false, ``},
		tkBitXor | tkUnary:       {30, true, `BitNot`},
		tkBitOr | tkUnary:        {30, true, `PrintShift`},
		tkSub | tkUnary:          {30, true, `Sign`},
		tkNot | tkUnary:          {30, true, `Not`},
		tkMul | tkUnary:          {30, true, `Len`},
		tkInc | tkUnary:          {30, true, ``},
		tkDec | tkUnary:          {30, true, ``},
		tkCtx | tkUnary:          {30, true, `Ctx`},
		tkDoubleCtx | tkUnary:    {30, true, `Ctx`},
		tkBitAnd | tkUnary:       {35, true, ``},
		tkStrExp:                 {35, false, `ExpStr`},
		tkLPar:                   {50, true, ``},
		tkRPar:                   {50, true, ``},
		tkLSBracket:              {50, true, ``},
		tkRSBracket:              {50, true, ``},
	}
)

func init() {
	makeLexTable()
	makeCompileTable()
}

func (cmpl *compiler) curOwner() *core.CmdBlock {
	return cmpl.owners[len(cmpl.owners)-1].(*core.CmdBlock)
}

func (cmpl *compiler) appendObj(obj core.IObject) (ret int) {
	//	cmpl.unit.NewObject(obj)
	cmpl.unit.VM.Objects = append(cmpl.unit.VM.Objects, obj)
	ret = len(cmpl.vm.Objects) - 1
	if obj.GetType() == core.ObjFunc {
		cmpl.curFunc = ret
	}
	return
}

func (cmpl *compiler) latestFunc() *core.FuncObject {
	return cmpl.vm.Objects[cmpl.curFunc].(*core.FuncObject)
}

// Compile compiles the source code
func Compile(vm *core.VirtualMachine, input, path string) (int, error) {

	countObjects := len(vm.Objects)
	countUnits := len(vm.Units)

	lp, errID := LexParsing([]rune(input))
	lp.Path = path
	cmpl := &compiler{
		vm:      vm,
		unit:    vm.InitUnit(),
		lexems:  []int{0}, // added lp in Lexeme
		runID:   core.Undefined,
		owners:  make([]core.ICmd, 0, 128),
		exp:     make([]core.ICmd, 0, 128),
		expbuf:  make([]ExpBuf, 0, 128),
		curIota: core.NotIota,
	}
	cmpl.unit.Lexeme = []*core.Lex{lp}
	cmpl.copyNameSpace(vm.StdLib(), true)
	cmplError := func(err interface{}) (int, error) {
		// Rollback vm
		vm.Objects = vm.Objects[:countObjects]
		vm.Units = vm.Units[:countUnits]
		for key, unitID := range vm.UnitNames {
			if unitID >= countUnits {
				delete(vm.UnitNames, key)
			}
		}

		if v, ok := err.(int); ok {
			err = cmpl.Error(v)
		}
		return core.Undefined, err.(error)
	}

	if len(lp.Tokens) == 0 {
		return cmplError(ErrEmptyCode)
	}
	if errID != 0 {
		cmpl.pos = len(lp.Tokens) - 1
		return cmplError(errID)
	}

	stackState := make([]StateStack, 0, 32)
	state := cmMain
main:
	for i := 0; i < len(lp.Tokens); i++ {
		if cmpl.inits == 0 && lp.Tokens[i].Type == tkColon {
			if err := colonToLine(cmpl, i); err != nil {
				return cmplError(err)
			}
		}
		cmpl.pos = i
		token := lp.Tokens[i]
		if state == cmBody && token.Type == tkIdent && i+1 < len(lp.Tokens) &&
			lp.Tokens[i+1].Type != tkLPar {
			obj, _ := getType(cmpl)
			if obj != nil {
				token.Type = tkType
			}
		}
		cmpl.next = compileTable[state][token.Type]
		cmpl.states = &stackState
		cmpl.dynamic = nil
		cmpl.newPos = 0
		//		fmt.Printf("NEXT i=%d state=%d token=%d v=%v inits=%d nextstate=%v %v\r\n", i, state, token.Type,
		//			getToken(cmpl.getLex(), i), cmpl.inits, cmpl.next, stackState)
		if (state == cmExp || state == cmExpOper) && token.Type == tkLine {
			if state == cmExp && lp.Tokens[i-1].Type >= tkAdd && lp.Tokens[i-1].Type <= tkComma {
				continue
			}
			for _, expBuf := range cmpl.expbuf {
				if expBuf.Oper == tkLPar || expBuf.Oper == tkLSBracket {
					continue main
				}
			}
		}
		if cmpl.next.Func != nil {
			if err := cmpl.next.Func(cmpl); err != nil {
				return cmplError(err)
			}
			if cmpl.newPos != 0 {
				i = cmpl.newPos
			}
			if cmpl.dynamic != nil {
				stackState = append(stackState, StateStack{Origin: cmpl.dynamic, Pos: i, State: state})
				state = cmpl.dynamic.State
				if cmpl.dynamic.Flags&cfStay != 0 {
					i--
				}
				continue
			}
		}
		if cmpl.next.State == 0 {
			continue
		}
		if cmpl.next.Flags&cfStay != 0 {
			i--
		}
		if cmpl.next.State == cmBack {
			if len(stackState) == 0 {
				return cmplError(cmpl.Error(ErrCompiler, `Compile`))
			}
			for len(stackState) > 0 {
				prev := stackState[len(stackState)-1]
				state = prev.State
				if prev.Origin.Callback != nil {
					//cmpl.pos = prev.Pos
					if err := prev.Origin.Callback(cmpl); err != nil {
						return cmplError(err)
					}
					if cmpl.dynamic != nil {
						stackState = append(stackState, StateStack{Origin: cmpl.dynamic, Pos: i, State: state})
						state = cmpl.dynamic.State
						if cmpl.dynamic.Flags&cfStay != 0 {
							i--
						}
					}
				}
				stackState = stackState[:len(stackState)-1]
				if prev.Origin.Flags&cfStopBack != 0 || cmpl.dynamic != nil {
					break
				}
			}
			continue
		}

		stackState = append(stackState, StateStack{Origin: cmpl.next, Pos: i, State: state})
		state = cmpl.next.State
	}
	if len(stackState) > 0 {
		return cmplError(cmpl.ErrorPos(len(lp.Tokens), ErrEnd))
	}

	if cmpl.runID != core.Undefined {
		cmpl.unit.RunID = cmpl.runID
		if len(cmpl.unit.Name) == 0 {
			cmpl.unit.Name = path
		}
		if unitIndex, ok := vm.UnitNames[cmpl.unit.Name]; ok {
			if vm.Units[unitIndex].Lexeme[0].Path != path {
				return cmplError(cmpl.Error(ErrLink, cmpl.unit.Name))
			}
		}
		/*				vm.Units[unitIndex] = cmpl.unit
						vm.Compiled = unitIndex
					} else {*/
	}
	vm.Units = append(vm.Units, cmpl.unit)
	unitID := len(vm.Units) - 1
	vm.UnitNames[cmpl.unit.Name] = unitID
	vm.Units[unitID].Index = uint32(unitID)

	return unitID, nil
}

func colonToLine(cmpl *compiler, i int) error {
	if i < cmpl.endColon {
		return cmpl.ErrorPos(i, ErrDoubleColon)
	}
	lp := cmpl.unit.Lexeme[0]
	lp.Tokens[i].Type = tkLCurly
	end := i + 1
	for ; end < len(lp.Tokens); end++ {
		if lp.Tokens[end].Type == tkLine && lp.Source[lp.Tokens[end].Offset] != ';' {
			break
		}
	}
	if end == len(lp.Tokens) {
		lp.Tokens = append(lp.Tokens, core.Token{Type: int32(tkRCurly),
			Offset: lp.Tokens[len(lp.Tokens)-1].Offset, Length: 1})
	} else {
		lp.Tokens = append(lp.Tokens[:end], append([]core.Token{{Type: int32(tkRCurly),
			Offset: lp.Tokens[end].Offset, Length: 1}}, lp.Tokens[end:]...)...)
	}
	cmpl.endColon = end
	return nil
}

func coIndex(cmpl *compiler) error {
	coExpVar(cmpl)
	return appendExpBuf(cmpl, tkIndex)
}

func isEqualTypes(left *core.TypeObject, right *core.TypeObject) bool {
	if left == nil || right == nil {
		return left == right
	}
	switch left.Original {
	case reflect.TypeOf(core.Fn{}):
		if right.Original != reflect.TypeOf(core.Fn{}) ||
			len(left.Func.Params) != len(right.Func.Params) ||
			!isEqualTypes(left.Func.Result, right.Func.Result) {
			return false
		}
		for i, param := range left.Func.Params {
			if !isEqualTypes(param, right.Func.Params[i]) {
				return false
			}
		}
		return true
	case reflect.TypeOf(core.Array{}):
		if right.Original != reflect.TypeOf(core.Array{}) {
			return false
		}
		// compare for arr*
		if left.IndexOf == nil || right.IndexOf == nil {
			return true
		}
		return isEqualTypes(left.IndexOf, right.IndexOf)
	case reflect.TypeOf(core.Map{}):
		if right.Original != reflect.TypeOf(core.Map{}) {
			return false
		}
		// compare for map*
		if left.IndexOf == nil || right.IndexOf == nil {
			return true
		}
		return isEqualTypes(left.IndexOf, right.IndexOf)
	}
	return left == right
}

func autoType(cmpl *compiler, name string) (obj core.IObject, err error) {
	if strings.HasSuffix(name, `.arr`) || strings.HasSuffix(name, `.map`) {
		name += `.str`
	}
	obj = cmpl.unit.FindType(name)
	if obj == nil {
		ins := strings.SplitN(name, `.`, 2)
		if len(ins) == 2 {
			if ins[0] == `arr` {
				var indexOf core.IObject
				indexOf, err = autoType(cmpl, ins[1])
				if indexOf != nil {
					if obj = cmpl.unit.NewType(name, reflect.TypeOf(core.Array{}), indexOf); obj != nil {
						return
					}
				}
			} else if ins[0] == `map` {
				var indexOf core.IObject
				indexOf, err = autoType(cmpl, ins[1])
				if indexOf != nil {
					if obj = cmpl.unit.NewType(name, reflect.TypeOf(core.Map{}), indexOf); obj != nil {
						return
					}
				}
			}
		}
		return nil, cmpl.Error(ErrType)
	}
	return
}

func appendCmd(cmpl *compiler, cmd core.ICmd) {
	owner := cmpl.curOwner()
	if cmd.GetType() == core.CtStack {
		cmd.(*core.CmdBlock).Parent = owner
	}
	owner.Children = append(owner.Children, cmd)
}

func getType(cmpl *compiler) (obj core.IObject, err error) {
	return autoType(cmpl, getToken(cmpl.getLex(), cmpl.pos))
}

func (cmpl *compiler) getLex() *core.Lex {
	return cmpl.unit.Lexeme[len(cmpl.unit.Lexeme)-1]
}

func findVar(cmpl *compiler, token string) (*core.CmdBlock, int) {
	block := cmpl.curOwner()
	for block != nil {
		if ind, ok := block.VarNames[token]; ok {
			return block, ind
		}
		block = block.Parent
	}
	return nil, 0
}

func coError(cmpl *compiler) error {
	return cmpl.Error(cmpl.next.State)
}

func isInState(cmpl *compiler, state, shift int) bool {
	return len(cmpl.expbuf) == 0 && len(*cmpl.states) > 2 &&
		(*cmpl.states)[len(*cmpl.states)-1-shift].State == state
}

func coType(cmpl *compiler) error {
	obj, err := getType(cmpl)
	if err != nil {
		return err
	}
	cmpl.curType = obj.(*core.TypeObject)
	return nil
}

func coVarToken(cmpl *compiler, token string) error {
	if isCapital(token) {
		return cmpl.Error(ErrCapitalLetters)
	}
	if strings.IndexRune(token, '.') >= 0 {
		return cmpl.Error(ErrIdent)
	}
	if cmpl.unit.FindType(token) != nil {
		return cmpl.Error(ErrUsedName, token)
	}
	block := cmpl.curOwner()
	for block != nil {
		if _, ok := block.VarNames[token]; ok {
			return cmpl.Error(ErrUsedName, token)
		}
		block = block.Parent
	}

	block = cmpl.curOwner()
	if block.VarNames == nil {
		block.VarNames = make(map[string]int)
	}
	block.VarNames[token] = len(block.Vars)
	block.Vars = append(block.Vars, cmpl.curType)
	return nil
}

func coVar(cmpl *compiler) error {
	return coVarToken(cmpl, getToken(cmpl.getLex(), cmpl.pos))
}

func coVariadic(cmpl *compiler) error {
	block := cmpl.curOwner()
	block.Variadic = true
	obj, err := autoType(cmpl, `arr.`+cmpl.curType.GetName())
	if err != nil {
		return err
	}
	block.Vars[len(block.Vars)-1] = obj.(*core.TypeObject)
	return nil
}

func coVarType(cmpl *compiler) error {
	obj, err := getType(cmpl)
	if err != nil {
		return err
	}
	cmpl.curType = obj.(*core.TypeObject)
	return nil
}

func coVarExp(cmpl *compiler) error {
	if err := coVar(cmpl); err != nil {
		return err
	}
	tokens := cmpl.getLex().Tokens
	if len(tokens) == cmpl.pos+1 {
		return nil
	}
	if tokens[cmpl.pos+1].Type == tkAssign || tokens[cmpl.pos+1].Type == tkBitAndEq {
		if len(tokens) > cmpl.pos+2 {
			if tokens[cmpl.pos+2].Type == tkColon {
				if err := colonToLine(cmpl, cmpl.pos+2); err != nil {
					return err
				}
			}
			if tokens[cmpl.pos+2].Type == tkLCurly {
				block := cmpl.curOwner()
				cmd := core.CmdBlock{ID: core.StackInit, CmdCommon: core.CmdCommon{
					TokenID: uint32(cmpl.pos + 1)}}
				appendCmd(cmpl, &cmd)
				cmpl.owners = append(cmpl.owners, &cmd)

				appendCmd(cmpl, &core.CmdVar{Block: block, Index: len(block.Vars) - 1,
					CmdCommon: core.CmdCommon{TokenID: uint32(cmpl.pos - 1)}})

				cmpl.newPos = cmpl.pos + 2
				if err := coInitStart(cmpl); err != nil {
					return err
				}
				cmpl.dynamic = &cmState{tkLCurly, cmInit, nil, nil, 0}
				return nil
			}
		}
		coExpStart(cmpl)
		cmpl.dynamic = &cmState{tkIdent, cmExp, nil, nil, cfStay}
		//cmpl.newState = cmExp | cfStay
	}
	return nil
}

func coExpEnv(cmpl *compiler) error {
	token := getToken(cmpl.getLex(), cmpl.pos)
	getEnv := cmpl.vm.StdLib().FindObj(core.DefGetEnv) //Names[`GetEnv`]
	icmd := &core.CmdValue{Value: token,
		CmdCommon: core.CmdCommon{TokenID: uint32(cmpl.pos)},
		Result:    getEnv.Result()}
	appendExp(cmpl, &core.CmdUnary{CmdCommon: core.CmdCommon{TokenID: uint32(cmpl.pos)},
		Object: getEnv, Result: getEnv.Result(), Operand: icmd})
	return nil
}

func isInLoop(cmpl *compiler, incase bool) bool {
	for _, item := range cmpl.owners {
		if item.GetType() == core.CtStack {
			id := item.(*core.CmdBlock).ID
			if id == core.StackWhile || id == core.StackFor ||
				(incase && (id == core.StackCase || id == core.StackDefault)) {
				return true
			}
		}
	}
	return false
}

func coBreak(cmpl *compiler) error {
	if !isInLoop(cmpl, true) {
		return cmpl.Error(ErrBreak)
	}
	appendCmd(cmpl, &core.CmdCommand{CmdCommon: core.CmdCommon{TokenID: uint32(cmpl.pos)},
		ID: core.RcBreak})
	return nil
}

func coContinue(cmpl *compiler) error {
	if !isInLoop(cmpl, false) {
		return cmpl.Error(ErrContinue)
	}
	appendCmd(cmpl, &core.CmdCommand{CmdCommon: core.CmdCommon{TokenID: uint32(cmpl.pos)},
		ID: core.RcContinue})
	return nil
}
