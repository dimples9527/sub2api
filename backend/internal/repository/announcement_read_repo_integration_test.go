//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/announcementread"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

type AnnouncementReadRepoSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	repo   *announcementReadRepository
}

func (s *AnnouncementReadRepoSuite) SetupTest() {
	s.ctx = context.Background()
	tx := testEntTx(s.T())
	s.client = tx.Client()
	s.repo = NewAnnouncementReadRepository(s.client).(*announcementReadRepository)
}

func TestAnnouncementReadRepoSuite(t *testing.T) {
	suite.Run(t, new(AnnouncementReadRepoSuite))
}

func (s *AnnouncementReadRepoSuite) TestMarkRead_IsIdempotent() {
	user := mustCreateUser(s.T(), s.client, &service.User{
		Email:    "announcement-read-" + time.Now().Format("150405.000000000") + "@example.com",
		Username: "announcement-read-user",
		Notes:    "",
	})
	ann, err := s.client.Announcement.Create().
		SetTitle("read test").
		SetContent("content").
		SetStatus(service.AnnouncementStatusActive).
		SetNotifyMode(service.AnnouncementNotifyModeSilent).
		Save(s.ctx)
	s.Require().NoError(err)

	firstReadAt := time.Now().Add(-time.Minute)
	s.Require().NoError(s.repo.MarkRead(s.ctx, ann.ID, user.ID, firstReadAt))

	secondReadAt := time.Now()
	s.Require().NoError(s.repo.MarkRead(s.ctx, ann.ID, user.ID, secondReadAt))

	row, err := s.client.AnnouncementRead.Query().
		Where(
			announcementread.AnnouncementIDEQ(ann.ID),
			announcementread.UserIDEQ(user.ID),
		).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(firstReadAt.UTC().Truncate(time.Microsecond), row.ReadAt.UTC().Truncate(time.Microsecond))
}
