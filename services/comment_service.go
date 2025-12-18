package services

import (
	"blog-backend/models"
	"blog-backend/repositories"
	"errors"
)

// CommentService encapsulates business logic for post comments.
type CommentService struct {
	repo     *repositories.CommentRepository
	postRepo *repositories.PostRepository
}

func NewCommentService(repo *repositories.CommentRepository, postRepo *repositories.PostRepository) *CommentService {
	return &CommentService{repo: repo, postRepo: postRepo}
}

func (s *CommentService) AddComment(author *models.User, postID uint, body, ip, userAgent, country string) (*models.Comment, error) {
	if author == nil || author.ID == 0 {
		return nil, errors.New("author is required")
	}
	if postID == 0 {
		return nil, errors.New("post id is required")
	}
	if body == "" {
		return nil, errors.New("comment body is required")
	}
	if _, err := s.postRepo.GetByID(postID); err != nil {
		return nil, err
	}
	comment := &models.Comment{
		PostID:    postID,
		UserID:    author.ID,
		Body:      body,
		Visible:   true,
		IPAddress: ip,
		UserAgent: userAgent,
		Country:   country,
	}
	if err := s.repo.Create(comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *CommentService) ListComments(postID uint, offset, limit int) ([]models.Comment, int64, error) {
	if postID == 0 {
		return nil, 0, errors.New("post id is required")
	}
	return s.repo.ListByPost(postID, offset, limit)
}

func (s *CommentService) DeleteComment(requestor *models.User, commentID uint) error {
	if requestor == nil || requestor.ID == 0 {
		return errors.New("requestor is required")
	}
	if commentID == 0 {
		return errors.New("comment id is required")
	}
	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		return err
	}
	if comment.UserID != requestor.ID && !userHasPermission(requestor, "delete_post") {
		return errors.New("permission denied")
	}
	return s.repo.Delete(commentID)
}

func (s *CommentService) SetVisibility(operator *models.User, commentID uint, visible bool) error {
	if !userHasPermission(operator, "moderate_comments") {
		return errors.New("permission denied")
	}
	if commentID == 0 {
		return errors.New("comment id is required")
	}
	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		return err
	}
	comment.Visible = visible
	return s.repo.Update(comment)
}
