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

// MessagingRepository implements [ports.MessagingRepository] (conversations + messages + dedup).
type MessagingRepository struct {
	Client *dynamodb.Client
	Table  string
}

type conversationRow struct {
	PK               string         `dynamodbav:"PK"`
	SK               string         `dynamodbav:"SK"`
	Entity           string         `dynamodbav:"entity"`
	ID               string         `dynamodbav:"id"`
	BusinessID       string         `dynamodbav:"business_id"`
	CustomerID       string         `dynamodbav:"customer_id"`
	Channel          string         `dynamodbav:"channel"`
	ProviderThreadID string         `dynamodbav:"provider_thread_id,omitempty"`
	State            string         `dynamodbav:"state"`
	Context          map[string]any `dynamodbav:"context,omitempty"`
	LastActivityAt   string         `dynamodbav:"last_activity_at"`
	CreatedAt        string         `dynamodbav:"created_at"`
	UpdatedAt        string         `dynamodbav:"updated_at"`
}

type convoIndexRow struct {
	PK             string `dynamodbav:"PK"`
	SK             string `dynamodbav:"SK"`
	Entity         string `dynamodbav:"entity"`
	ConversationID string `dynamodbav:"conversation_id"`
	CustomerID     string `dynamodbav:"customer_id"`
	Channel        string `dynamodbav:"channel"`
}

type messageRow struct {
	PK                string         `dynamodbav:"PK"`
	SK                string         `dynamodbav:"SK"`
	Entity            string         `dynamodbav:"entity"`
	ID                string         `dynamodbav:"id"`
	ConversationID    string         `dynamodbav:"conversation_id"`
	Direction         string         `dynamodbav:"direction"`
	Body              string         `dynamodbav:"body"`
	Structured        map[string]any `dynamodbav:"structured,omitempty"`
	ProviderMessageID string         `dynamodbav:"provider_message_id,omitempty"`
	CreatedAt         string         `dynamodbav:"created_at"`
}

type dedupRow struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	Entity     string `dynamodbav:"entity"`
	CreatedAt  string `dynamodbav:"created_at"`
	Provider   string `dynamodbav:"provider"`
	ExternalID string `dynamodbav:"external_id"`
}

func conversationMessagePK(conversationID string) string {
	return "CONVO#" + conversationID
}

func (r *MessagingRepository) GetConversation(ctx context.Context, businessID, conversationID string) (*domain.Conversation, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: conversationSK(conversationID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, domain.ErrNotFound
	}
	return unmarshalConversation(out.Item)
}

func (r *MessagingRepository) GetConversationIDByCustomerChannel(ctx context.Context, businessID, customerID string, channel domain.ConversationChannel) (string, error) {
	out, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.Table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: BusinessPK(businessID)},
			"SK": &types.AttributeValueMemberS{Value: conversationIndexSK(customerID, string(channel))},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	if out.Item == nil {
		return "", domain.ErrNotFound
	}
	var row convoIndexRow
	if err := attributevalue.UnmarshalMap(out.Item, &row); err != nil {
		return "", err
	}
	if row.ConversationID == "" {
		return "", domain.ErrNotFound
	}
	return row.ConversationID, nil
}

func (r *MessagingRepository) CreateConversationWithIndex(ctx context.Context, c *domain.Conversation) error {
	cItem, err := marshalConversation(c)
	if err != nil {
		return err
	}
	idx := convoIndexRow{
		PK:             BusinessPK(c.BusinessID),
		SK:             conversationIndexSK(c.CustomerID, string(c.Channel)),
		Entity:         entityConvoIndex,
		ConversationID: c.ID,
		CustomerID:     c.CustomerID,
		Channel:        string(c.Channel),
	}
	iItem, err := attributevalue.MarshalMap(idx)
	if err != nil {
		return err
	}
	cond := aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)")
	_, err = r.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{TableName: aws.String(r.Table), Item: cItem, ConditionExpression: cond}},
			{Put: &types.Put{TableName: aws.String(r.Table), Item: iItem, ConditionExpression: cond}},
		},
	})
	if err != nil {
		var tc *types.TransactionCanceledException
		if errors.As(err, &tc) {
			for _, reason := range tc.CancellationReasons {
				if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
					return fmt.Errorf("%w", domain.ErrConflict)
				}
			}
		}
	}
	return err
}

