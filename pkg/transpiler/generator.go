package transpiler

import (
	"fmt"
	"strings"
)

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
			asyncPrefix := ""
			if fn.Async {
				asyncPrefix = "async "
			}
			result.WriteString(fmt.Sprintf("  %sfunction %s() {\n", asyncPrefix, fn.Name))

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
