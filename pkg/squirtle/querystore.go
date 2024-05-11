package squirtle

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type QueryConfig struct {
	Table          string   `yaml:"table"`
	QueryFilePaths []string `yaml:"query_file"`
}

type QueryConfigStore []QueryConfig

type QueryMapper map[string]string

const DefaultQueryStoreConfigLocation = "./config/querystore.yaml"

func LoadAll(configFilePath ...string) QueryConfigStore {
	if len(configFilePath) == 0 {
		configFilePath = append(configFilePath, DefaultQueryStoreConfigLocation)
	}

	byt, err := os.ReadFile(configFilePath[0])
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read query config file")
	}

	store := QueryConfigStore{}

	if err := yaml.Unmarshal(byt, &store); err != nil {
		log.Fatal().Err(err).Msg("invalid config structure")
	}

	return store
}

func (qs QueryConfigStore) HydrateQueryStore(tableName string) (QueryMapper, error) {
	var config *QueryConfig
	var store = make(QueryMapper)

	for _, record := range qs {
		if record.Table == tableName {
			config = &record
			break
		}
	}

	if config == nil {
		return store, errors.New("config for table not found")
	}

	if len(config.QueryFilePaths) == 0 {
		return store, errors.New("missing query file. don't fool me")
	}

	// For now lets just do [0]
	byt, err := os.ReadFile(config.QueryFilePaths[0])
	if err != nil {
		return store, err
	}

	queries := string(byt)

	rx := regexp.MustCompile(`(?m)sql:([P<QueryName>\w]+)$`)

	for _, queryConfig := range strings.Split(queries, "--") {
		res := rx.FindStringSubmatch(queryConfig)
		// res := rx.SubexpNames()
		if len(res) < 2 {
			continue
		}

		match := res[1]

		store[match] = strings.TrimSpace(rx.ReplaceAllString(queryConfig, ""))
	}

	return store, nil
}

func (qmap QueryMapper) Keys() []string {
	keys := []string{}

	for k := range qmap {
		keys = append(keys, k)
	}

	return keys
}

func (qmap QueryMapper) GetQuery(queryName string) (string, bool) {
	v, ok := qmap[queryName]
	return v, ok
}
