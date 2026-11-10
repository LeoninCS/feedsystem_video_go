package video

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func mockTransactionDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		conn.Close()
	})
	return db, mock
}

func TestRepositoryTransaction(t *testing.T) {
	for _, repo := range []string{"like", "video", "comment"} {
		for _, outcome := range []string{"commit", "error", "panic"} {
			t.Run(repo+"/"+outcome, func(t *testing.T) {
				db, mock := mockTransactionDB(t)
				transaction := NewLikeRepository(db).Transaction
				if repo == "video" {
					transaction = NewVideoRepository(db).Transaction
				} else if repo == "comment" {
					transaction = NewCommentRepository(db).Transaction
				}
				mock.ExpectBegin()
				if outcome == "commit" {
					mock.ExpectCommit()
				} else {
					mock.ExpectRollback()
				}
				failure := errors.New("failed operation")
				type contextKey struct{}
				ctx := context.WithValue(context.Background(), contextKey{}, "request")
				defer func() {
					p := recover()
					if outcome == "panic" && p != failure {
						t.Errorf("panic = %v, want original panic", p)
					} else if outcome != "panic" && p != nil {
						t.Errorf("unexpected panic: %v", p)
					}
				}()
				err := transaction(ctx, func(tx *gorm.DB) error {
					if tx.Statement.Context.Value(contextKey{}) != "request" {
						t.Error("request context lost")
					}
					if outcome == "panic" {
						panic(failure)
					}
					if outcome == "error" {
						return failure
					}
					return nil
				})
				if outcome == "panic" {
					t.Fatal("panic was swallowed")
				}
				if outcome == "error" && !errors.Is(err, failure) || outcome == "commit" && err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	}
}

func TestLikeInTxDuplicateAndFailure(t *testing.T) {
	for _, duplicate := range []bool{true, false} {
		t.Run(fmt.Sprint("duplicate=", duplicate), func(t *testing.T) {
			db, mock := mockTransactionDB(t)
			failure := errors.New("database unavailable")
			var insertErr error = failure
			if duplicate {
				insertErr = fmt.Errorf("wrapped: %w", &mysqldriver.MySQLError{Number: 1062})
			}
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO `likes`").WillReturnError(insertErr)
			mock.ExpectRollback()
			created, err := NewLikeRepository(db).LikeInTx(context.Background(), nil, &Like{VideoID: 1, AccountID: 2})
			if created || duplicate && err != nil || !duplicate && !errors.Is(err, failure) {
				t.Fatalf("created=%v, err=%v", created, err)
			}
		})
	}
}

func TestInTxUsesSuppliedTransaction(t *testing.T) {
	db, mock := mockTransactionDB(t)
	ctx := context.Background()
	// Nil repository connections make accidental use of the fallback fail.
	likes := NewLikeRepository(nil)
	videos := NewVideoRepository(nil)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `likes`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE `videos` SET `likes_count`").WithArgs(int64(1), uint(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `videos` SET `popularity`").WithArgs(int64(1), uint(7)).WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()
	err := NewLikeRepository(db).Transaction(ctx, func(tx *gorm.DB) error {
		if created, err := likes.LikeInTx(ctx, tx, &Like{VideoID: 7, AccountID: 8}); err != nil || !created {
			t.Fatalf("created=%v, err=%v", created, err)
		}
		if err := videos.ChangeLikesCountInTx(ctx, tx, 7, 1); err != nil {
			return err
		}
		return videos.ChangePopularityInTx(ctx, tx, 7, 1)
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
}

func TestUnlikeInTxRowsAffected(t *testing.T) {
	for _, rows := range []int64{0, 1} {
		t.Run(fmt.Sprint(rows), func(t *testing.T) {
			db, mock := mockTransactionDB(t)
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM `likes`").WithArgs(uint(1), uint(2)).WillReturnResult(sqlmock.NewResult(0, rows))
			mock.ExpectCommit()
			deleted, err := NewLikeRepository(db).UnlikeInTx(context.Background(), nil, &Like{VideoID: 1, AccountID: 2})
			if err != nil || deleted != (rows > 0) {
				t.Fatalf("deleted=%v, err=%v", deleted, err)
			}
		})
	}
}
