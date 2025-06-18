package transpiler

// Estructuras para representar el componente React
type ReactComponent struct {
	Name       string
	Props      []PropDefinition
	States     []StateDefinition
	Effects    []EffectDefinition
	Functions  []FunctionDefinition
	JSXContent string
	Imports    []string
}

type PropDefinition struct {
	Name         string
	Type         string
	DefaultValue string
	Optional     bool
}

type StateDefinition struct {
	Name         string
	Type         string
	InitialValue string
}

type EffectDefinition struct {
	Dependencies []string
	Body         string
}

type FunctionDefinition struct {
	Name  string
	Body  string
	Async bool
}
