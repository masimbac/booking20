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

// PaymentRepository implements [ports.PaymentRepository].
type PaymentRepository struct {
	Client *dynamodb.Client
	Table  string
}

type idempotencyPaymentItem struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	Entity    string `dynamodbav:"entity"`
	PaymentID string `dynamodbav:"payment_id"`
}

type paymentRow struct {
	PK            string `dynamodbav:"PK"`
	SK            string `dynamodbav:"SK"`
	GSI3PK        string `dynamodbav:"GSI3PK"`
	GSI3SK        string `dynamodbav:"GSI3SK"`
	Entity        string `dynamodbav:"entity"`
	ID            string `dynamodbav:"id"`
	BusinessID    string `dynamodbav:"business_id"`
	BookingID     string `dynamodbav:"booking_id"`
	Amount        string `dynamodbav:"amount,omitempty"`
	Currency      string `dynamodbav:"currency,omitempty"`
	Kind          string `dynamodbav:"kind"`
	Provider      string `dynamodbav:"provider,omitempty"`
	ExternalRef   string `dynamodbav:"external_ref,omitempty"`
	CheckoutURL   string `dynamodbav:"checkout_url,omitempty"`
	Status        string `dynamodbav:"status"`
	CreatedAt     string `dynamodbav:"created_at"`
	UpdatedAt     string `dynamodbav:"updated_at"`
}

func (r *PaymentRepository) lookupPaymentIdempotency(ctx context.Context, businessID, idemSK string) (string, error) {
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
	var row idempotencyPaymentItem
	if err := attributevalue.UnmarshalMap(out.Item, &row); err != nil {
		return "", err
	}
	return row.PaymentID, nil
}

// CreateIfNotExistsWithIdempotency writes payment + idempotency row transactionally.
func (r *PaymentRepository) CreateIfNotExistsWithIdempotency(ctx context.Context, idempotencyKey string, p *domain.Payment) (*domain.Payment, bool, error) {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return nil, false, fmt.Errorf("%w: Idempotency-Key is required", domain.ErrInvalid)
	}
	idemSK := paymentIdempotencySortKey(key)

	if pid, err := r.lookupPaymentIdempotency(ctx, p.BusinessID, idemSK); err != nil {
		return nil, false, err
	} else if pid != "" {
		existing, err := r.Get(ctx, p.BusinessID, pid)
		return existing, false, err
	}

	pItem, err := marshalPayment(p)
	if err != nil {
		return nil, false, err
	}
	iItem, err := attributevalue.MarshalMap(idempotencyPaymentItem{
		PK:        BusinessPK(p.BusinessID),
		SK:        idemSK,
		Entity:    entityIdempotency,
		PaymentID: p.ID,
	})
	if err != nil {
		return nil, false, err
	}
	cond := aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)")
	_, err = r.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{TableName: aws.String(r.Table), Item: iItem, ConditionExpression: cond}},
			{Put: &types.Put{TableName: aws.String(r.Table), Item: pItem, ConditionExpression: cond}},
		},
	})
	if err == nil {
		return p, true, nil
	}
	var tc *types.TransactionCanceledException
	if errors.As(err, &tc) {
		for _, reason := range tc.CancellationReasons {
			if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
				if pid, e2 := r.lookupPaymentIdempotency(ctx, p.BusinessID, idemSK); e2 != nil {
					return nil, false, e2
				} else if pid != "" {
					existing, e3 := r.Get(ctx, p.BusinessID, pid)
					return existing, false, e3
				}
			}
		}
	}
	return nil, false, err
}

func (r *PaymentRepository) Get(ctx context.Context, businessID, paymentID string) (*domain.Payment, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: paymentSK(paymentID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalPayment(out.Item)
}

func (r *PaymentRepository) Save(ctx context.Context, p *domain.Payment) error {
	item, err := marshalPayment(p)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *PaymentRepository) ListByBooking(ctx context.Context, businessID, bookingID string, opt ports.ListPaymentsOptions) ([]domain.Payment, string, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.Table),
		IndexName:              aws.String("GSI3"),
		KeyConditionExpression: aws.String("GSI3PK = :pk AND begins_with(GSI3SK, :pfx)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":  &types.AttributeValueMemberS{Value: paymentGSI3PK(bookingID)},
			":pfx": &types.AttributeValueMemberS{Value: "PAYMENT#"},
		},
		Limit: aws.Int32(limit),
	}
	if opt.Cursor != "" {
		eks, err := decodeGSI3PageCursor(opt.Cursor)
		if err != nil {
			return nil, "", err
		}
		input.ExclusiveStartKey = eks
	}
	out, err := r.Client.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	var list []domain.Payment
	for _, it := range out.Items {
		p, err := unmarshalPayment(it)
		if err != nil {
			return nil, "", err
		}
		if p.BusinessID != businessID {
			continue
		}
		list = append(list, *p)
	}
	next := ""
	if len(out.LastEvaluatedKey) > 0 {
		next, err = encodeGSI3PageCursor(out.LastEvaluatedKey)
		if err != nil {
			return nil, "", err
		}
	}
	return list, next, nil
}

func marshalPayment(p *domain.Payment) (map[string]types.AttributeValue, error) {
	amt := ""
	cur := ""
	if p.Amount != nil {
		amt = p.Amount.Amount
		cur = p.Amount.Currency
	}
	row := paymentRow{
		PK:          BusinessPK(p.BusinessID),
		SK:          paymentSK(p.ID),
		GSI3PK:      paymentGSI3PK(p.BookingID),
		GSI3SK:      paymentGSI3SK(p.CreatedAt, p.ID),
		Entity:      entityPayment,
		ID:          p.ID,
		BusinessID:  p.BusinessID,
		BookingID:   p.BookingID,
		Amount:      amt,
		Currency:    cur,
		Kind:        string(p.Kind),
		Provider:    p.Provider,
		ExternalRef: p.ExternalRef,
		CheckoutURL: p.CheckoutURL,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalPayment(item map[string]types.AttributeValue) (*domain.Payment, error) {
	var row paymentRow
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	if row.Entity != entityPayment {
		return nil, fmt.Errorf("unexpected entity %q", row.Entity)
	}
	created, err := parseDynamoTime(row.CreatedAt)
	if err != nil {
		return nil, err
	}
	updated, err := parseDynamoTime(row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	var money *domain.Money
	if strings.TrimSpace(row.Amount) != "" || strings.TrimSpace(row.Currency) != "" {
		money = &domain.Money{Amount: row.Amount, Currency: row.Currency}
	}
	return &domain.Payment{
		ID:          row.ID,
		BusinessID:  row.BusinessID,
		BookingID:   row.BookingID,
		Amount:      money,
		Kind:        domain.PaymentKind(row.Kind),
		Provider:    row.Provider,
		ExternalRef: row.ExternalRef,
		CheckoutURL: row.CheckoutURL,
		Status:      domain.PaymentStatus(row.Status),
		CreatedAt:   created,
		UpdatedAt:   updated,
	}, nil
}

var _ ports.PaymentRepository = (*PaymentRepository)(nil)
