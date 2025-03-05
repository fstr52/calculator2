package parser

import (
	"final3/internal/models"
	"final3/pkg/stack"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var operatorPriority = map[rune]int{
	'(': 0,
	')': 0,
	'+': 1,
	'-': 1,
	'*': 2,
	'/': 2,
}

func ParseExpression(expression string) (map[int][]*models.Node, int, error) {
	poland, err := toPostfix(expression)
	if err != nil {
		return nil, 0, err
	}

	return createDAG(poland)
}

// Перевод выражения в постфиксную запись
func toPostfix(expression string) ([]string, error) {
	output := make([]string, 0)
	operatorStack := stack.NewStack[rune]()
	var curNum strings.Builder
	openBrackets := 0

	for _, n := range expression {
		if unicode.IsDigit(n) || (curNum.Len() > 0 && (n == '.' || n == ',')) {
			if n == '.' {
				n = ','
			}

			curNum.WriteRune(n)
			continue
		}

		if curNum.Len() > 0 {
			output = append(output, curNum.String())
			curNum.Reset()
		}

		if priority, ok := operatorPriority[n]; ok {
			if n == '(' {
				openBrackets++
				operatorStack.Push(n)
				continue
			}

			if n == ')' {
				openBrackets--
				for !operatorStack.IsEmpty() {
					op, _ := operatorStack.Pop()
					if op == '(' {
						break
					}
					output = append(output, string(op))
				}
				continue
			}

			for !operatorStack.IsEmpty() {
				topOp := operatorStack.Peek()
				if topOp != '(' && operatorPriority[topOp] >= priority {
					op, _ := operatorStack.Pop()
					output = append(output, string(op))
				} else {
					break
				}
			}

			operatorStack.Push(n)
		}
	}

	if curNum.Len() > 0 {
		output = append(output, curNum.String())
	}

	if openBrackets != 0 {
		return nil, fmt.Errorf("brackets aren't balanced")
	}

	for !operatorStack.IsEmpty() {
		op, _ := operatorStack.Pop()
		if op == '(' {
			return nil, fmt.Errorf("mismatched brackets")
		}
		output = append(output, string(op))
	}

	return output, nil
}

// Создание DAG записи
func createDAG(poland []string) (map[int][]*models.Node, int, error) {
	dagStack := stack.NewStack[*models.Node]()
	levelMap := make(map[int][]*models.Node)
	var maxLevel int

	for _, n := range poland {
		_, err := strconv.ParseFloat(n, 64)
		if err == nil {
			node := models.NewNode(n)
			node.Type = models.Number
			node.Level = 0
			dagStack.Push(node)
			levelMap[node.Level] = append(levelMap[node.Level], node)
			continue
		}

		rightOperand, ok := dagStack.Pop()
		if !ok {
			return nil, 0, fmt.Errorf("stack.pop not ok")
		}

		leftOperand, ok := dagStack.Pop()
		if !ok {
			return nil, 0, fmt.Errorf("stack.pop not ok")
		}

		node := models.NewNode(n)
		node.Type = models.Operator
		node.Level = max(rightOperand.Level, leftOperand.Level) + 1
		if maxLevel < node.Level {
			maxLevel = node.Level
		}
		node.Dependencies = []*models.Node{leftOperand, rightOperand}

		dagStack.Push(node)
		levelMap[node.Level] = append(levelMap[node.Level], node)
	}

	if dagStack.Len() != 1 {
		return nil, 0, fmt.Errorf("invalid expression: expected 1 root node, got %d", dagStack.Len())
	}

	return levelMap, maxLevel, nil
}
