package stremio

// ConfigurationField represents a field in the configuration UI
type ConfigurationField struct {
	Type        string `json:"type"`                  // The type of the field (e.g. "text", "number", "boolean", etc.)
	Label       string `json:"label"`                 // The label of the field
	Description string `json:"description,omitempty"` // The description of the field
	Default     any    `json:"default,omitempty"`     // The default value of the field
	Required    bool   `json:"required,omitempty"`    // Whether the field is required
	Options     []any  `json:"options,omitempty"`     // The options for the field (for select fields)
	Min         any    `json:"min,omitempty"`         // The minimum value (for number fields)
	Max         any    `json:"max,omitempty"`         // The maximum value (for number fields)
	Step        any    `json:"step,omitempty"`        // The step value (for number fields)
	Pattern     string `json:"pattern,omitempty"`     // The pattern for validation (for text fields)
	Placeholder string `json:"placeholder,omitempty"` // The placeholder text (for text fields)
}

// NewConfigurationUI creates a ConfigurationUI with the given type and properties
func NewConfigurationUI(uiType string, properties map[string]any) *ConfigurationUI {
	return &ConfigurationUI{
		Type:       uiType,
		Properties: properties,
	}
}

// AddRequiredField adds a required field to the configuration UI
func (c *ConfigurationUI) AddRequiredField(field string) {
	if c.Required == nil {
		c.Required = make([]string, 0)
	}
	c.Required = append(c.Required, field)
}

// SetDefault sets the default value for the configuration UI
func (c *ConfigurationUI) SetDefault(defaultValue any) {
	c.Default = defaultValue
}

// NewConfigurationField creates a ConfigurationField with the given type and label
func NewConfigurationField(fieldType string, label string) *ConfigurationField {
	return &ConfigurationField{
		Type:  fieldType,
		Label: label,
	}
}

// SetDescription sets the description of the field
func (f *ConfigurationField) SetDescription(description string) *ConfigurationField {
	f.Description = description
	return f
}

// SetDefault sets the default value of the field
func (f *ConfigurationField) SetDefault(defaultValue any) *ConfigurationField {
	f.Default = defaultValue
	return f
}

// SetRequired sets whether the field is required
func (f *ConfigurationField) SetRequired(required bool) *ConfigurationField {
	f.Required = required
	return f
}

// SetOptions sets the options for the field
func (f *ConfigurationField) SetOptions(options []any) *ConfigurationField {
	f.Options = options
	return f
}

// SetMin sets the minimum value for the field
func (f *ConfigurationField) SetMin(min any) *ConfigurationField {
	f.Min = min
	return f
}

// SetMax sets the maximum value for the field
func (f *ConfigurationField) SetMax(max any) *ConfigurationField {
	f.Max = max
	return f
}

// SetStep sets the step value for the field
func (f *ConfigurationField) SetStep(step any) *ConfigurationField {
	f.Step = step
	return f
}

// SetPattern sets the pattern for validation
func (f *ConfigurationField) SetPattern(pattern string) *ConfigurationField {
	f.Pattern = pattern
	return f
}

// SetPlaceholder sets the placeholder text
func (f *ConfigurationField) SetPlaceholder(placeholder string) *ConfigurationField {
	f.Placeholder = placeholder
	return f
}