func (r *MessagingRepository) SaveConversation(ctx context.Context, c *domain.Conversation) error {
	item, err := marshalConversation(c)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *MessagingRepository) AppendMessage(ctx context.Context, m *domain.Message) error {
	item, err := marshalMessage(m)
	if err != nil {
		return err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.Table),
		Item:      item,
	})
	return err
}

func (r *MessagingRepository) ListMessages(ctx context.Context, conversationID string, opt ports.ListMessagesOptions) ([]domain.Message, string, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.Table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :pfx)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":  &types.AttributeValueMemberS{Value: conversationMessagePK(conversationID)},
			":pfx": &types.AttributeValueMemberS{Value: "MSG#"},
		},
		Limit:            aws.Int32(limit),
		ScanIndexForward: aws.Bool(true),
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
	var list []domain.Message
	for _, it := range out.Items {
		m, err := unmarshalMessage(it)
		if err != nil {
			return nil, "", err
		}
		list = append(list, *m)
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

func (r *MessagingRepository) TryAcquireWebhookDedup(ctx context.Context, businessID, provider, providerMessageID string) (bool, error) {
	if strings.TrimSpace(providerMessageID) == "" {
		return true, nil
	}
	t := time.Now().UTC()
	row := dedupRow{
		PK:         BusinessPK(businessID),
		SK:         webhookDedupSK(provider, providerMessageID),
		Entity:     "WEBHOOK_DEDUP",
		CreatedAt:  t.Format(time.RFC3339Nano),
		Provider:   provider,
		ExternalID: providerMessageID,
	}
	item, err := attributevalue.MarshalMap(row)
	if err != nil {
		return false, err
	}
	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.Table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	})
	if err != nil {
		var cc *types.ConditionalCheckFailedException
		if errors.As(err, &cc) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func marshalConversation(c *domain.Conversation) (map[string]types.AttributeValue, error) {
	row := conversationRow{
		PK:               BusinessPK(c.BusinessID),
		SK:               conversationSK(c.ID),
		Entity:           entityConversation,
		ID:               c.ID,
		BusinessID:       c.BusinessID,
		CustomerID:       c.CustomerID,
		Channel:          string(c.Channel),
		ProviderThreadID: c.ProviderThreadID,
		State:            c.State,
		Context:          c.Context,
		LastActivityAt:   c.LastActivityAt.UTC().Format(time.RFC3339Nano),
		CreatedAt:        c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalConversation(item map[string]types.AttributeValue) (*domain.Conversation, error) {
	var row conversationRow
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	if row.Entity != entityConversation {
		return nil, fmt.Errorf("unexpected entity %q", row.Entity)
	}
	la, err := parseDynamoTime(row.LastActivityAt)
	if err != nil {
		return nil, err
	}
	ca, err := parseDynamoTime(row.CreatedAt)
	if err != nil {
		return nil, err
	}
	ua, err := parseDynamoTime(row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &domain.Conversation{
		ID:               row.ID,
		BusinessID:       row.BusinessID,
		CustomerID:       row.CustomerID,
		Channel:          domain.ConversationChannel(row.Channel),
		ProviderThreadID: row.ProviderThreadID,
		State:            row.State,
		Context:          row.Context,
		LastActivityAt:   la,
		CreatedAt:        ca,
		UpdatedAt:        ua,
	}, nil
}

func marshalMessage(m *domain.Message) (map[string]types.AttributeValue, error) {
	row := messageRow{
		PK:                conversationMessagePK(m.ConversationID),
		SK:                messageSK(m.CreatedAt, m.ID),
		Entity:            entityMessage,
		ID:                m.ID,
		ConversationID:    m.ConversationID,
		Direction:         string(m.Direction),
		Body:              m.Body,
		Structured:        m.Structured,
		ProviderMessageID: m.ProviderMessageID,
		CreatedAt:         m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	return attributevalue.MarshalMap(row)
}

func unmarshalMessage(item map[string]types.AttributeValue) (*domain.Message, error) {
	var row messageRow
	if err := attributevalue.UnmarshalMap(item, &row); err != nil {
		return nil, err
	}
	if row.Entity != entityMessage {
		return nil, fmt.Errorf("unexpected entity %q", row.Entity)
	}
	ca, err := parseDynamoTime(row.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &domain.Message{
		ID:                row.ID,
		ConversationID:    row.ConversationID,
		Direction:         domain.MessageDirection(row.Direction),
		Body:              row.Body,
		Structured:        row.Structured,
		ProviderMessageID: row.ProviderMessageID,
		CreatedAt:         ca,
	}, nil
}

var _ ports.MessagingRepository = (*MessagingRepository)(nil)
