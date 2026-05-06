package dynamo

import (
	"context"
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

// NotificationRepository implements [ports.NotificationRepository].
type NotificationRepository struct {
	Client *dynamodb.Client
	Table  string
}

type notificationRow struct {
	PK            string         `dynamodbav:"PK"`
	SK            string         `dynamodbav:"SK"`
	GSI4PK        string         `dynamodbav:"GSI4PK"`
	GSI4SK        string         `dynamodbav:"GSI4SK"`
	Entity        string         `dynamodbav:"entity"`
	ID            string         `dynamodbav:"id"`
	BusinessID    string         `dynamodbav:"business_id"`
	Kind          string         `dynamodbav:"kind"`
	Channel       string         `dynamodbav:"channel"`
	Status        string         `dynamodbav:"status"`
	ScheduledAt   string         `dynamodbav:"scheduled_at"`
	CustomerID    string         `dynamodbav:"customer_id,omitempty"`
	BookingID     string         `dynamodbav:"booking_id,omitempty"`
	Payload       map[string]any `dynamodbav:"payload,omitempty"`
	CreatedAt     string         `dynamodbav:"created_at"`
	UpdatedAt     string         `dynamodbav:"updated_at"`
}

func gsi4KeysForNotification(n *domain.Notification) (pk, sk string) {
	switch n.Status {
	case domain.NotificationScheduled:
		return gsi4NotificationDue, notificationGSI4ScheduledSK(n.ScheduledAt, n.ID)
	case domain.NotificationSent:
		return gsi4NotificationDue + "#SENT", notificationGSI4SentSK(n.ID)
	case domain.NotificationFailed:
		return gsi4NotificationDue + "#FAILED", notificationGSI4FailedSK(n.ID)
	default:
		return gsi4NotificationDue + "#UNKNOWN", notificationGSI4FailedSK(n.ID)
	}
}

func marshalNotification(n *domain.Notification) (map[string]types.AttributeValue, error) {
	g4pk, g4sk := gsi4KeysForNotification(n)
	row := notificationRow{
		PK:          BusinessPK(n.BusinessID),
		SK:          notificationSK(n.ID),
		GSI4PK:      g4pk,
		GSI4SK:      g4sk,
		Entity:      entityNotification,
		ID:          n.ID,
		BusinessID:  n.BusinessID,
		Kind:        string(n.Kind),
		Channel:     n.Channel,
		Status:      string(n.Status),
		ScheduledAt: n.ScheduledAt.UTC().Format(time.RFC3339Nano),
		CustomerID:  n.CustomerID,
		BookingID:   n.BookingID,
		Payload:     n.Payload,
		CreatedAt:   n.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   n.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalNotificationItem(item map[string]types.AttributeValue) (*domain.Notification, error) {
	var row notificationRow
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	scheduledAt, err := parseDynamoTime(row.ScheduledAt)
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
	return &domain.Notification{
		ID:          row.ID,
		BusinessID:  row.BusinessID,
		Kind:        domain.NotificationKind(row.Kind),
		Channel:     row.Channel,
		Status:      domain.NotificationStatus(row.Status),
		ScheduledAt: scheduledAt,
		CustomerID:  row.CustomerID,
		BookingID:   row.BookingID,
		Payload:     row.Payload,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// Create persists a new notification.
func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	if n.Status == "" {
		n.Status = domain.NotificationScheduled
	}
	item, err := marshalNotification(n)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.Table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	})
	return err
}

// Get returns one notification by id.
func (r *NotificationRepository) Get(ctx context.Context, businessID, notificationID string) (*domain.Notification, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: notificationSK(notificationID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalNotificationItem(out.Item)
}

// List returns notifications for a business (optionally filtered by status).
func (r *NotificationRepository) List(ctx context.Context, businessID string, opt ports.ListNotificationsOptions) ([]domain.Notification, string, error) {
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
			":prefix": &types.AttributeValueMemberS{Value: "NOTIF#"},
		},
		Limit: aws.Int32(limit),
	}
	if st := strings.TrimSpace(opt.Status); st != "" {
		input.FilterExpression = aws.String("#st = :st")
		input.ExpressionAttributeNames = map[string]string{"#st": "status"}
		input.ExpressionAttributeValues[":st"] = &types.AttributeValueMemberS{Value: st}
	}
	if opt.Cursor != "" {
		eks, err := decodeExclusiveStartKey(opt.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("%w: bad cursor", domain.ErrInvalid)
		}
		input.ExclusiveStartKey = eks
	}
	out, err := r.Client.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	var list []domain.Notification
	for _, item := range out.Items {
		n, err := unmarshalNotificationItem(item)
		if err != nil {
			return nil, "", err
		}
		list = append(list, *n)
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

// ListDueScheduled queries GSI4 for scheduled notifications due at or before beforeUTC.
func (r *NotificationRepository) ListDueScheduled(ctx context.Context, beforeUTC time.Time, limit int32) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	untilSK := fmt.Sprintf("SCHED#%020d#\uffff", beforeUTC.UTC().UnixNano())
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.Table),
		IndexName:              aws.String("GSI4"),
		KeyConditionExpression: aws.String("GSI4PK = :pk AND GSI4SK <= :until"),
		FilterExpression:       aws.String("#st = :scheduled"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":         &types.AttributeValueMemberS{Value: gsi4NotificationDue},
			":until":      &types.AttributeValueMemberS{Value: untilSK},
			":scheduled":  &types.AttributeValueMemberS{Value: string(domain.NotificationScheduled)},
		},
		Limit: aws.Int32(limit),
	}
	out, err := r.Client.Query(ctx, input)
	if err != nil {
		return nil, err
	}
	var list []domain.Notification
	for _, item := range out.Items {
		n, err := unmarshalNotificationItem(item)
		if err != nil {
			return nil, err
		}
		list = append(list, *n)
	}
	return list, nil
}

// Save replaces a notification row (including GSI4 projection).
func (r *NotificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	item, err := marshalNotification(n)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

var _ ports.NotificationRepository = (*NotificationRepository)(nil)
