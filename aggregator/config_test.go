package aggregator

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config[int]
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config[int]{
				MaxDuration:       10 * time.Second,
				MaxBufferedEvents: 5,
				QueueSize:         16,
				Handler:           func(events []int) {},
			},
			wantErr: false,
		},
		{
			name: "invalid MaxDuration",
			config: Config[int]{
				MaxDuration:       0,
				MaxBufferedEvents: 5,
				QueueSize:         10,
				Handler:           func(events []int) {},
			},
			wantErr: true,
		},
		{
			name: "invalid MaxBufferedEvents",
			config: Config[int]{
				MaxDuration:       10 * time.Second,
				MaxBufferedEvents: 0,
				QueueSize:         10,
				Handler:           func(events []int) {},
			},
			wantErr: true,
		},
		{
			name: "invalid QueueSize",
			config: Config[int]{
				MaxDuration:       10 * time.Second,
				MaxBufferedEvents: 5,
				QueueSize:         0,
				Handler:           func(events []int) {},
			},
			wantErr: true,
		},
		{
			name: "invalid QueueSize - not power of 2",
			config: Config[int]{
				MaxDuration:       10 * time.Second,
				MaxBufferedEvents: 5,
				QueueSize:         3,
				Handler:           func(events []int) {},
			},
			wantErr: true,
		},
		{
			name: "nil Handler",
			config: Config[int]{
				MaxDuration:       10 * time.Second,
				MaxBufferedEvents: 5,
				QueueSize:         10,
				Handler:           nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
