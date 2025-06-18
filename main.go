package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
)

type Transpiler struct {
	fset *token.FileSet
}

type ComponentInfo struct {
	Name       string
	Props      []PropInfo
	StateVars  []StateVar
	Effects    []string
	JSXContent string
}

type PropInfo struct {
	Name string
	Type string
}

type StateVar struct {
	Name         string
	InitialValue string
	SetterName   string
}

func NewTranspiler() *Transpiler {
	return &Transpiler{
		fset: token.NewFileSet(),
	}
}

func (t *Transpiler) TranspileReactToSvelte(reactCode string) (string, error) {
	// Preprocesar el código para manejar JSX
	processedCode := t.preprocessJSX(reactCode)

	// Parsear el código JavaScript/React como Go (aproximación)
	node, err := parser.ParseFile(t.fset, "", processedCode, parser.ParseComments)
	if err != nil {
		// Si falla el parsing de Go, usar regex para extraer información
		return t.fallbackTranspile(reactCode)
	}

	component := t.extractComponentInfo(node, reactCode)
	return t.generateSvelteCode(component), nil
}

func (t *Transpiler) preprocessJSX(code string) string {
	// Remover imports de React
	code = regexp.MustCompile(`import\s+.*?from\s+['"]react['"];?\s*`).ReplaceAllString(code, "")

	// Convertir JSX a comentarios para evitar errores de parsing
	jsxRegex := regexp.MustCompile(`return\s*\(([\s\S]*?)\);`)
	code = jsxRegex.ReplaceAllStringFunc(code, func(match string) string {
		return "/* JSX_CONTENT_START " + match + " JSX_CONTENT_END */"
	})

	return code
}

func (t *Transpiler) fallbackTranspile(reactCode string) (string, error) {
	component := ComponentInfo{}

	// Extraer nombre del componente
	componentRegex := regexp.MustCompile(`(?:function|const)\s+(\w+)\s*(?:\(|\=)`)
	if matches := componentRegex.FindStringSubmatch(reactCode); len(matches) > 1 {
		component.Name = matches[1]
	}

	// Extraer interface de props (TypeScript)
	interfaceRegex := regexp.MustCompile(`interface\s+(\w+Props?|\w+)\s*\{([^}]+)\}`)
	var propsInterface string
	if matches := interfaceRegex.FindStringSubmatch(reactCode); len(matches) > 2 {
		propsInterface = matches[2]
	}

	// También buscar tipos inline
	typeRegex := regexp.MustCompile(`(?:function\s+\w+\s*\(\s*\{\s*([^}]+)\s*\}\s*:\s*\{([^}]+)\}|const\s+\w+\s*:\s*React\.FC<\{([^}]+)\}>|\(\s*\{\s*([^}]+)\s*\}\s*:\s*\{([^}]+)\})`)
	typeMatches := typeRegex.FindStringSubmatch(reactCode)

	// Procesar props con tipos
	if propsInterface != "" {
		component.Props = t.parsePropsFromInterface(propsInterface)
	} else if len(typeMatches) > 0 {
		// Extraer de tipos inline
		var propsStr, typesStr string
		for i := 1; i < len(typeMatches); i += 2 {
			if typeMatches[i] != "" {
				propsStr = typeMatches[i]
				if i+1 < len(typeMatches) {
					typesStr = typeMatches[i+1]
				}
				break
			}
		}
		component.Props = t.parsePropsWithTypes(propsStr, typesStr)
	} else {
		// Fallback: buscar destructuring simple
		propsRegex := regexp.MustCompile(`\{\s*([^}]+)\s*\}`)
		if matches := propsRegex.FindStringSubmatch(reactCode); len(matches) > 1 {
			propNames := strings.Split(matches[1], ",")
			for _, name := range propNames {
				name = strings.TrimSpace(name)
				if name != "" {
					component.Props = append(component.Props, PropInfo{
						Name: name,
						Type: "any",
					})
				}
			}
		}
	}

	// Extraer useState con tipos
	useStateRegex := regexp.MustCompile(`const\s*\[(\w+),\s*(\w+)\]\s*=\s*useState(?:<([^>]+)>)?\s*\(([^)]*)\)`)
	matches := useStateRegex.FindAllStringSubmatch(reactCode, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			stateVar := StateVar{
				Name:         match[1],
				SetterName:   match[2],
				InitialValue: strings.Trim(match[4], `"'`),
			}
			component.StateVars = append(component.StateVars, stateVar)
		}
	}

	// Extraer JSX del return
	returnRegex := regexp.MustCompile(`return\s*\(([\s\S]*?)\);`)
	if matches := returnRegex.FindStringSubmatch(reactCode); len(matches) > 1 {
		component.JSXContent = strings.TrimSpace(matches[1])
	}

	return t.generateSvelteCode(component), nil
}

func (t *Transpiler) parsePropsFromInterface(interfaceContent string) []PropInfo {
	var props []PropInfo

	// Limpiar y dividir por líneas
	lines := strings.Split(interfaceContent, ";")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parsear propName: type
		propRegex := regexp.MustCompile(`(\w+)(\?)?\s*:\s*([^;,]+)`)
		if matches := propRegex.FindStringSubmatch(line); len(matches) >= 4 {
			propType := strings.TrimSpace(matches[3])
			props = append(props, PropInfo{
				Name: matches[1],
				Type: propType,
			})
		}
	}

	return props
}

