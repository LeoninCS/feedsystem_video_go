package account

import "golang.org/x/crypto/bcrypt"

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

func (us *UserService) ChangePassword(id uint, newPassword string) error {
	if err := us.userRepository.ChangePassword(id, newPassword); err != nil {
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
