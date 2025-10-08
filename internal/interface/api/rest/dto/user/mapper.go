package user

import (
	"errors"
	"time"

	"user-manager-api/internal/domain/user"
)

func ToResponseUser(uDomain user.User) User {
	var u = User{
		UUID:      uDomain.UUID,
		Email:     uDomain.Email,
		Name:      uDomain.Name,
		Lastname:  uDomain.Lastname,
		BirthDate: uDomain.BirthDate,
		Phone:     uDomain.Phone,
	}

	return u
}

func ToResponseUsers(usDomain user.Users) Users {
	us := make(Users, len(usDomain))
	for idx, u := range usDomain {
		us[idx] = ToResponseUser(*u)
	}

	return us
}

func ToDomainUser(uRequest Request) (user.User, error) {
	d, err := time.Parse("2006-01-02", uRequest.BirthDate)
	if err != nil {
		return user.User{}, errors.New("invalid birth_date format, want YYYY-MM-DD")
	}

	var u = user.User{
		Email:     uRequest.Email,
		Name:      uRequest.Name,
		Lastname:  uRequest.Lastname,
		BirthDate: d,
		Phone:     uRequest.Phone,
	}

	return u, nil
}
