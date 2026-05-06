package dynamo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type lekCursor struct {
	PK string `json:"PK"`
	SK string `json:"SK"`
}

func encodeExclusiveStartKey(key map[string]types.AttributeValue) (string, error) {
	pk, err := stringAttr(key["PK"])
	if err != nil {
		return "", err
	}
	sk, err := stringAttr(key["SK"])
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(lekCursor{PK: pk, SK: sk})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeExclusiveStartKey(cursor string) (map[string]types.AttributeValue, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	var c lekCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: c.PK},
		"SK": &types.AttributeValueMemberS{Value: c.SK},
	}, nil
}

func stringAttr(av types.AttributeValue) (string, error) {
	if av == nil {
		return "", fmt.Errorf("nil attribute")
	}
	s, ok := av.(*types.AttributeValueMemberS)
	if !ok {
		return "", fmt.Errorf("expected string attribute")
	}
	return s.Value, nil
}
