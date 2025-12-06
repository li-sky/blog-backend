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
	posts, _, err := fc.postService.ListPublishedPosts(0, 20, "created_at", true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	feed := fc.buildAtomFeed(c, posts)
	c.XML(http.StatusOK, feed)
}

// RSSFeed generates an RSS 2.0 XML feed for published posts.
func (fc *FeedController) RSSFeed(c *gin.Context) {
	posts, _, err := fc.postService.ListPublishedPosts(0, 20, "created_at", true)
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
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []AtomLink{
			{Rel: "self", Href: siteURL + "/feed/atom", Type: "application/atom+xml"},
			{Rel: "alternate", Href: siteURL, Type: "text/html"},
		},
		Subtitle: siteDesc,
	}

	for _, post := range posts {
		postURL := fc.postURL(siteURL, post.ID)
		entry := AtomEntry{
			Title:   post.Title,
			ID:      postURL,
			Updated: post.UpdatedAt.Format(time.RFC3339),
			Link:    AtomLink{Rel: "alternate", Href: postURL, Type: "text/html"},
			Summary: post.Summary,
			Content: AtomContent{Type: "html", Content: post.Body},
		}
		if post.PublishedAt != nil {
			entry.Published = post.PublishedAt.Format(time.RFC3339)
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
		Title:       siteTitle,
		Link:        siteURL,
		Description: siteDesc,
		Language:    "zh-cn",
		PubDate:     time.Now().UTC().Format(time.RFC1123Z),
		AtomLink:    RSSAtomLink{Href: siteURL + "/feed/rss", Rel: "self", Type: "application/rss+xml"},
	}

	for _, post := range posts {
		postURL := fc.postURL(siteURL, post.ID)
		item := RSSItem{
			Title:       post.Title,
			Link:        postURL,
			Description: post.Summary,
			GUID:        RSSGUID{IsPermaLink: true, Value: postURL},
		}
		if post.PublishedAt != nil {
			item.PubDate = post.PublishedAt.Format(time.RFC1123Z)
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

func (fc *FeedController) getSetting(key, defaultVal string) string {
	setting, err := fc.settingService.GetByKey(key)
	if err != nil || setting == nil || setting.Value == "" {
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
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description string      `xml:"description"`
	Language    string      `xml:"language,omitempty"`
	PubDate     string      `xml:"pubDate,omitempty"`
	AtomLink    RSSAtomLink `xml:"atom:link"`
	Items       []RSSItem   `xml:"item"`
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
	GUID        RSSGUID `xml:"guid"`
}

type RSSGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}
