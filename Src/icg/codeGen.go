package icg

import (
	"bytes"
	"container/list"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strconv"

	"WAH_prototype_go-master/Src/icg/symbolTable"
)

//Statements ...
type Statements int

const (
	ExprStmt = iota
	SendStmt
	IncDecStmt
	AssignStmt
	GoStmt
	DeferStmt
	ReturnStmt
	BranchStmt
	IfStmt
	CaseClause
	SwitchStmt
	TypeSwitchStmt
	CommClause
	SelectStmt
	ForStmt
	RangeStmt
	ValueSpec
	TypeSpec
	FuncDecl
	NotState
	EndState
)

func (s Statements) String() string {
	return [...]string{"ExprStmt", "SendStmt", "IncDecStmt", "AssignStmt", "GoStmt", "DeferStmt", "ReturnStmt", "BranchStmt", "IfStmt", "CaseClause", "SwitchStmt",
		"TypeSwitchStmt", "Commclause", "SelectStmt", "ForStmt", "RangeStmt", "ValueSpec", "TypeSpec", "FuncDecl", "Not", "EndState"}[s]
}

//CodeGen ...
func CodeGen(f *ast.File, fs *token.FileSet, info *types.Info, pool *symbolTable.StringPool,
	symTble *symbolTable.BlockSymbolTable, litTable *symbolTable.LiteralTable) *SILTable {
	var icg *ICG = &ICG{}
	icg.Init(fs, pool, info, symTble, litTable)

	silTable := icg.Visit(f)
	for _, codeInfoList := range silTable.FunctionCodeTable() {
		for i, sil := range codeInfoList {
			sil.SetLine(i)
		}

	}
	return silTable
}

//ICG ...
type ICG struct {
	_isGlobalSym   bool
	_codeInfoList  []CodeInfo
	_silTable      SILTable
	_isFirstBlock  bool
	_isReturnField bool

	_labelCount      int
	_loopStartLabel  int
	_loopEndLabel    int
	_isFunction      bool
	_funcDeclVarSize int
	_isReturn        bool
	_assignIndex     int
	_info            *types.Info
	_stringPool      *symbolTable.StringPool
	_symTble         *symbolTable.BlockSymbolTable
	_litTble         *symbolTable.LiteralTable
	_blockNum        int
	_isCallExpr      bool
	_fs              *token.FileSet
	_codeState       Statements
	_tmpSym          int
	_structSize      int
	ast.Visitor
}

func (icg *ICG) FindSymbolOffset(symName string) (res *symbolTable.SymbolInfo, ok bool) {
	symIndx := icg._stringPool.LookupSymbolNumber(symName)
	ok = false
	if symIndx != -1 {
		blockNum := icg._blockNum
		for blockNum != -1 {
			symTble, sok := icg._symTble.GetTable(blockNum)
			if !sok {
				fmt.Errorf("error")
			}
			symInfo, isGetOffset := symTble.GetOffset(symIndx)
			if !isGetOffset {
				blockNum = symTble.ParentBlock
			} else {
				res = symInfo
				ok = true
				break
			}

		}
	}

	return

}

// Init ...
func (icg *ICG) Init(fs *token.FileSet, pool *symbolTable.StringPool, info *types.Info, symtble *symbolTable.BlockSymbolTable, litTable *symbolTable.LiteralTable) {
	icg._isGlobalSym = false
	icg._isFunction = false
	icg._isReturn = false
	icg._isFirstBlock = false
	icg._isReturnField = false
	icg._stringPool = pool
	icg._silTable.Init(pool)
	icg._isCallExpr = false
	icg._info = info
	icg._symTble = symtble
	icg._blockNum = 0
	icg._litTble = litTable
	icg._labelCount = 0

	icg._loopEndLabel = -1
	icg._loopStartLabel = -1
	icg._fs = fs
	icg._tmpSym = 0
}

func TypeToSilType(goType types.Type) (kind SilType) {
	kind = Nt
	switch n := goType.(type) {
	case *types.Basic:
		switch n.Kind() {
		case types.Bool:
			kind = C
		case types.Float32:
			kind = F
		case types.Float64:
			kind = D
		case types.Int:
			kind = I
		case types.Int16:
			kind = S
		case types.Int32:
			kind = I
		case types.Int64:
			kind = D
		case types.Int8:
			kind = C
		case types.String:
			kind = Sp
		case types.Uint:
			kind = Ui
		case types.Uint16:
			kind = Us
		case types.Uint32:
			kind = Ui
		case types.Uint64:
			kind = Ul
		case types.Uint8:
			kind = Uc
		case types.Uintptr:
			kind = P
		default:
			kind = Nt
		}
	case *types.Array:
		kind = P
	case *types.Pointer:
		kind = P
	case *types.Map:
		kind = P
	case *types.Slice:
		kind = P
	case *types.Named:
		kind = TypeToSilType(n.Underlying())
	case *types.Struct:
		kind = T
	case *types.Interface:
		kind = P
	default:
		kind = Nt
	}
	return
}

//TypeToByte ...
func TypeToByte(kind SilType) int {
	var res = 0
	switch kind {
	case I:
		res = 4
	case C:
		res = 1
	case S:
		res = 2
	case Ui:
		res = 4
	case Uc:
		res = 1
	case Us:
		res = 2
	case Ul:
		res = 4
	case F:
		res = 4
	case D:
		res = 8
	case P:
		res = 4
	case Sp:
		res = 4
	case Nt:
		res = 0
	case T:
		res = 12

	}
	return res
}

func (icg *ICG) visitExprList(list []ast.Expr) {
	for _, x := range list {
		icg.Visit(x)
	}
}
func (icg *ICG) visitDeclList(list []ast.Decl) {
	for _, x := range list {
		icg.Visit(x)
	}
}
func (icg *ICG) visitStmtList(list []ast.Stmt) {
	for _, x := range list {
		icg.Visit(x)
	}
}

