package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LeoPani/agora/backend/internal/llm"
)

// Agent implements a ReAct (Reason + Act) loop using LLM tool calling.
type Agent struct {
	Name     string
	System   string
	Tools    []Tool
	LLM      *llm.Router
	MaxSteps int
}

// Result is the final output of an agent run.
type Result struct {
	Text       string
	StepsUsed  int
	ContextLog []StepRecord
}

// StepRecord logs one iteration of the ReAct loop.
type StepRecord struct {
	Step       int    `json:"step"`
	ToolName   string `json:"tool,omitempty"`
	ToolInput  string `json:"input,omitempty"`
	ToolOutput string `json:"output,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
}

// Run executes the agent loop until a final answer is produced or MaxSteps is reached.
func (a *Agent) Run(ctx context.Context, goal string) (*Result, error) {
	maxSteps := a.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 8
	}

	// Build the tool list for the LLM
	llmTools := make([]llm.Tool, len(a.Tools))
	for i, t := range a.Tools {
		llmTools[i] = llm.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		}
	}

	// Start with a system message and the goal
	messages := []llm.Message{
		{Role: "system", Content: a.System},
		{Role: "user", Content: goal},
	}

	result := &Result{}

	for step := 0; step < maxSteps; step++ {
		resp, err := a.LLM.Complete(ctx, llm.CompletionRequest{
			Purpose:     "agent_action",
			Messages:    messages,
			Tools:       llmTools,
			Temperature: 0.3,
			MaxTokens:   2048,
		})
		if err != nil {
			return nil, fmt.Errorf("agent %s step %d: %w", a.Name, step, err)
		}

		result.StepsUsed = step + 1

		// If no tool calls, this is the final answer
		if len(resp.ToolCalls) == 0 {
			result.Text = resp.Text
			return result, nil
		}

		// Append the assistant's tool-call message
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			tool := a.findTool(tc.Name)
			var toolOutput string
			if tool == nil {
				toolOutput = fmt.Sprintf(`{"error":"tool %q not found"}`, tc.Name)
			} else {
				var args map[string]any
				json.Unmarshal([]byte(tc.ArgsJSON), &args)
				out, err := tool.Execute(ctx, args)
				if err != nil {
					toolOutput = fmt.Sprintf(`{"error":%q}`, err.Error())
				} else {
					toolOutput = out
				}
			}

			result.ContextLog = append(result.ContextLog, StepRecord{
				Step:       step + 1,
				ToolName:   tc.Name,
				ToolInput:  tc.ArgsJSON,
				ToolOutput: toolOutput,
			})

			messages = append(messages, llm.Message{
				Role:       "tool",
				Content:    toolOutput,
				ToolCallID: tc.ID,
			})
		}
	}

	return nil, fmt.Errorf("agent %s: max steps (%d) reached without final answer", a.Name, maxSteps)
}

func (a *Agent) findTool(name string) Tool {
	for _, t := range a.Tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}
