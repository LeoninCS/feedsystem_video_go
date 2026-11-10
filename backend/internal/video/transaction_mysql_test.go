package video

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"feedsystem_video_go/internal/account"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Each test owns a new schema; existing schemas are never migrated or cleared.
func mysqlTransactionDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("INTX_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set INTX_MYSQL_DSN to run real MySQL transaction tests")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	cfg.ParseTime = true
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	schema := fmt.Sprintf("intx_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = schema
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.AutoMigrate(&Video{}, &Like{}, &Comment{}, &OutboxMsg{}, &Tag{}, &VideoTag{}, &account.Account{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedTransactionVideo(t *testing.T, db *gorm.DB) *Video {
	t.Helper()
	v := &Video{AuthorID: 1, Username: "author", Title: "test", PlayURL: "/video", CoverURL: "/cover"}
	if err := db.Create(v).Error; err != nil {
		t.Fatal(err)
	}
	return v
}

func assertVideoState(t *testing.T, db *gorm.DB, id uint, likes, popularity, rows int64) {
	t.Helper()
	var v Video
	if err := db.First(&v, id).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&Like{}).Where("video_id = ?", id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if v.LikesCount != likes || v.Popularity != popularity || count != rows {
		t.Fatalf("likes_count=%d popularity=%d likes rows=%d; want %d %d %d", v.LikesCount, v.Popularity, count, likes, popularity, rows)
	}
}

func TestMySQLLikeFallbackRollback(t *testing.T) {
	for _, column := range []string{"likes_count", "popularity"} {
		t.Run(column, func(t *testing.T) {
			db := mysqlTransactionDB(t)
			v := seedTransactionVideo(t, db)
			failure := errors.New("injected counter failure")
			if err := db.Callback().Update().Before("gorm:update").Register("test:fail_counter", func(tx *gorm.DB) {
				if tx.Statement.Table == "videos" && tx.Statement.Selects == nil {
					if changes, ok := tx.Statement.Dest.(map[string]interface{}); ok {
						if _, updating := changes[column]; updating {
							tx.AddError(failure)
						}
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			s := NewLikeService(NewLikeRepository(db), NewVideoRepository(db), nil, nil, nil)
			if err := s.Like(context.Background(), &Like{VideoID: v.ID, AccountID: 2}); !errors.Is(err, failure) {
				t.Fatalf("err=%v, want injected failure", err)
			}
			assertVideoState(t, db, v.ID, 0, 0, 0)
		})
	}
}

func TestMySQLLikeFallbackCommitAndUnlike(t *testing.T) {
	db := mysqlTransactionDB(t)
	v := seedTransactionVideo(t, db)
	// The optional VideoRepository remains supported.
	s := NewLikeService(NewLikeRepository(db), nil, nil, nil, nil)
	ctx := context.Background()
	like := &Like{VideoID: v.ID, AccountID: 2}
	if err := s.Like(ctx, like); err != nil {
		t.Fatal(err)
	}
	assertVideoState(t, db, v.ID, 1, 1, 1)
	if err := s.Like(ctx, &Like{VideoID: v.ID, AccountID: 2}); err == nil || err.Error() != "user has liked this video" {
		t.Fatalf("duplicate error=%v", err)
	}
	assertVideoState(t, db, v.ID, 1, 1, 1)
	if err := s.Unlike(ctx, like); err != nil {
		t.Fatal(err)
	}
	assertVideoState(t, db, v.ID, 0, 0, 0)
}

func TestMySQLInTxNilCommitsIndependently(t *testing.T) {
	db := mysqlTransactionDB(t)
	v := seedTransactionVideo(t, db)
	ctx := context.Background()
	likes, videos, comments := NewLikeRepository(db), NewVideoRepository(db), NewCommentRepository(db)
	failure := errors.New("outer transaction aborted")
	err := likes.Transaction(ctx, func(tx *gorm.DB) error {
		// No row locks are held here. All nil calls deliberately commit outside tx.
		if created, err := likes.LikeInTx(ctx, nil, &Like{VideoID: v.ID, AccountID: 2}); err != nil || !created {
			return fmt.Errorf("create: %v", err)
		}
		if err := videos.ChangeLikesCountInTx(ctx, nil, v.ID, 2); err != nil {
			return err
		}
		if err := videos.ChangePopularityInTx(ctx, nil, v.ID, 3); err != nil {
			return err
		}
		if err := comments.CreateCommentInTx(ctx, nil, &Comment{VideoID: v.ID, AuthorID: 2, Content: "hello"}); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	assertVideoState(t, db, v.ID, 2, 3, 1)
	if created, err := likes.LikeInTx(ctx, nil, &Like{VideoID: v.ID, AccountID: 2}); err != nil || created {
		t.Fatalf("duplicate: created=%v err=%v", created, err)
	}
	if deleted, err := likes.UnlikeInTx(ctx, nil, &Like{VideoID: v.ID, AccountID: 2}); err != nil || !deleted {
		t.Fatalf("delete: deleted=%v err=%v", deleted, err)
	}
	if deleted, err := likes.UnlikeInTx(ctx, nil, &Like{VideoID: v.ID, AccountID: 2}); err != nil || deleted {
		t.Fatalf("missing: deleted=%v err=%v", deleted, err)
	}
	if err := videos.ChangeLikesCountInTx(ctx, nil, v.ID, -10); err != nil {
		t.Fatal(err)
	}
	if err := videos.ChangePopularityInTx(ctx, nil, v.ID, -10); err != nil {
		t.Fatal(err)
	}
	assertVideoState(t, db, v.ID, 0, 0, 0)
	var count int64
	if err := db.Model(&Comment{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("comments=%d err=%v", count, err)
	}
}

func TestMySQLPublishAtomicity(t *testing.T) {
	for _, failTable := range []string{"outbox_msgs", "video_tags", ""} {
		t.Run("failure="+failTable, func(t *testing.T) {
			db := mysqlTransactionDB(t)
			failure := errors.New("injected publish failure")
			if failTable != "" {
				if err := db.Callback().Create().Before("gorm:create").Register("test:fail_publish", func(tx *gorm.DB) {
					if tx.Statement.Table == failTable {
						tx.AddError(failure)
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			v := &Video{AuthorID: 1, Username: "author", Title: "test #golang", PlayURL: "/video", CoverURL: "/cover"}
			err := NewVideoService(NewVideoRepository(db), nil, nil).Publish(context.Background(), v)
			if failTable != "" && !errors.Is(err, failure) || failTable == "" && err != nil {
				t.Fatalf("publish err=%v", err)
			}
			want := int64(1)
			if failTable != "" {
				want = 0
			}
			for _, model := range []interface{}{&Video{}, &OutboxMsg{}, &Tag{}, &VideoTag{}} {
				var count int64
				if err := db.Model(model).Count(&count).Error; err != nil || count != want {
					t.Fatalf("%T rows=%d want=%d err=%v", model, count, want, err)
				}
			}
		})
	}
}

func TestMySQLCommentFallbackRollback(t *testing.T) {
	db := mysqlTransactionDB(t)
	v := seedTransactionVideo(t, db)
	failure := errors.New("injected popularity failure")
	if err := db.Callback().Update().Before("gorm:update").Register("test:fail_popularity", func(tx *gorm.DB) { tx.AddError(failure) }); err != nil {
		t.Fatal(err)
	}
	s := NewCommentService(NewCommentRepository(db), NewVideoRepository(db), nil, nil, nil)
	err := s.Publish(context.Background(), &Comment{VideoID: v.ID, AuthorID: 2, Content: "hello"})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&Comment{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("comments=%d err=%v", count, err)
	}
	assertVideoState(t, db, v.ID, 0, 0, 0)
}

func TestMySQLAccountRenameRollback(t *testing.T) {
	db := mysqlTransactionDB(t)
	a := &account.Account{Username: "original", Token: "old-token"}
	if err := db.Create(a).Error; err != nil {
		t.Fatal(err)
	}
	failure := errors.New("injected token failure")
	if err := db.Callback().Update().Before("gorm:update").Register("test:fail_token", func(tx *gorm.DB) {
		if changes, ok := tx.Statement.Dest.(map[string]interface{}); ok {
			if _, updating := changes["token"]; updating {
				tx.AddError(failure)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	// Service generates a JWT before starting the transaction.
	t.Setenv("JWT_SECRET", strings.Repeat("test-secret", 4))
	s := account.NewAccountService(account.NewAccountRepository(db), nil)
	if _, err := s.Rename(context.Background(), a.ID, "renamed"); !errors.Is(err, failure) {
		t.Fatalf("rename err=%v", err)
	}
	var actual account.Account
	if err := db.First(&actual, a.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Username != "original" || actual.Token != "old-token" {
		t.Fatalf("partially committed account: %+v", actual)
	}
}
