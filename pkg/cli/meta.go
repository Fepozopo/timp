package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Fepozopo/timp/pkg/stdimg"
)

// ParamType is a small enum for parameter types used in metadata.
type ParamType string

const (
	ParamTypeInt     ParamType = "int"
	ParamTypeFloat   ParamType = "float"
	ParamTypeBool    ParamType = "bool"
	ParamTypeString  ParamType = "string"
	ParamTypeEnum    ParamType = "enum"
	ParamTypePercent ParamType = "percent"
)

// ValidationRule is a machine-friendly representation of the constraints
// that a UI or client can use to validate input before invoking a command.
type ValidationRule struct {
	Type        ParamType `json:"type"`
	Required    bool      `json:"required"`
	Min         *float64  `json:"min,omitempty"`
	Max         *float64  `json:"max,omitempty"`
	Unit        string    `json:"unit,omitempty"`
	Pattern     string    `json:"pattern,omitempty"`     // optional regex-like pattern or note
	EnumOptions []string  `json:"enumOptions,omitempty"` // valid when Type == ParamTypeEnum
	Example     string    `json:"example,omitempty"`
	Hint        string    `json:"hint,omitempty"`
}

// parseBoolLikeToString accepts common truthy/falsy forms and returns "true"/"false" string.
func parseBoolLikeToString(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "y", "yes", "on":
		return "true", nil
	case "0", "f", "false", "n", "no", "off":
		return "false", nil
	default:
		return "", fmt.Errorf("invalid boolean: %q", s)
	}
}

// parsePercentValue parses a percent string like "3%" or a bare number and returns numeric string.
func parsePercentValue(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		raw := strings.TrimSuffix(s, "%")
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return "", fmt.Errorf("invalid percent value: %q", s)
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}
	// bare number
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return "", fmt.Errorf("invalid percent/float value: %q", s)
	}
	return s, nil
}

// --- stdimg integration helpers ---

// GenerateTooltipFromStdSpec produces a tooltip string from a stdimg.CommandSpec.
func GenerateTooltipFromStdSpec(c stdimg.CommandSpec) string {
	var sb strings.Builder
	if c.Description != "" {
		sb.WriteString(c.Description)
	} else {
		sb.WriteString("No description")
	}
	if len(c.Args) == 0 {
		sb.WriteString(" — no parameters")
		return sb.String()
	}
	sb.WriteString(" — parameters:\n")
	for _, a := range c.Args {
		req := "optional"
		if a.Required {
			req = "required"
		}
		line := fmt.Sprintf("- %s (%s, %s)", a.Name, a.Type, req)
		sb.WriteString(line)
		if a.Description != "" {
			sb.WriteString(" — " + a.Description)
		}
		if a.Default != "" {
			sb.WriteString(" (default: " + a.Default + ")")
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

// GenerateValidationRulesFromStdSpec creates ValidationRule entries from a stdimg.CommandSpec.
func GenerateValidationRulesFromStdSpec(c stdimg.CommandSpec) map[string]ValidationRule {
	rules := make(map[string]ValidationRule, len(c.Args))
	for _, a := range c.Args {
		at := strings.ToLower(a.Type)
		var t ParamType
		switch {
		case at == "int":
			t = ParamTypeInt
		case at == "float":
			t = ParamTypeFloat
		case at == "bool":
			t = ParamTypeBool
		case strings.Contains(at, "percent"):
			t = ParamTypePercent
		case at == "enum":
			t = ParamTypeEnum
		default:
			t = ParamTypeString
		}
		r := ValidationRule{Type: t, Required: a.Required, Hint: a.Description, Example: a.Default}
		rules[a.Name] = r
	}
	return rules
}

// StdMetaStore is a MetaStore-like wrapper for stdimg.CommandSpec.
type StdMetaStore struct {
	Commands []stdimg.CommandSpec
	byName   map[string]stdimg.CommandSpec
}

// NewMetaStoreFromStdimg creates a StdMetaStore from stdimg.CommandSpec list.
func NewMetaStoreFromStdimg(cmds []stdimg.CommandSpec) *StdMetaStore {
	m := &StdMetaStore{Commands: cmds, byName: make(map[string]stdimg.CommandSpec, len(cmds))}
	for _, c := range cmds {
		m.byName[c.Name] = c
	}
	return m
}

// GetTooltip returns tooltip string for a stdimg command.
func (m *StdMetaStore) GetTooltip(name string) (string, error) {
	c, ok := m.byName[name]
	if !ok {
		return "", fmt.Errorf("unknown command: %s", name)
	}
	return GenerateTooltipFromStdSpec(c), nil
}

// GetValidationRules returns validation rules for a stdimg command.
func (m *StdMetaStore) GetValidationRules(name string) (map[string]ValidationRule, error) {
	c, ok := m.byName[name]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", name)
	}
	return GenerateValidationRulesFromStdSpec(c), nil
}

// GetCommandHelp returns both tooltip and validation rules for a stdimg command.
func (m *StdMetaStore) GetCommandHelp(name string) (string, map[string]ValidationRule, error) {
	c, ok := m.byName[name]
	if !ok {
		return "", nil, fmt.Errorf("unknown command: %s", name)
	}
	return GenerateTooltipFromStdSpec(c), GenerateValidationRulesFromStdSpec(c), nil
}

// NormalizeArgsFromStd normalizes args using stdimg command metadata provided via StdMetaStore.
// It mirrors NormalizeArgs but operates on StdMetaStore and rules from stdimg.CommandSpec.
func NormalizeArgsFromStd(store *StdMetaStore, cmdName string, args []string) ([]string, error) {
	if store == nil {
		return nil, fmt.Errorf("metadata store is nil")
	}
	c, ok := store.byName[cmdName]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
	rules := GenerateValidationRulesFromStdSpec(c)
	// Build a slice of parameter ordering based on c.Args
	paramOrder := c.Args
	out := make([]string, len(paramOrder))
	for i, a := range paramOrder {
		var raw string
		if i < len(args) {
			raw = strings.TrimSpace(args[i])
		} else {
			raw = ""
		}
		if raw == "" {
			if a.Required {
				return nil, fmt.Errorf("missing required parameter: %s", a.Name)
			}
			out[i] = ""
			continue
		}
		vr := rules[a.Name]
		switch vr.Type {
		case ParamTypeInt:
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parameter %s: expected integer, got %q", a.Name, raw)
			}
			if vr.Min != nil && float64(v) < *vr.Min {
				return nil, fmt.Errorf("parameter %s: %d < min %v", a.Name, v, *vr.Min)
			}
			if vr.Max != nil && float64(v) > *vr.Max {
				return nil, fmt.Errorf("parameter %s: %d > max %v", a.Name, v, *vr.Max)
			}
			out[i] = strconv.FormatInt(v, 10)
		case ParamTypeFloat:
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, fmt.Errorf("parameter %s: expected float, got %q", a.Name, raw)
			}
			if vr.Min != nil && f < *vr.Min {
				return nil, fmt.Errorf("parameter %s: %v < min %v", a.Name, f, *vr.Min)
			}
			if vr.Max != nil && f > *vr.Max {
				return nil, fmt.Errorf("parameter %s: %v > max %v", a.Name, f, *vr.Max)
			}
			out[i] = strconv.FormatFloat(f, 'f', -1, 64)
		case ParamTypePercent:
			n, err := parsePercentValue(raw)
			if err != nil {
				return nil, fmt.Errorf("parameter %s: %w", a.Name, err)
			}
			f, _ := strconv.ParseFloat(n, 64)
			if vr.Min != nil && f < *vr.Min {
				return nil, fmt.Errorf("parameter %s: %v < min %v", a.Name, f, *vr.Min)
			}
			if vr.Max != nil && f > *vr.Max {
				return nil, fmt.Errorf("parameter %s: %v > max %v", a.Name, f, *vr.Max)
			}
			out[i] = n
		case ParamTypeBool:
			bs, err := parseBoolLikeToString(raw)
			if err != nil {
				return nil, fmt.Errorf("parameter %s: %w", a.Name, err)
			}
			out[i] = bs
		case ParamTypeEnum:
			// Try numeric first
			if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
				out[i] = raw
				break
			}
			// Try known maps
			if mapped, ok := mapEnumToNumeric(a.Name, raw); ok {
				out[i] = mapped
				break
			}
			out[i] = raw
		case ParamTypeString:
			out[i] = raw
		default:
			return nil, fmt.Errorf("parameter %s: unsupported param type %q", a.Name, vr.Type)
		}
	}
	return out, nil
}

