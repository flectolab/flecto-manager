package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		t    AgentType
		want bool
	}{
		{
			name: "valid default type",
			t:    AgentTypeDefault,
			want: true,
		},
		{
			name: "valid traefik type",
			t:    AgentTypeTraefik,
			want: true,
		},
		{
			name: "invalid type",
			t:    AgentType("invalid"),
			want: false,
		},
		{
			name: "empty type",
			t:    AgentType(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.t.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAgentStatus_IsValid(t *testing.T) {
	tests := []struct {
		name string
		s    AgentStatus
		want bool
	}{
		{
			name: "valid success status",
			s:    AgentStatusSuccess,
			want: true,
		},
		{
			name: "valid error status",
			s:    AgentStatusError,
			want: true,
		},
		{
			name: "invalid status",
			s:    AgentStatus("invalid"),
			want: false,
		},
		{
			name: "empty status",
			s:    AgentStatus(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateAgent(t *testing.T) {
	tests := []struct {
		name    string
		agent   Agent
		wantErr string
	}{
		{
			name: "valid agent with status",
			agent: Agent{
				Name:         "my-agent",
				Type:         AgentTypeTraefik,
				Status:       AgentStatusSuccess,
				Version:      1,
				LoadDuration: 100 * time.Millisecond,
			},
			wantErr: "",
		},
		{
			name: "valid agent without status",
			agent: Agent{
				Name:         "my_agent_123",
				Type:         AgentTypeDefault,
				Version:      1,
				LoadDuration: 50 * time.Millisecond,
			},
			wantErr: "",
		},
		{
			name: "invalid name with spaces",
			agent: Agent{
				Name: "my agent",
				Type: AgentTypeDefault,
			},
			wantErr: "invalid agent name: only alphanumeric characters, underscores and hyphens are allowed",
		},
		{
			name: "invalid name with special chars",
			agent: Agent{
				Name: "agent@test",
				Type: AgentTypeDefault,
			},
			wantErr: "invalid agent name: only alphanumeric characters, underscores and hyphens are allowed",
		},
		{
			name: "empty name",
			agent: Agent{
				Name: "",
				Type: AgentTypeDefault,
			},
			wantErr: "invalid agent name: only alphanumeric characters, underscores and hyphens are allowed",
		},
		{
			name: "invalid type",
			agent: Agent{
				Name: "my-agent",
				Type: AgentType("unknown"),
			},
			wantErr: "invalid agent type: unknown",
		},
		{
			name: "empty type",
			agent: Agent{
				Name: "my-agent",
				Type: AgentType(""),
			},
			wantErr: "invalid agent type: ",
		},
		{
			name: "invalid status",
			agent: Agent{
				Name:    "my-agent",
				Type:    AgentTypeTraefik,
				Version: 1,
				Status:  AgentStatus("pending"),
			},
			wantErr: "invalid agent status: pending",
		},
		{
			name: "missing version",
			agent: Agent{
				Name: "my-agent",
				Type: AgentTypeTraefik,
			},
			wantErr: "agent version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgent(tt.agent)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
