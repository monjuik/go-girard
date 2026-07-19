package common

import "testing"

func TestIDFromString(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid snowflake id",
			value:   "1992328621821009920",
			wantErr: false,
		},
		{
			name:    "invalid snowflake id",
			value:   "not-a-snowflake-id",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "zero id",
			value:   "0",
			wantErr: true,
		},
		{
			name:    "negative id",
			value:   "-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := IDFromString(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatal("IDFromString() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("IDFromString() error = %v", err)
			}

			if !id.IsValid() {
				t.Fatalf("IDFromString() returned invalid ID %d", id)
			}

			if id.String() != tt.value {
				t.Fatalf("id.String() = %q, want %q", id.String(), tt.value)
			}
		})
	}
}
