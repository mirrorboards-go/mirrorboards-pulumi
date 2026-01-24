package namespace

import "strings"

type Namespace struct {
	parts []string
}

func NewNamespace(parts ...string) *Namespace {
	var validParts []string
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	return &Namespace{
		parts: validParts,
	}
}

func (n *Namespace) Get(name ...string) string {
	allParts := make([]string, len(n.parts))
	copy(allParts, n.parts)

	for _, part := range name {
		if part != "" {
			allParts = append(allParts, part)
		}
	}

	return strings.Join(allParts, "-")
}

func (n *Namespace) GetParts() []string {
	parts := make([]string, len(n.parts))
	copy(parts, n.parts)
	return parts
}

func (n *Namespace) Append(parts ...string) *Namespace {
	newParts := make([]string, len(n.parts))
	copy(newParts, n.parts)

	for _, part := range parts {
		if part != "" {
			newParts = append(newParts, part)
		}
	}

	return &Namespace{parts: newParts}
}
