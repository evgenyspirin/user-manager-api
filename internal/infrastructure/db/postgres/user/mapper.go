package user

import (
	domain "user-manager-api/internal/domain/user"
)

func fromDBModel(model *User) *domain.User {
	var u = &domain.User{
		UUID:         model.UUID,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		Role:         model.Role,
		Name:         model.Name,
		Lastname:     model.Lastname,
		BirthDate:    model.BirthDate,
		Phone:        model.Phone,

		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,

		DeletedAt:     model.DeletedAt,
		DeletedReason: model.DeletedReason,
		DeletedBy:     (*domain.ID)(model.DeletedBy),
	}

	return u
}

func fromDBModels(models *Users) domain.Users {
	us := make(domain.Users, len(*models))
	for idx, u := range *models {
		us[idx] = fromDBModel(u)
	}

	return us
}
