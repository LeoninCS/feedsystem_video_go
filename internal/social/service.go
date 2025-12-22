package social

import (
	"context"
	"errors"
	"feedsystem_video_go/internal/account"
)

type SocialService struct {
	repo        *SocialRepository
	accountrepo *account.AccountRepository
}

func NewSocialService(repo *SocialRepository, accountrepo *account.AccountRepository) *SocialService {
	return &SocialService{repo: repo, accountrepo: accountrepo}
}

func (s *SocialService) Follow(ctx context.Context, social *Social) error {
	_, err := s.accountrepo.FindByID(ctx, social.FollowerID)
	if err != nil {
		return err
	}
	_, err = s.accountrepo.FindByID(ctx, social.VloggerID)
	if err != nil {
		return err
	}
	if social.FollowerID == social.VloggerID {
		return errors.New("can not follow self")
	}
	isFollowed, err := s.repo.IsFollowed(ctx, social)
	if err != nil {
		return err
	}
	if isFollowed {
		return errors.New("already followed")
	}
	return s.repo.Follow(ctx, social)
}

func (s *SocialService) Unfollow(ctx context.Context, social *Social) error {
	_, err := s.accountrepo.FindByID(ctx, social.FollowerID)
	if err != nil {
		return err
	}
	_, err = s.accountrepo.FindByID(ctx, social.VloggerID)
	if err != nil {
		return err
	}
	isFollowed, err := s.repo.IsFollowed(ctx, social)
	if err != nil {
		return err
	}
	if !isFollowed {
		return errors.New("not followed")
	}
	return s.repo.Unfollow(ctx, social)
}

func (s *SocialService) GetAllFollowers(ctx context.Context, VloggerID uint) ([]*account.Account, error) {
	_, err := s.accountrepo.FindByID(ctx, VloggerID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAllFollowers(ctx, VloggerID)
}

func (s *SocialService) GetAllVloggers(ctx context.Context, FollowerID uint) ([]*account.Account, error) {
	_, err := s.accountrepo.FindByID(ctx, FollowerID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAllVloggers(ctx, FollowerID)
}

func (s *SocialService) IsFollowed(ctx context.Context, social *Social) (bool, error) {
	_, err := s.accountrepo.FindByID(ctx, social.FollowerID)
	if err != nil {
		return false, err
	}
	_, err = s.accountrepo.FindByID(ctx, social.VloggerID)
	if err != nil {
		return false, err
	}
	return s.repo.IsFollowed(ctx, social)
}
