package account

import (
	"errors"
	"feedsystem_video_go/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

type AccountService struct {
	accountRepository *AccountRepository
}

func NewAccountService(accountRepository *AccountRepository) *AccountService {
	return &AccountService{accountRepository: accountRepository}
}

func (as *AccountService) CreateAccount(account *Account) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	account.Password = string(passwordHash)
	if err := as.accountRepository.CreateAccount(account); err != nil {
		return err
	}
	return nil
}

func (as *AccountService) RenameByID(id uint, newUsername string) error {
	if err := as.accountRepository.RenameByID(id, newUsername); err != nil {
		return err
	}
	return nil
}

func (as *AccountService) ChangePassword(username, oldPassword, newPassword string) error {
	account, err := as.FindByUsername(username)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(oldPassword)); err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := as.accountRepository.ChangePassword(account.ID, string(passwordHash)); err != nil {
		return err
	}
	return nil
}

func (as *AccountService) FindByID(id uint) (*Account, error) {
	if account, err := as.accountRepository.FindByID(id); err != nil {
		return nil, err
	} else {
		return account, nil
	}
}

func (as *AccountService) FindByUsername(username string) (*Account, error) {
	if account, err := as.accountRepository.FindByUsername(username); err != nil {
		return nil, err
	} else {
		return account, nil
	}
}

func (as *AccountService) Login(username, password string) (string, error) {
	account, err := as.FindByUsername(username)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		return "", err
	}
	// generate token
	token, err := auth.GenerateToken(account.ID, account.Username)
	if err != nil {
		return "", err
	}
	if err := as.accountRepository.Login(account.ID, token); err != nil {
		return "", err
	}

	return token, nil
}

func (as *AccountService) Logout(accountID uint) error {
	account, err := as.FindByID(accountID)
	if err != nil {
		return err
	}
	if account.Token == "" {
		return errors.New("account already logged out")
	}
	return as.accountRepository.Logout(account.ID, account.Token)
}
