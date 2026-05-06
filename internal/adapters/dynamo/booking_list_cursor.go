package dynamo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// bookingListCursor captures table + GSI keys for paginated GSI1 queries.
type bookingListCursor struct {
	PK     string `json:"PK"`
	SK     string `json:"SK"`
	GSI1PK string `json:"GSI1PK"`
	GSI1SK string `json:"GSI1SK"`
}

func encodeBookingListCursor(key map[string]types.AttributeValue) (string, error) {
	var c bookingListCursor
	var err error
	c.PK, err = stringAttr(key["PK"])
	if err != nil {
		return "", err
	}
	c.SK, err = stringAttr(key["SK"])
	if err != nil {
		return "", err
	}
	c.GSI1PK, err = stringAttr(key["GSI1PK"])
	if err != nil {
		return "", err
	}
	c.GSI1SK, err = stringAttr(key["GSI1SK"])
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeBookingListCursor(cursor string) (map[string]types.AttributeValue, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	var c bookingListCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	return map[string]types.AttributeValue{
		"PK":     &types.AttributeValueMemberS{Value: c.PK},
		"SK":     &types.AttributeValueMemberS{Value: c.SK},
		"GSI1PK": &types.AttributeValueMemberS{Value: c.GSI1PK},
		"GSI1SK": &types.AttributeValueMemberS{Value: c.GSI1SK},
	}, nil
}
