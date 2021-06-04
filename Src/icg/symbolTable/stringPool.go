package symbolTable

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

//StringPool ...
type StringPool struct {
	table map[string]int
}

func (pool *StringPool) Init() {
	pool.table = make(map[string]int)
}
func (pool *StringPool) IsContain(symbol string) bool {
	if _, ok := pool.table[symbol]; ok {
		return true
	}

	return false
}
func (pool *StringPool) LookupSymbolNumber(symbol string) int {
	return pool.table[symbol]
}
func (pool *StringPool) LookupSymbolName(symbolNumber int) string {
	var res = ""
	for k, v := range pool.table {
		if v == symbolNumber {
			res = k
			break
		}
	}

	return res
}
func (pool *StringPool) Insert(symbol string, symbolNumber int) {
	if _, ok := pool.table[symbol]; !ok {
		pool.table[symbol] = symbolNumber
	}
}

// StringPoolGenerator ...
type StringPoolGenerator struct {
	_pool          *StringPool
	isReceiver     bool
	index          int
	symbolOffset   int
	symbolTypeInfo *types.Info
	isFirstBlock   bool
	blockNum       int
	isField        bool
	isFunction     bool
	base           int
	literalIndex   int
	offsetTable    *BlockSymbolTable
	tmpTable       *SymbolTable
	literalTable   *LiteralTable
	tmpSym         int
	blockCount     int

	ast.Visitor
}

// Init ...
func (pg *StringPoolGenerator) Init(info *types.Info) {
	pg.index = 0
	pg._pool = &StringPool{}
	pg._pool.Init()
	pg.symbolOffset = 0
	pg.symbolTypeInfo = info
	pg.base = 0
	pg.isFirstBlock = false
	pg.isField = false
	pg.blockNum = 0
	pg.offsetTable = &BlockSymbolTable{} // block num, offsettable
	pg.offsetTable.Init()
	pg.blockCount = 0
	pg.tmpTable = &SymbolTable{}
	pg.tmpTable.Init()
	pg.tmpSym = 0

	pg.isFunction = false
	pg.literalIndex = 0
	pg.literalTable = &LiteralTable{}
	pg.literalTable.Init()
}

