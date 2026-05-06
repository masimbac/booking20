package dynamo

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

// AvailabilityRepository implements [ports.AvailabilityRepository].
type AvailabilityRepository struct {
	Client *dynamodb.Client
	Table  string
}

type availabilityDoc struct {
	PK        string                `dynamodbav:"PK"`
	SK        string                `dynamodbav:"SK"`
	Entity    string                `dynamodbav:"entity"`
	Rules     []availabilityRuleRow `dynamodbav:"rules"`
	UpdatedAt string                `dynamodbav:"updated_at"`
}

type availabilityRuleRow struct {
	StaffID    string         `dynamodbav:"staff_id,omitempty"`
	ServiceID  string         `dynamodbav:"service_id,omitempty"`
	DayOfWeek  int            `dynamodbav:"day_of_week"`
	StartLocal string         `dynamodbav:"start_local"`
	EndLocal   string         `dynamodbav:"end_local"`
	Metadata   map[string]any `dynamodbav:"metadata,omitempty"`
}

func (r *AvailabilityRepository) PutRules(ctx context.Context, businessID string, rules []domain.AvailabilityRule, updatedAt time.Time) error {
	rows := make([]availabilityRuleRow, 0, len(rules))
	for _, ru := range rules {
		rows = append(rows, availabilityRuleRow{
			StaffID:    ru.StaffID,
			ServiceID:  ru.ServiceID,
			DayOfWeek:  ru.DayOfWeek,
			StartLocal: ru.StartLocal,
			EndLocal:   ru.EndLocal,
			Metadata:   ru.Metadata,
		})
	}
	doc := availabilityDoc{
		PK:        BusinessPK(businessID),
		SK:        availRulesSK(),
		Entity:    entityAvail,
		Rules:     rows,
		UpdatedAt: updatedAt.UTC().Format(time.RFC3339Nano),
	}
	item, err := attributevalue.MarshalMap(doc)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *AvailabilityRepository) GetRules(ctx context.Context, businessID string) ([]domain.AvailabilityRule, time.Time, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: availRulesSK()},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	if out.Item == nil {
		return nil, time.Time{}, nil
	}
	var doc availabilityDoc
	if err := attributevalue.UnmarshalMap(out.Item, &doc); err != nil {
		return nil, time.Time{}, err
	}
	updatedAt, err := parseDynamoTime(doc.UpdatedAt)
	if err != nil {
		return nil, time.Time{}, err
	}
	rules := make([]domain.AvailabilityRule, 0, len(doc.Rules))
	for _, row := range doc.Rules {
		rules = append(rules, domain.AvailabilityRule{
			StaffID:    row.StaffID,
			ServiceID:  row.ServiceID,
			DayOfWeek:  row.DayOfWeek,
			StartLocal: row.StartLocal,
			EndLocal:   row.EndLocal,
			Metadata:   row.Metadata,
		})
	}
	return rules, updatedAt, nil
}

var _ ports.AvailabilityRepository = (*AvailabilityRepository)(nil)
