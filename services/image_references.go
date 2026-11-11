package services

import (
	"blog-backend/models"
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

var ErrImageInUse = errors.New("image_in_use")

type ImageReference struct {
	PostID   uint   `json:"postId"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Location string `json:"location"`
	authorID uint
}
type ImageReferencesResponse struct {
	Items []ImageReference `json:"items"`
	Total int              `json:"total"`
}
type ImageInUseError struct{ References ImageReferencesResponse }

func (e *ImageInUseError) Error() string { return ErrImageInUse.Error() }
func (e *ImageInUseError) Unwrap() error { return ErrImageInUse }

// Optional capability keeps existing repository implementations and test doubles
// compatible. The production ImageRepo implements this read path.
type imageReferencePostReader interface{ ListImageReferencePosts() ([]models.Post, error) }

func (s *ImageService) imageReferences(image *models.Image) ([]ImageReference, error) {
	reader, ok := s.imageRepo.(imageReferencePostReader)
	if !ok {
		return nil, nil
	}
	posts, err := reader.ListImageReferencePosts()
	if err != nil {
		return nil, err
	}
	matcher := newImageURLMatcher(image)
	references := make([]ImageReference, 0)
	for _, post := range posts {
		bodyReferenced := matcher.referencedBy(post.Body)
		if bodyReferenced {
			location := "draft"
			if post.Status == "published" {
				location = "published"
			}
			references = append(references, ImageReference{PostID: post.ID, Title: post.Title, Status: post.Status, Location: location, authorID: post.UserID})
		}
		if post.Draft != nil && matcher.referencedBy(post.Draft.Body) {
			// One entry per article location, not per repeated occurrence in the body.
			if post.Status != "published" && bodyReferenced {
				continue
			}
			references = append(references, ImageReference{PostID: post.ID, Title: post.Draft.Title, Status: post.Status, Location: "draft", authorID: post.UserID})
		}
	}
	return references, nil
}
func visibleImageReferences(user *models.User, references []ImageReference) ImageReferencesResponse {
	result := ImageReferencesResponse{Items: make([]ImageReference, 0), Total: len(references)}
	for _, reference := range references {
		if reference.Location == "draft" && !userHasPermission(user, "edit_post") && (user == nil || user.ID != reference.authorID) {
			continue
		}
		result.Items = append(result.Items, reference)
	}
	return result
}
func (s *ImageService) GetImageReferences(user *models.User, id uint) (ImageReferencesResponse, error) {
	if user == nil || (!userHasPermission(user, "upload_image") && !userHasPermission(user, "edit_any_image") && !userHasPermission(user, "delete_any_image")) {
		return ImageReferencesResponse{}, errors.New("permission denied: image management required")
	}
	image, err := s.imageRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ImageReferencesResponse{}, ErrImageNotFound
		}
		return ImageReferencesResponse{}, err
	}
	if image.UserID != user.ID && !userHasPermission(user, "edit_any_image") && !userHasPermission(user, "delete_any_image") {
		return ImageReferencesResponse{}, errors.New("permission denied: image owner or manager required")
	}
	refs, err := s.imageReferences(image)
	if err != nil {
		return ImageReferencesResponse{}, err
	}
	return visibleImageReferences(user, refs), nil
}

type imageURLMatcher struct {
	paths map[string]struct{}
	hosts map[string]struct{}
}

func canonicalImageHost(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" && port != "80" && port != "443" {
		host += ":" + port
	}
	return host
}
func canonicalImagePath(u *url.URL) string { return path.Clean("/" + strings.TrimLeft(u.Path, "/")) }
func newImageURLMatcher(image *models.Image) imageURLMatcher {
	m := imageURLMatcher{paths: map[string]struct{}{}, hosts: map[string]struct{}{}}
	for _, raw := range []string{image.URL, image.WebP600URL, image.WebP1200URL, image.WebPFullURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Path == "" {
			continue
		}
		m.paths[canonicalImagePath(u)] = struct{}{}
		if filename := path.Base(u.Path); filename != "." && filename != "/" {
			m.paths[path.Join("/images", filename)] = struct{}{}
		}
		if u.Host != "" {
			m.hosts[canonicalImageHost(u)] = struct{}{}
		}
	}
	for _, filename := range []string{image.Filename, image.WebP600Filename, image.WebP1200Filename, image.WebPFullFilename} {
		if filename != "" {
			m.paths[path.Join("/images", path.Base(filename))] = struct{}{}
		}
	}
	return m
}
func (m imageURLMatcher) matches(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || (u.Scheme != "" && !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https")) {
		return false
	}
	if u.Host != "" {
		if _, ok := m.hosts[canonicalImageHost(u)]; !ok {
			return false
		}
	}
	_, found := m.paths[canonicalImagePath(u)]
	return found
}

func (m imageURLMatcher) referencedBy(source string) bool {
	if source == "" {
		return false
	}
	// Render only in memory. CommonMark resolves reference-style destinations and
	// escapes code samples; the tokenizer then sees actual image/link attributes.
	var rendered bytes.Buffer
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()))
	if err := markdown.Convert([]byte(source), &rendered); err != nil {
		return false
	}
	tokenizer := html.NewTokenizer(&rendered)
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return false
		}
		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}
		token := tokenizer.Token()
		for _, attribute := range token.Attr {
			switch attribute.Key {
			case "src", "href", "poster":
				if m.matches(attribute.Val) {
					return true
				}
			case "title":
				// Link-card icons are rendered from this existing Markdown extension.
				if strings.HasPrefix(attribute.Val, "link-card:v2:") {
					if data, err := url.PathUnescape(strings.TrimPrefix(attribute.Val, "link-card:v2:")); err == nil {
						var metadata struct {
							Icon string `json:"icon"`
						}
						if json.Unmarshal([]byte(data), &metadata) == nil && m.matches(metadata.Icon) {
							return true
						}
					}
				} else if strings.HasPrefix(attribute.Val, "link-card:") && m.matches(strings.TrimPrefix(attribute.Val, "link-card:")) {
					return true
				}
			case "srcset":
				for _, candidate := range strings.Split(attribute.Val, ",") {
					fields := strings.Fields(candidate)
					if len(fields) > 0 && m.matches(fields[0]) {
						return true
					}
				}
			}
		}
	}
}