// GenerateSymbolInfo ...
func (pg *StringPoolGenerator) GenerateSymbolInfo(name string, varType types.Type) (width int) {

	width = 0
	switch varType.(type) {
	case *types.Struct:
		res := varType.(*types.Struct)
		if !pg.isField {

			symInfo := &SymbolInfo{}
			symInfo.Base = pg.base
			symInfo.Offset = pg.symbolOffset
			symInfo.IsReceiver = pg.isReceiver

			for i := 0; i < res.NumFields(); i++ {
				pg.isField = true
				symInfo.FieldType = append(symInfo.FieldType, res.Field(i).Type())
				fieldWidth := pg.GenerateSymbolInfo(name, res.Field(i).Type())
				symInfo.FieldWidth = append(symInfo.FieldWidth, fieldWidth)
				width += fieldWidth

			}
			symInfo.Width = width
			pg.symbolOffset += symInfo.Width

			if pg._pool.IsContain(name) {
				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			} else {
				pg._pool.Insert(name, pg.index)
				pg.index++

				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			}

			pg.isField = false
		} else {
			for i := 0; i < res.NumFields(); i++ {
				width += pg.GenerateSymbolInfo(name, res.Field(i).Type())
			}
		}
	case *types.Basic:
		res := varType.(*types.Basic)
		if !pg.isField {
			symInfo := &SymbolInfo{}
			symInfo.Base = pg.base
			symInfo.Offset = pg.symbolOffset
			symInfo.Width = TypeToByte(res.Kind())
			pg.symbolOffset += symInfo.Width
			symInfo.IsReceiver = pg.isReceiver
			width = symInfo.Width
			if pg._pool.IsContain(name) {
				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			} else {
				pg._pool.Insert(name, pg.index)
				pg.index++

				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			}
		} else {
			width = TypeToByte(res.Kind())
		}
	case *types.Named:
		res := varType.(*types.Named)
		width = pg.GenerateSymbolInfo(name, res.Underlying())
	case *types.Pointer:
		res := varType.(*types.Pointer)
		if !pg.isField {
			symInfo := &SymbolInfo{}
			symInfo.Base = pg.base
			symInfo.Offset = pg.symbolOffset
			symInfo.Width = TypeToByte(types.Uintptr)
			pg.symbolOffset += symInfo.Width
			symInfo.IsReceiver = pg.isReceiver
			width = symInfo.Width

			baseType := res.Elem()
			if _, ok := baseType.(*types.Pointer); ok {
				//var typeInfo types.Type
				for {
					baseType = baseType.Underlying()
					_, ok := baseType.(*types.Pointer)
					if !ok {
						break
					}
				}
			}

			if usr, ok := baseType.(*types.Named); ok {
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
					for i := 0; i < structType.NumFields(); i++ {
						pg.isField = true
						symInfo.FieldType = append(symInfo.FieldType, structType.Field(i).Type())
						fieldWidth := pg.GenerateSymbolInfo(name, structType.Field(i).Type())
						symInfo.FieldWidth = append(symInfo.FieldWidth, fieldWidth)
					}
					pg.isField = false
				}
			}

			if pg._pool.IsContain(name) {
				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			} else {
				pg._pool.Insert(name, pg.index)
				pg.index++

				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			}
		} else {
			width = TypeToByte(types.Uintptr)
		}

	default:
		if !pg.isField {
			symInfo := &SymbolInfo{}
			symInfo.Base = pg.base
			symInfo.Offset = pg.symbolOffset
			symInfo.Width = TypeToByte(types.Uintptr)
			pg.symbolOffset += symInfo.Width
			symInfo.IsReceiver = pg.isReceiver
			width = symInfo.Width
			if pg._pool.IsContain(name) {
				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			} else {
				pg._pool.Insert(name, pg.index)
				pg.index++

				pg.tmpTable.Insert(pg._pool.LookupSymbolNumber(name), symInfo)
			}
		} else {
			width = TypeToByte(types.Uintptr)
		}
	}

	return

}
func (pg *StringPoolGenerator) GenIdentList(list []*ast.Ident) {
	for _, x := range list {
		pg.Gen(x)
	}
}
func (pg *StringPoolGenerator) GenExprList(list []ast.Expr) {
	for _, x := range list {
		pg.Gen(x)
	}
}
func (pg *StringPoolGenerator) GenStmtList(list []ast.Stmt) {
	for _, x := range list {
		pg.Gen(x)
	}
}

