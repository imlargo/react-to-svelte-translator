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

func (t *Transpiler) extractFunctions(code string) []FunctionDefinition {
	var functions []FunctionDefinition

	// Regex para detectar funciones flecha, con async opcional, parámetros con o sin paréntesis
	funcRegex := regexp.MustCompile(`const\s+(\w+)\s*=\s*(async\s*)?(?:\([^\)]*\)|\w+)\s*=>\s*{((?:[^{}]*|\{[^{}]*\})*)}`)

	matches := funcRegex.FindAllStringSubmatch(code, -1)

	// Obtener el nombre del componente principal para excluirlo
	componentName := t.extractComponentName(code)

	for _, match := range matches {
		if len(match) > 3 {
			funcName := match[1]
			isAsync := strings.TrimSpace(match[2]) == "async"
			funcBody := match[3]

			// Excluir componente principal y funciones con hooks
			if funcName != componentName &&
				!strings.Contains(funcBody, "useState") &&
				!strings.Contains(funcBody, "useEffect") {

				cleanBody := t.cleanFunctionBody(funcBody)

				functions = append(functions, FunctionDefinition{
					Name:  funcName,
					Body:  cleanBody,
					Async: isAsync,
				})
			}
		}
	}

	return functions
}

func (t *Transpiler) cleanFunctionBody(body string) string {
	// Eliminar useState
	body = regexp.MustCompile(`const\s*\[[^]]+\]\s*=\s*useState\([^)]*\);\s*`).ReplaceAllString(body, "")

	// Detectar y convertir setters
	setCalls := t.extractSetterCalls(body)
	for _, call := range setCalls {
		converted := t.convertSetterCall(call)
		body = strings.Replace(body, call, converted, 1)
	}

	// Limpiar líneas vacías
	lines := strings.Split(body, "\n")
	var cleanLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

func (t *Transpiler) convertSetterCall(call string) string {
	setStateRegex := regexp.MustCompile(`^set([A-Z]\w*)\s*\(\s*((?s).*?)\s*\)$`)
	submatches := setStateRegex.FindStringSubmatch(strings.TrimSpace(call))
	if len(submatches) < 3 {
		return call
	}

	setter := submatches[1]
	rawValue := strings.TrimSpace(submatches[2])
	stateVar := strings.ToLower(setter[:1]) + setter[1:]

	// Ver si el valor es una función flecha tipo (prev) => ({ ... })
	if strings.HasPrefix(rawValue, "(") && strings.Contains(rawValue, "=>") {
		// Separar en partes: (prev) => ({...})
		parts := strings.SplitN(rawValue, "=>", 2)
		body := strings.TrimSpace(parts[1])

		// Eliminar paréntesis externos si los hay
		if strings.HasPrefix(body, "(") && strings.HasSuffix(body, ")") {
			body = strings.TrimPrefix(body, "(")
			body = strings.TrimSuffix(body, ")")
		}

		// Reemplazar "prev" o argumento por el estado real
		body = strings.ReplaceAll(body, "prev", stateVar)

		return fmt.Sprintf("%s = %s", stateVar, strings.TrimSpace(body))
	}

	// Si no es función flecha, devolver como asignación directa
	return fmt.Sprintf("%s = %s", stateVar, rawValue)
}

func (t *Transpiler) extractSetterCalls(code string) []string {
	var results []string

	// Buscar todas las ocurrencias de "setX("
	baseRegex := regexp.MustCompile(`set([A-Z]\w*)\s*\(`)
	indexes := baseRegex.FindAllStringIndex(code, -1)

	for _, idx := range indexes {
		start := idx[0]
		i := idx[1] // posición después del paréntesis de apertura

		openParens := 1
		end := i

		for end < len(code) && openParens > 0 {
			c := code[end]
			switch c {
			case '(':
				openParens++
			case ')':
				openParens--
			}
			end++
		}

		if openParens == 0 {
			call := code[start:end]
			results = append(results, call)
		}
	}

	return results
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
	processed := t.replaceComments(jsx)

	// Convertir className a class
	processed = regexp.MustCompile(`className=`).ReplaceAllString(processed, `class=`)

	processed = t.deleteFragments(processed)

	processed = t.replaceEvents(processed)

	processed = t.replaceLoops(processed)

	processed = t.replaceConditionals(processed)

	return processed, nil
}

func (t *Transpiler) replaceLoops(jsx string) string {
	mapRegex := regexp.MustCompile(`\{\s*([a-zA-Z0-9_.$]+)\.map\s*\(\s*\(?\s*([a-zA-Z0-9_]+)\s*(?:,\s*([a-zA-Z0-9_]+))?\s*\)?\s*=>\s*\(\s*([\s\S]+?)\s*\)\s*\)\s*\}`)

	return mapRegex.ReplaceAllStringFunc(jsx, func(match string) string {
		m := mapRegex.FindStringSubmatch(match)
		if len(m) < 5 {
			return match
		}

		collection := strings.TrimSpace(m[1]) // ej: packages
		item := strings.TrimSpace(m[2])       // ej: pkg
		index := strings.TrimSpace(m[3])      // ej: i (opcional)
		body := strings.TrimSpace(m[4])       // JSX
		key := ""

		// Buscar key
		keyRegex := regexp.MustCompile(`key\s*=\s*\{([^}]+)\}`)
		if keyMatch := keyRegex.FindStringSubmatch(body); len(keyMatch) > 1 {
			key = strings.TrimSpace(keyMatch[1])
			body = keyRegex.ReplaceAllString(body, "") // eliminar key del JSX
		}

		indexPart := ""
		if index != "" {
			indexPart = ", " + index
		}

		keyPart := ""
		if key != "" {
			keyPart = fmt.Sprintf(" (%s)", key)
		}

		return fmt.Sprintf("\n{#each %s as %s%s%s}\n%s\n{/each}\n", collection, item, indexPart, keyPart, body)
	})
}

func (t *Transpiler) replaceConditionals(jsx string) string {
	// 1. Ternario
	ternaryRegex := regexp.MustCompile(`\{\s*([^{}?]+?)\s*\?\s*\(([\s\S]+?)\)\s*:\s*\(([\s\S]+?)\)\s*\}`)
	jsx = ternaryRegex.ReplaceAllString(jsx, "\n{#if $1}\n$2\n{:else}\n$3\n{/if}\n")

	// 2. && con paréntesis multilínea
	andBlockRegex := regexp.MustCompile(`\{\s*([^{}]+?)\s*&&\s*\(\s*([\s\S]+?)\s*\)\s*\}`)
	jsx = andBlockRegex.ReplaceAllStringFunc(jsx, func(match string) string {
		m := andBlockRegex.FindStringSubmatch(match)
		if len(m) < 3 {
			return match
		}
		return fmt.Sprintf("\n{#if %s}\n%s\n{/if}\n", strings.TrimSpace(m[1]), strings.TrimSpace(m[2]))
	})

	// 3. && con JSX inline (como <Shield ... /> o <Tag></Tag>)
	andInlineJSXRegex := regexp.MustCompile(`\{\s*([^{}]+?)\s*&&\s*(<[^>]+/?>)\s*\}`)
	jsx = andInlineJSXRegex.ReplaceAllStringFunc(jsx, func(match string) string {
		m := andInlineJSXRegex.FindStringSubmatch(match)
		if len(m) < 3 {
			return match
		}
		return fmt.Sprintf("\n{#if %s}\n%s\n{/if}\n", strings.TrimSpace(m[1]), strings.TrimSpace(m[2]))
	})

	return jsx
}

func (t *Transpiler) replaceEvents(jsx string) string {
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

	processed := jsx

	for reactEvent, svelteEvent := range eventMap {
		processed = regexp.MustCompile(reactEvent+`=`).ReplaceAllString(processed, svelteEvent+`=`)
	}

	return processed
}

func (t *Transpiler) deleteFragments(jsx string) string {
	// Convertir fragmentos <React.Fragment> o <> a elementos div (simplificación)
	processed := regexp.MustCompile(`<React\.Fragment[^>]*>`).ReplaceAllString(jsx, `<div>`)
	processed = regexp.MustCompile(`</React\.Fragment>`).ReplaceAllString(processed, `</div>`)
	processed = regexp.MustCompile(`<>`).ReplaceAllString(processed, `<div>`)
	processed = regexp.MustCompile(`</>`).ReplaceAllString(processed, `</div>`)

	return processed
}

func (t *Transpiler) replaceComments(jsx string) string {
	// 1. Convertir comentarios JSX a comentarios HTML
	jsxCommentRegex := regexp.MustCompile(`\{\s*/\*\s*(.*?)\s*\*/\s*\}`)
	jsx = jsxCommentRegex.ReplaceAllString(jsx, `<!-- $1 -->`)

	// 2. Eliminar comentarios de línea JS (// ...) que no están en JSX
	singleLineCommentRegex := regexp.MustCompile(`(?m)^\s*//.*$`)
	jsx = singleLineCommentRegex.ReplaceAllString(jsx, ``)

	// 3. Eliminar comentarios de bloque JS (/* ... */)
	blockCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	jsx = blockCommentRegex.ReplaceAllString(jsx, ``)

	return jsx
}
