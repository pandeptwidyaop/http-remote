// Package validation provides input validation and sanitization utilities.
package validation

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	// ErrPasswordTooShort indicates password is less than minimum length.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	// ErrPasswordNoUppercase indicates password has no uppercase letter.
	ErrPasswordNoUppercase = errors.New("password must contain at least one uppercase letter")
	// ErrPasswordNoLowercase indicates password has no lowercase letter.
	ErrPasswordNoLowercase = errors.New("password must contain at least one lowercase letter")
	// ErrPasswordNoDigit indicates password has no digit.
	ErrPasswordNoDigit = errors.New("password must contain at least one digit")
	// ErrPasswordNoSpecial indicates password has no special character.
	ErrPasswordNoSpecial = errors.New("password must contain at least one special character")
	// ErrPasswordCommon indicates password is too common.
	ErrPasswordCommon = errors.New("password is too common, please choose a stronger password")
	// ErrInputTooLong indicates input exceeds maximum length.
	ErrInputTooLong = errors.New("input exceeds maximum length")
	// ErrInputInvalid indicates input contains invalid characters.
	ErrInputInvalid = errors.New("input contains invalid characters")
)

// PasswordPolicy defines password requirements.
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireDigit     bool
	RequireSpecial   bool
	CheckCommon      bool
}

// DefaultPasswordPolicy returns the recommended password policy.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
		CheckCommon:      true,
	}
}

// Common passwords that should be rejected
var commonPasswords = map[string]bool{
	"password":    true,
	"123456":      true,
	"12345678":    true,
	"qwerty":      true,
	"abc123":      true,
	"password1":   true,
	"password123": true,
	"admin":       true,
	"letmein":     true,
	"welcome":     true,
	"monkey":      true,
	"dragon":      true,
	"master":      true,
	"login":       true,
	"princess":    true,
	"qwerty123":   true,
	"solo":        true,
	"passw0rd":    true,
	"starwars":    true,
	"iloveyou":    true,
}

// ValidatePassword validates a password against the policy.
func ValidatePassword(password string, policy PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if policy.RequireUppercase && !hasUpper {
		return ErrPasswordNoUppercase
	}
	if policy.RequireLowercase && !hasLower {
		return ErrPasswordNoLowercase
	}
	if policy.RequireDigit && !hasDigit {
		return ErrPasswordNoDigit
	}
	if policy.RequireSpecial && !hasSpecial {
		return ErrPasswordNoSpecial
	}

	if policy.CheckCommon {
		lower := strings.ToLower(password)
		if commonPasswords[lower] {
			return ErrPasswordCommon
		}
	}

	return nil
}

// ValidatePasswordWithDefault validates using default policy.
func ValidatePasswordWithDefault(password string) error {
	return ValidatePassword(password, DefaultPasswordPolicy())
}

// PasswordStrength returns a score from 0-100 indicating password strength.
func PasswordStrength(password string) int {
	score := 0
	length := len(password)

	// Length scoring
	if length >= 8 {
		score += 20
	}
	if length >= 12 {
		score += 10
	}
	if length >= 16 {
		score += 10
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Character type scoring
	if hasUpper {
		score += 15
	}
	if hasLower {
		score += 15
	}
	if hasDigit {
		score += 15
	}
	if hasSpecial {
		score += 15
	}

	// Penalize common passwords
	if commonPasswords[strings.ToLower(password)] {
		score = 0
	}

	if score > 100 {
		score = 100
	}

	return score
}

// SanitizeString removes potentially dangerous characters from input.
// This is for XSS prevention in text fields.
func SanitizeString(input string) string {
	// HTML escape to prevent XSS
	sanitized := html.EscapeString(input)
	// Trim whitespace
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// SanitizeStringPreserveNewlines sanitizes but keeps newlines for multiline fields.
func SanitizeStringPreserveNewlines(input string) string {
	// Replace newlines with placeholder
	input = strings.ReplaceAll(input, "\r\n", "\n")
	lines := strings.Split(input, "\n")

	// Sanitize each line
	for i, line := range lines {
		lines[i] = html.EscapeString(strings.TrimSpace(line))
	}

	return strings.Join(lines, "\n")
}

// ValidateName validates a name field (app name, command name, etc.)
func ValidateName(name string, maxLength int) error {
	if len(name) > maxLength {
		return ErrInputTooLong
	}

	// Allow alphanumeric, spaces, hyphens, underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9\s\-_]+$`)
	if !validName.MatchString(name) {
		return ErrInputInvalid
	}

	return nil
}

// ValidateDescription validates a description field.
func ValidateDescription(desc string, maxLength int) error {
	if len(desc) > maxLength {
		return ErrInputTooLong
	}
	return nil
}

// ValidatePath validates a file system path.
func ValidatePath(path string) error {
	// Prevent path traversal
	if strings.Contains(path, "..") {
		return ErrInputInvalid
	}

	// Must be absolute path
	if !strings.HasPrefix(path, "/") {
		return ErrInputInvalid
	}

	// Disallow null bytes and other dangerous characters
	if strings.ContainsAny(path, "\x00\n\r") {
		return ErrInputInvalid
	}

	return nil
}

// ValidateCommand validates a shell command string.
// Note: This is basic validation - proper escaping should be handled at execution.
func ValidateCommand(command string, maxLength int) error {
	if len(command) > maxLength {
		return ErrInputTooLong
	}

	// Disallow null bytes
	if strings.Contains(command, "\x00") {
		return ErrInputInvalid
	}

	return nil
}

// StripHTML removes all HTML tags from a string.
func StripHTML(input string) string {
	// Simple regex to strip HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// ValidateUsername validates a username.
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 50 {
		return ErrInputTooLong
	}

	// Alphanumeric and underscores only
	validUsername := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	if !validUsername.MatchString(username) {
		return errors.New("username must start with a letter and contain only letters, numbers, and underscores")
	}

	return nil
}
