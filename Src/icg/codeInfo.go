package icg

import (
	"container/list"
	"fmt"
	"strings"

	"WAH_prototype_go-master/Src/icg/symbolTable"
)

type SilType int

const (
	I  = 0
	C  = 1
	S  = 2
	L  = 3
	Ui = 4
	Uc = 5
	Us = 6
	Ul = 7
	F  = 8
	D  = 9
	P  = 10
	T  = 11
	Sp = 12 // string pointer
	Nt = 13 // non type
)

func (silType SilType) String() string {
	names := [...]string{
		"i", "c", "s", "l", "ui", "uc", "us", "ul", "f", "d", "p", "t", "p", ""}

	return names[silType]
}

type Opcode int

const (
	Nop   = 0
	Pop   = 1
	Pop2  = 2
	Dup   = 3
	Dup2  = 4
	Swap  = 5
	Swap2 = 6
	Ldc   = 7
	Lod   = 8
	Ldi   = 9
	Lda   = 10
	Ldftn = 11
	Str   = 12
	Sti   = 13

	//Arithmetic operation
	Add  = 14
	Sub  = 15
	Mul  = 16
	Div  = 17
	Mod  = 18
	Neg  = 19
	Eq   = 20
	Ne   = 21
	Ge   = 22
	Gt   = 23
	Le   = 24
	Lt   = 25
	Band = 26
	Bor  = 27
	Bxor = 28
	Bcom = 29
	Shl  = 30
	Shr  = 31
	Ushr = 32
	And  = 33
	Or   = 34
	Not  = 35
	Inc  = 36
	Dec  = 37

	//control op
	Label = 38
	Tjp   = 39
	Fjp   = 40
	Ujp   = 41
	Ret   = 42
	Retv  = 43
	Retmv = 44

	Proc   = 45
	Ldp    = 46
	Call   = 47
	Calli  = 48
	Calls  = 49
	Callv  = 50
	Procva = 51
	End    = 52

	//type conversion(Arith)
	Cvc  = 53
	Cvs  = 54
	Cvi  = 55
	Cvui = 56
	Cvl  = 57
	Cvul = 58
	Cvp  = 59
	Cvf  = 60
	Cvd  = 61
)

func (opcode Opcode) String() string {
	names := [...]string{
		"nop", "pop", "pop2", "dup", "dup2", "swap", "swap2", "ldc", "lod", "ldi", "lda", "ldftn", "str", "sti",
		"add", "sub", "mul", "div", "mod", "neg", "eq", "ne", "ge", "gt", "le", "lt", "band", "bor", "bxor",
		"bcom", "shl", "shr", "ushr", "and", "or", "not", "inc", "dec", "label", "tjp", "fjp", "ujp", "ret", "retv", "retmv", "proc",
		"ldp", "call", "calli", "calls", "callv", "procva", "end", "cvc", "cvs", "cvi", "cvui", "cvl", "cvul", "cvp", "cvf", "cvd"}

	return names[opcode]
}
func (opcode Opcode) PushParameterNum() int {
	pushNum := [...]int{
		0, 0, 0, 1, 2, 0, 0, 1, 1, 1, 1, 1, 0, 0,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	}

	return pushNum[opcode]
}
func (opcode Opcode) PopParameterNum() int {
	popNum := [...]int{
		0, 1, 2, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 2,
		2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2,
		1, 2, 2, 2, 2, 2, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 0,
		0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	}

	return popNum[opcode]
}

// CodeInfo ...
type CodeInfo interface {
	String() string
	ParentStmt() Statements
	Type() SilType
	SetLine(l int)
	GetLine() int
	Opcode() Opcode
	GetPushParameterNum() int
	GetPopParameterNum() int
	GetSourceLine() int
}

// StackOpcode ...
type StackOpcode struct {
	_opcode        Opcode
	_type          SilType
	_params        *list.List
	_parentStmt    Statements
	_lineNum       int
	_isAlias       bool
	_pushParamNum  int
	_popParamNum   int
	_isReceiver    bool
	_sourceLineNum int
}

func (code *StackOpcode) GetPushParameterNum() int {
	return code._pushParamNum
}
func (code *StackOpcode) GetPopParameterNum() int {
	return code._popParamNum
}
func (code *StackOpcode) IsAlias() bool {
	return code._isAlias
}
func (code *StackOpcode) IsReceiver() bool {
	return code._isReceiver
}
func (code *StackOpcode) Opcode() Opcode {
	return code._opcode
}
func (code *StackOpcode) Type() SilType {
	return code._type
}
func (code *StackOpcode) Params() *list.List {
	return code._params
}
func (code *StackOpcode) Init() {

	code._pushParamNum = code._opcode.PushParameterNum()
	code._popParamNum = code._opcode.PopParameterNum()
}
func (code *StackOpcode) SetLine(l int) {
	code._lineNum = l
}
func (code *StackOpcode) GetLine() int {
	return code._lineNum
}
func (code StackOpcode) String() string {
	builder := strings.Builder{}
	builder.WriteString(code._opcode.String())
	if code._type != -1 {
		str := code._type.String()
		if len(str) > 0 {
			builder.WriteString("." + str)
		}
	}

	for temp := code._params.Front(); temp != nil; temp = temp.Next() {
		str := fmt.Sprint(temp.Value)
		builder.WriteString("\t" + str)
	}
	//builder.WriteString("\t (" + code._parentStmt.String() + ")")
	return builder.String()
}
func (code *StackOpcode) ParentStmt() Statements {
	return code._parentStmt
}
func (code *StackOpcode) GetSourceLine() int {
	return code._sourceLineNum
}

