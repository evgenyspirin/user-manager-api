package validator

import (
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"user-manager-api/internal/interface/api/rest/dto/auth"

	"github.com/google/uuid"

	"user-manager-api/internal/interface/api/rest/dto/user"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt safe
)

var (
	e164Re = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)
)

func ValidatePage(page string) (int, error) {
	p := 1
	if page != "" {
		p, err := strconv.Atoi(page)
		if err != nil && p < 0 {
			return p, errors.New("invalid page")
		}
		return p, nil
	}

	return p, nil
}

func IsUUID(s string) (bool, uuid.UUID) {
	id, err := uuid.Parse(s)
	return err == nil, id
}

func ValidateUser(r user.Request) map[string]string {
	errs := make(map[string]string)

	// Normalize
	email := strings.ToLower(strings.TrimSpace(r.Email))
	name := strings.TrimSpace(r.Name)
	last := strings.TrimSpace(r.Lastname)
	bdate := strings.TrimSpace(r.BirthDate)
	phone := strings.TrimSpace(r.Phone)

	// email (required + format)
	if email == "" {
		errs["email"] = "email is required"
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs["email"] = "invalid email format"
	}

	// name (required + length + allowed chars)
	if name == "" {
		errs["name"] = "name is required"
	} else if l := utf8.RuneCountInString(name); l < 2 || l > 64 {
		errs["name"] = "name length must be 2–64 characters"
	} else if !isHumanName(name) {
		errs["name"] = "allowed characters: letters, space, '-', '''"
	}

	// lastname (required + length + allowed chars)
	if last == "" {
		errs["lastname"] = "lastname is required"
	} else if l := utf8.RuneCountInString(last); l < 2 || l > 64 {
		errs["lastname"] = "lastname length must be 2–64 characters"
	} else if !isHumanName(last) {
		errs["lastname"] = "allowed characters: letters, space, '-', '''"
	}

	// birth_date (required + format + 18+)
	if bdate == "" {
		errs["birth_date"] = "birth_date is required"
	} else if dob, err := time.Parse("2006-01-02", bdate); err != nil {
		errs["birth_date"] = "must be YYYY-MM-DD"
	} else if dob.After(time.Now().UTC().AddDate(-18, 0, 0)) {
		errs["birth_date"] = "user must be 18+ years old"
	}

	// phone (required + E.164)
	if phone == "" {
		errs["phone"] = "phone is required"
	} else if !e164Re.MatchString(phone) {
		errs["phone"] = "must be in E.164 format (e.g., +33788888888)"
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func isHumanName(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' {
			continue
		}
		return false
	}
	return true
}

func ValidateLogin(r auth.LoginRequest) map[string]string {
	errs := make(map[string]string)

	// Normalize
	email := strings.ToLower(strings.TrimSpace(r.Email))
	password := r.Password // не триммим пароль, но проверим, что не пустой

	// email (required + format)
	if email == "" {
		errs["email"] = "email is required"
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs["email"] = "invalid email format"
	}

	// password (required + length)
	if strings.TrimSpace(password) == "" {
		errs["password"] = "password is required"
	} else if l := utf8.RuneCountInString(password); l < minPasswordLen || l > maxPasswordLen {
		errs["password"] = "password length must be 8–72 characters"
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
