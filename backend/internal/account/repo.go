package account

import (
	"context"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (ar *AccountRepository) CreateAccount(ctx context.Context, account *Account) error {
	if err := ar.db.WithContext(ctx).Create(account).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) Rename(ctx context.Context, id uint, newUsername string) error {
	result := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("username", newUsername)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (ar *AccountRepository) RenameWithToken(ctx context.Context, id uint, newUsername string, token string) error {
	return ar.Transaction(ctx, func(tx *gorm.DB) error {
		if err := ar.RenameInTx(ctx, tx, id, newUsername); err != nil {
			return err
		}
		return ar.UpdateTokenInTx(ctx, tx, id, token)
	})
}

func (ar *AccountRepository) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	if err := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("password", newPassword).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	if err := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": token, "refresh_token": refreshToken}).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) Logout(ctx context.Context, id uint) error {
	if err := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": "", "refresh_token": ""}).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", accountID).Update("avatar_url", avatarURL).Error
}

func (ar *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("token", token).Error
}

func (ar *AccountRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(updates).Error
}

func (ar *AccountRepository) FindAll(ctx context.Context) ([]*Account, error) {
	var accounts []*Account
	if err := ar.db.WithContext(ctx).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// Transaction commits on nil and rolls back on an error or panic.
// All repository operations in fn must receive the supplied tx.
func (ar *AccountRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return ar.db.WithContext(ctx).Transaction(fn)
}

// RenameInTx changes the username or returns gorm.ErrRecordNotFound.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (ar *AccountRepository) RenameInTx(ctx context.Context, tx *gorm.DB, id uint, newUsername string) error {
	if tx == nil {
		tx = ar.db
	}
	result := tx.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("username", newUsername)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateTokenInTx replaces an account token.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (ar *AccountRepository) UpdateTokenInTx(ctx context.Context, tx *gorm.DB, id uint, token string) error {
	if tx == nil {
		tx = ar.db
	}
	return tx.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("token", token).Error
}
