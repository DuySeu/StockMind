package agent

import (
	"context"
	"fmt"
	"log"
	"stockmind/internal/database"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	openai "github.com/sashabaranov/go-openai"
)

type LLMClientWrapper struct {
	OfOpenAI    *openai.Client
	OfAnthropic *anthropic.Client
}

type AgentService struct {
	config  LLMProviderConfig
	queries *database.Queries
	ctx     context.Context
}

func NewService(ctx context.Context, dbPool *pgxpool.Pool, config LLMProviderConfig) (*AgentService, error) {
	log.Println("Initializing LLM service...")
	
	return &AgentService{
		config:  config,
		ctx:     ctx,
		queries: database.New(dbPool),
	}, nil
}

// getImplByProvider returns an initialized struct that implements LLMProvider interface
// The returned struct must NOT have the 'agent' field set yet, as the agent is not created.
// This requires a slight adjustment: implementing structs (OpenAIProvider/AnthropicProvider)
// should allow setting the agent later or the interface needs to account for this.
// Given the existing design where Agent accesses Config/Tools, passing Agent to the Provider is correct.
// But we can't create Provider with Agent because Agent needs Provider. Circular dependency.
// Solution:
// 1. Create LLMClientWrapper (RAW client) first.
// 2. Create Agent with RAW client or placeholder.
// 3. Construct concrete Provider (wrapping RAW client + Agent) and assign to Agent.
func (s *AgentService) getClientByProvider(provider database.ModelProvider) (*LLMClientWrapper, error) {
	var client *LLMClientWrapper
	var err error
	switch provider {
	case database.ModelProviderOpenAI:
		client, err = createOpenAIClient(s.config.OpenAI)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenRouter client: %v", err)
		}
	case database.ModelProviderAnthropic:
		client, err = createAnthropicClient(s.ctx, s.config.Anthropic)
		if err != nil {
			return nil, fmt.Errorf("failed to create Anthropic client: %v", err)
		}
	default:
		log.Printf("Unsupported model provider: %s", string(provider))
		return nil, fmt.Errorf("unsupported model provider: %s", string(provider))
	}
	return client, nil
}

func (s *AgentService) GetOrCreateSession(userID, agentFlowID, sessionID *uuid.UUID, sessionName *string) (*SessionManager, error) {
	// Create a new session in the database
	var session database.Session
	var agentFlow database.AgentFlow
	var err error
	if sessionID == nil {
		newUUID := uuid.Must(uuid.NewV7())
		sessionID = &newUUID
		if userID == nil || agentFlowID == nil {
			return nil, fmt.Errorf("userID and agentFlowID are required to create a new session")
		}
		// Get Agent Flow Configuration
		agentFlow, err = s.queries.GetAgentFlowById(s.ctx, *agentFlowID)
		if err != nil {
			log.Printf("failed to get agent flow by ID: %v (agent_flow_id: %s)", err, *agentFlowID)
			return nil, err
		}
		log.Printf("creating new session (user_id: %s, agent_flow_id: %s, agent_flow_name: %s)", *userID, *agentFlowID, agentFlow.Name)
		newSessionName := "New Session"
		if sessionName != nil && *sessionName != "" {
			newSessionName = *sessionName
		}
		session, err = s.queries.CreateSession(s.ctx, database.CreateSessionParams{
			ID:          *sessionID,
			CreatedBy:   *userID,
			AgentFlowID: *agentFlowID,
			Title:       newSessionName,
		})
		if err != nil {
			log.Printf("failed to create new session: %v", err)
			return nil, err
		}
		log.Printf("new session created (session_id: %s, user_id: %s)", *sessionID, *userID)
	} else {
		// Fetch existing session from the database
		session, err = s.queries.GetSessionByID(s.ctx, *sessionID)
		if err != nil {
			log.Printf("failed to get session by ID: %v (session_id: %s)", err, *sessionID)
			return nil, err
		}
		// Get Agent Flow Configuration
		agentFlow, err = s.queries.GetAgentFlowById(s.ctx, session.AgentFlowID)
		if err != nil {
			log.Printf("failed to get agent flow by ID: %v (agent_flow_id: %s)", err, session.AgentFlowID)
			return nil, err
		}
		log.Printf("existing session fetched (session_id: %s, user_id: %s, agent_flow_id: %s, agent_flow_name: %s)", *sessionID, session.CreatedBy, session.AgentFlowID, agentFlow.Name)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	sm := &SessionManager{
		ctx:          ctx,
		cancel:       cancel,
		session:      session,
		agentFlowCfg: agentFlow.Config,
		llm:          s,
		history:      []database.SessionHistory{},
		nodes:        make(map[string]database.Node),
	}
	err = sm.Initialize()
	return sm, err
}
