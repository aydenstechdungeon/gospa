package compiler

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// PropsRegex matches GoSPA $props() declarations.
	PropsRegex = regexp.MustCompile(`(?m)var\s+\{\s*(.*?)\s*\}\s*=\s*\$props\(\)`)
	// StateRegex matches GoSPA $state() declarations.
	StateRegex = regexp.MustCompile(`(?m)(?:var|const)\s+([a-zA-Z0-9_]+)\s*=\s*\$state\((.*?)\)`)
	// DerivedRegex matches GoSPA $derived() declarations.
	DerivedRegex = regexp.MustCompile(`\$derived\((.*?)\)`)
	// EffectRegex matches GoSPA $effect() declarations.
	EffectRegex = regexp.MustCompile(`(?s)\$effect\(\s*func\(\)\s*\{(.*?)\}\s*\)`)
	// CSSDotRegex matches CSS class selectors.
	CSSDotRegex = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9-_]*)`)
	// ReactiveLabelRegex matches GoSPA $: reactive labels.
	ReactiveLabelRegex = regexp.MustCompile(`\$:\s*([a-zA-Z0-9_]+)\s*=\s*([^;\n]+)`)
	// CSSElementRegex matches CSS element selectors.
	CSSElementRegex = regexp.MustCompile(`(?m)^([a-z0-9]+)\s*\{`)
)

// Prop represents a component prop.
type Prop struct {
	Name string
	Type string
}

// State represents a component reactive state.
type State struct {
	Name         string
	InitialValue string
	Type         string
}

// ExtractTypes parses the script content to find props and state.
func ExtractTypes(script string) ([]Prop, []State) {
	var props []Prop
	var states []State

	// Extract Props
	if matches := PropsRegex.FindStringSubmatch(script); len(matches) > 1 {
		parts := strings.Split(matches[1], ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			f := strings.Fields(trimmed)
			if len(f) >= 2 {
				props = append(props, Prop{Name: f[0], Type: f[1]})
			} else if len(f) == 1 {
				props = append(props, Prop{Name: f[0], Type: "any"})
			}
		}
	}

	// Extract State
	stateMatches := StateRegex.FindAllStringSubmatch(script, -1)
	for _, m := range stateMatches {
		states = append(states, State{
			Name:         m[1],
			InitialValue: m[2],
			Type:         "any", // Inferring type from initial value would be better but complex for Go
		})
	}

	return props, states
}

// GenerateGoStruct generates a Go struct for the component props.
func GenerateGoStruct(name string, props []Prop) string {
	var sb strings.Builder
	sb.WriteString("type ")
	sb.WriteString(name)
	sb.WriteString("Props struct {\n")
	for _, p := range props {
		sb.WriteString("\t")
		sb.WriteString(capitalize(p.Name))
		sb.WriteString(" ")
		t := p.Type
		if t == "any" {
			t = "any"
		}
		sb.WriteString(t)
		sb.WriteString(" `json:\"")
		sb.WriteString(p.Name)
		sb.WriteString("\"`")
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

// GenerateTSInterface generates a TypeScript interface for the component state.
func GenerateTSInterface(name string, props []Prop, states []State) string {
	var sb strings.Builder
	sb.WriteString("export interface ")
	sb.WriteString(name)
	sb.WriteString("State {\n")

	sb.WriteString("\tprops: {\n")
	for _, p := range props {
		sb.WriteString("\t\t")
		sb.WriteString(p.Name)
		sb.WriteString(": ")
		sb.WriteString(tsType(p.Type))
		sb.WriteString(";\n")
	}
	sb.WriteString("\t};\n")

	for _, s := range states {
		sb.WriteString("\t")
		sb.WriteString(s.Name)
		sb.WriteString(": ")
		sb.WriteString(tsType(s.Type))
		sb.WriteString(";\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

func tsType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int64", "float64", "float32":
		return "number"
	case "bool":
		return "boolean"
	case "any", "interface{}":
		return "any"
	default:
		return "any"
	}
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
