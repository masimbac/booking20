package dynamo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/parama/booking/internal/app/ports"
	"github.com/parama/booking/internal/domain"
)

// CustomerRepository implements [ports.CustomerRepository].
type CustomerRepository struct {
	Client *dynamodb.Client
	Table  string
}

func normalizeE164(s string) string {
	return strings.TrimSpace(s)
}

func (r *CustomerRepository) Create(ctx context.Context, c *domain.Customer) error {
	phone := normalizeE164(c.PhoneE164)
	if phone == "" {
		return fmt.Errorf("%w: phone_e164 is required", domain.ErrInvalid)
	}
	existing, err := r.queryPhone(ctx, c.BusinessID, phone)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("%w: phone already registered for this business", domain.ErrConflict)
	}
	c.PhoneE164 = phone
	item, err := marshalCustomer(c)
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
		return fmt.Errorf("%w: customer id in use", domain.ErrConflict)
	}
	return err
}

func (r *CustomerRepository) queryPhone(ctx context.Context, businessID, phone string) (*domain.Customer, error) {
	out, err := r.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.Table),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk AND GSI2SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: phoneGSI2PK(phone)},
			":sk": &types.AttributeValueMemberS{Value: phoneGSI2SK(businessID)},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, nil
	}
	return unmarshalCustomer(out.Items[0])
}

func (r *CustomerRepository) Get(ctx context.Context, businessID, customerID string) (*domain.Customer, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: customerSK(customerID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalCustomer(out.Item)
}

func (r *CustomerRepository) GetByPhone(ctx context.Context, businessID, phone string) (*domain.Customer, error) {
	c, err := r.queryPhone(ctx, businessID, normalizeE164(phone))
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (r *CustomerRepository) List(ctx context.Context, businessID string, opt ports.ListCustomersOptions) ([]domain.Customer, string, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.Table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			":prefix": &types.AttributeValueMemberS{Value: "CUSTOMER#"},
		},
		Limit: aws.Int32(limit),
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
	var list []domain.Customer
	for _, item := range out.Items {
		c, err := unmarshalCustomer(item)
		if err != nil {
			return nil, "", err
		}
		list = append(list, *c)
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

func (r *CustomerRepository) Save(ctx context.Context, c *domain.Customer) error {
	c.PhoneE164 = normalizeE164(c.PhoneE164)
	item, err := marshalCustomer(c)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

type customerItem struct {
	PK             string         `dynamodbav:"PK"`
	SK             string         `dynamodbav:"SK"`
	GSI2PK         string         `dynamodbav:"GSI2PK"`
	GSI2SK         string         `dynamodbav:"GSI2SK"`
	Entity         string         `dynamodbav:"entity"`
	ID             string         `dynamodbav:"id"`
	BusinessID     string         `dynamodbav:"business_id"`
	PhoneE164      string         `dynamodbav:"phone_e164"`
	DisplayName    string         `dynamodbav:"display_name,omitempty"`
	Preferences    map[string]any `dynamodbav:"preferences,omitempty"`
	MarketingOptIn bool           `dynamodbav:"marketing_opt_in"`
	CreatedAt      string         `dynamodbav:"created_at"`
	UpdatedAt      string         `dynamodbav:"updated_at"`
}

func marshalCustomer(c *domain.Customer) (map[string]types.AttributeValue, error) {
	row := customerItem{
		PK:             BusinessPK(c.BusinessID),
		SK:             customerSK(c.ID),
		GSI2PK:         phoneGSI2PK(c.PhoneE164),
		GSI2SK:         phoneGSI2SK(c.BusinessID),
		Entity:         entityCustomer,
		ID:             c.ID,
		BusinessID:     c.BusinessID,
		PhoneE164:      c.PhoneE164,
		DisplayName:    c.DisplayName,
		Preferences:    c.Preferences,
		MarketingOptIn: c.MarketingOptIn,
		CreatedAt:      c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalCustomer(item map[string]types.AttributeValue) (*domain.Customer, error) {
	var row customerItem
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
	return &domain.Customer{
		ID:             row.ID,
		BusinessID:     row.BusinessID,
		PhoneE164:      row.PhoneE164,
		DisplayName:    row.DisplayName,
		Preferences:    row.Preferences,
		MarketingOptIn: row.MarketingOptIn,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

var _ ports.CustomerRepository = (*CustomerRepository)(nil)
