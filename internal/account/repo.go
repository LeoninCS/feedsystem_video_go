package account

import (
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (ar *AccountRepository) CreateAccount(account *Account) error {
	if err := ar.db.Create(account).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) RenameByID(id uint, newUsername string) error {
	if err := ar.db.Model(&Account{}).Where("id = ?", id).Update("username", newUsername).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) ChangePassword(id uint, newPassword string) error {
	if err := ar.db.Model(&Account{}).Where("id = ?", id).Update("password", newPassword).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) FindByID(id uint) (*Account, error) {
	var account Account
	if err := ar.db.First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) FindByUsername(username string) (*Account, error) {
	var account Account
	if err := ar.db.Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) Login(id uint, token string) error {
	if err := ar.db.Model(&Account{}).Where("id = ?", id).Update("token", token).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) Logout(id uint, token string) error {
	if err := ar.db.Model(&Account{}).Where("id = ?", id).Update("token", "").Error; err != nil {
		return err
	}
	return nil
}
