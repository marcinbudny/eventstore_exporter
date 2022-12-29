package client

import (
	"fmt"
	"strconv"
	"strings"
)

type EventPosition string

// parseEventPosition extracts commit and prepare position from strings like "C:1234/P:5678"
func (position EventPosition) ParseCommitPreparePosition() (commit int64, prepare int64, err error) {
	if position == "" {
		return -1, -1, fmt.Errorf("empty position")
	}

	parts := strings.Split(string(position), "/")
	if len(parts) != 2 || parts[0][0] != 'C' || parts[1][0] != 'P' {
		return -1, -1, fmt.Errorf("invalid event position: %s", position)
	}

	commit, err = strconv.ParseInt(parts[0][2:], 10, 64)
	if err != nil {
		return -1, -1, fmt.Errorf("invalid commit position in event position: %s", position)
	}

	prepare, err = strconv.ParseInt(parts[1][2:], 10, 64)
	if err != nil {
		return -1, -1, fmt.Errorf("invalid prepare position in event position: %s", position)
	}

	return commit, prepare, nil
}
