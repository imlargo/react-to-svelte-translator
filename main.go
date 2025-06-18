package main

import (
	"fmt"
	"os"

	"github.com/imlargo/react-svelte-transpiler/pkg/transpiler"
)

// Función de ejemplo para probar el transpilador
func main() {
	transpiler := transpiler.NewTranspiler()

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
