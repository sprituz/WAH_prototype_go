package symbolTable

import "go/types"

// LiteralTable ...
// this table is save string literal
type LiteralTable struct {
	table map[string]int
}

// Init ...
func (l *LiteralTable) Init() {
	l.table = make(map[string]int)
}

// Insert ...
func (l *LiteralTable) Insert(k string, v int) {
	if _, ok := l.table[k]; !ok {
		l.table[k] = v
	}
}

//GetLiteralAddress ...
func (l *LiteralTable) GetLiteralAddress(k string) (int, bool) {
	res, ok := l.table[k]
	return res, ok

}
func (l *LiteralTable) GetLiteral(k int) (string, bool) {
	res := ""
	ok := false
	for key,val := range l.table {
		if val == k {
			res = key
			ok = true
		}
	}
	return res , ok
}

// BlockSymbolTable ...
type BlockSymbolTable struct {
	table map[int]*SymbolTable
}

// Init ...
func (o *BlockSymbolTable) Init() {
	o.table = make(map[int]*SymbolTable)

}

//Insert ...
func (o *BlockSymbolTable) Insert(k int, v *SymbolTable) {
	if _, ok := o.table[k]; !ok {
		o.table[k] = v
	}
}

//GetTable ...
func (o *BlockSymbolTable) GetTable(k int) (*SymbolTable, bool) {
	if val, ok := o.table[k]; ok {
		return val, true
	} else {
		res := &SymbolTable{}
		return res, false
	}
}

//SymbolInfo ...
type SymbolInfo struct {
	Base   int
	Offset int
	Width  int

	FieldWidth []int // struct type일 때만 ..
	FieldType  []types.Type
	IsReceiver bool
}

//SymbolTable ...
type SymbolTable struct {
	table       map[int]*SymbolInfo
	ParentBlock int
	BlockNum    int
}

func (o *SymbolTable) Init() {
	o.table = make(map[int]*SymbolInfo)
	o.ParentBlock = -1
	o.BlockNum = 0
}

//Insert ...
func (o *SymbolTable) Insert(k int, v *SymbolInfo) {
	if _, ok := o.table[k]; !ok {
		o.table[k] = v
	}
}

//GetOffset ...
func (o *SymbolTable) GetOffset(k int) (*SymbolInfo, bool) {
	if val, ok := o.table[k]; ok {
		return val, true
	} else {
		res := &SymbolInfo{}
		return res, false
	}
}

//IsEmpty ...
func (o *SymbolTable) IsEmpty() bool {
	if len(o.table) == 0 {
		return true
	}
	return false
}
