package dynamo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// BookingRepository implements [ports.BookingRepository].
type BookingRepository struct {
	Client *dynamodb.Client
	Table  string
}

func hashIdempotencyKey(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func idempotencySortKeyFromKey(raw string) string {
	return "IDEMPOTENCY#" + hashIdempotencyKey(raw)
}

type idempotencyItem struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	Entity    string `dynamodbav:"entity"`
	BookingID string `dynamodbav:"booking_id"`
}

type bookingItem struct {
	PK           string `dynamodbav:"PK"`
	SK           string `dynamodbav:"SK"`
	GSI1PK       string `dynamodbav:"GSI1PK"`
	GSI1SK       string `dynamodbav:"GSI1SK"`
	Entity       string `dynamodbav:"entity"`
	ID           string `dynamodbav:"id"`
	BusinessID   string `dynamodbav:"business_id"`
	CustomerID   string `dynamodbav:"customer_id"`
	ServiceID    string `dynamodbav:"service_id"`
	StaffID      string `dynamodbav:"staff_id,omitempty"`
	StartAt      string `dynamodbav:"start_at"`
	EndAt        string `dynamodbav:"end_at"`
	Status       string `dynamodbav:"status"`
	CancelReason string `dynamodbav:"cancel_reason,omitempty"`
	CreatedAt    string `dynamodbav:"created_at"`
	UpdatedAt    string `dynamodbav:"updated_at"`
}

const entityIdempotency = "IDEMPOTENCY"

// CreateIfNotExistsWithIdempotency writes booking + idempotency row in one transaction, or returns the prior booking on replay.
func (r *BookingRepository) CreateIfNotExistsWithIdempotency(ctx context.Context, idempotencyKey string, b *domain.Booking) (*domain.Booking, bool, error) {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return nil, false, fmt.Errorf("%w: Idempotency-Key is required", domain.ErrInvalid)
	}
	idemSK := idempotencySortKeyFromKey(key)

	if bid, err := r.lookupIdempotency(ctx, b.BusinessID, idemSK); err != nil {
		return nil, false, err
	} else if bid != "" {
		existing, err := r.Get(ctx, b.BusinessID, bid)
		return existing, false, err
	}

	bItem, err := marshalBooking(b)
	if err != nil {
		return nil, false, err
	}
	iItem, err := attributevalue.MarshalMap(idempotencyItem{
		PK:        BusinessPK(b.BusinessID),
		SK:        idemSK,
		Entity:    entityIdempotency,
		BookingID: b.ID,
	})
	if err != nil {
		return nil, false, err
	}

	cond := aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)")
	_, err = r.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName:           aws.String(r.Table),
					Item:                iItem,
					ConditionExpression: cond,
				},
			},
			{
				Put: &types.Put{
					TableName:           aws.String(r.Table),
					Item:                bItem,
					ConditionExpression: cond,
				},
			},
		},
	})
	if err == nil {
		return b, true, nil
	}

	var tc *types.TransactionCanceledException
	if errors.As(err, &tc) {
		for _, reason := range tc.CancellationReasons {
			if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
				if bid, e2 := r.lookupIdempotency(ctx, b.BusinessID, idemSK); e2 != nil {
					return nil, false, e2
				} else if bid != "" {
					existing, e3 := r.Get(ctx, b.BusinessID, bid)
					return existing, false, e3
				}
			}
		}
	}
	return nil, false, err
}

func (r *BookingRepository) lookupIdempotency(ctx context.Context, businessID, idemSK string) (string, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: idemSK},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	if out.Item == nil {
		return "", nil
	}
	var row idempotencyItem
	if err := attributevalue.UnmarshalMap(out.Item, &row); err != nil {
		return "", err
	}
	return row.BookingID, nil
}

