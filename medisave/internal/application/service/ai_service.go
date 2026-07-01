package service

import (
	"context"
	"fmt"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	aiclient "github.com/medisave/app/internal/infrastructure/external/ai"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/utils"
)

const (
	dailyMessageLimit = 20
)

type AIService interface {
	Chat(ctx context.Context, userID uint, req *dto.AIChatRequest) (*dto.AIMessageResponse, error)
	GetConversations(ctx context.Context, userID uint) ([]*entity.AIConversation, error)
	GetMessages(ctx context.Context, userID uint, conversationID uint) ([]*entity.AIMessage, error)
	ClearConversation(ctx context.Context, userID uint, conversationID uint) error
}

type aiService struct {
	aiRepo      repository.AIRepository
	patientRepo repository.PatientRepository
	aiClient    *aiclient.Client
}

func NewAIService(
	aiRepo repository.AIRepository,
	patientRepo repository.PatientRepository,
	aiClient *aiclient.Client,
) AIService {
	return &aiService{
		aiRepo:      aiRepo,
		patientRepo: patientRepo,
		aiClient:    aiClient,
	}
}

func (s *aiService) Chat(ctx context.Context, userID uint, req *dto.AIChatRequest) (*dto.AIMessageResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	// Enforce daily limit
	count, _ := s.aiRepo.CountMessagesToday(ctx, patient.ID)
	if count >= dailyMessageLimit {
		return nil, fmt.Errorf("daily AI message limit of %d reached. Try again tomorrow", dailyMessageLimit)
	}

	// Get or create active conversation
	conv, err := s.aiRepo.FindActiveConversation(ctx, patient.ID)
	if err == pkgerrors.ErrNotFound {
		conv = &entity.AIConversation{
			PatientID: patient.ID,
			SessionID: utils.NewUUID(),
			IsActive:  true,
		}
		if err := s.aiRepo.CreateConversation(ctx, conv); err != nil {
			return nil, pkgerrors.ErrInternalServer
		}
	} else if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Save user message
	userMsg := &entity.AIMessage{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        req.Message,
	}
	if err := s.aiRepo.CreateMessage(ctx, userMsg); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Build conversation history for AI
	history, _ := s.aiRepo.ListMessages(ctx, conv.ID)
	chatHistory := make([]aiclient.ChatMessage, 0, len(history))
	for _, m := range history {
		chatHistory = append(chatHistory, aiclient.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Call AI (external or local fallback)
	result, err := s.aiClient.Analyze(chatHistory, req.Message)
	if err != nil {
		result = &aiclient.AnalysisResult{
			Content:  "I'm having trouble connecting right now. Please try again in a moment, or consult a doctor directly if your symptoms are urgent.",
			Severity: "low",
		}
	}

	// Save assistant message
	var sev *entity.AISeverity
	if result.Severity != "" {
		s := entity.AISeverity(result.Severity)
		sev = &s
	}
	assistantMsg := &entity.AIMessage{
		ConversationID:    conv.ID,
		Role:              "assistant",
		Content:           result.Content,
		Severity:          sev,
		PossibleCondition: result.PossibleCondition,
		RecommendedAction: result.RecommendedAction,
		LabTests:          result.LabTests,
		HomeCare:          result.HomeCare,
		EmergencyWarning:  result.EmergencyWarning,
		ShowBookDoctor:    result.ShowBookDoctor,
		ShowFindHospital:  result.ShowFindHospital,
		ShowEmergencySOS:  result.ShowEmergencySOS,
	}
	if err := s.aiRepo.CreateMessage(ctx, assistantMsg); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return &dto.AIMessageResponse{
		ID:                assistantMsg.ID,
		Role:              assistantMsg.Role,
		Content:           assistantMsg.Content,
		Severity:          assistantMsg.Severity,
		PossibleCondition: assistantMsg.PossibleCondition,
		RecommendedAction: assistantMsg.RecommendedAction,
		LabTests:          assistantMsg.LabTests,
		HomeCare:          assistantMsg.HomeCare,
		EmergencyWarning:  assistantMsg.EmergencyWarning,
		ShowBookDoctor:    assistantMsg.ShowBookDoctor,
		ShowFindHospital:  assistantMsg.ShowFindHospital,
		ShowEmergencySOS:  assistantMsg.ShowEmergencySOS,
		Disclaimer:        "MediSave AI is for informational purposes only and does not replace professional medical advice.",
		CreatedAt:         assistantMsg.CreatedAt,
	}, nil
}

func (s *aiService) GetConversations(ctx context.Context, userID uint) ([]*entity.AIConversation, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}
	return s.aiRepo.ListConversations(ctx, patient.ID)
}

func (s *aiService) GetMessages(ctx context.Context, userID uint, conversationID uint) ([]*entity.AIMessage, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	conv, err := s.aiRepo.FindConversationByID(ctx, conversationID)
	if err != nil {
		return nil, pkgerrors.ErrNotFound
	}
	if conv.PatientID != patient.ID {
		return nil, pkgerrors.ErrForbidden
	}

	return s.aiRepo.ListMessages(ctx, conv.ID)
}

func (s *aiService) ClearConversation(ctx context.Context, userID uint, conversationID uint) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	conv, err := s.aiRepo.FindConversationByID(ctx, conversationID)
	if err != nil {
		return pkgerrors.ErrNotFound
	}
	if conv.PatientID != patient.ID {
		return pkgerrors.ErrForbidden
	}

	return s.aiRepo.CloseConversation(ctx, conv.ID)
}
