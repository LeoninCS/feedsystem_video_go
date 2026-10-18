package account

import (
	"context"
	"errors"
	"testing"
	"time"

	rediscache "feedsystem_video_go/internal/middleware/redis"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) (*AccountService, sqlmock.Sqlmock, *miniredis.Miniredis) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	gdb, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Discard, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}

	mr := miniredis.RunT(t)
	cache := rediscache.NewClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), "v1:")
	t.Cleanup(func() { cache.Close() })

	return NewAccountService(NewAccountRepository(gdb), cache), mock, mr
}

const selectByRefreshTokenPattern = `SELECT .?id.?,.?username.? FROM .accounts. WHERE refresh_token = `
const selectByIDAndRefreshTokenPattern = `SELECT .?id.?,.?username.? FROM .accounts. WHERE id = . AND refresh_token = `
const updateTokenPattern = `UPDATE .accounts. SET .token.`

func TestRefreshAccessTokenCacheMissUsesIndexedQueryAndWritesBack(t *testing.T) {
	svc, mock, mr := newTestService(t)
	ctx := context.Background()
	token := "missed-refresh-token"

	mock.ExpectQuery(selectByRefreshTokenPattern).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(7, "alice"))
	mock.ExpectExec(updateTokenPattern).WillReturnResult(sqlmock.NewResult(0, 1))

	newToken, id, username, err := svc.RefreshAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if id != 7 || username != "alice" {
		t.Fatalf("expected id 7 alice, got id %d username %q", id, username)
	}
	if newToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if got, _ := mr.Get("v1:refresh:" + token); got != "7" {
		t.Fatalf("expected refresh lookup write-back to id 7, got %q", got)
	}
	ttl := mr.TTL("v1:refresh:" + token)
	if ttl < refreshTokenTTL-time.Minute || ttl > refreshTokenTTL {
		t.Fatalf("expected write-back ttl aligned to %s, got %s", refreshTokenTTL, ttl)
	}
	if got, _ := mr.Get("v1:account:7:refresh"); got != token {
		t.Fatalf("expected account refresh cache %q, got %q", token, got)
	}
	if got, _ := mr.Get("v1:account:7"); got != newToken {
		t.Fatalf("expected account cache to hold new token, got %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRefreshAccessTokenCacheHitVerifiesAgainstDB(t *testing.T) {
	svc, mock, mr := newTestService(t)
	ctx := context.Background()
	token := "cached-refresh-token"

	mr.Set("v1:refresh:"+token, "7")

	mock.ExpectQuery(selectByIDAndRefreshTokenPattern).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(7, "alice"))
	mock.ExpectExec(updateTokenPattern).WillReturnResult(sqlmock.NewResult(0, 1))

	newToken, id, username, err := svc.RefreshAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if id != 7 || username != "alice" {
		t.Fatalf("expected id 7 alice, got id %d username %q", id, username)
	}
	if got, _ := mr.Get("v1:account:7"); got != newToken {
		t.Fatalf("expected account cache to hold new token, got %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRefreshAccessTokenRejectsRotatedToken(t *testing.T) {
	svc, mock, mr := newTestService(t)
	ctx := context.Background()
	oldToken, currentToken := "rotated-out-token", "current-refresh-token"

	// Stale cache entry left behind by rotation: cache still maps the old
	// token to the account, but the database already holds the new token.
	mr.Set("v1:refresh:"+oldToken, "7")

	mock.ExpectQuery(selectByIDAndRefreshTokenPattern).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}))
	mock.ExpectQuery(selectByRefreshTokenPattern).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}))

	_, _, _, err := svc.RefreshAccessToken(ctx, oldToken)
	if err == nil || err.Error() != "invalid refresh token" {
		t.Fatalf("expected invalid refresh token, got %v", err)
	}

	mock.ExpectQuery(selectByRefreshTokenPattern).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(7, "alice"))
	mock.ExpectExec(updateTokenPattern).WillReturnResult(sqlmock.NewResult(0, 1))

	newToken, id, _, err := svc.RefreshAccessToken(ctx, currentToken)
	if err != nil {
		t.Fatalf("refresh with current token: %v", err)
	}
	if id != 7 || newToken == "" {
		t.Fatalf("expected id 7 with new access token, got id %d token %q", id, newToken)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRefreshTokenEmptyShortCircuitsBeforeSQL(t *testing.T) {
	svc, mock, _ := newTestService(t)
	ctx := context.Background()

	_, _, _, err := svc.RefreshAccessToken(ctx, "")
	if err == nil || err.Error() != "refresh token is empty" {
		t.Fatalf("expected refresh token is empty, got %v", err)
	}

	repo := svc.accountRepository
	if _, err := repo.FindByRefreshToken(ctx, ""); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound without SQL, got %v", err)
	}
	if _, err := svc.accountRepository.FindByIDAndRefreshToken(ctx, 0, "some-token"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound without SQL, got %v", err)
	}
	if _, err := svc.accountRepository.FindByIDAndRefreshToken(ctx, 7, ""); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound without SQL, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL executed: %v", err)
	}
}

func TestLoginReplacesStaleRefreshKey(t *testing.T) {
	svc, mock, mr := newTestService(t)
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	mr.Set("v1:refresh:stale-token", "7")

	mock.ExpectQuery(`FROM .accounts. WHERE username = `).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password", "token", "refresh_token", "avatar_url", "bio"}).
			AddRow(7, "alice", string(hash), "old-access", "stale-token", "", ""))
	mock.ExpectExec(`UPDATE .accounts. SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	_, refresh, err := svc.Login(ctx, "alice", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if refresh == "" || refresh == "stale-token" {
		t.Fatalf("expected a fresh refresh token, got %q", refresh)
	}
	if mr.Exists("v1:refresh:stale-token") {
		t.Fatal("expected stale refresh lookup key to be deleted after rotation")
	}
	if got, _ := mr.Get("v1:refresh:" + refresh); got != "7" {
		t.Fatalf("expected new refresh lookup to map to id 7, got %q", got)
	}
	if got, _ := mr.Get("v1:account:7:refresh"); got != refresh {
		t.Fatalf("expected account refresh cache to hold new token, got %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
