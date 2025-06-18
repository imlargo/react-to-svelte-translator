package transpiler

import (
	"fmt"
)

// Transpilador principal
type Transpiler struct{}

func NewTranspiler() *Transpiler {
	return &Transpiler{}
}

// Función principal de transpilación
func (t *Transpiler) TranspileComponent(reactCode string) (string, error) {
	// Separar el código JSX del código JavaScript/TypeScript
	jsCode, jsxContent, err := t.separateJSXFromCode(reactCode)
	if err != nil {
		return "", fmt.Errorf("error separando JSX del código: %v", err)
	}

	// Parsear el código JavaScript/TypeScript
	component, err := t.parseReactCode(jsCode)
	if err != nil {
		return "", fmt.Errorf("error parseando código React: %v", err)
	}

	fmt.Println(jsxContent)

	// Procesar el JSX
	component.JSXContent = jsxContent
	processedJSX, err := t.processJSX(jsxContent)
	if err != nil {
		return "", fmt.Errorf("error procesando JSX: %v", err)
	}

	// Generar código Svelte
	svelteCode := t.generateSvelteCode(component, processedJSX)
	return svelteCode, nil
}
