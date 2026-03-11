package auth

import (
	"testing"
)

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		wantErr  bool
	}{
		// Length checks
		{"Short123!", true},          // Length 9
		{"Short12345!", true},        // Length 11
		{"Shorter1234!", false},       // Length 12, 4 categories

		// Complexity checks (exactly 12 characters)
		{"alllowercase", true},        // 1 category (lower)
		{"123456789012", true},        // 1 category (num)
		{"!!!!!!!!!!!!", true},        // 1 category (spec)
		{"ABCDEFGHIJKL", true},        // 1 category (upper)

		{"lower1234567", true},        // 2 categories (lower, num)
		{"LOWERUPPER12", true},        // 2 categories (upper, num)
		{"lowerUPPER!!", false},       // 3 categories (lower, upper, spec)

		{"lowerupper!!", true},        // 2 categories (lower, spec)
		{"1234567890!!", true},        // 2 categories (num, spec)

		{"Password1234", false},       // 3 categories (upper, lower, num)
		{"Lowerupper!!", false},       // 3 categories (upper, lower, spec)
		{"Lower12345!!", false},       // 3 categories (lower, num, spec)
		{"UPPER12345!!", false},       // 3 categories (upper, num, spec)

		{"Password123!", false},       // 4 categories (upper, lower, num, spec)

		// Edge cases
		{"VeryLongPasswordWithManyCategories123!", false},
		{"ExactlyTwelve", true},        // Only 2 categories (upper, lower)
		{"ExactlyTwel1", false},       // 3 categories (upper, lower, num)
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTableName(t *testing.T) {
	tests := []struct {
		table   string
		wantErr bool
	}{
		{"parts", false},
		{"users", false},
		{"ecos", false},
		{"nonexistent", true},
		{"parts; DROP TABLE users", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			err := ValidateTableName(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTableName(%q) error = %v, wantErr %v", tt.table, err, tt.wantErr)
			}
		})
	}
}

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		column  string
		wantErr bool
	}{
		{"id", false},
		{"username", false},
		{"created_at", false},
		{"invalid_col", true},
		{"id; SELECT *", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.column, func(t *testing.T) {
			err := ValidateColumnName(tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateColumnName(%q) error = %v, wantErr %v", tt.column, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		identifier string
		want       string
		wantErr    bool
	}{
		{"valid_123", "valid_123", false},
		{"UserID", "UserID", false},
		{"invalid-dash", "", true},
		{"space identifier", "", true},
		{"semi;colon", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.identifier, func(t *testing.T) {
			got, err := SanitizeIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeIdentifier(%q) error = %v, wantErr %v", tt.identifier, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeIdentifier(%q) got = %v, want %v", tt.identifier, got, tt.want)
			}
		})
	}
}
