package templating

import (
	"fmt"
	"strings"
)

type RendeFunc func(...any) string

func NewTemplateString(tmplStr string) RendeFunc {
	return func(values ...any) string {
		if strings.Contains(tmplStr, "%") {
			return fmt.Sprintf(tmplStr, values...)
		}

		return tmplStr
	}
}
