package services

import (
	"blog-backend/models"
	"errors"
	"fmt"
	"time"
)

type PostService struct {
	repo PostRepository
}

func NewPostService(repo PostRepository) *PostService {
	return &PostService{repo: repo}
}

func (s *PostService) CreatePost(author *models.User, post *models.Post) error {
	if !userHasPermission(author, "create_post") {
		return errors.New("permission denied")
	}
	if post == nil {
		return errors.New("post is required")
	}
	if err := validatePostInput(post); err != nil {
		return err
	}
	post.UserID = author.ID
	if post.ClientRequestID != nil {
		if len(*post.ClientRequestID) > 100 || *post.ClientRequestID == "" {
			return errors.New("invalid client request id")
		}
		if repo, ok := s.repo.(createRequestRepository); ok {
			if existing, err := repo.GetByCreateRequest(author.ID, *post.ClientRequestID); err == nil {
				*post = *existing
				return nil
			}
		}
	}
	post.PublicationTimeKnown = true
	post.DraftVersion = 1
	nowDraft := time.Now().UTC().Truncate(time.Microsecond)
	post.DraftUpdatedAt = &nowDraft
	draft := postSnapshot(post)
	post.Draft = &draft
	if post.Status == "" {
		post.Status = defaultPostStatus
	}
	if !isAllowedPostStatus(post.Status) {
		return fmt.Errorf("invalid post status %q", post.Status)
	}
	if post.Status == publishedPostStatus && !userHasPermission(author, "publish_post") {
		return errors.New("permission denied")
	}
	if post.Status == publishedPostStatus && post.PublishedAt == nil {
		now := time.Now().UTC().Truncate(time.Microsecond)
		post.PublishedAt = &now
	}
	err := s.repo.Create(post)
	if err != nil && post.ClientRequestID != nil {
		if repo, ok := s.repo.(createRequestRepository); ok {
			if existing, lookupErr := repo.GetByCreateRequest(author.ID, *post.ClientRequestID); lookupErr == nil {
				*post = *existing
				return nil
			}
		}
	}
	return err
}

// UpdatePost remains a compatibility endpoint for older authoring clients.
// Its payload is always an editing draft, including status:"draft" from old UIs.
func (s *PostService) UpdatePost(editor *models.User, updated *models.Post) error {
	if updated == nil || updated.ID == 0 {
		return errors.New("post id is required")
	}
	if updated.Status != "" && !isAllowedPostStatus(updated.Status) {
		return fmt.Errorf("invalid post status %q", updated.Status)
	}
	post, err := s.GetPost(updated.ID)
	if err != nil {
		return err
	}
	_, err = s.SaveDraft(editor, updated.ID, postSnapshot(updated), post.DraftVersion)
	return err
}

// PublishPost supports legacy callers without a version. New clients use
// PublishDraft to reject concurrent changes explicitly.
func (s *PostService) PublishPost(editor *models.User, postID uint) error {
	if !userHasPermission(editor, "publish_post") {
		return errors.New("permission denied")
	}
	post, err := s.GetPost(postID)
	if err != nil {
		return err
	}
	if post.Status == "published" && post.Draft == nil {
		return nil
	}
	_, err = s.PublishDraft(editor, postID, post.DraftVersion)
	return err
}

func (s *PostService) GetPost(id uint) (*models.Post, error) {
	if id == 0 {
		return nil, errors.New("post id is required")
	}
	return s.repo.GetByID(id)
}

func (s *PostService) GetPostForUser(user *models.User, id uint) (*models.Post, error) {
	if id == 0 {
		return nil, errors.New("post id is required")
	}

	post, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// If published, everyone can see
	if post.Status == publishedPostStatus {
		return post, nil
	}

	// If not published, check permissions
	if user != nil {
		// Admin/Editor or Author
		if userHasPermission(user, "edit_post") || post.UserID == user.ID {
			return post, nil
		}
	}

	// Post exists but user is not allowed to see it
	return nil, errors.New("record not found")
}

func (s *PostService) GetPublishedPost(id uint) (*models.Post, error) {
	if id == 0 {
		return nil, errors.New("post id is required")
	}
	return s.repo.GetByIDAndStatus(id, publishedPostStatus)
}

func (s *PostService) ListPublishedPosts(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return s.repo.ListByStatus(publishedPostStatus, offset, limit, sortBy, desc)
}

func (s *PostService) ListAllPosts(operator *models.User, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if !userHasPermission(operator, "edit_post") && !userHasPermission(operator, "publish_post") {
		return nil, 0, errors.New("permission denied")
	}
	return s.repo.List(offset, limit, sortBy, desc)
}

func (s *PostService) DeletePost(operator *models.User, id uint) error {
	if !userHasPermission(operator, "delete_post") {
		return errors.New("permission denied")
	}
	if id == 0 {
		return errors.New("post id is required")
	}
	return s.repo.Delete(id)
}

const (
	defaultPostStatus   = "draft"
	publishedPostStatus = "published"
)

var allowedPostStatuses = map[string]struct{}{
	"draft":     {},
	"published": {},
	"archived":  {},
}

func isAllowedPostStatus(status string) bool {
	if status == "" {
		return false
	}
	_, ok := allowedPostStatuses[status]
	return ok
}

func validatePostInput(post *models.Post) error {
	if post == nil {
		return errors.New("post is required")
	}
	return nil
}
