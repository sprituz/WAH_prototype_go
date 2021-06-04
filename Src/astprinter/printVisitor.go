package astprinter

import (
	"fmt"
	"go/ast"
	"strings"
)

type visitor int

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	fmt.Printf("%s%T\n", strings.Repeat("\t", int(v)), n)

	if ident, ok := n.(*ast.Ident); ok {
		fmt.Printf("%s%s\n", strings.Repeat("\t", int(v+1)), ident.Name)
	} else if str, ok := n.(*ast.BasicLit); ok {
		fmt.Printf("%s%s\n", strings.Repeat("\t", int(v+1)), str.Value)
	}

	return v + 1
}

//PrintAst ...
func PrintAst(f *ast.File) {
	var astPrinter visitor

	ast.Walk(astPrinter, f)
}
