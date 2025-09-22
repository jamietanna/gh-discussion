package discussionform

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type BodyItem struct {
	Type        string
	Item        any
	Validations map[string]bool
}

func (b *BodyItem) UnmarshalYAML(value *yaml.Node) error {
	var typeHolder struct {
		Type string `yaml:"type"`
	}
	if err := value.Decode(&typeHolder); err != nil {
		return err
	}
	b.Type = typeHolder.Type

	var validationsHolder struct {
		Validations map[string]bool `yaml:"validations"`
	}
	if err := value.Decode(&validationsHolder); err != nil {
		return err
	}
	b.Validations = validationsHolder.Validations

	switch typeHolder.Type {
	case "dropdown":
		var dropdown Dropdown
		if err := value.Decode(&dropdown); err != nil {
			return err
		}
		b.Item = dropdown
	case "input":
		var input Input
		if err := value.Decode(&input); err != nil {
			return err
		}
		b.Item = input
	case "textarea":
		var textarea Textarea
		if err := value.Decode(&textarea); err != nil {
			return err
		}
		b.Item = textarea
	default:
		return fmt.Errorf("unknown type: %s", typeHolder.Type)
	}
	return nil
}

type Dropdown struct {
	Type       string             `yaml:"type"`
	ID         string             `yaml:"id"`
	Attributes DropdownAttributes `yaml:"attributes"`
}
type DropdownAttributes struct {
	Label   string   `yaml:"label"`
	Options []string `yaml:"options"`
}

type Input struct {
	Type       string          `yaml:"type"`
	ID         string          `yaml:"id"`
	Attributes InputAttributes `yaml:"attributes"`
}
type InputAttributes struct {
	Label string `yaml:"label"`
}

type Textarea struct {
	Type       string             `yaml:"type"`
	ID         string             `yaml:"id"`
	Attributes TextareaAttributes `yaml:"attributes"`
}
type TextareaAttributes struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	Value       string `yaml:"value"`
}

type Template struct {
	Body []BodyItem `yaml:"body"`
}