func (r *BookingRepository) Get(ctx context.Context, businessID, bookingID string) (*domain.Booking, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: bookingSK(bookingID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalBooking(out.Item)
}

func (r *BookingRepository) Save(ctx context.Context, b *domain.Booking) error {
	item, err := marshalBooking(b)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *BookingRepository) ListByStartRange(ctx context.Context, businessID string, fromUTC, toUTC time.Time, opt ports.ListBookingsOptions) ([]domain.Booking, string, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	lo := "BOOKING_DATE#" + fromUTC.UTC().Format(time.RFC3339Nano)
	hi := "BOOKING_DATE#" + toUTC.UTC().Format(time.RFC3339Nano) + "#ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.Table),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND GSI1SK BETWEEN :lo AND :hi"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			":lo": &types.AttributeValueMemberS{Value: lo},
			":hi": &types.AttributeValueMemberS{Value: hi},
		},
		Limit: aws.Int32(limit),
	}
	if opt.Cursor != "" {
		eks, err := decodeBookingListCursor(opt.Cursor)
		if err != nil {
			return nil, "", err
		}
		input.ExclusiveStartKey = eks
	}
	out, err := r.Client.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	var list []domain.Booking
	for _, it := range out.Items {
		b, err := unmarshalBooking(it)
		if err != nil {
			return nil, "", err
		}
		list = append(list, *b)
	}
	next := ""
	if len(out.LastEvaluatedKey) > 0 {
		next, err = encodeBookingListCursor(out.LastEvaluatedKey)
		if err != nil {
			return nil, "", err
		}
	}
	return list, next, nil
}

func (r *BookingRepository) FindStaffCandidatesBefore(ctx context.Context, businessID, staffID string, windowEnd time.Time, limit int32) ([]domain.Booking, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	upper := "BOOKING_DATE#" + windowEnd.UTC().Format(time.RFC3339Nano) + "#\uffff"
	var all []domain.Booking
	var eks map[string]types.AttributeValue
	for i := 0; i < 20; i++ {
		input := &dynamodb.QueryInput{
			TableName:              aws.String(r.Table),
			IndexName:              aws.String("GSI1"),
			KeyConditionExpression: aws.String("GSI1PK = :pk AND GSI1SK < :upper"),
			FilterExpression:       aws.String("staff_id = :sid"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk":    &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
				":upper": &types.AttributeValueMemberS{Value: upper},
				":sid":   &types.AttributeValueMemberS{Value: staffID},
			},
			Limit: aws.Int32(limit),
		}
		if eks != nil {
			input.ExclusiveStartKey = eks
		}
		out, err := r.Client.Query(ctx, input)
		if err != nil {
			return nil, err
		}
		for _, it := range out.Items {
			b, err := unmarshalBooking(it)
			if err != nil {
				return nil, err
			}
			all = append(all, *b)
		}
		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		eks = out.LastEvaluatedKey
	}
	return all, nil
}

func marshalBooking(b *domain.Booking) (map[string]types.AttributeValue, error) {
	g1sk := bookingGSI1SK(b.StartAt, b.ID)
	row := bookingItem{
		PK:           BusinessPK(b.BusinessID),
		SK:           bookingSK(b.ID),
		GSI1PK:       BusinessPK(b.BusinessID),
		GSI1SK:       g1sk,
		Entity:       entityBooking,
		ID:           b.ID,
		BusinessID:   b.BusinessID,
		CustomerID:   b.CustomerID,
		ServiceID:    b.ServiceID,
		StaffID:      b.StaffID,
		StartAt:      b.StartAt.UTC().Format(time.RFC3339Nano),
		EndAt:        b.EndAt.UTC().Format(time.RFC3339Nano),
		Status:       string(b.Status),
		CancelReason: b.CancelReason,
		CreatedAt:    b.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:    b.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalBooking(item map[string]types.AttributeValue) (*domain.Booking, error) {
	var row bookingItem
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	startAt, err := parseDynamoTime(row.StartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := parseDynamoTime(row.EndAt)
	if err != nil {
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
	return &domain.Booking{
		ID:           row.ID,
		BusinessID:   row.BusinessID,
		CustomerID:   row.CustomerID,
		ServiceID:    row.ServiceID,
		StaffID:      row.StaffID,
		StartAt:      startAt,
		EndAt:        endAt,
		Status:       domain.BookingStatus(row.Status),
		CancelReason: row.CancelReason,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

var _ ports.BookingRepository = (*BookingRepository)(nil)
