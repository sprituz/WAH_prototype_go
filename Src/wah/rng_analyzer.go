package wah

import (
	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/icg"
	"fmt"
	"strings"
)

type RNGAnalyzer struct {
	analysisFile  string
	isSeedCall    bool
	analysisCount int
}

func (analyzer *RNGAnalyzer) Init(analysisFile string) {
	analyzer.analysisFile = analysisFile
	analyzer.isSeedCall = false
	analyzer.analysisCount = 0
}

func (analyzer *RNGAnalyzer) Copy() *RNGAnalyzer {
	newAnalyzer := new(RNGAnalyzer)

	newAnalyzer.analysisFile = analyzer.analysisFile
	newAnalyzer.isSeedCall = analyzer.isSeedCall

	return newAnalyzer
}

func (analyzer *RNGAnalyzer) printAlarm(linenum int) {
	var ccw CCW = RANDOM_NUMBER_GENERATION
	if isFirstDetect {
		fmt.Printf("chaincode weakness detected:\n")
		isFirstDetect = false
	}
	fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
	fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
}
func (analyzer *RNGAnalyzer) RNGAnalysis(block cfg.CFGBlock) int {
	switch b := block.(type) {
	case *cfg.CallBlock:
		opcode := b.CodeList()[0]
		if controlOp, ok := opcode.(*icg.ControlOpcode); ok {
			if controlOp.Opcode() == icg.Call {
				funcName := fmt.Sprint(controlOp.Params().Front().Value)
				if strings.Contains(funcName, "rand.Seed") {
					analyzer.isSeedCall = true
				} else if strings.Contains(funcName, "rand.") && !analyzer.isSeedCall {
					analyzer.printAlarm(controlOp.GetSourceLine())
					analyzer.analysisCount++
				}
			}
		}
	}

	return analyzer.analysisCount
}
