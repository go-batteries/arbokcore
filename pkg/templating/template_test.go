package templating

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_StringTemplating(t *testing.T) {
	t.Run(
		"returns original string if no formatter is present",
		func(t *testing.T) {
			renderer := NewTemplateString("foobar")
			require.Equal(t, "foobar", renderer())
		})

	t.Run(
		"return formatted string when formatter is present",
		func(t *testing.T) {
			renderer := NewTemplateString("foo::%s")
			require.Equal(t, "foo::bar", renderer("bar"))
		})
}
