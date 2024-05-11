package squirtle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_QueryStoreLoadAll(t *testing.T) {
	cfg := LoadAll("../../config/querystore.yaml")

	require.Equal(t, len(cfg), 3)
}

func Test_HydrateQueryStore(t *testing.T) {
	store := QueryConfigStore{
		{
			Table:          "users",
			QueryFilePaths: []string{"./queries.sql"},
		},
	}

	_, err := store.HydrateQueryStore("lol")
	require.Error(t, err, "fails to load with invalid table key")

	qs, err := store.HydrateQueryStore("users")
	require.NoError(t, err, "should not have failed to get query mapping")

	require.Equal(t, 2, len(qs.Keys()))

	assert.ElementsMatch(t, []string{"CreateUserQuery", "GetUserByEmail"}, qs.Keys())

	query, ok := qs.GetQuery("CreateUserQuery")
	require.True(t, ok)
	require.True(t, strings.HasPrefix(query, "INSERT INTO"))
}
