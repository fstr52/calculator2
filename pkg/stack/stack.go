package stack

type Stack[Type any] struct {
	elements []Type
}

// Создать новый стэк заданного типа
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{
		elements: make([]T, 0),
	}
}

// Занести элемент в стэк
func (s *Stack[Type]) Push(element Type) {
	s.elements = append(s.elements, element)
}

// Получить элемент стэка
//
// - Возвращает значение и перепенную, отвечающую за успешность операции
func (s *Stack[Type]) Pop() (Type, bool) {
	var zero Type
	if len(s.elements) == 0 {
		return zero, false
	}

	element := s.elements[len(s.elements)-1]
	s.elements = s.elements[:len(s.elements)-1]
	return element, true
}

// Получить самый первый элемент стэка
func (s *Stack[Type]) Peek() Type {
	element := s.elements[len(s.elements)-1]
	return element
}

// Получить список элементов стэка
func (s *Stack[Type]) Elements() []Type {
	result := make([]Type, 0, len(s.elements))
	for i := len(s.elements); i >= 0; i-- {
		n := s.elements[i]
		result = append(result, n)
	}
	return result
}

// Возрвращает длину стэка
func (s *Stack[Type]) Len() int {
	return len(s.elements)
}

// Проверить стэк на пустоту
func (s *Stack[Type]) IsEmpty() bool {
	return len(s.elements) == 0
}
