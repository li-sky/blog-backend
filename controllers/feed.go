package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type FeedController struct {
	postService    *services.PostService
	settingService *services.SettingService
}

func NewFeedController(postService *services.PostService, settingService *services.SettingService) *FeedController {
	return &FeedController{
		postService:    postService,
		settingService: settingService,
	}
}

// AtomFeed generates an Atom 1.0 XML feed for published posts.
func (fc *FeedController) AtomFeed(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	posts, _, err := fc.postService.ListPublishedPosts(0, 20, "published_at", true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	feed := fc.buildAtomFeed(c, posts)
	c.XML(http.StatusOK, feed)
}

// RSSFeed generates an RSS 2.0 XML feed for published posts.
func (fc *FeedController) RSSFeed(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	posts, _, err := fc.postService.ListPublishedPosts(0, 20, "published_at", true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	feed := fc.buildRSSFeed(c, posts)
	c.XML(http.StatusOK, feed)
}

// buildAtomFeed constructs the Atom feed structure from posts.
func (fc *FeedController) buildAtomFeed(c *gin.Context, posts []models.Post) AtomFeed {
	siteTitle := fc.getSetting("site_title", "My Blog")
	siteURL := fc.getSetting("site_url", fc.defaultSiteURL(c))
	siteDesc := fc.getSetting("site_description", "A blog about technology and life")

	feed := AtomFeed{
		XMLName: xml.Name{Space: "http://www.w3.org/2005/Atom", Local: "feed"},
		Title:   siteTitle,
		ID:      siteURL,
		Updated: latestPublicEvent(posts).Format(time.RFC3339),
		Links: []AtomLink{
			{Rel: "self", Href: siteURL + "/feed/atom", Type: "application/atom+xml"},
			{Rel: "alternate", Href: siteURL, Type: "text/html"},
		},
		Subtitle: siteDesc,
		Author:   AtomAuthor{Name: siteTitle},
	}

	for _, post := range posts {
		if post.Status != "published" {
			continue
		}
		updated := publicPostUpdatedAt(post)
		// Atom requires entry.updated. An old PublishedAt may still represent a
		// recorded public event, even when its being the first publication is unknown.
		// Database creation/save times are never used as public-event timestamps.
		if updated == nil {
			continue
		}
		postURL := fc.postURL(siteURL, post.ID)
		entry := AtomEntry{
			Title:   post.Title,
			ID:      postURL,
			Updated: updated.UTC().Format(time.RFC3339),
			Link:    AtomLink{Rel: "alternate", Href: postURL, Type: "text/html"},
			Summary: post.Summary,
			Content: AtomContent{Type: "html", Content: post.Body},
		}
		if post.PublicationTimeKnown && validPublicTime(post.PublishedAt) {
			entry.Published = post.PublishedAt.UTC().Format(time.RFC3339)
		}
		feed.Entries = append(feed.Entries, entry)
	}

	return feed
}

// buildRSSFeed constructs the RSS feed structure from posts.
func (fc *FeedController) buildRSSFeed(c *gin.Context, posts []models.Post) RSSFeed {
	siteTitle := fc.getSetting("site_title", "My Blog")
	siteURL := fc.getSetting("site_url", fc.defaultSiteURL(c))
	siteDesc := fc.getSetting("site_description", "A blog about technology and life")

	channel := RSSChannel{
		Title:         siteTitle,
		Link:          siteURL,
		Description:   siteDesc,
		Language:      "zh-cn",
		PubDate:       latestFirstPublication(posts),
		LastBuildDate: latestRSSPublicEvent(posts),
		AtomLink:      RSSAtomLink{Href: siteURL + "/feed/rss", Rel: "self", Type: "application/rss+xml"},
	}

	for _, post := range posts {
		if post.Status != "published" {
			continue
		}
		postURL := fc.postURL(siteURL, post.ID)
		item := RSSItem{
			Title:       post.Title,
			Link:        postURL,
			Description: post.Summary,
			GUID:        RSSGUID{IsPermaLink: true, Value: postURL},
		}
		if post.PublicationTimeKnown && validPublicTime(post.PublishedAt) {
			item.PubDate = post.PublishedAt.UTC().Format(time.RFC1123Z)
		}
		if validPublicTime(post.PublishedUpdatedAt) && (!validPublicTime(post.PublishedAt) || post.PublishedUpdatedAt.After(*post.PublishedAt)) {
			item.Updated = post.PublishedUpdatedAt.UTC().Format(time.RFC3339)
		}
		channel.Items = append(channel.Items, item)
	}

	return RSSFeed{
		XMLName: xml.Name{Local: "rss"},
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: channel,
	}
}

func validPublicTime(value *time.Time) bool { return value != nil && !value.IsZero() }

func publicPostUpdatedAt(post models.Post) *time.Time {
	if validPublicTime(post.PublishedUpdatedAt) && (!validPublicTime(post.PublishedAt) || post.PublishedUpdatedAt.After(*post.PublishedAt)) {
		return post.PublishedUpdatedAt
	}
	if validPublicTime(post.PublishedAt) {
		return post.PublishedAt
	}
	return nil
}

func latestKnownPublicEvent(posts []models.Post) *time.Time {
	var latest *time.Time
	for _, post := range posts {
		if post.Status != "published" {
			continue
		}
		if timestamp := publicPostUpdatedAt(post); timestamp != nil && (latest == nil || timestamp.After(*latest)) {
			latest = timestamp
		}
	}
	return latest
}

func latestPublicEvent(posts []models.Post) time.Time {
	if latest := latestKnownPublicEvent(posts); latest != nil {
		return latest.UTC()
	}
	// Atom still requires feed.updated for an empty feed. This is the generated
	// representation's timestamp, never an inferred article publication date.
	return time.Now().UTC()
}
func latestRSSPublicEvent(posts []models.Post) string {
	if latest := latestKnownPublicEvent(posts); latest != nil {
		return latest.UTC().Format(time.RFC1123Z)
	}
	return ""
}
func latestFirstPublication(posts []models.Post) string {
	var latest *time.Time
	for _, post := range posts {
		if post.Status == "published" && post.PublicationTimeKnown && validPublicTime(post.PublishedAt) && (latest == nil || post.PublishedAt.After(*latest)) {
			latest = post.PublishedAt
		}
	}
	if latest != nil {
		return latest.UTC().Format(time.RFC1123Z)
	}
	return ""
}

func (fc *FeedController) getSetting(key, defaultVal string) string {
	setting, err := fc.settingService.GetByKey(key)
	if setting == nil || setting.Value == "" {
		// write default value back to settings
		setting, err = fc.settingService.CreateOrUpdate(key, defaultVal)
		if err != nil {
			return defaultVal
		}
	}
	if err != nil {
		return defaultVal
	}
	return setting.Value
}

func (fc *FeedController) defaultSiteURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func (fc *FeedController) postURL(base string, postID uint) string {
	return fmt.Sprintf("%s/post/%d", base, postID)
}

// AtomFeed represents the root element of an Atom feed.
type AtomFeed struct {
	XMLName  xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title    string      `xml:"title"`
	ID       string      `xml:"id"`
	Updated  string      `xml:"updated"`
	Links    []AtomLink  `xml:"link"`
	Subtitle string      `xml:"subtitle,omitempty"`
	Entries  []AtomEntry `xml:"entry"`
	Author   AtomAuthor  `xml:"author"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomLink struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type AtomEntry struct {
	Title     string      `xml:"title"`
	ID        string      `xml:"id"`
	Updated   string      `xml:"updated"`
	Published string      `xml:"published,omitempty"`
	Link      AtomLink    `xml:"link"`
	Summary   string      `xml:"summary,omitempty"`
	Content   AtomContent `xml:"content,omitempty"`
}

type AtomContent struct {
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// RSSFeed represents the root element of an RSS 2.0 feed.
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title         string      `xml:"title"`
	Link          string      `xml:"link"`
	Description   string      `xml:"description"`
	Language      string      `xml:"language,omitempty"`
	PubDate       string      `xml:"pubDate,omitempty"`
	LastBuildDate string      `xml:"lastBuildDate,omitempty"`
	AtomLink      RSSAtomLink `xml:"atom:link"`
	Items         []RSSItem   `xml:"item"`
}

type RSSAtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type RSSItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	PubDate     string  `xml:"pubDate,omitempty"`
	Updated     string  `xml:"atom:updated,omitempty"`
	GUID        RSSGUID `xml:"guid"`
}

type RSSGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}
