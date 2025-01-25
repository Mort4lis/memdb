package compute

import (
	"errors"
	"fmt"
	"strings"
)

func ParseQuery(rawQuery string) (Query, error) {
	query := Query{}
	queryParts := strings.Split(rawQuery, " ")
	if len(queryParts) == 0 {
		return query, errors.New("empty query")
	}

	cmdID, ok := nameCommandIDMapping[(queryParts[0])]
	if !ok {
		return query, fmt.Errorf("unsupport command %s", queryParts[0])
	}

	numArgs := commandIDArgNumbersMapping[cmdID]
	if len(queryParts[1:]) != numArgs {
		return query, errors.New("invalid the number of arguments")
	}

	query.cmdID = cmdID
	query.args = queryParts[1:]
	return query, nil
}
