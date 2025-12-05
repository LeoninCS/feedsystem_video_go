package account

import (
	"errors"
	"feedsystem_video_go/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepository *UserRepository
}

func NewUserService(userRepository *UserRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

func (us *UserService) CreateUser(user *User) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(passwordHash)
	if err := us.userRepository.CreateUser(user); err != nil {
		return err
	}
	return nil
}

func (us *UserService) RenameByID(id uint, newUsername string) error {
	if err := us.userRepository.RenameByID(id, newUsername); err != nil {
		return err
	}
	return nil
}

func (us *UserService) ChangePassword(username, oldPassword, newPassword string) error {
	user, err := us.FindByUsername(username)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := us.userRepository.ChangePassword(user.ID, string(passwordHash)); err != nil {
		return err
	}
	return nil
}

func (us *UserService) FindByID(id uint) (*User, error) {
	if user, err := us.userRepository.FindByID(id); err != nil {
		return nil, err
	} else {
		return user, nil
	}
}

func (us *UserService) FindByUsername(username string) (*User, error) {
	if user, err := us.userRepository.FindByUsername(username); err != nil {
		return nil, err
	} else {
		return user, nil
	}
}

func (us *UserService) Login(username, password string) (string, error) {
	user, err := us.FindByUsername(username)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", err
	}
	// generate token
	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", err
	}
	if err := us.userRepository.Login(user.ID, token); err != nil {
		return "", err
	}

	return token, nil
}

func (us *UserService) Logout(userID uint) error {
	user, err := us.FindByID(userID)
	if err != nil {
		return err
	}
	if user.Token == "" {
		return errors.New("user already logged out")
	}
	return us.userRepository.Logout(user.ID, user.Token)
}
