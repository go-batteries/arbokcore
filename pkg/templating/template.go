package templating

import "fmt"

type RendeFunc func(...any) string

func NewTemplateString(tmplStr string) RendeFunc {
	return func(values ...any) string {
		return fmt.Sprintf(tmplStr, values...)
	}
}
