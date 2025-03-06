package parser

import (
	"final3/internal/models"
	"testing"
)

func TestParseExpression(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expectError bool
		maxLevel    int
	}{
		{
			name:        "Simple Addition",
			expression:  "2+3",
			expectError: false,
			maxLevel:    1,
		},
		{
			name:        "Complex Expression",
			expression:  "2+3*4",
			expectError: false,
			maxLevel:    2,
		},
		{
			name:        "Expression With Brackets",
			expression:  "(2+3)*4",
			expectError: false,
			maxLevel:    2,
		},
		{
			name:        "Decimal Numbers",
			expression:  "2.5+3.7",
			expectError: false,
			maxLevel:    1,
		},
		{
			name:        "Unbalanced Brackets",
			expression:  "2+(3*4",
			expectError: true,
			maxLevel:    0,
		},
		{
			name:        "Complex Expression With Multiple Operations",
			expression:  "((2+3)*(4-1))/5",
			expectError: false,
			maxLevel:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levelMap, maxLevel, err := ParseExpression(tt.expression)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if maxLevel != tt.maxLevel {
					t.Errorf("Expected maxLevel %d but got %d", tt.maxLevel, maxLevel)
				}

				if len(levelMap) != maxLevel+1 {
					t.Errorf("Expected %d levels in levelMap but got %d", maxLevel+1, len(levelMap))
				}

				if len(levelMap[maxLevel]) != 1 {
					t.Errorf("Expected 1 node at top level but got %d", len(levelMap[maxLevel]))
				}

				for level, nodes := range levelMap {
					for _, node := range nodes {
						if level == 0 {
							if node.Type != models.Number {
								t.Errorf("Expected node at level 0 to be Number, got %v", node.Type)
							}
						} else {
							if node.Type != models.Operator {
								t.Errorf("Expected node at level %d to be Operator, got %v", level, node.Type)
							}

							if len(node.Dependencies) != 2 {
								t.Errorf("Expected operator node to have 2 dependencies, got %d", len(node.Dependencies))
							}

							for _, dep := range node.Dependencies {
								if dep.Level > level-1 {
									t.Errorf("Invalid dependency level: node level %d, dependency level %d", level, dep.Level)
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestParseExpressionIntegration(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		nodeCount  int
	}{
		{
			name:       "Simple Addition",
			expression: "2+3",
			nodeCount:  3,
		},
		{
			name:       "Complex Expression",
			expression: "2+3*4",
			nodeCount:  5,
		},
		{
			name:       "Expression With Brackets",
			expression: "(2+3)*4",
			nodeCount:  5,
		},
		{
			name:       "Complex Expression With Multiple Brackets",
			expression: "((2+3)*(4-1))/5",
			nodeCount:  9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levelMap, _, err := ParseExpression(tt.expression)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			totalNodes := 0
			for _, nodes := range levelMap {
				totalNodes += len(nodes)
			}

			if totalNodes != tt.nodeCount {
				t.Errorf("Expected %d total nodes, got %d", tt.nodeCount, totalNodes)
			}

			checkDAGStructure(t, levelMap)
		})
	}
}

func checkDAGStructure(t *testing.T, levelMap map[int][]*models.Node) {
	for level, nodes := range levelMap {
		for _, node := range nodes {
			if node.Type == models.Operator {
				if len(node.Dependencies) != 2 {
					t.Errorf("Operator node at level %d has %d dependencies, expected 2",
						level, len(node.Dependencies))
				}

				for _, dep := range node.Dependencies {
					if dep.Level >= level {
						t.Errorf("Invalid dependency level: node level %d, dependency level %d",
							level, dep.Level)
					}
				}
			}
		}
	}

	for _, node := range levelMap[0] {
		if node.Type != models.Number {
			t.Errorf("Expected all nodes at level 0 to be numbers, found %v", node.Type)
		}
	}
}
