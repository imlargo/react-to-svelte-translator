package transpiler

import (
	"fmt"
	"regexp"
	"strings"
)

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

	var cleanImports []string
	for _, match := range matches {
		isReactImport := strings.Contains(match, "react") || strings.Contains(match, "React")
		isNextImport := false
		if strings.Contains(match, "next") || strings.Contains(match, "Next") {
			nextImportRegex := regexp.MustCompile(`from\s+['"]next[\/\w-]*['"]`)
			isNextImport = nextImportRegex.MatchString(match)
		}

		if !isReactImport && !isNextImport {
			cleanImports = append(cleanImports, match)
		}
	}

	return cleanImports
}
func (t *Transpiler) extractProps(code string) []PropDefinition {
	var props []PropDefinition
	propsMap := make(map[string]bool) // Para evitar duplicados

	// 1. Destructuring directo en parámetros de la función
	paramRegex := regexp.MustCompile(`function\s+\w+\s*\(\s*{\s*([^}]+)\s*}\s*(?::\s*(\w+Props))?\s*\)`)
	paramMatches := paramRegex.FindStringSubmatch(code)

	if len(paramMatches) > 1 {
		propsList := strings.Split(paramMatches[1], ",")
		for _, prop := range propsList {
			prop = strings.TrimSpace(prop)

			if prop == "" {
				continue
			}

			// Manejar default value
			if strings.Contains(prop, "=") {
				parts := strings.SplitN(prop, "=", 2)
				propName := strings.TrimSpace(parts[0])
				defaultValue := strings.TrimSpace(parts[1])
				props = append(props, PropDefinition{
					Name:         propName,
					Type:         "any",
					DefaultValue: defaultValue,
					Optional:     true,
				})
				propsMap[propName] = true
			} else {
				props = append(props, PropDefinition{
					Name:     prop,
					Type:     "any",
					Optional: false,
				})
				propsMap[prop] = true
			}
		}
	}

	// 2. Destructuring de props dentro del cuerpo: const { x, y = z } = props;
	propsRegex := regexp.MustCompile(`const\s*{\s*([^}]+)\s*}\s*=\s*props`)
	matches := propsRegex.FindStringSubmatch(code)

	if len(matches) > 1 {
		propsList := strings.Split(matches[1], ",")
		for _, prop := range propsList {
			prop = strings.TrimSpace(prop)

			if prop == "" {
				continue
			}

			if strings.Contains(prop, "=") {
				parts := strings.SplitN(prop, "=", 2)
				propName := strings.TrimSpace(parts[0])
				defaultValue := strings.TrimSpace(parts[1])
				if !propsMap[propName] {
					props = append(props, PropDefinition{
						Name:         propName,
						Type:         "any",
						DefaultValue: defaultValue,
						Optional:     true,
					})
					propsMap[propName] = true
				}
			} else {
				if !propsMap[prop] {
					props = append(props, PropDefinition{
						Name:     prop,
						Type:     "any",
						Optional: false,
					})
					propsMap[prop] = true
				}
			}
		}
	}

	// 3. Interface o type definition: interface MyProps { a: string; b?: number }
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

			// Ejemplo: name?: string;
			propRegex := regexp.MustCompile(`^(\w+)(\??):\s*([^;]+);?$`)
			propMatch := propRegex.FindStringSubmatch(line)

			if len(propMatch) > 3 {
				name := propMatch[1]
				optional := propMatch[2] == "?"
				typ := strings.TrimSpace(propMatch[3])

				if !propsMap[name] {
					props = append(props, PropDefinition{
						Name:     name,
						Type:     typ,
						Optional: optional,
					})
					propsMap[name] = true
				}
			}
		}
	}

	return props
}

// Extraer states usando useState
func (t *Transpiler) extractStates(code string) []StateDefinition {
	var states []StateDefinition

	// Buscar useState hooks: const [stateName, setStateName] = useState(initialValue)
	stateRegex := regexp.MustCompile(`const\s*\[\s*(\w+)\s*,\s*set\w+\s*\]\s*=\s*useState\s*\(\s*((?:[^()]|\([^()]*\))*)\s*\)`)
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