//Visit ...
func (icg *ICG) Visit(node ast.Node) *SILTable {
	if node == nil {
		return nil
	}

	var position token.Position
	if node.Pos() != token.NoPos {
		position = icg._fs.Position(node.Pos())
	}

	// walk children
	// (the order of the cases matches the order
	// of the corresponding node types in ast.go)
	switch n := node.(type) {
	// Comments and fields
	case *ast.Comment:
		// nothing to do

	case *ast.CommentGroup:
	case *ast.Field:
		field := node.(*ast.Field)
		if !icg._isReturnField {
			for _, ident := range field.Names {
				symType := icg._info.Types[field.Type].Type

				opcode := &StackOpcode{}

				opcode._opcode = Str
				opcode._type = TypeToSilType(symType)
				opcode._parentStmt = icg._codeState
				opcode._isAlias = false
				opcode._params = list.New()
				opcode._sourceLineNum = position.Line
				opcode.Init()

				oInfo, _ := icg.FindSymbolOffset(ident.Name)

				opcode._params.PushBack(oInfo.Base)
				if oInfo.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(oInfo.Offset)
				}

				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
				icg._funcDeclVarSize += oInfo.Width

			}
		} else {
			if field.Names != nil {
				// param return
				for _, ident := range field.Names {
					oInfo, _ := icg.FindSymbolOffset(ident.Name)
					icg._funcDeclVarSize += oInfo.Width
				}
			}

			icg._isReturnField = false
		}

	case *ast.FieldList:

	// Expressions
	case *ast.Ident:
	case *ast.BadExpr:
	case *ast.BasicLit:
		switch n.Kind {
		case token.INT:
			opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			val, _ := strconv.Atoi(n.Value)
			opcode._params.PushBack(val)

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.FLOAT:
			opcode := &StackOpcode{Ldc, D, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			val, _ := strconv.ParseFloat(n.Value, 64)
			opcode._params.PushBack(val)

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.STRING:
			opcode := &StackOpcode{Lda, Sp, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			address, ok := icg._litTble.GetLiteralAddress(n.Value)
			if !ok {
				fmt.Errorf("error not exist literal")
			}
			opcode._params.PushBack(0)
			opcode._params.PushBack("@" + strconv.Itoa(address))
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		default:
			opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(0)

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}

	case *ast.Ellipsis:

	case *ast.FuncLit:

	case *ast.CompositeLit:
		comlit := node.(*ast.CompositeLit)
		litType := icg._info.Types[comlit.Type].Type

		if named, ok := litType.(*types.Named); ok {
			if _, stOk := named.Underlying().(*types.Struct); stOk {
				//isFirst := true
				tmpSym := strconv.Itoa(icg._tmpSym)
				icg._tmpSym++

				tmpInfo, _ := icg.FindSymbolOffset(tmpSym)

				typeName := comlit.Type.(*ast.Ident).Name
				totalSize := 0
				for _, width := range tmpInfo.FieldWidth {
					totalSize += width
				}
				if icg._isFunction {
					icg._funcDeclVarSize += totalSize
				}
				elemWidth := 0

				// element 초기화 문
				if comlit.Elts != nil {
					for index, elt := range comlit.Elts {

						//몇번째 element를 초기화할건지 element 주소 계산
						// if isFirst {
						// 	isFirst = false
						// 	op := &StackOpcode{Ldc, types.Uintptr, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						// 	op._params.PushBack(0)
						// 	icg._codeInfoList = append(icg._codeInfoList, op)

						// 	aop := &ArithmeticOpcode{Add, types.Uintptr}
						// 	icg._codeInfoList = append(icg._codeInfoList, aop)
						// } else if !isFirst {
						ldaOp := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						ldaOp._isReceiver = tmpInfo.IsReceiver
						ldaOp._params.PushBack(tmpInfo.Base)
						if tmpInfo.Base == 0 {
							ldaOp._params.PushBack("$" + tmpSym)
						} else {
							ldaOp._params.PushBack(tmpInfo.Offset)
						}
						ldaOp.Init()
						icg._codeInfoList = append(icg._codeInfoList, ldaOp)

						op := &StackOpcode{Ldc, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						op._params.PushBack(elemWidth)
						op.Init()
						icg._codeInfoList = append(icg._codeInfoList, op)

						aop := &ArithmeticOpcode{Add, P, icg._codeState, -1, -1, -1, position.Line}
						aop.Init()
						icg._codeInfoList = append(icg._codeInfoList, aop)
						//}
						eltType := icg._info.Types[elt].Type

						isKeyVar := false
						if keyVar, ok := elt.(*ast.KeyValueExpr); ok {
							eltType = icg._info.Types[keyVar.Value].Type
							isKeyVar = true
						}

						typeInfo, _ := icg.FindSymbolOffset(typeName)
						elemWidth += typeInfo.FieldWidth[index]
						// ident 일땐 lod str 해줘야함
						if !isKeyVar {
							if ident, isIdent := elt.(*ast.Ident); isIdent {
								// sym 정보
								symInfo, _ := icg.FindSymbolOffset(ident.Name)
								iType := icg._info.Types[ident].Type

								// user type
								if userType, isNamed := iType.(*types.Named); isNamed {
									//struct
									if _, isStruct := userType.Underlying().(*types.Struct); isStruct {
										opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

										opcode._params.PushBack(symInfo.Base)
										if symInfo.Base == 0 {
											opcode._params.PushBack("$" + ident.Name)
										} else {
											opcode._params.PushBack(symInfo.Offset)
										}
										opcode._params.PushBack(symInfo.Width)
										opcode.Init()
										opcode._pushParamNum = 3
										opcode._popParamNum = 0
										icg._codeInfoList = append(icg._codeInfoList, opcode)

										opcode = &StackOpcode{Sti, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

										opcode.Init()
										opcode._pushParamNum = 0
										opcode._popParamNum = 4
										icg._codeInfoList = append(icg._codeInfoList, opcode)

									}
								} else if basicType, isBasic := iType.(*types.Basic); isBasic { // basic type
									opcode := &StackOpcode{Lod, TypeToSilType(basicType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

									opcode._params.PushBack(symInfo.Base)
									if symInfo.Base == 0 {
										opcode._params.PushBack("$" + ident.Name)
									} else {
										opcode._params.PushBack(symInfo.Offset)
									}
									opcode.Init()
									icg._codeInfoList = append(icg._codeInfoList, opcode)

									opcode = &StackOpcode{Sti, TypeToSilType(basicType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
									opcode.Init()
									icg._codeInfoList = append(icg._codeInfoList, opcode)
								}
								continue
							}
						} else {
							keyVar := elt.(*ast.KeyValueExpr)
							if ident, isIdent := keyVar.Value.(*ast.Ident); isIdent {
								// sym 정보
								symInfo, _ := icg.FindSymbolOffset(ident.Name)
								iType := icg._info.Types[ident].Type

								// user type
								if userType, isNamed := iType.(*types.Named); isNamed {
									//struct
									if _, isStruct := userType.Underlying().(*types.Struct); isStruct {
										opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

										opcode._params.PushBack(symInfo.Base)
										if symInfo.Base == 0 {
											opcode._params.PushBack("$" + ident.Name)
										} else {
											opcode._params.PushBack(symInfo.Offset)
										}
										opcode._params.PushBack(symInfo.Width)
										opcode.Init()
										opcode._pushParamNum = 3
										opcode._popParamNum = 0
										icg._codeInfoList = append(icg._codeInfoList, opcode)

										opcode = &StackOpcode{Sti, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

										opcode.Init()
										opcode._pushParamNum = 0
										opcode._popParamNum = 4
										icg._codeInfoList = append(icg._codeInfoList, opcode)

									}
								} else if basicType, isBasic := iType.(*types.Basic); isBasic { // basic type
									opcode := &StackOpcode{Lod, TypeToSilType(basicType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

									opcode._params.PushBack(symInfo.Base)
									if symInfo.Base == 0 {
										opcode._params.PushBack("$" + ident.Name)
									} else {
										opcode._params.PushBack(symInfo.Offset)
									}
									opcode.Init()
									icg._codeInfoList = append(icg._codeInfoList, opcode)
									opcode = &StackOpcode{Sti, TypeToSilType(basicType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
									opcode.Init()
									icg._codeInfoList = append(icg._codeInfoList, opcode)
								}
								continue
							}
						}

						if userType, isNamed := eltType.(*types.Named); isNamed {
							if _, isStruct := userType.Underlying().(*types.Struct); isStruct {

								// elt의 sym
								tmpSym := strconv.Itoa(icg._tmpSym)
								symInfo, _ := icg.FindSymbolOffset(tmpSym)
								if isKeyVar {
									icg.Visit(elt.(*ast.KeyValueExpr).Value)
								} else {
									icg.Visit(elt)
								}

								opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								opcode._params.PushBack(symInfo.Base)
								if symInfo.Base == 0 {
									opcode._params.PushBack("$" + tmpSym)
								} else {
									opcode._params.PushBack(symInfo.Offset)
								}
								opcode._params.PushBack(symInfo.Width)
								opcode.Init()
								opcode._pushParamNum = 3
								opcode._popParamNum = 0
								icg._codeInfoList = append(icg._codeInfoList, opcode)

								opcode = &StackOpcode{Sti, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

								opcode.Init()
								opcode._pushParamNum = 0
								opcode._popParamNum = 4
								icg._codeInfoList = append(icg._codeInfoList, opcode)
							}
						} else if basicType, isBasic := eltType.(*types.Basic); isBasic {
							if isKeyVar {
								icg.Visit(elt.(*ast.KeyValueExpr).Value)
							} else {
								icg.Visit(elt)
							}
							opcode := &StackOpcode{Sti, TypeToSilType(basicType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)

						}
					}

				}
				// opcode := &StackOpcode{Lod, types.NewStruct(nil,nil), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				// opcode._params.PushBack(tmpInfo.Base)

				// if tmpInfo.Base == 0 {
				// 	opcode._params.PushBack("$" + tmpSym)
				// } else {
				// 	opcode._params.PushBack(tmpInfo.Offset)
				// }
				// opcode.Init()

			}
		} else if _, ok := litType.(*types.Array); ok {

		}
		// ...

	case *ast.ParenExpr:

	case *ast.SelectorExpr:

		info := icg._info.Types[n.X].Type

		if _, ok := n.X.(*ast.SelectorExpr); ok {
			//selector 처리

			icg.Visit(n.X)

			if usr, ok := info.(*types.Named); ok {
				var typeInfo types.Type
				for {
					usrType := usr.Underlying()
					_, ok := usrType.(*types.Named)
					if !ok {
						typeInfo = usrType
						break
					}
				}

				if structType, ok := typeInfo.(*types.Struct); ok {

					var stInfo *symbolTable.SymbolInfo
					//var stName = ""
					if basic, ok := n.X.(*ast.Ident); ok {
						stInfo, _ = icg.FindSymbolOffset(basic.Name)
						//stName = basic.Name
					} else if sel, ok := n.X.(*ast.SelectorExpr); ok {
						stInfo, _ = icg.FindSymbolOffset(sel.Sel.Name)
						//stName = sel.Sel.Name
					}

					location := 0
					totalWidth := 0

					for _, width := range stInfo.FieldWidth {
						totalWidth += width
					}

					icg._structSize = totalWidth
					ldiOpcode := &StackOpcode{Ldi, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					ldiOpcode.Params().PushBack(totalWidth)
					ldiOpcode.Init()
					ldiOpcode._pushParamNum = 3
					ldiOpcode._popParamNum = 1
					icg._codeInfoList = append(icg._codeInfoList, ldiOpcode)

					popOpcode := &StackOpcode{Pop2, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					popOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, popOpcode)

					for index := 0; index < structType.NumFields(); index++ {

						if n.Sel.Name == structType.Field(index).Name() {
							break
						}
						location += stInfo.FieldWidth[index]
					}
					opcode := &StackOpcode{Ldc, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(location)
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)

					aOp := &ArithmeticOpcode{Add, P, icg._codeState, -1, -1, -1, position.Line}
					aOp.Init()
					icg._codeInfoList = append(icg._codeInfoList, aOp)
				}
			}

		} else {
			if pointer, ok := info.(*types.Pointer); ok {
				{
					if usr, ok := pointer.Elem().(*types.Named); ok {
						var typeInfo types.Type
						usrType := usr.Underlying()
						for {
							usrType = usrType.Underlying()
							_, ok := usrType.(*types.Named)
							if !ok {
								typeInfo = usrType
								break
							}
						}

						if structType, ok := typeInfo.(*types.Struct); ok {
							var stInfo *symbolTable.SymbolInfo
							var stName = ""
							if basic, ok := n.X.(*ast.Ident); ok {
								stInfo, _ = icg.FindSymbolOffset(basic.Name)
								stName = basic.Name
							} else if sel, ok := n.X.(*ast.SelectorExpr); ok {
								stInfo, _ = icg.FindSymbolOffset(sel.Sel.Name)
								stName = sel.Sel.Name
							}
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(stInfo.Base)
							if stInfo.Base == 0 {
								opcode._params.PushBack("$" + stName)
							} else {
								opcode._params.PushBack(stInfo.Offset)
							}

							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)

							totalWidth := 0
							for _, width := range stInfo.FieldWidth {
								totalWidth += width
							}

							icg._structSize = totalWidth
							ldiOpcode := &StackOpcode{Ldi, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							ldiOpcode.Params().PushBack(totalWidth)
							ldiOpcode.Init()
							ldiOpcode._pushParamNum = 3
							ldiOpcode._popParamNum = 1
							icg._codeInfoList = append(icg._codeInfoList, ldiOpcode)

							opcode = &StackOpcode{Pop2, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
							location := 0
							for index := 0; index < structType.NumFields(); index++ {

								if n.Sel.Name == structType.Field(index).Name() {
									break
								}
								location += stInfo.FieldWidth[index]
							}
							opcode = &StackOpcode{Ldc, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(location)
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
							aOp := &ArithmeticOpcode{Add, P, icg._codeState, -1, -1, -1, position.Line}
							aOp.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOp)
						}
					}
				}
			}
			if usr, ok := info.(*types.Named); ok {
				var typeInfo types.Type
				for {
					usrType := usr.Underlying()
					_, ok := usrType.(*types.Named)
					if !ok {
						typeInfo = usrType
						break
					}
				}

				if structType, ok := typeInfo.(*types.Struct); ok {
					var stInfo *symbolTable.SymbolInfo
					var stName = ""
					if basic, ok := n.X.(*ast.Ident); ok {
						stInfo, _ = icg.FindSymbolOffset(basic.Name)
						stName = basic.Name
					} else if sel, ok := n.X.(*ast.SelectorExpr); ok {
						stInfo, _ = icg.FindSymbolOffset(sel.Sel.Name)
						stName = sel.Sel.Name
					}
					opcode := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._isReceiver = stInfo.IsReceiver
					opcode._params.PushBack(stInfo.Base)
					if stInfo.Base == 0 {
						opcode._params.PushBack("$" + stName)
					} else {
						opcode._params.PushBack(stInfo.Offset)
					}

					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
					location := 0
					for index := 0; index < structType.NumFields(); index++ {

						if n.Sel.Name == structType.Field(index).Name() {
							break
						}
						location += stInfo.FieldWidth[index]
					}
					opcode = &StackOpcode{Ldc, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(location)
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
					aOp := &ArithmeticOpcode{Add, P, icg._codeState, -1, -1, -1, position.Line}
					aOp.Init()
					icg._codeInfoList = append(icg._codeInfoList, aOp)
				}
			}

			// X is Package ( package.Sel)
			if info == nil {
				typeInfo := icg._info.Types[n.Sel].Type

				fmt.Println(typeInfo)
			}

		}

	case *ast.IndexExpr:

		if ident, ok := n.X.(*ast.Ident); ok {
			symInfo, _ := icg.FindSymbolOffset(ident.Name)
			opcode := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._isReceiver = symInfo.IsReceiver
			opcode._params.PushBack(symInfo.Base)
			if symInfo.Base == 0 {
				opcode._params.PushBack("$" + ident.Name)
			} else {
				opcode._params.PushBack(symInfo.Offset)
			}

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)

		} else {
			typeInfo := icg._info.Types[n.X].Type
			fmt.Println(typeInfo)
			icg.Visit(n.X)
		}

		if ident, ok := n.Index.(*ast.Ident); ok {
			symInfo, _ := icg.FindSymbolOffset(ident.Name)
			opcode := &StackOpcode{Lod, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(symInfo.Base)
			if symInfo.Base == 0 {
				opcode._params.PushBack("$" + ident.Name)
			} else {
				opcode._params.PushBack(symInfo.Offset)
			}

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
			opcode = &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(4)
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)

			mulOp := &ArithmeticOpcode{Mul, I, icg._codeState, -1, -1, -1, position.Line}
			mulOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, mulOp)
		} else if basicLit, ok := n.Index.(*ast.BasicLit); ok {
			val, _ := strconv.Atoi(basicLit.Value)
			val *= 4
			opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(val)
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}

		opcode := &ArithmeticOpcode{Cvi, Ui, icg._codeState, -1, -1, -1, position.Line}
		opcode.Init()
		icg._codeInfoList = append(icg._codeInfoList, opcode)
		opcode = &ArithmeticOpcode{Cvui, P, icg._codeState, -1, -1, -1, position.Line}
		opcode.Init()
		icg._codeInfoList = append(icg._codeInfoList, opcode)

		opcode = &ArithmeticOpcode{Add, P, icg._codeState, -1, -1, -1, position.Line}
		opcode.Init()
		icg._codeInfoList = append(icg._codeInfoList, opcode)

		targetType := icg._info.Types[n].Type
		//n.X

		if basic, ok := targetType.(*types.Basic); ok {
			opcode := &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		} else if usr, ok := targetType.(*types.Named); ok {
			var typeInfo types.Type
			for {
				usrType := usr.Underlying()
				_, ok := usrType.(*types.Named)
				if !ok {
					typeInfo = usrType
					break
				}
			}

			if _, ok := typeInfo.(*types.Struct); ok {
				opcode := &StackOpcode{Ldi, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

				opcode.Init()
				opcode._pushParamNum = 3
				opcode._popParamNum = 1
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else if usrBasic, ok := typeInfo.(*types.Basic); ok {
				opcode := &StackOpcode{Ldi, TypeToSilType(usrBasic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else {
				opcode := &StackOpcode{Ldi, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}
		}
	case *ast.SliceExpr:

	case *ast.TypeAssertExpr:

	case *ast.CallExpr:
		icg._isCallExpr = true

		opcode := &ControlOpcode{Ldp, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		opcode.Init()
		icg._codeInfoList = append(icg._codeInfoList, opcode)

		if n.Args != nil {
			for _, arg := range n.Args {
				icg.Visit(arg)
				typeInfo := icg._info.Types[arg].Type

				if _, ok := arg.(*ast.IndexExpr); ok {

					continue
				}

				if ident, ok := arg.(*ast.Ident); ok {
					//typeInfo := icg._info.Types[arg].Type
					symInfo, _ := icg.FindSymbolOffset(ident.Name)

					// basic type
					if basic, ok := typeInfo.(*types.Basic); ok {
						if ident.Name != "nil" {
							opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

							opcode._params.PushBack(symInfo.Base)

							if symInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(symInfo.Offset)
							}

							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						} else {
							opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(0)
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)

						}
					} else if usr, ok := typeInfo.(*types.Named); ok {
						if _, ok := usr.Underlying().(*types.Struct); ok {
							opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

							opcode._params.PushBack(symInfo.Base)

							if symInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(symInfo.Offset)
							}
							opcode._params.PushBack(symInfo.Width)
							opcode.Init()
							opcode._pushParamNum = 3
							opcode._popParamNum = 0
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						} else if basic, ok := usr.Underlying().(*types.Basic); ok {
							if ident.Name != "nil" {
								opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

								opcode._params.PushBack(symInfo.Base)

								if symInfo.Base == 0 {
									opcode._params.PushBack("$" + ident.Name)
								} else {
									opcode._params.PushBack(symInfo.Offset)
								}

								opcode.Init()
								icg._codeInfoList = append(icg._codeInfoList, opcode)
							} else {
								opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								opcode._params.PushBack(0)
								opcode.Init()
								icg._codeInfoList = append(icg._codeInfoList, opcode)

							}
						} else {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

							opcode._params.PushBack(symInfo.Base)

							if symInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(symInfo.Offset)
							}

							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					} else {
						opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}

						opcode._params.PushBack(symInfo.Base)

						if symInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(symInfo.Offset)
						}

						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}

				if _, ok := arg.(*ast.SelectorExpr); ok {
					if basic, ok := typeInfo.(*types.Basic); ok {
						opcode := &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
					if usr, ok := typeInfo.(*types.Named); ok {
						var info types.Type
						for {
							usrType := usr.Underlying()
							_, ok := usrType.(*types.Named)
							if !ok {
								typeInfo = usrType
								break
							}
						}

						if _, ok := info.(*types.Struct); ok {
							opcode := &StackOpcode{Ldi, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode.Params().PushBack(icg._structSize)
							opcode.Init()
							opcode._pushParamNum = 3
							opcode._popParamNum = 1
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				}

			}
		}

		callOp := &ControlOpcode{Call, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		callOp._params.PushBack(NodeString(icg._fs, n.Fun))

		callOp.Init()

		//fmt.Println(icg._info.Types[n.Fun].Type.(*types.Signature))
		if sig, ok := icg._info.Types[n.Fun].Type.(*types.Signature); ok {
			callOp._pushParamNum = sig.Results().Len()
		} else {
			callOp._pushParamNum = 1
		}

		callOp._popParamNum = len(n.Args)
		icg._codeInfoList = append(icg._codeInfoList, callOp)

		icg._isCallExpr = false
	case *ast.StarExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			symInfo, _ := icg.FindSymbolOffset(ident.Name)
			symType := icg._info.Types[n.X].Type.(*types.Pointer).Elem()
			opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(symInfo.Base)
			if symInfo.Base == 0 {
				opcode._params.PushBack("$" + ident.Name)
			} else {
				opcode._params.PushBack(symInfo.Offset)
			}

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
			// basic type
			if basic, ok := symType.(*types.Basic); ok {
				opcode := &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else if usr, ok := symType.(*types.Named); ok { // usr type
				var typeInfo types.Type
				for {
					usrType := usr.Underlying()
					_, ok := usrType.(*types.Named)
					if !ok {
						typeInfo = usrType
						break
					}
				}
				// basic usr
				if basic, ok := typeInfo.(*types.Basic); ok {
					opcode := &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
				} else if _, ok := typeInfo.(*types.Struct); ok {
					opcode := &StackOpcode{Ldi, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode.Init()
					opcode._pushParamNum = 3
					opcode._popParamNum = 1
					icg._codeInfoList = append(icg._codeInfoList, opcode)
				} else {
					opcode := &StackOpcode{Ldi, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
				}

			} else { // etc ..
				opcode := &StackOpcode{Ldi, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}

		}
	case *ast.UnaryExpr:
		unary := node.(*ast.UnaryExpr)
		switch unary.Op {
		case token.SUB:
			icg.Visit(unary.X)
			typeInfo := icg._info.Types[unary.X].Type

			if ident, ok := unary.X.(*ast.Ident); ok {
				inf, _ := icg.FindSymbolOffset(ident.Name)

				opcode := &StackOpcode{Lod, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode._params.PushBack(inf.Base)

				if inf.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(inf.Offset)
				}

				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else if _, ok := unary.X.(*ast.SelectorExpr); ok {
				opcode := &StackOpcode{Ldi, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}
			negOp := &ArithmeticOpcode{Neg, TypeToSilType(typeInfo), icg._codeState, -1, -1, -1, position.Line}
			negOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, negOp)
		case token.ADD:
			icg.Visit(unary.X)
			typeInfo := icg._info.Types[unary.X].Type
			if ident, ok := unary.X.(*ast.Ident); ok {
				inf, _ := icg.FindSymbolOffset(ident.Name)

				opcode := &StackOpcode{Lod, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode._params.PushBack(inf.Base)

				if inf.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(inf.Offset)
				}

				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else if _, ok := unary.X.(*ast.SelectorExpr); ok {
				opcode := &StackOpcode{Ldi, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}
		case token.XOR:
			typeInfo := icg._info.Types[unary.X].Type
			kind := TypeToSilType(typeInfo)
			ldcOp := &StackOpcode{Ldc, kind, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			bcomOp := &ArithmeticOpcode{Bcom, kind, icg._codeState, -1, -1, -1, position.Line}
			ldcOp.Init()
			bcomOp.Init()
			if Ui <= kind && kind <= Ul {
				ldcOp._params.PushBack(0)

				icg._codeInfoList = append(icg._codeInfoList, ldcOp)

				icg._codeInfoList = append(icg._codeInfoList, bcomOp)
			} else if I <= kind && kind <= L {
				ldcOp._params.PushBack(-1)
				icg._codeInfoList = append(icg._codeInfoList, ldcOp)
			}
			icg.Visit(unary.X)

			if ident, ok := unary.X.(*ast.Ident); ok {
				inf, _ := icg.FindSymbolOffset(ident.Name)

				opcode := &StackOpcode{Lod, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode._params.PushBack(inf.Base)

				if inf.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(inf.Offset)
				}

				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else if _, ok := unary.X.(*ast.SelectorExpr); ok {
				opcode := &StackOpcode{Ldi, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}

			if kind == Ui {
				aOp := &ArithmeticOpcode{Cvui, I, icg._codeState, -1, -1, -1, position.Line}
				aOp.Init()
				icg._codeInfoList = append(icg._codeInfoList, aOp)
			} else if kind == Ul {
				aOp := &ArithmeticOpcode{Cvul, L, icg._codeState, -1, -1, -1, position.Line}
				aOp.Init()
				icg._codeInfoList = append(icg._codeInfoList, aOp)
			}

			modOp := &ArithmeticOpcode{Bxor, kind, icg._codeState, -1, -1, -1, position.Line}
			modOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, modOp)
		case token.NOT:
			icg.Visit(unary.X)
			typeInfo := icg._info.Types[unary.X].Type
			if ident, ok := unary.X.(*ast.Ident); ok {
				inf, _ := icg.FindSymbolOffset(ident.Name)

				opcode := &StackOpcode{Lod, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode._params.PushBack(inf.Base)

				if inf.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(inf.Offset)
				}

				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else if _, ok := unary.X.(*ast.SelectorExpr); ok {
				opcode := &StackOpcode{Ldi, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}

			notOp := &ArithmeticOpcode{Not, TypeToSilType(typeInfo), icg._codeState, -1, -1, -1, position.Line}
			notOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, notOp)
		case token.AND:
			typeInfo := icg._info.Types[n.X].Type
			tmpSym := icg._tmpSym
			icg.Visit(n.X)

			if ident, ok := n.X.(*ast.Ident); ok {
				info, _ := icg.FindSymbolOffset(ident.Name)
				opcode := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, true, -1, -1, false, position.Line}
				opcode._isReceiver = info.IsReceiver
				opcode._params.PushBack(info.Base)
				if info.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(info.Offset)
				}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else {
				if usr, ok := typeInfo.(*types.Named); ok {
					if _, ok := usr.Underlying().(*types.Struct); ok {
						info, _ := icg.FindSymbolOffset(strconv.Itoa(tmpSym))
						opcode := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, true, -1, -1, false, position.Line}
						opcode._isReceiver = info.IsReceiver
						opcode._params.PushBack(info.Base)
						opcode._params.PushBack(info.Offset)
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

					}
				} else {

				}
			}

		}

	case *ast.BinaryExpr:
		expr := node.(*ast.BinaryExpr)

		icg.Visit(expr.X)
		lType := icg._info.Types[expr.X].Type

		if ident, ok := expr.X.(*ast.Ident); ok {
			inf, _ := icg.FindSymbolOffset(ident.Name)

			opcode := &StackOpcode{Lod, TypeToSilType(lType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(inf.Base)

			if inf.Base == 0 {
				opcode._params.PushBack("$" + ident.Name)
			} else {
				opcode._params.PushBack(inf.Offset)
			}

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		} else if _, ok := expr.X.(*ast.SelectorExpr); ok {
			opcode := &StackOpcode{Ldi, TypeToSilType(lType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}

		icg.Visit(expr.Y)

		rType := icg._info.Types[expr.Y].Type
		isNil := false
		if ident, ok := expr.Y.(*ast.Ident); ok {
			var opcode *StackOpcode
			if ident.Name != "nil" {
				inf, _ := icg.FindSymbolOffset(ident.Name)

				opcode = &StackOpcode{Lod, TypeToSilType(rType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode._params.PushBack(inf.Base)

				if inf.Base == 0 {
					opcode._params.PushBack("$" + ident.Name)
				} else {
					opcode._params.PushBack(inf.Offset)
				}
			} else {
				opcode = &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
				opcode._params.PushBack(0)
				isNil = true
			}

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		} else if _, ok := expr.Y.(*ast.SelectorExpr); ok {
			opcode := &StackOpcode{Ldi, TypeToSilType(rType), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}

		switch expr.Op {
		case token.ADD:
			opcode := &ArithmeticOpcode{Add, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.SUB:
			opcode := &ArithmeticOpcode{Sub, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.MUL:
			opcode := &ArithmeticOpcode{Mul, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.QUO:
			opcode := &ArithmeticOpcode{Div, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.REM:
			opcode := &ArithmeticOpcode{Mod, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.SHL:
			opcode := &ArithmeticOpcode{Shl, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.SHR:
			opcode := &ArithmeticOpcode{Shr, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.AND:
			opcode := &ArithmeticOpcode{Band, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.AND_NOT:
			opcode := &ArithmeticOpcode{Bcom, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
			opcode = &ArithmeticOpcode{Band, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.OR:
			opcode := &ArithmeticOpcode{Bor, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.XOR:
			opcode := &ArithmeticOpcode{Bxor, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.EQL:
			if !isNil {
				opcode := &ArithmeticOpcode{Eq, TypeToSilType(rType), icg._codeState, -1, -1, -1, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else {
				opcode := &ArithmeticOpcode{Eq, I, icg._codeState, -1, -1, -1, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}
		case token.NEQ:
			if !isNil {
				opcode := &ArithmeticOpcode{Ne, TypeToSilType(rType), icg._codeState, -1, -1, -1, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			} else {
				opcode := &ArithmeticOpcode{Ne, I, icg._codeState, -1, -1, -1, position.Line}
				opcode.Init()
				icg._codeInfoList = append(icg._codeInfoList, opcode)
			}
		case token.LSS:
			opcode := &ArithmeticOpcode{Lt, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.GTR:
			opcode := &ArithmeticOpcode{Gt, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.LEQ:
			opcode := &ArithmeticOpcode{Le, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.GEQ:
			opcode := &ArithmeticOpcode{Ge, TypeToSilType(lType), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}
	case *ast.KeyValueExpr:

	// Types
	case *ast.ArrayType:

	case *ast.StructType:

	case *ast.FuncType:

	case *ast.InterfaceType:

	case *ast.MapType:

	case *ast.ChanType:

	// Statements
	case *ast.BadStmt:
		// nothing to do

	case *ast.DeclStmt:
		decl := node.(*ast.DeclStmt)
		icg.Visit(decl.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:

	case *ast.ExprStmt:
		icg._codeState = ExprStmt
		icg.Visit(n.X)
	case *ast.SendStmt:

	case *ast.IncDecStmt:
		icg._codeState = IncDecStmt
		icg.Visit(n.X)
		typeInfo := icg._info.Types[n.X].Type
		strOp := &StackOpcode{}
		if ident, ok := n.X.(*ast.Ident); ok {
			inf, _ := icg.FindSymbolOffset(ident.Name)

			opcode := &StackOpcode{Lod, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			strOp._opcode = Str
			strOp._type = TypeToSilType(typeInfo)
			strOp._sourceLineNum = position.Line
			strOp.Init()

			opcode._params.PushBack(inf.Base)
			opcode._parentStmt = icg._codeState

			strOp._params.PushBack(inf.Base)
			strOp._parentStmt = icg._codeState

			if inf.Base == 0 {
				opcode._params.PushBack("$" + ident.Name)
				strOp._params.PushBack("$" + ident.Name)
			} else {
				opcode._params.PushBack(inf.Offset)
				strOp._params.PushBack(inf.Offset)
			}

			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		} else if _, ok := n.X.(*ast.SelectorExpr); ok {
			opcode := &StackOpcode{Ldi, TypeToSilType(typeInfo), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			strOp.Init()
			strOp._opcode = Sti
			strOp._type = TypeToSilType(typeInfo)
			strOp._parentStmt = icg._codeState
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}
		switch n.Tok {
		case token.INC:
			opcode := &ArithmeticOpcode{Inc, TypeToSilType(typeInfo), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		case token.DEC:
			opcode := &ArithmeticOpcode{Dec, TypeToSilType(typeInfo), icg._codeState, -1, -1, -1, position.Line}
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}
		strOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, strOp)

	case *ast.AssignStmt:
		assign := node.(*ast.AssignStmt)
		icg._codeState = AssignStmt
		switch assign.Tok {
		case token.ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {

				icg.Visit(assign.Lhs[index])

				isSelect := false
				isIndex := false

				lType := icg._info.Types[assign.Lhs[index]].Type
				lOk := false
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, lOk = icg.FindSymbolOffset(ident.Name)

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true

				} else if _, ok := assign.Lhs[index].(*ast.IndexExpr); ok {
					isIndex = true
					icg._codeInfoList = icg._codeInfoList[0 : len(icg._codeInfoList)-1]
				}
				//lIdent := assign.Lhs[i].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				var rType types.Type
				befTmp := icg._tmpSym
				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

					// right가 심볼이 아닌데 struct 인 경우
					if !rOk {
						if stType, ok := rType.(*types.Named); ok {
							if _, sOk := stType.Underlying().(*types.Struct); sOk {

								tmpSym := strconv.Itoa(befTmp)
								tmpInfo, _ := icg.FindSymbolOffset(tmpSym)
								//elemWidth := 0

								// if lOk {
								// 	opcode := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								// 	opcode._params.PushBack(lInfo.Base)
								// 	if lInfo.Base == 0 {
								// 		opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
								// 	} else {
								// 		opcode._params.PushBack(lInfo.Offset)
								// 	}
								// 	opcode.Init()
								//icg._codeInfoList = append(icg._codeInfoList, opcode)
								// }

								opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								opcode.Params().PushBack(tmpInfo.Base)
								if tmpInfo.Base == 0 {
									opcode._params.PushBack("$" + tmpSym)
								} else {
									opcode._params.PushBack(tmpInfo.Offset)
								}

								opcode.Params().PushBack(tmpInfo.Width)
								opcode.Init()
								opcode._pushParamNum = 3
								opcode._popParamNum = 1
								icg._codeInfoList = append(icg._codeInfoList, opcode)

								if !lOk {
									stiOp := &StackOpcode{Sti, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
									stiOp.Init()
									stiOp._pushParamNum = 0
									stiOp._popParamNum = 4
									icg._codeInfoList = append(icg._codeInfoList, stiOp)
								} else {
									opcode := &StackOpcode{Str, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
									opcode._params.PushBack(lInfo.Base)
									if lInfo.Base == 0 {
										opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
									} else {
										opcode._params.PushBack(lInfo.Offset)
									}
									opcode.Init()
									opcode._pushParamNum = 0
									opcode._popParamNum = 3
									icg._codeInfoList = append(icg._codeInfoList, opcode)
								}

								continue
							}
						}
					}

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				// right var가 symbol
				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode._params.PushBack(rInfo.Width)
							opcode.Init()
							opcode._pushParamNum = 3
							opcode._popParamNum = 0
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					if !isSelect && !isIndex {
						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()

						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if !isIndex {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Str, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							opcode._pushParamNum = 0
							opcode._popParamNum = 3
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					} else {
						opcode := &StackOpcode{Sti, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()

						opcode._pushParamNum = 0
						opcode._popParamNum = 4
						icg._codeInfoList = append(icg._codeInfoList, opcode)

					}
				} else {
					opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(lInfo.Base)

					if lInfo.Base == 0 {
						opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
					} else {
						opcode._params.PushBack(lInfo.Offset)
					}
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
				}
			}
		case token.DEFINE:
			isFunctionExpr := false
			for index := range assign.Lhs {

				icg.Visit(assign.Lhs[index])

				lInfo, lOk := icg.FindSymbolOffset(assign.Lhs[index].(*ast.Ident).Name)
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				if icg._isFunction {
					icg._funcDeclVarSize += lInfo.Width
				}
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				var rType types.Type
				//var lType types.Type

				rIdent := ""
				befTmp := icg._tmpSym
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

					// right가 심볼이 아닌데 struct 인 경우
					if !rOk {
						if stType, ok := rType.(*types.Named); ok {
							if _, sOk := stType.Underlying().(*types.Struct); sOk {

								tmpSym := strconv.Itoa(befTmp)
								tmpInfo, _ := icg.FindSymbolOffset(tmpSym)
								//elemWidth := 0

								// if lOk {
								// 	opcode := &StackOpcode{Lda, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								// 	opcode._params.PushBack(lInfo.Base)
								// 	if lInfo.Base == 0 {
								// 		opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
								// 	} else {
								// 		opcode._params.PushBack(lInfo.Offset)
								// 	}
								// 	opcode.Init()
								//icg._codeInfoList = append(icg._codeInfoList, opcode)
								// }

								opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								opcode.Params().PushBack(tmpInfo.Base)
								if tmpInfo.Base == 0 {
									opcode._params.PushBack("$" + tmpSym)
								} else {
									opcode._params.PushBack(tmpInfo.Offset)
								}

								opcode.Params().PushBack(tmpInfo.Width)
								opcode.Init()
								opcode._pushParamNum = 3
								opcode._popParamNum = 0
								icg._codeInfoList = append(icg._codeInfoList, opcode)

								if !lOk {
									stiOp := &StackOpcode{Sti, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
									stiOp.Init()

									stiOp._pushParamNum = 0
									stiOp._popParamNum = 4
									icg._codeInfoList = append(icg._codeInfoList, stiOp)
								} else {
									opcode := &StackOpcode{Str, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
									opcode._params.PushBack(lInfo.Base)
									if lInfo.Base == 0 {
										opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
									} else {
										opcode._params.PushBack(lInfo.Offset)
									}
									opcode.Init()
									opcode._pushParamNum = 0
									opcode._popParamNum = 3
									icg._codeInfoList = append(icg._codeInfoList, opcode)
								}

								continue
							}
						}
					}

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
						icg.Visit(assign.Rhs[0])
					}
				}

				//right var가 ident인 경우 right var lod
				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode._params.PushBack(rInfo.Width)
							opcode.Init()
							opcode._pushParamNum = 3
							opcode._popParamNum = 0
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					} else {
						opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
				//lType = rType

				// 할당하는 부분
				if basic, ok := rType.(*types.Basic); ok {
					opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(lInfo.Base)
					if lInfo.Base == 0 {
						opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
					} else {
						opcode._params.PushBack(lInfo.Offset)
					}
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, T, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						opcode._pushParamNum = 0
						opcode._popParamNum = 3
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else {
					opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(lInfo.Base)
					if lInfo.Base == 0 {
						opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
					} else {
						opcode._params.PushBack(lInfo.Offset)
					}
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)
				}
			}
		case token.ADD_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}

				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Add, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {
						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.AND_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Band, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.AND_NOT_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}

						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Bcom, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)

					arithOpcode = &ArithmeticOpcode{Band, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.MUL_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//	lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Mul, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)

					opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(lInfo.Base)
					if !isSelect {

						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.OR_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//	lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Bor, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}

						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}

		case token.QUO_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Div, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)

					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.REM_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Mod, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.SHL_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Add, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.SHR_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {

				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//	lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Sub, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)
					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		case token.XOR_ASSIGN:
			isFunctionExpr := false
			for index := range assign.Lhs {
				icg.Visit(assign.Lhs[index])

				isSelect := false
				lType := icg._info.Types[assign.Lhs[index]].Type
				var lInfo *symbolTable.SymbolInfo
				if ident, ok := assign.Lhs[index].(*ast.Ident); ok {
					lInfo, _ = icg.FindSymbolOffset(ident.Name)
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := lType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(lInfo.Base)
							if lInfo.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(lInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}

				} else if _, ok := assign.Lhs[index].(*ast.SelectorExpr); ok {
					isSelect = true
					if basic, ok := lType.(*types.Basic); ok {
						opcode := &StackOpcode{Dup, Nt, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)

						opcode = &StackOpcode{Ldi, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}

				}
				//lIdent := assign.Lhs[index].(*ast.Ident).Name
				var rInfo *symbolTable.SymbolInfo
				var rOk = false

				//	lType := icg._info.Types[assign.Lhs[index]].Type
				var rType types.Type

				rIdent := ""
				if len(assign.Lhs) == len(assign.Rhs) {
					rType = icg._info.Types[assign.Rhs[index]].Type
					if ident, ok := assign.Rhs[index].(*ast.Ident); ok {
						rInfo, rOk = icg.FindSymbolOffset(ident.Name)
						rIdent = ident.Name
					}

					icg.Visit(assign.Rhs[index])

				} else if len(assign.Lhs) > len(assign.Rhs) {
					if !isFunctionExpr {
						rType = icg._info.Types[assign.Rhs[0]].Type

						icg.Visit(assign.Rhs[0])
						if ident, ok := assign.Rhs[0].(*ast.Ident); ok {
							rInfo, rOk = icg.FindSymbolOffset(ident.Name)
							rIdent = ident.Name
						}
						isFunctionExpr = true
					}

				}

				if rOk {
					// right expr 가 basic type
					if basic, ok := rType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(rInfo.Base)
						if rInfo.Base == 0 {
							opcode._params.PushBack("$" + rIdent)
						} else {
							opcode._params.PushBack(rInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := rType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(rInfo.Base)
							if rInfo.Base == 0 {
								opcode._params.PushBack("$" + rIdent)
							} else {
								opcode._params.PushBack(rInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				//str
				if basic, ok := lType.(*types.Basic); ok {
					arithOpcode := &ArithmeticOpcode{Bxor, TypeToSilType(basic), icg._codeState, -1, -1, -1, position.Line}
					arithOpcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, arithOpcode)

					if !isSelect {

						opcode := &StackOpcode{Str, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else {
						opcode := &StackOpcode{Sti, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else if stType, ok := rType.(*types.Named); ok {
					if _, sOk := stType.Underlying().(*types.Struct); sOk {
						opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(lInfo.Base)
						if lInfo.Base == 0 {
							opcode._params.PushBack("$" + assign.Lhs[index].(*ast.Ident).Name)
						} else {
							opcode._params.PushBack(lInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				}
			}
		}
	case *ast.GoStmt:
		icg._codeState = GoStmt

	case *ast.DeferStmt:
		icg._codeState = DeferStmt
	case *ast.ReturnStmt:
		icg._codeState = ReturnStmt
		var listBuffer []CodeInfo
		var retvSize = 0

		for _, expr := range n.Results {
			icg.Visit(expr)
			typeInfo := icg._info.Types[expr].Type
			ret := &ControlOpcode{_opcode: Retv, _type: TypeToSilType(typeInfo), _params: list.New(), _parentStmt: icg._codeState}
			ret.Init()
			if ident, ok := expr.(*ast.Ident); ok {
				if ident.Name != "nil" {
					symInfo, _ := icg.FindSymbolOffset(ident.Name)

					//basic type
					if basic, ok := typeInfo.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(symInfo.Base)

						if symInfo.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(symInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					}
				} else {
					opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
					opcode._params.PushBack(0)
					opcode.Init()
					icg._codeInfoList = append(icg._codeInfoList, opcode)

				}
			}

			listBuffer = append(listBuffer, ret)

		}

		if len(listBuffer) > 1 {
			for _, code := range listBuffer {
				retvSize += TypeToByte(code.(*ControlOpcode)._type)
			}
			ret := &ControlOpcode{Retmv, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
			ret._params.PushBack(retvSize)
			ret.Init()
			icg._codeInfoList = append(icg._codeInfoList, ret)

			icg._isReturn = true
		} else if len(listBuffer) == 1 {
			icg._codeInfoList = append(icg._codeInfoList, listBuffer...)
			icg._isReturn = true
		}
	case *ast.BranchStmt:
		icg._codeState = BranchStmt

	case *ast.BlockStmt:
		block := node.(*ast.BlockStmt)
		if icg._isFirstBlock {
			icg._isFirstBlock = false
		} else {
			icg._blockNum++
		}

		icg.visitStmtList(block.List)
	case *ast.IfStmt:
		icg._codeState = IfStmt
		if n.Init != nil {
			icg.Visit(n.Init)
		}

		icg.Visit(n.Cond)

		fjpOp := &ControlOpcode{Fjp, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		icg._labelCount++
		fjpOp._params.PushBack(icg._labelCount)
		fjpOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, fjpOp)

		cLabel := icg._labelCount
		var ujpOp *ControlOpcode = nil
		ujpLabel := 0
		if n.Else != nil {
			ujpOp = &ControlOpcode{Ujp, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
			icg._labelCount++
			ujpOp._params.PushBack(icg._labelCount)
			ujpLabel = icg._labelCount
		}

		icg.Visit(n.Body)

		if ujpOp != nil {
			ujpOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, ujpOp)
		}

		labelOp := &ControlOpcode{Label, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		labelOp._params.PushBack(cLabel)
		labelOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, labelOp)

		if n.Else != nil {
			icg.Visit(n.Else)
			labelOp := &ControlOpcode{Label, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
			labelOp._params.PushBack(ujpLabel)
			labelOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, labelOp)
		}
	case *ast.CaseClause:
		icg._codeState = CaseClause
	case *ast.SwitchStmt:
		icg._codeState = SwitchStmt
	case *ast.TypeSwitchStmt:
		icg._codeState = TypeSwitchStmt
	case *ast.CommClause:
		icg._codeState = CommClause
	case *ast.SelectStmt:
		icg._codeState = SelectStmt
	case *ast.ForStmt:
		// expr for
		icg._codeState = ForStmt
		if n.Init != nil {
			icg.Visit(n.Init)
		}

		icg._labelCount++
		labelOp := &ControlOpcode{Label, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		labelOp._params.PushBack(icg._labelCount)
		labelOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, labelOp)

		ujpLabel := icg._labelCount
		fjpLabel := icg._labelCount + 1

		if n.Cond != nil {
			icg.Visit(n.Cond)
		} else {
			opcode := &StackOpcode{Ldc, I, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
			opcode._params.PushBack(1)
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		}

		fjpOp := &ControlOpcode{Fjp, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		icg._labelCount++
		fjpOp._params.PushBack(icg._labelCount)
		fjpOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, fjpOp)

		befLoopEnd := icg._loopEndLabel
		befLoopStart := icg._loopStartLabel
		icg._loopEndLabel = fjpLabel
		icg._loopStartLabel = ujpLabel

		icg.Visit(n.Body)

		icg._loopEndLabel = befLoopEnd
		icg._loopStartLabel = befLoopStart

		if n.Post != nil {
			labelOp = &ControlOpcode{Label, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
			icg._labelCount++
			labelOp._params.PushBack(icg._labelCount)
			labelOp.Init()
			icg._codeInfoList = append(icg._codeInfoList, labelOp)
			icg.Visit(n.Post)
		}

		ujpOp := &ControlOpcode{Ujp, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		ujpOp._params.PushBack(ujpLabel)
		ujpOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, ujpOp)

		labelOp = &ControlOpcode{Label, Nt, list.New(), icg._codeState, -1, -1, -1, position.Line}
		labelOp._params.PushBack(fjpLabel)
		labelOp.Init()
		icg._codeInfoList = append(icg._codeInfoList, labelOp)

	case *ast.RangeStmt:
		icg._codeState = RangeStmt
	// Declarations
	case *ast.ImportSpec:

	case *ast.ValueSpec:
		icg._codeState = ValueSpec
		isFunctionExpr := false
		varSpec := node.(*ast.ValueSpec)
		var valueInfo *symbolTable.SymbolInfo
		var valueOk = false
		var valueType types.Type
		if icg._blockNum > 0 {
			for index, ident := range varSpec.Names {
				info, ok := icg.FindSymbolOffset(ident.Name)

				if icg._isFunction && ok {
					icg._funcDeclVarSize += info.Width
				}

				valIdent := ""
				if len(varSpec.Values) == len(varSpec.Names) && len(varSpec.Values) > 0 {

					valueType = icg._info.Types[varSpec.Values[index]].Type
					if vIdent, ok := varSpec.Values[index].(*ast.Ident); ok {
						valueInfo, valueOk = icg.FindSymbolOffset(vIdent.Name)
						valIdent = vIdent.Name
					} else {
						valueOk = false
					}

					if !valueOk {
						if stType, ok := valueType.(*types.Named); ok { // right가 심볼이 아닌데 struct 인 경우
							if _, sOk := stType.Underlying().(*types.Struct); sOk {
								opcode := &StackOpcode{Lda, Ui, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
								opcode._params.PushBack(info.Base)
								if info.Base == 0 {
									opcode._params.PushBack("$" + ident.Name)
								} else {
									opcode._params.PushBack(info.Offset)
								}
								opcode.Init()
								icg._codeInfoList = append(icg._codeInfoList, opcode)
							}
						}
					}
					icg.Visit(varSpec.Values[index])

				} else if (len(varSpec.Values) < len(varSpec.Names)) && len(varSpec.Values) != 0 {
					if !isFunctionExpr {
						valueType = icg._info.Types[varSpec.Values[0]].Type
						icg.Visit(varSpec.Values[0])
						if vIdent, ok := varSpec.Values[0].(*ast.Ident); ok {
							valueInfo, valueOk = icg.FindSymbolOffset(vIdent.Name)
							valIdent = vIdent.Name
						} else {
							valueOk = false
						}

						isFunctionExpr = true

					}
				}

				if valueOk {
					if basic, ok := valueType.(*types.Basic); ok {
						opcode := &StackOpcode{Lod, TypeToSilType(basic), list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
						opcode._params.PushBack(valueInfo.Base)
						if valueInfo.Base == 0 {
							opcode._params.PushBack("$" + valIdent)
						} else {
							opcode._params.PushBack(valueInfo.Offset)
						}
						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := valueType.(*types.Named); ok {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Lod, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(valueInfo.Base)
							if valueInfo.Base == 0 {
								opcode._params.PushBack("$" + valIdent)
							} else {
								opcode._params.PushBack(valueInfo.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}

				var identType types.Type

				if varSpec.Type != nil {
					identType = icg._info.Types[varSpec.Type].Type
				} else {
					identType = valueType
				}

				if len(n.Values) != 0 {
					if iType, ok := identType.(*types.Basic); ok {
						opcode := &StackOpcode{}
						opcode.Init()
						opcode._params = list.New()
						opcode._opcode = Str
						opcode._sourceLineNum = position.Line
						switch iType.Kind() {

						case types.Uint8:
							aOpcode := &ArithmeticOpcode{}
							aOpcode._opcode = Cvui
							aOpcode._sourceLineNum = position.Line
							aOpcode._type = I
							aOpcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOpcode)

							aOpcode = &ArithmeticOpcode{}
							aOpcode._opcode = Cvi
							aOpcode._type = Ui
							aOpcode._sourceLineNum = position.Line
							aOpcode.Init()

							icg._codeInfoList = append(icg._codeInfoList, aOpcode)

							opcode._type = Ui

						case types.Int16:

						case types.Uint16:
							aOpcode := &ArithmeticOpcode{}
							aOpcode._opcode = Cvui

							aOpcode._type = I
							aOpcode._sourceLineNum = position.Line
							aOpcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOpcode)

							aOpcode = &ArithmeticOpcode{}
							aOpcode._opcode = Cvi
							aOpcode._sourceLineNum = position.Line
							aOpcode._type = S
							aOpcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOpcode)

							opcode._type = S
						case types.Int32:

						case types.Uint32:
							aOpcode := &ArithmeticOpcode{}
							aOpcode._sourceLineNum = position.Line
							aOpcode._opcode = Cvui

							aOpcode._type = I
							aOpcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOpcode)
							opcode._type = I

						case types.Int64:

						case types.Uint64:
							aOpcode := &ArithmeticOpcode{}
							aOpcode._sourceLineNum = position.Line
							aOpcode._opcode = Cvul
							aOpcode._type = L
							aOpcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOpcode)
							opcode._type = L
						case types.Uint:
							aOpcode := &ArithmeticOpcode{}
							aOpcode._sourceLineNum = position.Line
							aOpcode._opcode = Cvui
							aOpcode._type = I
							aOpcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, aOpcode)
							opcode._type = I

						default:
							opcode._type = TypeToSilType(iType)
						}

						opcode._params.PushBack(info.Base)

						if info.Base == 0 {
							opcode._params.PushBack("$" + ident.Name)
						} else {
							opcode._params.PushBack(info.Offset)
						}

						opcode.Init()
						icg._codeInfoList = append(icg._codeInfoList, opcode)
					} else if stType, ok := valueType.(*types.Named); ok && valueOk {
						if _, sOk := stType.Underlying().(*types.Struct); sOk {
							opcode := &StackOpcode{Str, P, list.New(), icg._codeState, -1, false, -1, -1, false, position.Line}
							opcode._params.PushBack(info.Base)
							if info.Base == 0 {
								opcode._params.PushBack("$" + ident.Name)
							} else {
								opcode._params.PushBack(info.Offset)
							}
							opcode.Init()
							icg._codeInfoList = append(icg._codeInfoList, opcode)
						}
					}
				}
			}
		}
	case *ast.TypeSpec:

	case *ast.BadDecl:
		// nothing to do

	case *ast.GenDecl:
		genDecl := node.(*ast.GenDecl)
		for _, s := range genDecl.Specs {
			icg.Visit(s)
		}
	case *ast.FuncDecl:
		icg._codeState = FuncDecl
		icg._blockNum++
		icg._isFirstBlock = true
		funcDecl := node.(*ast.FuncDecl)
		funcKey := icg._stringPool.LookupSymbolNumber(funcDecl.Name.Name)
		icg._isFunction = true

		var opcode = &ControlOpcode{}
		opcode.Init()
		opcode._opcode = Proc
		opcode._type = Nt
		opcode._sourceLineNum = position.Line
		opcode._parentStmt = icg._codeState
		opcode._params = list.New()
		//procIndx := len(icg._codeInfoList)
		params := funcDecl.Type.Params.List
		reverseParams := []*ast.Field{}
		for i := range params {
			n := params[len(params)-1-i]
			reverseParams = append(reverseParams, n)
		}

		for _, param := range reverseParams {
			icg.Visit(param)
		}

		icg.Visit(funcDecl.Body)
		icg._isFunction = false

		if funcDecl.Type.Results == nil {
			opcode := &ControlOpcode{}
			opcode._opcode = Ret
			opcode._type = Nt
			icg._codeState = ReturnStmt
			opcode._parentStmt = icg._codeState
			opcode._params = list.New()
			opcode._sourceLineNum = position.Line
			opcode.Init()
			icg._codeInfoList = append(icg._codeInfoList, opcode)
		} else {
			if !icg._isReturn {
				icg._codeState = ReturnStmt
				for _, result := range funcDecl.Type.Results.List {
					icg._isReturnField = true
					icg.Visit(result)
				}
			}
		}

		opcode._params.PushBack(icg._funcDeclVarSize)
		opcode._params.PushBack(1)
		opcode._params.PushBack(1)

		icg._codeInfoList = append([]CodeInfo{opcode}, icg._codeInfoList...)
		icg._funcDeclVarSize = 0
		icg._silTable.Insert(funcKey, icg._codeInfoList)

		icg._codeInfoList = []CodeInfo{}
	// Files and packages
	case *ast.File:

		file := node.(*ast.File)
		icg.visitDeclList(file.Decls)
		return &icg._silTable

	case *ast.Package:

	default:
		panic(fmt.Sprintf("ast.Walk: unexpected node type %T", n))
	}

	return nil
}

func NodeString(fs *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	format.Node(&buf, fs, n)
	return buf.String()
}
