package services

import (
	"blog-backend/models"
	"blog-backend/repositories"
	"errors"
	"fmt"
	"time"
)

type PostService struct {
	repo *repositories.PostRepository
}

func NewPostService(repo *repositories.PostRepository) *PostService {
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
	if post.Status == "" {
		post.Status = defaultPostStatus
	}
	if !isAllowedPostStatus(post.Status) {
		return fmt.Errorf("invalid post status %q", post.Status)
	}
	if post.Status == publishedPostStatus && post.PublishedAt == nil {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	return s.repo.Create(post)
}

func (s *PostService) UpdatePost(editor *models.User, updated *models.Post) error {
	if !userHasPermission(editor, "edit_post") {
		return errors.New("permission denied")
	}
	if updated == nil || updated.ID == 0 {
		return errors.New("post id is required")
	}
	if err := validatePostInput(updated); err != nil {
		return err
	}
	if updated.Status != "" && !isAllowedPostStatus(updated.Status) {
		return fmt.Errorf("invalid post status %q", updated.Status)
	}
	existing, err := s.repo.GetByID(updated.ID)
	if err != nil {
		return err
	}
	existing.Title = updated.Title
	existing.Summary = updated.Summary
	existing.Body = updated.Body
	if updated.Status != "" {
		existing.Status = updated.Status
		if updated.Status == publishedPostStatus {
			now := time.Now().UTC()
			existing.PublishedAt = &now
		} else {
			existing.PublishedAt = nil
		}
	}
	return s.repo.Update(existing)
}

func (s *PostService) PublishPost(editor *models.User, postID uint) error {
	if !userHasPermission(editor, "publish_post") {
		return errors.New("permission denied")
	}
	if postID == 0 {
		return errors.New("post id is required")
	}
	post, err := s.repo.GetByID(postID)
	if err != nil {
		return err
	}
	if post.Status == publishedPostStatus {
		return nil
	}
	now := time.Now().UTC()
	post.Status = publishedPostStatus
	post.PublishedAt = &now
	return s.repo.Update(post)
}

func (s *PostService) GetPost(id uint) (*models.Post, error) {
	if id == 0 {
		return nil, errors.New("post id is required")
	}
	return s.repo.GetByID(id)
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
