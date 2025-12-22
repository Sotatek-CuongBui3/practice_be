package handler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/cuongbtq/practice-be/internal/api/storage"
)

// DecodeJobCursor decodes a base64-encoded cursor string into a JobCursor struct
func DecodeJobCursor(cursorStr string) (*storage.JobCursor, error) {
	if cursorStr == "" {
		return nil, nil
	}

	// Decode from base64
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

// EncodeJobCursor encodes a JobCursor struct into a base64-encoded string
func EncodeJobCursor(cursor *storage.JobCursor) (string, error) {
	// Encode to base64
	cs := fmt.Sprintf("%d|%s", cursor.CreatedAt.UnixNano(), cursor.JobID)
	return base64.StdEncoding.EncodeToString([]byte(cs)), nil
}
