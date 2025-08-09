package types

import (
	"testing"
)

func TestInvocationRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   InvocationRequest
		wantError bool
	}{
		{
			name: "valid request",
			request: InvocationRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "user", Content: "hello"},
				},
			},
			wantError: false,
		},
		{
			name:      "empty model",
			request:   InvocationRequest{Messages: []Message{{Role: "user", Content: "hello"}}},
			wantError: true,
		},
		{
			name:      "no messages",
			request:   InvocationRequest{Model: "gpt-3.5-turbo"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestInvocationRequest_SetDefaults(t *testing.T) {
	req := &InvocationRequest{}
	req.SetDefaults()

	if req.Stream == nil || !*req.Stream {
		t.Error("SetDefaults() should set Stream to true")
	}

	if req.Metadata == nil {
		t.Error("SetDefaults() should initialize Metadata map")
	}
}