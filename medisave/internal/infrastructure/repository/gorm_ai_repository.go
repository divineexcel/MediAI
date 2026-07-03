package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
)

type GORMAIRepository struct {
	db *gorm.DB
}

func NewGORMAIRepository(db *gorm.DB) domainrepo.AIRepository {
	return &GORMAIRepository{db: db}
}

func (r *GORMAIRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMAIRepository) CreateConversation(ctx context.Context, c *entity.AIConversation) error {
	return r.dbc(ctx).Create(c).Error
}

func (r *GORMAIRepository) FindConversationByID(ctx context.Context, id uint) (*entity.AIConversation, error) {
	var c entity.AIConversation
	err := r.dbc(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &c, err
}

func (r *GORMAIRepository) FindActiveConversation(ctx context.Context, patientID uint) (*entity.AIConversation, error) {
	var c entity.AIConversation
	err := r.dbc(ctx).
		Where("patient_id = ? AND is_active = true", patientID).
		Order("created_at DESC").
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &c, err
}

func (r *GORMAIRepository) CloseConversation(ctx context.Context, id uint) error {
	return r.dbc(ctx).
		Model(&entity.AIConversation{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

func (r *GORMAIRepository) CreateMessage(ctx context.Context, m *entity.AIMessage) error {
	return r.dbc(ctx).Create(m).Error
}

func (r *GORMAIRepository) ListMessages(ctx context.Context, conversationID uint) ([]*entity.AIMessage, error) {
	var msgs []*entity.AIMessage
	err := r.dbc(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

func (r *GORMAIRepository) ListConversations(ctx context.Context, patientID uint) ([]*entity.AIConversation, error) {
	var list []*entity.AIConversation
	err := r.dbc(ctx).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

func (r *GORMAIRepository) CountMessagesToday(ctx context.Context, patientID uint) (int64, error) {
	var count int64
	err := r.dbc(ctx).
		Model(&entity.AIMessage{}).
		Joins("JOIN ai_conversations ON ai_conversations.id = ai_messages.conversation_id").
		Where("ai_conversations.patient_id = ? AND ai_messages.role = 'user' AND DATE(ai_messages.created_at) = DATE('now')", patientID).
		Count(&count).Error
	return count, err
}
