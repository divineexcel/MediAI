package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
)

type AIRepository interface {
	CreateConversation(ctx context.Context, c *entity.AIConversation) error
	FindConversationByID(ctx context.Context, id uint) (*entity.AIConversation, error)
	FindActiveConversation(ctx context.Context, patientID uint) (*entity.AIConversation, error)
	CloseConversation(ctx context.Context, id uint) error
	CreateMessage(ctx context.Context, m *entity.AIMessage) error
	ListMessages(ctx context.Context, conversationID uint) ([]*entity.AIMessage, error)
	ListConversations(ctx context.Context, patientID uint) ([]*entity.AIConversation, error)
	CountMessagesToday(ctx context.Context, patientID uint) (int64, error)
}
