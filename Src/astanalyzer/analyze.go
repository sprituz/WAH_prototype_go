package astanalyzer

import (
	"fmt"
	"go/ast"
	"go/types"
)

type visitor int

//Analyze ...
func Analyze(f *ast.File, info *types.Info) {
	var isFirst = true
	ast.Inspect(f, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncDecl); ok {
			if !isFirst {
				fmt.Println("$function end")
			}
			isFirst = false
			fmt.Println("$ " + function.Name.Name + " function analysis start")

		}

		// map Iteration 검사
		if rangeFor, ok := node.(*ast.RangeStmt); ok {
			if tv, ok := info.Types[rangeFor.X]; ok {
				_, isMap := tv.Type.(*types.Map)
				if isMap {
					fmt.Printf("  * Warning : Not use a map type \"%s\" in loop range\n", rangeFor.X)
				}
			}
		}

		if goStmt, ok := node.(*ast.GoStmt); ok {
			if tv, ok := info.Types[goStmt.Call.Fun]; ok {
				fmt.Printf("  * Warning : Not Use go routine \"go %v\"\n", tv.Type.Underlying())
			}
		}

		if assign, ok := node.(*ast.AssignStmt); ok {
			errLocation := -1
			for i, rhs := range assign.Rhs {

				if results, ok := info.Types[rhs].Type.(*types.Tuple); ok {
					for j := 0; j < results.Len(); j++ {
						res := results.At(j)
						resType := res.Type()
						if resType.String() == "error" {
							errLocation = i + j
							fmt.Printf("error Location : %d\n", errLocation)
						}
					}
				} else if res, ok := info.Types[rhs].Type.(*types.Named); ok {
					if res.String() == "error" {
						errLocation = i
						fmt.Printf("error Location : %d\n", errLocation)
					}
				}
			}

			if errLocation != -1 {
				if ident, ok := assign.Lhs[errLocation].(*ast.Ident); ok {
					if ident.Name == "_" {
						fmt.Println("Security weakness Unhandled error has been detected.")
						fmt.Printf("The %d-th return type of rhs is error, but is not assigned to the %d-th lhs.\n", errLocation, errLocation)

					}
				}
			}
		}
		return true
	})
	fmt.Println("$function end")
}
