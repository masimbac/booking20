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

// BusinessRepository implements [ports.BusinessRepository].
type BusinessRepository struct {
	Client *dynamodb.Client
	Table  string
}

func (r *BusinessRepository) Create(ctx context.Context, b *domain.Business) error {
	item, err := marshalBusiness(b)
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
		return fmt.Errorf("%w: business already exists", domain.ErrConflict)
	}
	return err
}

func (r *BusinessRepository) Get(ctx context.Context, id string) (*domain.Business, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(id)},
			"SK": &types.AttributeValueMemberS{Value: BusinessMetaSK(id)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalBusiness(out.Item)
}

func (r *BusinessRepository) Save(ctx context.Context, b *domain.Business) error {
	item, err := marshalBusiness(b)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func marshalBusiness(b *domain.Business) (map[string]types.AttributeValue, error) {
	row := businessItem{
		PK:        BusinessPK(b.ID),
		SK:        BusinessMetaSK(b.ID),
		Entity:    entityBusiness,
		ID:        b.ID,
		Name:      b.Name,
		LegalName: b.LegalName,
		Timezone:  b.Timezone,
		Contact:   b.Contact,
		Status:    string(b.Status),
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

type businessItem struct {
	PK        string         `dynamodbav:"PK"`
	SK        string         `dynamodbav:"SK"`
	Entity    string         `dynamodbav:"entity"`
	ID        string         `dynamodbav:"id"`
	Name      string         `dynamodbav:"name"`
	LegalName string         `dynamodbav:"legal_name,omitempty"`
	Timezone  string         `dynamodbav:"timezone"`
	Contact   map[string]any `dynamodbav:"contact,omitempty"`
	Status    string         `dynamodbav:"status"`
	CreatedAt string         `dynamodbav:"created_at"`
	UpdatedAt string         `dynamodbav:"updated_at"`
}

func unmarshalBusiness(item map[string]types.AttributeValue) (*domain.Business, error) {
	var row businessItem
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
	if err != nil {
		createdAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, row.UpdatedAt)
	if err != nil {
		updatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	}
	return &domain.Business{
		ID:        row.ID,
		Name:      row.Name,
		LegalName: row.LegalName,
		Timezone:  row.Timezone,
		Contact:   row.Contact,
		Status:    domain.BusinessStatus(row.Status),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

var _ ports.BusinessRepository = (*BusinessRepository)(nil)
