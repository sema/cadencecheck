package entities

import "strings"

type FunctionPattern struct {
	Package string
	Type    string // Optional
	Method  string
}

func (f *FunctionPattern) String() string {
	parts := []string{f.Package}
	if f.Type != "" {
		parts = append(parts)
	}
	parts = append(parts, f.Method)

	return strings.Join(parts, ".")
}

func StripVendor(packageName string) string {
	parts := strings.Split(packageName, "/vendor/")
	return parts[len(parts)-1]
}
