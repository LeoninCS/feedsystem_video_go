package video

import (
	"context"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) CreateComment(ctx context.Context, comment *Comment) error {
	return r.CreateCommentInTx(ctx, nil, comment)
}

func (r *CommentRepository) DeleteComment(ctx context.Context, comment *Comment) error {
	return r.db.WithContext(ctx).Delete(comment).Error
}

func (r *CommentRepository) GetAllComments(ctx context.Context, videoID uint) ([]Comment, error) {
	var comments []Comment
	err := r.db.WithContext(ctx).
		Where("video_id = ?", videoID).
		Order("created_at asc").
		Limit(200).
		Find(&comments).Error
	return comments, err
}

func (r *CommentRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var comment Comment
	if err := r.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id uint) (*Comment, error) {
	var comment Comment
	if err := r.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

// CreateCommentInTx creates a comment.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (r *CommentRepository) CreateCommentInTx(ctx context.Context, tx *gorm.DB, comment *Comment) error {
	return transactionDB(ctx, tx, r.db).Create(comment).Error
}

// FindAccountIDByUsername looks up the recipient of a mention.
func (r *CommentRepository) FindAccountIDByUsername(ctx context.Context, username string) (uint, error) {
	var id uint
	err := r.db.WithContext(ctx).Table("accounts").Where("username = ?", username).Select("id").Scan(&id).Error
	return id, err
}

// CreateMentionNotification stores a mention after the comment has committed.
func (r *CommentRepository) CreateMentionNotification(ctx context.Context, recipientID, senderID, videoID uint, content string) error {
	notification := struct {
		RecipientID uint
		SenderID    uint
		Type        string
		TargetID    uint
		Content     string
	}{recipientID, senderID, "mention", videoID, content}
	return r.db.WithContext(ctx).Table("notifications").Create(&notification).Error
}
