package wah

import (
   "fmt"
   "go/ast"
   "go/token"
   "go/types"
   "strings"

   "WAH_prototype_go-master/Src/icg"
)

type ASTAnalyzer struct {
   analysisFile string
   FuncRetTable map[string]int // functionName , error 위치  error리턴이 없는경우 -1
   fs           *token.FileSet
   ueCompleteFuncList []string
   analysisCount int
}

func (analyzer *ASTAnalyzer) Init(analysisFile string, fset *token.FileSet) {
   analyzer.analysisFile = analysisFile
   analyzer.FuncRetTable = make(map[string]int)
   analyzer.fs = fset
   analyzer.analysisCount = 0
}

//MapStructureIteration...
func (analyzer *ASTAnalyzer) MSIAnalysis(node ast.Node, info *types.Info) {
   var ccw CCW = MAP_STRUCTURE_ITER
   var position token.Position

   if node != nil {
      position = analyzer.fs.Position(node.Pos())
   }

   if rangeFor, ok := node.(*ast.RangeStmt); ok {
      if tv, ok := info.Types[rangeFor.X]; ok {
         _, isMap := tv.Type.(*types.Map)
         if isMap {
            linenum := position.Line
            if isFirstDetect {
               fmt.Printf("chaincode weakness detected:\n")
               isFirstDetect = false
            }
            fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
            fmt.Printf("\t not use a map type \"%s\" in loop range\n", rangeFor.X)
            fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
            analyzer.analysisCount ++
         }
      }
   }
}
func (analyzer *ASTAnalyzer) UsedGoroutineAnalysis(node ast.Node, info *types.Info) {
   var ccw CCW = USED_GOROUTINE
   var position token.Position

   if node != nil {
      position = analyzer.fs.Position(node.Pos())
   }
   if goStmt, ok := node.(*ast.GoStmt); ok {
      if tv, ok := info.Types[goStmt.Call.Fun]; ok {
         linenum := position.Line
         if isFirstDetect {
            fmt.Printf("chaincode weakness detected:\n")
            isFirstDetect = false
         }
         fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
         fmt.Printf("\t not use go routine \"go %v\"\n", tv.Type.Underlying())
         fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
         analyzer.analysisCount ++
      }
   }
}

func (analyzer *ASTAnalyzer) UnhandledErrorsAnalysis(node ast.Node, info *types.Info) int {
   errLocation := -1
   var ccw CCW = UNHANDLED_ERROR
   var position token.Position

   if node != nil {
      position = analyzer.fs.Position(node.Pos())
   }
   if assign, ok := node.(*ast.AssignStmt); ok {
      for i, rhs := range assign.Rhs {
         funcName := ""
         if call, ok := rhs.(*ast.CallExpr); ok {
            funcName = icg.NodeString(analyzer.fs, call.Fun)
            if results, ok := info.Types[rhs].Type.(*types.Tuple); ok {
               for j := 0; j < results.Len(); j++ {
                  res := results.At(j)
                  resType := res.Type()
                  if resType.String() == "error" {
                     errLocation = i + j
                  }
               }
            } else if res, ok := info.Types[rhs].Type.(*types.Named); ok {
               if res.String() == "error" {
                  errLocation = i
               }
            }

            if errLocation != -1 {
               if ident, ok := assign.Lhs[errLocation].(*ast.Ident); ok {
                  if ident.Name == "_" {
                     linenum := position.Line
                     if isFirstDetect {
                        fmt.Printf("chaincode weakness detected:\n")
                        isFirstDetect = false
                     }
                     fmt.Printf( "\t CCW-%03d : %s\n", ccw, ccw.String())

                     fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
                     analyzer.analysisCount ++
                     //fmt.Printf("\t   The %d return type of rhs( %s ) is error, but is not assigned to the %d lhs ( _ ).\n\n", errLocation, funcName, errLocation)
                     errLocation = -1
                  }
               }
            }
            analyzer.FuncRetTable[funcName] = errLocation
         }
      }

   }

   return errLocation
}