// ArithmeticOpcode ...
type ArithmeticOpcode struct {
	_opcode        Opcode
	_type          SilType
	_parentStmt    Statements
	_lineNum       int
	_pushParamNum  int
	_popParamNum   int
	_sourceLineNum int
}

func (code *ArithmeticOpcode) GetPushParameterNum() int {
	return code._pushParamNum
}
func (code *ArithmeticOpcode) Init() {

	code._pushParamNum = code._opcode.PushParameterNum()
	code._popParamNum = code._opcode.PopParameterNum()
}
func (code *ArithmeticOpcode) GetPopParameterNum() int {
	return code._popParamNum
}
func (code *ArithmeticOpcode) GetLine() int {
	return code._lineNum
}
func (code *ArithmeticOpcode) Opcode() Opcode {
	return code._opcode
}
func (code *ArithmeticOpcode) Type() SilType {
	return code._type
}
func (code *ArithmeticOpcode) ParentStmt() Statements {
	return code._parentStmt
}
func (code *ArithmeticOpcode) SetLine(l int) {
	code._lineNum = l
}
func (code *ArithmeticOpcode) String() string {
	builder := strings.Builder{}
	builder.WriteString(code._opcode.String())
	if code._type != -1 {
		str := code._type.String()
		if len(str) > 0 {
			builder.WriteString("." + str)
		}

	}
	return builder.String()
}
func (code *ArithmeticOpcode) GetSourceLine() int {
	return code._sourceLineNum
}

// ControlOpcode ...
type ControlOpcode struct {
	_opcode        Opcode
	_type          SilType
	_params        *list.List
	_parentStmt    Statements
	_lineNum       int
	_pushParamNum  int
	_popParamNum   int
	_sourceLineNum int
}

func (code *ControlOpcode) GetSourceLine() int {
	return code._sourceLineNum
}
func (code *ControlOpcode) Init() {
	//code._params = list.New()
	code._pushParamNum = code._opcode.PushParameterNum()
	code._popParamNum = code._opcode.PopParameterNum()
}
func (code *ControlOpcode) GetPushParameterNum() int {
	return code._pushParamNum
}
func (code *ControlOpcode) GetPopParameterNum() int {
	return code._popParamNum
}
func (code *ControlOpcode) GetLine() int {
	return code._lineNum
}
func (code *ControlOpcode) ParentStmt() Statements {
	return code._parentStmt
}
func (code *ControlOpcode) Opcode() Opcode {
	return code._opcode
}
func (code *ControlOpcode) Type() SilType {
	return code._type
}
func (code *ControlOpcode) Params() *list.List {
	return code._params
}

func (code *ControlOpcode) SetLine(l int) {
	code._lineNum = l
}
func (code *ControlOpcode) String() string {
	builder := strings.Builder{}
	builder.WriteString(code._opcode.String())
	if code._type != -1 {

		str := code._type.String()
		if len(str) > 0 {
			builder.WriteString("." + str)
		}

	}

	if code._params != nil {
		for temp := code._params.Front(); temp != nil; temp = temp.Next() {

			var str string = fmt.Sprint(temp.Value)
			if (Tjp <= code._opcode && code._opcode <= Ujp) || code._opcode == Label {
				str = "##" + fmt.Sprint(temp.Value)
			} else if Call == code._opcode {
				str = "&" + fmt.Sprint(temp.Value)
			}

			builder.WriteString("\t" + str)
		}
	}

	return builder.String()
}

// SILTable ...
type SILTable struct {
	_FunctionCodeTable map[int][]CodeInfo
	_Mfkey             int
	_Pool              *symbolTable.StringPool
}

func (tble *SILTable) Init(pool *symbolTable.StringPool) {
	tble._FunctionCodeTable = make(map[int][]CodeInfo)
	tble._Pool = pool
}

func (tble *SILTable) Insert(fKey int, codeList []CodeInfo) {
	if tble._FunctionCodeTable == nil {
		tble._FunctionCodeTable = make(map[int][]CodeInfo)
	}
	tble._FunctionCodeTable[fKey] = codeList

	if tble._Pool.LookupSymbolName(fKey) == "main" {
		tble._Mfkey = fKey
	}
}
func (tble *SILTable) IsExist(fKey int) bool {
	if len := len(tble._FunctionCodeTable[fKey]); len > 0 {
		return true
	}
	return false
}
func (tble *SILTable) FunctionCodeTable() map[int][]CodeInfo {
	return tble._FunctionCodeTable
}
func (tble *SILTable) StringPool() *symbolTable.StringPool {
	return tble._Pool
}
func (tble *SILTable) MainKey() int {
	return tble._Mfkey
}
