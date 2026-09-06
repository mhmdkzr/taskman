package schedule

import "testing"

func TestInputValidateRejectsMultipleScheduleValues(t *testing.T) {
	tests := []struct {
		name  string
		input Input
	}{
		{
			name: "once with delay",
			input: Input{
				ScheduleType: ScheduleTypeOnce,
				RunAt:        "2026-08-14T15:04:05Z",
				RunAfter:     "30m",
			},
		},
		{
			name: "delay with interval",
			input: Input{
				ScheduleType: ScheduleTypeDelay,
				RunAfter:     "30m",
				RunEvery:     "1h",
			},
		},
		{
			name: "interval with once",
			input: Input{
				ScheduleType: ScheduleTypeInterval,
				RunAt:        "2026-08-14T15:04:05Z",
				RunEvery:     "1h",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.input.Validate(); err == nil {
				t.Fatal("Validate() returned nil for multiple schedule values")
			}
		})
	}
}
