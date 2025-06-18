package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

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
	Name string
	Body string
}

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

// Separar JSX del código JavaScript/TypeScript
func (t *Transpiler) separateJSXFromCode(code string) (string, string, error) {
	start := strings.Index(code, "return (")
	if start == -1 {
		return code, "", nil
	}

	start += len("return (")
	count := 1
	end := start

	for end < len(code) && count > 0 {
		if code[end] == '(' {
			count++
		} else if code[end] == ')' {
			count--
		}
		end++
	}

	if count != 0 {
		return "", "", fmt.Errorf("no se pudo emparejar los paréntesis del JSX")
	}

	jsxContent := strings.TrimSpace(code[start : end-1])
	jsCode := strings.TrimSpace(code[:start-len("return (")] + code[end:])
	return jsCode, jsxContent, nil
}

// Parsear el código React (JavaScript/TypeScript)
func (t *Transpiler) parseReactCode(jsCode string) (*ReactComponent, error) {
	component := &ReactComponent{}

	// Extraer imports
	component.Imports = t.extractImports(jsCode)

	// Extraer props usando regex
	component.Props = t.extractProps(jsCode)

	// Extraer states
	component.States = t.extractStates(jsCode)

	// Extraer effects
	component.Effects = t.extractEffects(jsCode)

	// Extraer funciones (excluyendo el componente principal)
	component.Functions = t.extractFunctions(jsCode)

	// Extraer nombre del componente
	component.Name = t.extractComponentName(jsCode)

	return component, nil
}

// Extraer imports
func (t *Transpiler) extractImports(code string) []string {
	importRegex := regexp.MustCompile(`import\s+.*?from\s+['"].*?['"]`)
	matches := importRegex.FindAllString(code, -1)
	return matches
}

// Extraer props
func (t *Transpiler) extractProps(code string) []PropDefinition {
	var props []PropDefinition

	// Buscar destructuring de props: const { prop1, prop2 } = props
	propsRegex := regexp.MustCompile(`const\s*{\s*([^}]+)\s*}\s*=\s*props`)
	matches := propsRegex.FindStringSubmatch(code)

	if len(matches) > 1 {
		propsList := strings.Split(matches[1], ",")
		for _, prop := range propsList {
			prop = strings.TrimSpace(prop)

			// Manejar props con valores por defecto: prop = defaultValue
			if strings.Contains(prop, "=") {
				parts := strings.Split(prop, "=")
				propName := strings.TrimSpace(parts[0])
				defaultValue := strings.TrimSpace(parts[1])
				props = append(props, PropDefinition{
					Name:         propName,
					Type:         "any",
					DefaultValue: defaultValue,
					Optional:     true,
				})
			} else {
				props = append(props, PropDefinition{
					Name:     prop,
					Type:     "any",
					Optional: false,
				})
			}
		}
	}

	// También buscar interface/type para props
	interfaceRegex := regexp.MustCompile(`(?:interface|type)\s+(\w+Props)\s*{\s*([^}]+)\s*}`)
	interfaceMatches := interfaceRegex.FindStringSubmatch(code)

	if len(interfaceMatches) > 2 {
		propsBody := interfaceMatches[2]
		propLines := strings.Split(propsBody, "\n")

		for _, line := range propLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "//") {
				continue
			}

			// Parsear prop: name: type; o name?: type;
			propRegex := regexp.MustCompile(`(\w+)(\??):\s*([^;]+)`)
			propMatch := propRegex.FindStringSubmatch(line)

			if len(propMatch) > 3 {
				props = append(props, PropDefinition{
					Name:     propMatch[1],
					Type:     strings.TrimSpace(propMatch[3]),
					Optional: propMatch[2] == "?",
				})
			}
		}
	}

	return props
}

// Extraer states usando useState
func (t *Transpiler) extractStates(code string) []StateDefinition {
	var states []StateDefinition

	// Buscar useState hooks: const [stateName, setStateName] = useState(initialValue)
	stateRegex := regexp.MustCompile(`const\s*\[\s*(\w+)\s*,\s*set\w+\s*\]\s*=\s*useState\s*\(\s*([^)]*)\s*\)`)
	matches := stateRegex.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 2 {
			states = append(states, StateDefinition{
				Name:         match[1],
				Type:         "any",
				InitialValue: match[2],
			})
		}
	}

	return states
}