/*
Package-level enum maps and helpers.

We extract the textual<->numeric maps into package-level variables so that
they can be reused by both directions of mapping:

- mapEnumToNumeric: textual -> numeric (existing behavior)
- mapNumericToEnumName: numeric -> textual (new helper used elsewhere, e.g. GetImageInfo)

Add or extend maps here as new enum types are required.
*/

var (
	// Simple textual enum maps. Values are canonical uppercase strings used by the
	// pure-Go engine and metadata normalization. These maps exist to allow aliasing
	// or accepting different textual forms in the UI.
	noiseTypeNameToValue = map[string]string{
		"UNDEFINED":      "UNDEFINED",
		"UNIFORM":        "UNIFORM",
		"GAUSSIAN":       "GAUSSIAN",
		"MULTIPLICATIVE": "MULTIPLICATIVE",
		"IMPULSE":        "IMPULSE",
		"LAPLACIAN":      "LAPLACIAN",
		"POISSON":        "POISSON",
		"RANDOM":         "RANDOM",
	}

	composeOpNameToValue = map[string]string{
		"UNDEFINED":  "UNDEFINED",
		"OVER":       "OVER",
		"MULTIPLY":   "MULTIPLY",
		"SCREEN":     "SCREEN",
		"OVERLAY":    "OVERLAY",
		"DISSOLVE":   "DISSOLVE",
		"ADD":        "ADD",
		"DIFFERENCE": "DIFFERENCE",
	}

	compressionNameToValue = map[string]string{
		"UNDEFINED": "UNDEFINED",
		"NO":        "NO",
		"ZIP":       "ZIP",
		"JPEG":      "JPEG",
		"LZW":       "LZW",
	}
)

// mapEnumToNumeric attempts to translate some known enum textual values into a
// canonical textual representation used by the stdlib engine. If the input is
// already numeric, it is returned unchanged (to preserve backward compatibility).
func mapEnumToNumeric(paramName string, val string) (string, bool) {
	v := strings.TrimSpace(val)
	// If the caller provided an explicit numeric value, keep it.
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return v, true
	}
	up := strings.ToUpper(v)
	switch strings.ToLower(paramName) {
	case "noisetype", "noise_type", "noise":
		if out, ok := noiseTypeNameToValue[up]; ok {
			return out, true
		}
	case "composeoperator", "compose_operator", "compose":
		if out, ok := composeOpNameToValue[up]; ok {
			return out, true
		}
	case "type", "compression", "compressiontype", "compress":
		if out, ok := compressionNameToValue[up]; ok {
			return out, true
		}
	}
	return "", false
}
