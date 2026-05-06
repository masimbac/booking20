package dynamo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type gsi3PageCursor struct {
	PK     string `json:"PK"`
	SK     string `json:"SK"`
	GSI3PK string `json:"GSI3PK"`
	GSI3SK string `json:"GSI3SK"`
}

func encodeGSI3PageCursor(key map[string]types.AttributeValue) (string, error) {
	var c gsi3PageCursor
	var err error
	c.PK, err = stringAttr(key["PK"])
	if err != nil {
		return "", err
	}
	c.SK, err = stringAttr(key["SK"])
	if err != nil {
		return "", err
	}
	c.GSI3PK, err = stringAttr(key["GSI3PK"])
	if err != nil {
		return "", err
	}
	c.GSI3SK, err = stringAttr(key["GSI3SK"])
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeGSI3PageCursor(cursor string) (map[string]types.AttributeValue, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	var c gsi3PageCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	return map[string]types.AttributeValue{
		"PK":     &types.AttributeValueMemberS{Value: c.PK},
		"SK":     &types.AttributeValueMemberS{Value: c.SK},
		"GSI3PK": &types.AttributeValueMemberS{Value: c.GSI3PK},
		"GSI3SK": &types.AttributeValueMemberS{Value: c.GSI3SK},
	}, nil
}