// Extraer useEffect hooks
func (t *Transpiler) extractEffects(code string) []EffectDefinition {
	var effects []EffectDefinition

	// Buscar useEffect: useEffect(() => { ... }, [dependencies])
	effectRegex := regexp.MustCompile(`useEffect\s*\(\s*\(\s*\)\s*=>\s*{\s*([\s\S]*?)\s*}\s*,\s*\[\s*([^\]]*)\s*\]\s*\)`)
	matches := effectRegex.FindAllStringSubmatch(code, -1)

	for _, match := range matches {
		if len(match) > 2 {
			deps := []string{}
			if match[2] != "" {
				depsList := strings.Split(match[2], ",")
				for _, dep := range depsList {
					deps = append(deps, strings.TrimSpace(dep))
				}
			}

			effects = append(effects, EffectDefinition{
				Dependencies: deps,
				Body:         match[1],
			})
		}
	}

	return effects
}

// Extraer funciones
func (t *Transpiler) extractFunctions(code string) []FunctionDefinition {
	var functions []FunctionDefinition

	// Buscar funciones arrow dentro del componente: const functionName = () => { ... }
	// Pero excluir el componente principal
	funcRegex := regexp.MustCompile(`const\s+(\w+)\s*=\s*\([^)]*\)\s*=>\s*{([\s\S]*?)}`)
	matches := funcRegex.FindAllStringSubmatch(code, -1)

	componentName := t.extractComponentName(code)

	for _, match := range matches {
		if len(match) > 2 {
			funcName := match[1]
			funcBody := match[2]

			// Excluir el componente principal y funciones que contengan hooks
			if funcName != componentName &&
				!strings.Contains(funcBody, "useState") &&
				!strings.Contains(funcBody, "useEffect") {

				// Limpiar el cuerpo de la función de hooks de React
				cleanBody := t.cleanFunctionBody(funcBody)
				functions = append(functions, FunctionDefinition{
					Name: funcName,
					Body: cleanBody,
				})
			}
		}
	}

	return functions
}

