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

func FuzzIDFromString(f *testing.F) {
	for _, seed := range []string{
		"1992328621821009920",
		"1",
		"+1",
		"0001",
		"0",
		"-1",
		"",
		"not-a-snowflake-id",
		"9223372036854775807",
		"9223372036854775808",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		id, err := IDFromString(value)
		if err != nil {
			return
		}

		if !id.IsValid() {
			t.Fatalf(
				"IDFromString(%q) returned invalid ID %d without error",
				value,
				id,
			)
		}

		canonical := id.String()
		reparsed, err := IDFromString(canonical)
		if err != nil {
			t.Fatalf(
				"IDFromString(%q) succeeded, but canonical form %q failed: %v",
				value,
				canonical,
				err,
			)
		}

		if reparsed != id {
			t.Fatalf(
				"round-trip ID = %d, want %d; input = %q, canonical = %q",
				reparsed,
				id,
				value,
				canonical,
			)
		}
	})
}