func (analyzer *ASTAnalyzer) PhantomReadAnalysis(node ast.Node, info *types.Info) {
   var ccw CCW = PHANTOM_READS
   var position token.Position

   if node != nil {
      position = analyzer.fs.Position(node.Pos())
   }
   if assign, ok := node.(*ast.AssignStmt); ok {
      for _, rhs := range assign.Rhs {
         funcName := ""
         if call, ok := rhs.(*ast.CallExpr); ok {
            funcName = icg.NodeString(analyzer.fs, call.Fun)

            if strings.Contains(funcName, "GetHistoryForKey") || strings.Contains(funcName, "GetQueryResult") {
               linenum := position.Line
               if isFirstDetect {
                  fmt.Printf("chaincode weakness detected:\n")
                  isFirstDetect = false
               }
               fmt.Printf( "\t CCW-%03d : %s\n", ccw, ccw.String())

               fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
               analyzer.analysisCount ++
            }
         }
      }
   }
}

func (analyzer *ASTAnalyzer) RQRAnalysis(node ast.Node, info *types.Info) {
   var ccw CCW = RANGE_QUERY_RISK
   var position token.Position

   if node != nil {
      position = analyzer.fs.Position(node.Pos())
   }
   if assign, ok := node.(*ast.AssignStmt); ok {
      for _, rhs := range assign.Rhs {
         funcName := ""
         if call, ok := rhs.(*ast.CallExpr); ok {
            funcName = icg.NodeString(analyzer.fs, call.Fun)

            if strings.Contains(funcName, "GetHistoryForKey") || strings.Contains(funcName, "GetQueryResult")  || strings.Contains(funcName, "GetPrivateDataQueryResult"){
               linenum := position.Line
               if isFirstDetect {
                  fmt.Printf("chaincode weakness detected:\n")
                  isFirstDetect = false
               }
               fmt.Printf( "\t CCW-%03d : %s\n", ccw, ccw.String())

               fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
               analyzer.analysisCount ++
            }
         }
      }
   }
}

func (analyzer *ASTAnalyzer) SystemCommandAnalysis(node ast.Node, info *types.Info) {
   var ccw CCW = SYSTEM_COMMANDS
   var position token.Position

	 if node != nil {
			position = analyzer.fs.Position(node.Pos())
	 }

   switch x := node.(type) {
    case *ast.BasicLit:
      if strings.Contains(x.Value, "os/exec"){
         linenum := position.Line
         if isFirstDetect {
            fmt.Printf("chaincode weakness detected:\n")
            isFirstDetect = false
         }
         fmt.Printf( "\t CCW-%03d : %s\n", ccw, ccw.String())

         fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
         analyzer.analysisCount ++
      }
    }
}

func (analyzer *ASTAnalyzer) SystemTimestampAnalysis(node ast.Node, info *types.Info) {
   var ccw CCW = SYSTEM_TIMESTAMP
   var position token.Position

	 if node != nil {
			position = analyzer.fs.Position(node.Pos())
	 }

   switch x := node.(type) {
    case *ast.BasicLit:
      if strings.Contains(x.Value, "time"){
         linenum := position.Line
         if isFirstDetect {
            fmt.Printf("chaincode weakness detected:\n")
            isFirstDetect = false
         }
         fmt.Printf( "\t CCW-%03d : %s\n", ccw, ccw.String())

         fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
         analyzer.analysisCount ++
      }
    }
}

//Analyze ...
func (analyzer *ASTAnalyzer) Analysis(f *ast.File, info *types.Info) int{

   ast.Inspect(f, func(node ast.Node) bool {
      analyzer.MSIAnalysis(node, info)
      analyzer.UsedGoroutineAnalysis(node, info)
      analyzer.UnhandledErrorsAnalysis(node, info)
      analyzer.PhantomReadAnalysis(node,info)
      analyzer.RQRAnalysis(node,info)
			analyzer.SystemCommandAnalysis(node, info)
			analyzer.SystemTimestampAnalysis(node, info)
      return true
   })

   return analyzer.analysisCount
}
