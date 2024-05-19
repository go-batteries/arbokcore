package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ClauseStr(t *testing.T) {

	t.Run("when clauses are empty", func(t *testing.T) {
		clause := FindClause{}

		// We actually don't know what to do here.
		// Maybe We don't put Where in the SQL template str
		// SO we can return blank string
		assert.Equal(
			t,
			"",
			clause.String(),
		)
	})

	t.Run("with proper clauses, uses :key as placeholder by default", func(t *testing.T) {
		clause := FindClause{
			{Key: "id", Operator: "=", Val: "F1"},
			{Key: "upload_status", PlaceHolder: ":us", Operator: "=", Val: StatusUploading},
		}

		assert.Equal(
			t,
			"id = :id AND upload_status = :us",
			clause.String(),
		)
	})

}
