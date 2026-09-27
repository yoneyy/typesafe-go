package typesafe

type Pandora interface {
	i()
}

type (
	String string
	Object map[string]any
	Array  []any
)

type (
	TArray[V any]                []V
	TObject[K comparable, V any] map[K]V
)

func (String) i()        {}
func (Object) i()        {}
func (Array) i()         {}
func (TArray[V]) i()     {}
func (TObject[K, V]) i() {}