func (t *Transpiler) parsePropsWithTypes(propsStr, typesStr string) []PropInfo {
	var props []PropInfo

	propNames := strings.Split(propsStr, ",")
	typeDecls := strings.Split(typesStr, ";")

	// Crear mapa de tipos
	typeMap := make(map[string]string)
	for _, typeDecl := range typeDecls {
		typeRegex := regexp.MustCompile(`(\w+)(\?)?\s*:\s*([^;,]+)`)
		if matches := typeRegex.FindStringSubmatch(typeDecl); len(matches) >= 4 {
			typeMap[matches[1]] = strings.TrimSpace(matches[3])
		}
	}

	// Asignar tipos a props
	for _, name := range propNames {
		name = strings.TrimSpace(name)
		if name != "" {
			propType := "any"
			if t, exists := typeMap[name]; exists {
				propType = t
			}
			props = append(props, PropInfo{
				Name: name,
				Type: propType,
			})
		}
	}

	return props
}

func (t *Transpiler) extractComponentInfo(node *ast.File, originalCode string) ComponentInfo {
	// Esta es una implementación simplificada
	// En una implementación real, necesitarías un parser JSX completo

	// Por ahora, usar el método fallback
	fallbackResult, _ := t.fallbackTranspile(originalCode)
	return ComponentInfo{JSXContent: fallbackResult}
}

func (t *Transpiler) generateSvelteCode(component ComponentInfo) string {
	var svelteCode strings.Builder

	// Script section con TypeScript
	svelteCode.WriteString("<script lang=\"ts\">\n")

	// Props usando la nueva sintaxis de Svelte 5
	if len(component.Props) > 0 {
		// Definir el tipo Props
		svelteCode.WriteString("  type Props = {\n")
		for _, prop := range component.Props {
			svelteCode.WriteString(fmt.Sprintf("    %s: %s;\n", prop.Name, prop.Type))
		}
		svelteCode.WriteString("  };\n\n")

		// Destructuring con $props()
		propNames := make([]string, len(component.Props))
		for i, prop := range component.Props {
			propNames[i] = prop.Name
		}
		svelteCode.WriteString(fmt.Sprintf("  const { %s }: Props = $props();\n\n", strings.Join(propNames, ", ")))
	}

	// State variables (usando $state en Svelte 5)
	for _, stateVar := range component.StateVars {
		initialValue := stateVar.InitialValue
		if initialValue == "" {
			initialValue = "''"
		}
		svelteCode.WriteString(fmt.Sprintf("  let %s = $state(%s);\n", stateVar.Name, initialValue))
	}

	if len(component.StateVars) > 0 {
		svelteCode.WriteString("\n")
	}

	svelteCode.WriteString("</script>\n\n")

	// Template section
	if component.JSXContent != "" {
		htmlContent := t.convertJSXToSvelte(component.JSXContent, component.StateVars)
		svelteCode.WriteString(htmlContent)
	}

	return svelteCode.String()
}

func (t *Transpiler) convertJSXToSvelte(jsx string, stateVars []StateVar) string {
	// Remover el return y paréntesis
	jsx = regexp.MustCompile(`^\s*return\s*\(\s*`).ReplaceAllString(jsx, "")
	jsx = regexp.MustCompile(`\s*\)\s*;?\s*$`).ReplaceAllString(jsx, "")

	// Convertir className a class
	jsx = regexp.MustCompile(`className=`).ReplaceAllString(jsx, "class=")

	// Convertir eventos onClick a on:click
	jsx = regexp.MustCompile(`onClick=\{([^}]+)\}`).ReplaceAllString(jsx, "onclick={$1}")
	jsx = regexp.MustCompile(`onChange=\{([^}]+)\}`).ReplaceAllString(jsx, "onchange={$1}")
	jsx = regexp.MustCompile(`onSubmit=\{([^}]+)\}`).ReplaceAllString(jsx, "onsubmit={$1}")

	// Convertir llamadas a setters de estado
	for _, stateVar := range stateVars {
		// Convertir setter(value) a variable = value
		setterPattern := regexp.MustCompile(fmt.Sprintf(`%s\s*\(\s*([^)]+)\s*\)`, stateVar.SetterName))
		jsx = setterPattern.ReplaceAllString(jsx, fmt.Sprintf("%s = $1", stateVar.Name))
	}

	// Convertir interpolaciones {variable} (ya están en formato correcto)

	// Limpiar espacios extra
	jsx = regexp.MustCompile(`\s+`).ReplaceAllString(jsx, " ")
	jsx = strings.TrimSpace(jsx)

	return jsx
}

// Ejemplos de uso
func main() {
	transpiler := NewTranspiler()
	// Leer el archivo example.tsx
	content, err := os.ReadFile("example.tsx")
	if err != nil {
		fmt.Printf("Error leyendo example.tsx: %v\n", err)
		return
	}

	reactCode := string(content)
	fmt.Println("React Code from example.tsx:")
	fmt.Println(reactCode)

	svelteCode, err := transpiler.TranspileReactToSvelte(reactCode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nSvelte 5 Code:")
	fmt.Println(svelteCode)

	// Guardar el código Svelte en un archivo
	err = os.WriteFile("output.svelte", []byte(svelteCode), 0644)
	if err != nil {
		fmt.Printf("Error escribiendo output.svelte: %v\n", err)
		return
	}
	fmt.Println("\nCódigo Svelte guardado en output.svelte")
}
