package services

import (
	"blog-backend/models"
	"errors"
	"fmt"
	"time"
)

var ErrDraftConflict = errors.New("draft_conflict")

type DraftResponse struct {
	PostID                uint       `json:"postId"`
	Title                 string     `json:"title"`
	Summary               string     `json:"summary"`
	Body                  string     `json:"body"`
	Version               uint64     `json:"version"`
	UpdatedAt             *time.Time `json:"updatedAt"`
	PublishedAt           *time.Time `json:"publishedAt"`
	PublishedUpdatedAt    *time.Time `json:"publishedUpdatedAt"`
	Status                string     `json:"status"`
	PublicationTimeKnown  bool       `json:"publicationTimeKnown"`
	HasUnpublishedChanges bool       `json:"hasUnpublishedChanges"`
}

type versionedPostRepository interface {
	UpdateVersioned(*models.Post, uint64) error
}
type createRequestRepository interface {
	GetByCreateRequest(uint, string) (*models.Post, error)
}

func postSnapshot(post *models.Post) models.PostDraft {
	return models.PostDraft{Title: post.Title, Summary: post.Summary, Body: post.Body}
}
func currentDraft(post *models.Post) models.PostDraft {
	if post.Draft != nil {
		return *post.Draft
	}
	return postSnapshot(post)
}
func draftResponse(post *models.Post) *DraftResponse {
	draft := currentDraft(post)
	updated := post.DraftUpdatedAt
	if updated == nil && !post.UpdatedAt.IsZero() {
		value := post.UpdatedAt
		updated = &value
	}
	return &DraftResponse{PostID: post.ID, Title: draft.Title, Summary: draft.Summary, Body: draft.Body, Version: post.DraftVersion, UpdatedAt: updated, PublishedAt: post.PublishedAt, PublishedUpdatedAt: post.PublishedUpdatedAt, Status: post.Status, PublicationTimeKnown: post.PublicationTimeKnown, HasUnpublishedChanges: post.Status != "published" || draft != postSnapshot(post)}
}

func (s *PostService) persistRevision(post *models.Post, expected uint64) error {
	post.DraftVersion = expected + 1
	post.UpdatedAt = time.Now().UTC()
	if repo, ok := s.repo.(versionedPostRepository); ok {
		return repo.UpdateVersioned(post, expected)
	}
	// Legacy test repositories implement Update; production uses the atomic CAS path.
	return s.repo.Update(post)
}
func (s *PostService) GetDraft(user *models.User, id uint) (*DraftResponse, error) {
	if !userHasPermission(user, "edit_post") {
		return nil, errors.New("permission denied")
	}
	post, err := s.GetPost(id)
	if err != nil {
		return nil, err
	}
	return draftResponse(post), nil
}
func (s *PostService) SaveDraft(user *models.User, id uint, draft models.PostDraft, version uint64) (*DraftResponse, error) {
	if !userHasPermission(user, "edit_post") {
		return nil, errors.New("permission denied")
	}
	post, err := s.GetPost(id)
	if err != nil {
		return nil, err
	}
	if post.DraftVersion != version {
		return nil, ErrDraftConflict
	}
	post.Draft = &draft
	now := time.Now().UTC()
	post.DraftUpdatedAt = &now
	// Never change public content or publication state while saving a draft.
	if err = s.persistRevision(post, version); err != nil {
		return nil, err
	}
	return draftResponse(post), nil
}
func (s *PostService) PublishDraft(user *models.User, id uint, version uint64) (*DraftResponse, error) {
	if !userHasPermission(user, "publish_post") {
		return nil, errors.New("permission denied")
	}
	post, err := s.GetPost(id)
	if err != nil {
		return nil, err
	}
	if post.DraftVersion != version {
		return nil, ErrDraftConflict
	}
	draft := currentDraft(post)
	if draft.Title == "" || draft.Body == "" {
		return nil, fmt.Errorf("title and body are required to publish")
	}
	changed := draft != postSnapshot(post)
	now := time.Now().UTC()
	if post.PublishedAt == nil && post.PublicationTimeKnown {
		post.PublishedAt = &now
	} else if changed || post.Status != "published" {
		post.PublishedUpdatedAt = &now
	}
	// A previously published legacy row with missing first publication remains
	// unknown. Never manufacture it from CreatedAt or the time of this edit.
	post.Title = draft.Title
	post.Summary = draft.Summary
	post.Body = draft.Body
	post.Status = "published"
	post.Draft = nil
	if err = s.persistRevision(post, version); err != nil {
		return nil, err
	}
	return draftResponse(post), nil
}
func (s *PostService) DiscardDraft(user *models.User, id uint, version uint64) (*DraftResponse, error) {
	if !userHasPermission(user, "edit_post") {
		return nil, errors.New("permission denied")
	}
	post, err := s.GetPost(id)
	if err != nil {
		return nil, err
	}
	if post.DraftVersion != version {
		return nil, ErrDraftConflict
	}
	draft := postSnapshot(post)
	if post.Status != "published" {
		draft = models.PostDraft{}
	}
	post.Draft = &draft
	now := time.Now().UTC()
	post.DraftUpdatedAt = &now
	if err = s.persistRevision(post, version); err != nil {
		return nil, err
	}
	return draftResponse(post), nil
}