func (pg *StringPoolGenerator) GenDeclList(list []ast.Decl) {
	for _, x := range list {
		pg.Gen(x)
	}
}
func (pg *StringPoolGenerator) Gen(node ast.Node) (*StringPool, *BlockSymbolTable, *LiteralTable) {
	switch n := node.(type) {
	// Comments and fields
	case *ast.Comment:
		// nothing to do

	case *ast.CommentGroup:

	case *ast.Field:
		pg.GenIdentList(n.Names)
		_, _, _ = pg.Gen(n.Type)
		if n.Tag != nil {
			pg.Gen(n.Tag)
		}
		if n.Comment != nil {
			pg.Gen(n.Comment)
		}

	case *ast.FieldList:
		for _, f := range n.List {
			pg.Gen(f)
		}

	case *ast.BadExpr, *ast.Ident:
	// Expressions
	case *ast.BasicLit:
		// nothing to do
		if n.Kind == token.STRING {
			pg.literalTable.Insert(n.Value, pg.literalIndex)
			pg.literalIndex++
		}

	case *ast.Ellipsis:
		if n.Elt != nil {
			pg.Gen(n.Elt)
		}

	case *ast.FuncLit:
		pg.Gen(n.Type)
		pg.Gen(n.Body)

	case *ast.CompositeLit:
		if n.Type != nil {
			if usr, ok := pg.symbolTypeInfo.Types[n.Type].Type.(*types.Named); ok {
				if _, ok := usr.Underlying().(*types.Struct); ok {
					stType := pg.symbolTypeInfo.Types[n.Type].Type
					tmpName := strconv.Itoa(pg.tmpSym)

					pg.GenerateSymbolInfo(tmpName, stType)
					pg.tmpSym++
				}

			}
			pg.Gen(n.Type)

		}
		pg.GenExprList(n.Elts)

	case *ast.ParenExpr:
		pg.Gen(n.X)

	case *ast.SelectorExpr:
		pg.Gen(n.X)
		pg.Gen(n.Sel)

	case *ast.IndexExpr:
		pg.Gen(n.X)
		pg.Gen(n.Index)

	case *ast.SliceExpr:
		pg.Gen(n.X)
		if n.Low != nil {
			pg.Gen(n.Low)
		}
		if n.High != nil {
			pg.Gen(n.High)
		}
		if n.Max != nil {
			pg.Gen(n.Max)
		}

	case *ast.TypeAssertExpr:
		pg.Gen(n.X)
		if n.Type != nil {
			pg.Gen(n.Type)
		}

	case *ast.CallExpr:
		pg.Gen(n.Fun)
		pg.GenExprList(n.Args)

	case *ast.StarExpr:
		pg.Gen(n.X)

	case *ast.UnaryExpr:
		pg.Gen(n.X)

	case *ast.BinaryExpr:
		pg.Gen(n.X)
		pg.Gen(n.Y)

	case *ast.KeyValueExpr:
		pg.Gen(n.Key)
		pg.Gen(n.Value)

	// Types
	case *ast.ArrayType:
		if n.Len != nil {
			pg.Gen(n.Len)
		}
		pg.Gen(n.Elt)

	case *ast.StructType:
		pg.Gen(n.Fields)

	case *ast.FuncType:
		funcField := n
		for _, params := range funcField.Params.List {
			for _, ident := range params.Names {
				pg._pool.Insert(ident.Name, pg.index)

				typeInfo := pg.symbolTypeInfo.Types[params.Type]

				pg.GenerateSymbolInfo(ident.Name, typeInfo.Type)
				pg.index++
			}

		}

		if funcField.Results != nil {
			for _, params := range funcField.Results.List {
				if params.Names != nil {
					for _, ident := range params.Names {
						pg._pool.Insert(ident.Name, pg.index)

						typeInfo := pg.symbolTypeInfo.Types[params.Type]

						pg.GenerateSymbolInfo(ident.Name, typeInfo.Type)
						pg.index++
					}
				}
			}
		}
	case *ast.InterfaceType:
		pg.Gen(n.Methods)

	case *ast.MapType:
		pg.Gen(n.Key)
		pg.Gen(n.Value)

	case *ast.ChanType:
		pg.Gen(n.Value)

	// Statements
	case *ast.BadStmt:
		// nothing to do

	case *ast.DeclStmt:
		pg.Gen(n.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		pg.Gen(n.Label)
		pg.Gen(n.Stmt)

	case *ast.ExprStmt:
		pg.Gen(n.X)

	case *ast.SendStmt:
		pg.Gen(n.Chan)
		pg.Gen(n.Value)

	case *ast.IncDecStmt:
		pg.Gen(n.X)

	case *ast.AssignStmt:
		assign := n
		if assign.Tok == token.DEFINE {
			for i, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					pg._pool.Insert(ident.Name, pg.index)

					var typeInfo types.TypeAndValue

					if len(assign.Lhs) == len(assign.Rhs) {
						typeInfo = pg.symbolTypeInfo.Types[assign.Rhs[i]]

					} else if len(assign.Lhs) > len(assign.Rhs) {
						typeInfo = pg.symbolTypeInfo.Types[assign.Rhs[0]]
					}

					pg.GenerateSymbolInfo(ident.Name, typeInfo.Type)
					pg.index++
				}

			}
		}
		pg.GenExprList(n.Lhs)
		pg.GenExprList(n.Rhs)

	case *ast.GoStmt:
		pg.Gen(n.Call)

	case *ast.DeferStmt:
		pg.Gen(n.Call)

	case *ast.ReturnStmt:
		pg.GenExprList(n.Results)

	case *ast.BranchStmt:
		if n.Label != nil {
			pg.Gen(n.Label)
		}

	case *ast.BlockStmt:
		block := n
		var sblock *SymbolTable
		isExistPBlock := false
		if pg.isFirstBlock {
			pg.isFirstBlock = false

		} else { // 이전 블록이 있는경우
			sblock = pg.tmpTable
			isExistPBlock = true
			pBlock := pg.tmpTable.BlockNum
			pg.tmpTable = &SymbolTable{}
			pg.tmpTable.Init()
			pg.blockCount++
			pg.tmpTable.BlockNum = pg.blockCount
			pg.tmpTable.ParentBlock = pBlock
		}

		pg.GenStmtList(block.List)

		pg.offsetTable.Insert(pg.tmpTable.BlockNum, pg.tmpTable)
		//pg.symbolOffset = 0

		if isExistPBlock {
			pg.tmpTable = sblock
			pg.blockNum = 0
		}

	case *ast.IfStmt:
		if n.Init != nil {
			pg.Gen(n.Init)
		}
		pg.Gen(n.Cond)
		pg.Gen(n.Body)
		if n.Else != nil {
			pg.Gen(n.Else)
		}

	case *ast.CaseClause:
		pg.GenExprList(n.List)
		pg.GenStmtList(n.Body)

	case *ast.SwitchStmt:
		if n.Init != nil {
			pg.Gen(n.Init)
		}
		if n.Tag != nil {
			pg.Gen(n.Tag)
		}
		pg.Gen(n.Body)

	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			pg.Gen(n.Init)
		}
		pg.Gen(n.Assign)
		pg.Gen(n.Body)

	case *ast.CommClause:
		if n.Comm != nil {
			pg.Gen(n.Comm)
		}
		pg.GenStmtList(n.Body)

	case *ast.SelectStmt:
		pg.Gen(n.Body)

	case *ast.ForStmt:
		if n.Init != nil {
			pg.Gen(n.Init)
		}
		if n.Cond != nil {
			pg.Gen(n.Cond)
		}
		if n.Post != nil {
			pg.Gen(n.Post)
		}
		pg.Gen(n.Body)

	case *ast.RangeStmt:
		rangeStmt := n
		if key, ok := rangeStmt.Key.(*ast.Ident); ok {
			pg._pool.Insert(key.Name, pg.index)
			typeInfo := pg.symbolTypeInfo.Types[rangeStmt.Key]

			pg.GenerateSymbolInfo(key.Name, typeInfo.Type)
			pg.index++
		}

		if value, ok := rangeStmt.Value.(*ast.Ident); ok {
			pg._pool.Insert(value.Name, pg.index)

			typeInfo := pg.symbolTypeInfo.Types[rangeStmt.Value]

			pg.GenerateSymbolInfo(value.Name, typeInfo.Type)
			pg.index++
		}

	// Declarations
	case *ast.ImportSpec:
		if n.Doc != nil {
			pg.Gen(n.Doc)
		}
		if n.Name != nil {
			pg.Gen(n.Name)
		}
		pg.Gen(n.Path)
		if n.Comment != nil {
			pg.Gen(n.Comment)
		}

	case *ast.ValueSpec:
		varSpec := n

		for i, ident := range varSpec.Names {
			pg._pool.Insert(ident.Name, pg.index)
			typeInfo := pg.symbolTypeInfo.Types[varSpec.Type]

			if varSpec.Type == nil {
				if len(varSpec.Values) == len(varSpec.Names) && len(varSpec.Values) > 0 {
					typeInfo = pg.symbolTypeInfo.Types[varSpec.Values[i]]

				} else if len(varSpec.Values) < len(varSpec.Names) {
					typeInfo = pg.symbolTypeInfo.Types[varSpec.Values[0]]
				}
			}

			pg.GenerateSymbolInfo(ident.Name, typeInfo.Type)
			pg.index++
		}

	case *ast.TypeSpec:
		if n.Doc != nil {
			pg.Gen(n.Doc)
		}
		pg.Gen(n.Name)
		pg.Gen(n.Type)
		if n.Comment != nil {
			pg.Gen(n.Comment)
		}

	case *ast.BadDecl:
		// nothing to do

	case *ast.GenDecl:
		genDecl := n
		for _, spec := range genDecl.Specs {
			if varSpec, ok := spec.(*ast.ValueSpec); ok {
				for i, ident := range varSpec.Names {
					pg._pool.Insert(ident.Name, pg.index)
					typeInfo := pg.symbolTypeInfo.Types[varSpec.Type]

					if varSpec.Type == nil {
						if len(varSpec.Values) == len(varSpec.Names) && len(varSpec.Values) > 0 {
							typeInfo = pg.symbolTypeInfo.Types[varSpec.Values[i]]

						} else if len(varSpec.Values) < len(varSpec.Names) {
							typeInfo = pg.symbolTypeInfo.Types[varSpec.Values[0]]
						}

					}

					pg.GenerateSymbolInfo(ident.Name, typeInfo.Type)
					pg.index++
				}
			} else if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				pg._pool.Insert(typeSpec.Name.Name, pg.index)
				typeInfo := pg.symbolTypeInfo.Types[typeSpec.Type]
				sOffset := pg.symbolOffset
				pg.GenerateSymbolInfo(typeSpec.Name.Name, typeInfo.Type)
				pg.index++
				pg.symbolOffset = sOffset

			}
		}

	case *ast.FuncDecl:
		function := n
		pg._pool.Insert(function.Name.Name, pg.index)
		pg.isFirstBlock = true
		pg.isFunction = true
		// if !pg.tmpTable.IsEmpty() {
		pg.offsetTable.Insert(pg.tmpTable.BlockNum, pg.tmpTable)
		pg.tmpTable = &SymbolTable{}
		pg.tmpTable.Init()
		pg.blockCount++
		pg.tmpTable.BlockNum = pg.blockCount

		pg.tmpTable.ParentBlock = 0

		pg.symbolOffset = 0

		// } else {
		// 	pg.tmpTable = &SymbolTable{}
		// 	pg.tmpTable.Init()
		// 	pg.tmpTable.BlockNum = pg.blockCount

		// 	pg.tmpTable.ParentBlock = 0

		// 	pg.symbolOffset = 0
		// }

		pg.index++
		pg.isFunction = true

		pg.base = 1
		if n.Recv != nil {

			for _, field := range n.Recv.List {
				for _, ident := range field.Names {
					pg._pool.Insert(ident.Name, pg.index)

					typeInfo := pg.symbolTypeInfo.Types[field.Type]
					pg.isReceiver = true
					pg.GenerateSymbolInfo(ident.Name, typeInfo.Type)
					pg.isReceiver = false
					pg.index++
				}

			}

		}

		pg.Gen(n.Type)
		if n.Body != nil {
			pg.isFirstBlock = true
			pg.Gen(n.Body)
		}
		pg.isFunction = false
		pg.base = 0
		pg.symbolOffset = 0
	// Files and packages
	case *ast.File:
		file := n
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				pg.Gen(genDecl)
			}
		}
		for _, decl := range file.Decls {
			_, ok := decl.(*ast.GenDecl)
			if !ok {
				pg.Gen(decl)
			}
		}

		// ast.Inspect(node, func(node ast.Node) bool {
		// 	if basic, ok := node.(*ast.BasicLit); ok {
		// 		if basic.Kind == token.STRING {
		// 			pg.literalTable.Insert(basic.Value, pg.literalIndex)
		// 			pg.literalIndex++
		// 		}
		// 	}
		// 	return true
		// })
		// don't pg.Gen n.Comments - they have been
		// visited already through the individual
		// nodes
	case *ast.Package:

	default:
		panic(fmt.Sprintf("ast.Gen: unexpected node type %T", n))
	}

	return pg._pool, pg.offsetTable, pg.literalTable
}

func TypeToByte(kind types.BasicKind) int {
	var res = 0
	switch kind {
	case types.Bool:
		res = 4
	case types.Int:
		res = 4
	case types.Int8:
		res = 1
	case types.Uint8:
		res = 1
	case types.Int16:
		res = 2
	case types.Uint16:
		res = 2
	case types.Int32:
		res = 4
	case types.Uint32:
		res = 4
	case types.Int64:
		res = 8
	case types.Uint64:
		res = 8
	case types.Uint:
		res = 4
	case types.Uintptr:
		res = 4
	case types.Float32:
		res = 4
	case types.Float64:
		res = 8
	case types.String:
		res = 4
	default:
		res = 0
	}
	return res
}
