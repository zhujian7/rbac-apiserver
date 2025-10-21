//go:build tools

package tools

import (
	_ "k8s.io/code-generator"
	_ "k8s.io/code-generator/cmd/validation-gen"
)
