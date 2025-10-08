package services

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"user-manager-api/internal/application/ports"
	domain "user-manager-api/internal/domain/user"
	"user-manager-api/internal/domain/user_file"
	"user-manager-api/internal/infrastructure/mq"
	"user-manager-api/internal/interface/api/rest/dto/user"
)

type UserService struct {
	userRepository     domain.Repository
	userFileRepository user_file.Repository
	mq                 ports.RabbitMQ
	mCounter           *prometheus.CounterVec
}

func NewUserService(
	userRepository domain.Repository,
	userFileRepository user_file.Repository,
	mq ports.RabbitMQ,
	mCounter *prometheus.CounterVec,
) ports.UserService {
	return &UserService{
		userRepository:     userRepository,
		userFileRepository: userFileRepository,
		mq:                 mq,
		mCounter:           mCounter,
	}
}

func (us *UserService) FindUserByID(ctx context.Context, uuid domain.UUID) (*domain.User, error) {
	u, err := us.userRepository.FetchUserByID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (us *UserService) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := us.userRepository.FetchUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (us *UserService) FindUsers(ctx context.Context, page int) (domain.Users, error) {
	users, err := us.userRepository.FetchUsers(ctx, page)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (us *UserService) CreateUser(ctx context.Context, u domain.User) (*domain.User, error) {
	uRet, err := us.userRepository.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	if uRet != nil {
		us.mq.GetInputChan() <- mq.Event{
			Id:      uuid.New(),
			TS:      time.Now(),
			Method:  http.MethodPost,
			UserID:  uRet.UUID.String(),
			Payload: user.ToResponseUser(*uRet),
		}
	}

	us.mCounter.WithLabelValues("user_created_total").Inc()

	return uRet, nil
}

func (us *UserService) UpdateUser(ctx context.Context, u domain.User) (*domain.User, error) {
	uRet, err := us.userRepository.UpdateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	if uRet != nil {
		us.mq.GetInputChan() <- mq.Event{
			Id:      uuid.New(),
			TS:      time.Now(),
			Method:  http.MethodPut,
			UserID:  uRet.UUID.String(),
			Payload: user.ToResponseUser(*uRet),
		}
	}

	us.mCounter.WithLabelValues("user_updated_total").Inc()

	return uRet, nil
}

func (us *UserService) DeleteUser(ctx context.Context, userUUID domain.UUID) error {
	id, err := us.userRepository.FetchInternalID(ctx, userUUID)
	if err != nil {
		return err
	}

	// todo: should be run in transaction

	// example: delete objs from s3
	// ufs.s3.DeleteObjects(ufs.userFileRepository.FetchUserFiles(...))

	if err = us.userFileRepository.DeleteUserFiles(ctx, id); err != nil {
		return err
	}
	u, err := us.userRepository.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	if u != nil {
		us.mq.GetInputChan() <- mq.Event{
			Id:      uuid.New(),
			TS:      time.Now(),
			Method:  http.MethodDelete,
			UserID:  u.UUID.String(),
			Payload: user.ToResponseUser(*u),
		}
	}

	us.mCounter.WithLabelValues("user_deleted_total").Inc()

	return nil
}
