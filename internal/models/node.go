package models

type NodeType string

const (
	Number   NodeType = "number"
	Operator NodeType = "operator"
)

type NodeStatus string

const (
	StatusCreated  NodeStatus = "just_created"
	StatusInQueue  NodeStatus = "in_queue"
	StatusAtWorker NodeStatus = "at_worker"
	StatusDone     NodeStatus = "done"
)

type Node struct {
	ExpressionID int32
	Type         NodeType
	Value        string
	Dependencies []*Node
	Level        int
	Status       NodeStatus
}

func NewNode(value string) *Node {
	return &Node{
		Value:        value,
		Dependencies: make([]*Node, 0),
		Status:       StatusCreated,
	}
}
