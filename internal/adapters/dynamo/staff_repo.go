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

// StaffRepository implements staff persistence.
type StaffRepository struct {
	Client *dynamodb.Client
	Table  string
}

func (r *StaffRepository) Create(ctx context.Context, st *domain.Staff) error {
	item, err := marshalStaff(st)
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
		return fmt.Errorf("%w: staff id in use", domain.ErrConflict)
	}
	return err
}

func (r *StaffRepository) Get(ctx context.Context, businessID, staffID string) (*domain.Staff, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: staffSK(staffID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalStaff(out.Item)
}

func (r *StaffRepository) List(ctx context.Context, businessID string, opt ports.ListStaffOptions) ([]domain.Staff, string, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	vals := map[string]types.AttributeValue{
		":pk":     &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
		":prefix": &types.AttributeValueMemberS{Value: "STAFF#"},
	}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.Table),
		KeyConditionExpression:    aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: vals,
		Limit:                     aws.Int32(limit),
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
	var list []domain.Staff
	for _, item := range out.Items {
		s, err := unmarshalStaff(item)
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

func (r *StaffRepository) Save(ctx context.Context, st *domain.Staff) error {
	item, err := marshalStaff(st)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *StaffRepository) Delete(ctx context.Context, businessID, staffID string) error {
	_, err := r.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: staffSK(staffID)},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND begins_with(SK, :p)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":p": &types.AttributeValueMemberS{Value: "STAFF#"},
		},
	})
	var cc *types.ConditionalCheckFailedException
	if err != nil && errors.As(err, &cc) {
		return domain.ErrNotFound
	}
	return err
}

type staffItem struct {
	PK          string         `dynamodbav:"PK"`
	SK          string         `dynamodbav:"SK"`
	Entity      string         `dynamodbav:"entity"`
	ID          string         `dynamodbav:"id"`
	BusinessID  string         `dynamodbav:"business_id"`
	DisplayName string         `dynamodbav:"display_name"`
	Role        string         `dynamodbav:"role,omitempty"`
	ServiceIDs  []string       `dynamodbav:"service_ids,stringset,omitempty"`
	Active      bool           `dynamodbav:"active"`
	Metadata    map[string]any `dynamodbav:"metadata,omitempty"`
	CreatedAt   string         `dynamodbav:"created_at"`
	UpdatedAt   string         `dynamodbav:"updated_at"`
}

func marshalStaff(s *domain.Staff) (map[string]types.AttributeValue, error) {
	row := staffItem{
		PK:          BusinessPK(s.BusinessID),
		SK:          staffSK(s.ID),
		Entity:      entityStaff,
		ID:          s.ID,
		BusinessID:  s.BusinessID,
		DisplayName: s.DisplayName,
		Role:        s.Role,
		Active:      s.Active,
		Metadata:    s.Metadata,
		CreatedAt:   s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if len(s.ServiceIDs) > 0 {
		row.ServiceIDs = s.ServiceIDs
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalStaff(item map[string]types.AttributeValue) (*domain.Staff, error) {
	var row staffItem
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
	return &domain.Staff{
		ID:          row.ID,
		BusinessID:  row.BusinessID,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		ServiceIDs:  row.ServiceIDs,
		Active:      row.Active,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

var _ ports.StaffRepository = (*StaffRepository)(nil)