// Limpiar el cuerpo de funciones de código React
func (t *Transpiler) cleanFunctionBody(body string) string {
	// Remover useState calls
	body = regexp.MustCompile(`const\s*\[[^]]+\]\s*=\s*useState\([^)]*\);\s*`).ReplaceAllString(body, "")

	// Convertir setStateName calls a assignments
	// setCount(value) -> count = value
	setStateRegex := regexp.MustCompile(`set([A-Z]\w*)\(([^)]+)\)`)
	body = setStateRegex.ReplaceAllStringFunc(body, func(match string) string {
		submatches := setStateRegex.FindStringSubmatch(match)
		if len(submatches) > 2 {
			stateName := strings.ToLower(string(submatches[1][0])) + submatches[1][1:]
			value := submatches[2]
			return fmt.Sprintf("%s = %s", stateName, value)
		}
		return match
	})

	// Limpiar líneas vacías extra
	lines := strings.Split(body, "\n")
	var cleanLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// Extraer nombre del componente
func (t *Transpiler) extractComponentName(code string) string {
	// Buscar export default function ComponentName
	nameRegex := regexp.MustCompile(`(?:export\s+default\s+)?function\s+(\w+)`)
	matches := nameRegex.FindStringSubmatch(code)

	if len(matches) > 1 {
		return matches[1]
	}

	// Buscar const ComponentName = () =>
	arrowRegex := regexp.MustCompile(`(?:export\s+)?const\s+(\w+)\s*=\s*\([^)]*\)\s*=>`)
	arrowMatches := arrowRegex.FindStringSubmatch(code)

	if len(arrowMatches) > 1 {
		return arrowMatches[1]
	}

	return "Component"
}

// Procesar JSX y convertirlo a sintaxis de Svelte
func (t *Transpiler) processJSX(jsx string) (string, error) {
	if jsx == "" {
		return "", nil
	}

	// Convertir sintaxis de JSX a Svelte
	processed := jsx

	// Convertir className a class
	processed = regexp.MustCompile(`className=`).ReplaceAllString(processed, `class=`)

	// Convertir onClick a onclick (y otros eventos)
	eventMap := map[string]string{
		"onClick":     "onclick",
		"onChange":    "onchange",
		"onSubmit":    "onsubmit",
		"onFocus":     "onfocus",
		"onBlur":      "onblur",
		"onKeyDown":   "onkeydown",
		"onKeyUp":     "onkeyup",
		"onMouseOver": "onmouseover",
		"onMouseOut":  "onmouseout",
	}

	for reactEvent, svelteEvent := range eventMap {
		processed = regexp.MustCompile(reactEvent+`=`).ReplaceAllString(processed, svelteEvent+`=`)
	}

	// Convertir fragmentos <React.Fragment> o <> a elementos div (simplificación)
	processed = regexp.MustCompile(`<React\.Fragment[^>]*>`).ReplaceAllString(processed, `<div>`)
	processed = regexp.MustCompile(`</React\.Fragment>`).ReplaceAllString(processed, `</div>`)
	processed = regexp.MustCompile(`<>`).ReplaceAllString(processed, `<div>`)
	processed = regexp.MustCompile(`</>`).ReplaceAllString(processed, `</div>`)

	return processed, nil
}

// Generar código Svelte final
func (t *Transpiler) generateSvelteCode(component *ReactComponent, processedJSX string) string {
	var result strings.Builder

	// Script tag
	result.WriteString("<script lang=\"ts\">\n")

	// Props
	if len(component.Props) > 0 {
		result.WriteString("  // Props\n")
		result.WriteString("  type Props = {\n")
		for _, prop := range component.Props {
			optional := ""
			if prop.Optional {
				optional = "?"
			}
			result.WriteString(fmt.Sprintf("    %s%s: %s;\n", prop.Name, optional, prop.Type))
		}
		result.WriteString("  };\n")

		// Destructuring de props con valores por defecto
		result.WriteString("  let { ")
		propNames := make([]string, len(component.Props))
		for i, prop := range component.Props {
			if prop.DefaultValue != "" {
				propNames[i] = fmt.Sprintf("%s = %s", prop.Name, prop.DefaultValue)
			} else {
				propNames[i] = prop.Name
			}
		}
		result.WriteString(strings.Join(propNames, ", "))
		result.WriteString(" }: Props = $props();\n\n")
	}

	// States
	if len(component.States) > 0 {
		result.WriteString("  // States\n")
		for _, state := range component.States {
			result.WriteString(fmt.Sprintf("  let %s = $state(%s);\n", state.Name, state.InitialValue))
		}
		result.WriteString("\n")
	}

	// Functions
	if len(component.Functions) > 0 {
		result.WriteString("  // Functions\n")
		for _, fn := range component.Functions {
			result.WriteString(fmt.Sprintf("  function %s() {\n", fn.Name))
			// Indentar el cuerpo de la función
			bodyLines := strings.Split(strings.TrimSpace(fn.Body), "\n")
			for _, line := range bodyLines {
				trimmedLine := strings.TrimSpace(line)
				if trimmedLine != "" {
					result.WriteString(fmt.Sprintf("    %s\n", trimmedLine))
				}
			}
			result.WriteString("  }\n\n")
		}
	}

	// Effects (convertir a $effect)
	if len(component.Effects) > 0 {
		result.WriteString("  // Effects\n")
		for _, effect := range component.Effects {
			result.WriteString("  $effect(() => {\n")
			// Indentar el cuerpo del efecto
			bodyLines := strings.Split(strings.TrimSpace(effect.Body), "\n")
			for _, line := range bodyLines {
				trimmedLine := strings.TrimSpace(line)
				if trimmedLine != "" {
					result.WriteString(fmt.Sprintf("    %s\n", trimmedLine))
				}
			}
			result.WriteString("  });\n\n")
		}
	}

	result.WriteString("</script>\n\n")

	// HTML (JSX procesado)
	if processedJSX != "" {
		result.WriteString(processedJSX)
		result.WriteString("\n")
	}

	return result.String()
}

// Función de ejemplo para probar el transpilador
func main() {
	transpiler := NewTranspiler()

	// Cargar input desde un archivo
	fileContent, err := os.ReadFile("input.tsx")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error leyendo archivo: %v\n", err)
		return
	}

	svelteCode, err := transpiler.TranspileComponent(string(fileContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	// Save to file
	err = os.WriteFile("output.svelte", []byte(svelteCode), 0644)
	if err != nil {
		fmt.Printf("Error escribiendo archivo: %v\n", err)
		return
	}

}
