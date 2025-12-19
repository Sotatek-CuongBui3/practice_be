package handler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/cuongbtq/practice-be/internal/api/storage"
)

func DecodeJobCursor(cursorStr string) (*storage.JobCursor, error) {
	if cursorStr == "" {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, err
	}

	// Further decoding logic to parse decoded string into storage.JobCursor
	decodedParts := strings.Split(string(decoded), "|")
	if len(decodedParts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}

	var createdAt int64
	_, err = fmt.Sscanf(decodedParts[0], "%d", &createdAt)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt in cursor: %w", err)
	}

	return &storage.JobCursor{
		CreatedAt: time.Unix(0, createdAt),
		JobID:     decodedParts[1],
	}, nil
}

func EncodeJobCursor(cursor *storage.JobCursor) (string, error) {
	cs := fmt.Sprintf("%d|%s", cursor.CreatedAt.UnixNano(), cursor.JobID)
	return base64.StdEncoding.EncodeToString([]byte(cs)), nil
}
