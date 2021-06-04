package cfg

import (
	"strconv"
	"strings"
)

var isRoot = true
var currentBlockNum = 0
var branchType BranchType = UnconditionBranch

func InitPrinter() {
	isRoot = true
	currentBlockNum = 0
	branchType = UnconditionBranch

}
func Print(node CFGBlock) (nodeStr string) {
	if !node.IsPrint() {

		switch n := node.(type) {
		case *BasicBlock:
			builder := strings.Builder{}
			if isRoot {
				builder.WriteString("digraph {\n\tnode[rx = 5ry = 5labelStyle = \"font: 300 14px 'Helvetica Neue', Helvetica\"]\n\tedge[labelStyle = \"font: 30014px 'Helvetica Neue', Helvetica\"]\n")
				isRoot = false
			} else {
				str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
				builder.WriteString(str)

				if branchType == TrueBranch {
					builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
				} else if branchType == FalseBranch {
					builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
				}
				builder.WriteString("\n")
				branchType = UnconditionBranch
			}
			currentBlockNum = n.BlockNumber()
			str := strconv.Itoa(currentBlockNum) + " [label = \"" + n.String() + "\"];\n"
			builder.WriteString(str)

			builder.WriteString(Print(n.LinkedBlock()))

			// if n.BlockNumber() == 0 {
			// 	builder.WriteString("}")
			// 	isRoot = true
			// 	currentBlockNum = 0
			// }

			nodeStr = builder.String()

		case *BranchBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)

			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")

			branchType = UnconditionBranch

			currentBlockNum = n.BlockNumber()
			str = strconv.Itoa(currentBlockNum) + " [label = \"" + n.String() + "\"];\n"
			builder.WriteString(str)

			current := currentBlockNum
			switch b := n.TargetBlock().(type) {
			case *BasicBlock:
				if n.BlockNumber() < b.BlockNumber() {
					branchType = n.BranchType()

					builder.WriteString(Print(n.TargetBlock()))
					currentBlockNum = current
				} else {
					str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.TargetBlock().(*BasicBlock).BlockNumber()) + "\n"
					builder.WriteString(str)
				}
			case *BranchBlock:
				if n.BlockNumber() < b.BlockNumber() {
					branchType = n.BranchType()

					builder.WriteString(Print(n.TargetBlock()))
					currentBlockNum = current
				} else {
					str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.TargetBlock().(*BasicBlock).BlockNumber()) + "\n"
					builder.WriteString(str)
				}
			case *ReturnBlock:
				if n.BlockNumber() < b.BlockNumber() {
					branchType = n.BranchType()

					builder.WriteString(Print(n.TargetBlock()))
					currentBlockNum = current
				} else {
					str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.TargetBlock().(*BasicBlock).BlockNumber()) + "\n"
					builder.WriteString(str)
				}
			case *CallBlock:
				if n.BlockNumber() < b.BlockNumber() {
					branchType = n.BranchType()

					builder.WriteString(Print(n.TargetBlock()))
					currentBlockNum = current
				} else {
					str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.TargetBlock().(*BasicBlock).BlockNumber()) + "\n"
					builder.WriteString(str)
				}
			}

			if n.BranchType() != UnconditionBranch {
				builder.WriteString(Print(n.UjpBlock()))
				currentBlockNum = current
			}

			nodeStr = builder.String()
		case *CallBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)
			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")

			branchType = UnconditionBranch
			branchType = CallBranch

			currentBlockNum = n.BlockNumber()
			str = strconv.Itoa(currentBlockNum) + " [label = \"" + n.String() + "\"];\n"
			builder.WriteString(str)

			current := currentBlockNum
			if n.TargetBlock() != nil {
				builder.WriteString(Print(n.TargetBlock()))
				currentBlockNum = current
			}

			if n.UjpBlock() != nil {
				builder.WriteString(Print(n.UjpBlock()))
				currentBlockNum = current
			}

			nodeStr = builder.String()
		case *ReturnBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)
			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")

			branchType = UnconditionBranch
			currentBlockNum = n.BlockNumber()
			str = strconv.Itoa(currentBlockNum) + " [label = \"" + n.String() + "\"];\n"
			builder.WriteString(str)

			nodeStr = builder.String()

		default:
			nodeStr = ""
		}
		node.SetIsPrint(true)
	} else {
		switch n := node.(type) {
		case *BasicBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)
			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")
			branchType = UnconditionBranch
			nodeStr = builder.String()
		case *BranchBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)
			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")
			branchType = UnconditionBranch
			nodeStr = builder.String()
		case *ReturnBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)
			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")
			branchType = UnconditionBranch
			nodeStr = builder.String()
		case *CallBlock:
			builder := strings.Builder{}
			str := strconv.Itoa(currentBlockNum) + "->" + strconv.Itoa(n.BlockNumber())
			builder.WriteString(str)
			if branchType == TrueBranch {
				builder.WriteString("[label = \"true\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			} else if branchType == FalseBranch {
				builder.WriteString("[label = \"false\" labelStyle = \"fill: #f77; font-weight: bold;\"];")
			}
			builder.WriteString("\n")
			branchType = UnconditionBranch
			nodeStr = builder.String()

		}

	}
	return
}
