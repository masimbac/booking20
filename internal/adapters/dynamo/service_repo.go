package dynamo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

// ServiceRepository implements catalog service persistence.
type ServiceRepository struct {
	Client *dynamodb.Client
	Table  string
}

func (r *ServiceRepository) Create(ctx context.Context, s *domain.CatalogService) error {
	item, err := marshalService(s)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.Table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	})
	var cc *types.ConditionalCheckFailedException
	if err != nil && errors.As(err, &cc) {
		return fmt.Errorf("%w: service id in use", domain.ErrConflict)
	}
	return err
}

func (r *ServiceRepository) Get(ctx context.Context, businessID, serviceID string) (*domain.CatalogService, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: serviceSK(serviceID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalService(out.Item)
}

func (r *ServiceRepository) List(ctx context.Context, businessID string, opt ports.ListServicesOptions) ([]domain.CatalogService, string, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	vals := map[string]types.AttributeValue{
		":pk":     &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
		":prefix": &types.AttributeValueMemberS{Value: "SERVICE#"},
	}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.Table),
		KeyConditionExpression:    aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: vals,
		Limit:                     aws.Int32(limit),
	}
	if opt.ActiveOnly {
		input.FilterExpression = aws.String("active = :act")
		vals[":act"] = &types.AttributeValueMemberBOOL{Value: true}
	}
	if opt.Cursor != "" {
		eks, err := decodeExclusiveStartKey(opt.Cursor)
		if err != nil {
			return nil, "", err
		}
		input.ExclusiveStartKey = eks
	}

	out, err := r.Client.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	var list []domain.CatalogService
	for _, item := range out.Items {
		s, err := unmarshalService(item)
		if err != nil {
			return nil, "", err
		}
		list = append(list, *s)
	}
	next := ""
	if len(out.LastEvaluatedKey) > 0 {
		next, err = encodeExclusiveStartKey(out.LastEvaluatedKey)
		if err != nil {
			return nil, "", err
		}
	}
	return list, next, nil
}

func (r *ServiceRepository) Save(ctx context.Context, s *domain.CatalogService) error {
	item, err := marshalService(s)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *ServiceRepository) Delete(ctx context.Context, businessID, serviceID string) error {
	_, err := r.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: serviceSK(serviceID)},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND begins_with(SK, :p)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":p": &types.AttributeValueMemberS{Value: "SERVICE#"},
		},
	})
	var cc *types.ConditionalCheckFailedException
	if err != nil && errors.As(err, &cc) {
		return domain.ErrNotFound
	}
	return err
}

type serviceItem struct {
	PK              string         `dynamodbav:"PK"`
	SK              string         `dynamodbav:"SK"`
	Entity          string         `dynamodbav:"entity"`
	ID              string         `dynamodbav:"id"`
	BusinessID      string         `dynamodbav:"business_id"`
	Name            string         `dynamodbav:"name"`
	DurationMinutes int            `dynamodbav:"duration_minutes"`
	PriceAmount     *string        `dynamodbav:"price_amount,omitempty"`
	PriceCurrency   *string        `dynamodbav:"price_currency,omitempty"`
	Active          bool           `dynamodbav:"active"`
	Metadata        map[string]any `dynamodbav:"metadata,omitempty"`
	CreatedAt       string         `dynamodbav:"created_at"`
	UpdatedAt       string         `dynamodbav:"updated_at"`
}

func marshalService(s *domain.CatalogService) (map[string]types.AttributeValue, error) {
	row := serviceItem{
		PK:              BusinessPK(s.BusinessID),
		SK:              serviceSK(s.ID),
		Entity:          entityService,
		ID:              s.ID,
		BusinessID:      s.BusinessID,
		Name:            s.Name,
		DurationMinutes: s.DurationMinutes,
		Active:          s.Active,
		Metadata:        s.Metadata,
		CreatedAt:       s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:       s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if s.Price != nil {
		row.PriceAmount = aws.String(s.Price.Amount)
		row.PriceCurrency = aws.String(s.Price.Currency)
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalService(item map[string]types.AttributeValue) (*domain.CatalogService, error) {
	var row serviceItem
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	createdAt, err := parseDynamoTime(row.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseDynamoTime(row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	out := &domain.CatalogService{
		ID:              row.ID,
		BusinessID:      row.BusinessID,
		Name:            row.Name,
		DurationMinutes: row.DurationMinutes,
		Active:          row.Active,
		Metadata:        row.Metadata,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
	if row.PriceAmount != nil && row.PriceCurrency != nil && *row.PriceAmount != "" && *row.PriceCurrency != "" {
		out.Price = &domain.Money{Amount: *row.PriceAmount, Currency: *row.PriceCurrency}
	}
	return out, nil
}

func parseDynamoTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Parse(time.RFC3339, s)
	}
	return t, nil
}

var _ ports.ServiceRepository = (*ServiceRepository)(nil)
