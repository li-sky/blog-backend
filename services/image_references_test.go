package services

import (
	"blog-backend/models"
	"math"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageReferenceMatchingUsesRenderedURLsNotFilenameSubstrings(t *testing.T) {
	image := &models.Image{Filename: "asset.jpg", URL: "https://skyli.xyz/current/asset.jpg", WebP600Filename: "asset_w600.webp", WebP600URL: "https://skyli.xyz/current/asset_w600.webp", WebP1200URL: "https://skyli.xyz/current/asset_w1200.webp", WebPFullFilename: "asset_full.webp", WebPFullURL: "https://skyli.xyz/current/asset_full.webp"}
	match := newImageURLMatcher(image)
	cases := []struct {
		name, source string
		used         bool
	}{
		{"original Markdown image", "![image](https://skyli.xyz/current/asset.jpg)", true},
		{"query and fragment", "![image](/current/asset.jpg?v=1#preview)", true},
		{"relative historical", "![image](../images/asset.jpg)", true},
		{"reference style", "![image][ref]\n\n[ref]: /images/asset_w600.webp \"description\"", true},
		{"image download link", "[download](https://skyli.xyz/current/asset_full.webp?dl=1)", true},
		{"GFM bare link", "https://skyli.xyz/current/asset.jpg", true},
		{"HTML image", "<img src=\"/current/asset_w1200.webp?x=1&amp;y=2\">", true},
		{"HTML srcset", "<picture><source srcset=\"/unrelated.webp 1x, /images/asset_w1200.webp 2x\"></picture>", true},
		{"scheme relative", "![image](//skyli.xyz/current/asset.jpg)", true},
		{"escaped path", "![image](/images/%61sset.jpg)", true},
		{"same filename external", "![image](https://outside.test/images/asset.jpg)", false},
		{"same host different file", "![image](/images/prefix-asset.jpg)", false},
		{"same filename wrong path", "![image](/elsewhere/asset.jpg)", false},
		{"filename only prose", "This is asset.jpg, not an image reference.", false},
		{"inline code", "`![image](/images/asset.jpg)`", false},
		{"fenced code", "```markdown\n![image](/images/asset.jpg)\n```", false},
		{"HTML comment", "<!-- <img src=\"/images/asset.jpg\"> -->", false},
		{"external URL query contains image", "[unrelated](https://outside.test/?image=https://skyli.xyz/current/asset.jpg)", false},
		{"external port", "![image](https://skyli.xyz:9999/current/asset.jpg)", false},
		{"script literal", "<script>const example='<img src=\"/images/asset.jpg\">';</script>", false},
		{"legacy link card icon", "[site](https://outside.test \"link-card:https://skyli.xyz/images/asset.jpg\")", true},
		{"encoded link card icon", "[site](https://outside.test \"link-card:v2:" + url.PathEscape(`{"icon":"https://skyli.xyz/current/asset.jpg"}`) + "\")", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.used, match.referencedBy(tc.source)) })
	}
}

func TestBoundedFullImageDimensionsAndLegacyFullRemainCompatible(t *testing.T) {
	for _, original := range [][2]int{{18121, 3633}, {3633, 18121}, {20000, 20000}, {10000, 30000}, {1600, 900}, {1, 20000}} {
		width, height := original[0], original[1]
		scale := math.Min(1, math.Min(8192/float64(max(width, height)), math.Sqrt(16000000/(float64(width)*float64(height)))))
		boundedWidth, boundedHeight := max(1, int(math.Floor(float64(width)*scale))), max(1, int(math.Floor(float64(height)*scale)))
		require.True(t, matchesVariantDimensions(width, height, width, height, 0), "legacy full must remain valid")
		require.True(t, matchesVariantDimensions(boundedWidth, boundedHeight, width, height, 0), "bounded full should match browser dimensions")
		require.LessOrEqual(t, max(boundedWidth, boundedHeight), 8192)
		require.LessOrEqual(t, int64(boundedWidth)*int64(boundedHeight), int64(16000000))
		require.False(t, matchesVariantDimensions(boundedWidth+2, boundedHeight+2, width, height, 0))
	}
	require.True(t, matchesVariantDimensions(600, 1200, 10000, 20000, 600))
	require.True(t, matchesVariantDimensions(1200, 2400, 10000, 20000, 1200))
	require.False(t, matchesVariantDimensions(600, 600, 10000, 20000, 600))
	require.False(t, matchesVariantDimensions(1, 1, 0, 0, 0))
}

func TestSummaryCoverReferencesMatchOnlyFirstCoverURL(t *testing.T) {
	match := newImageURLMatcher(&models.Image{Filename: "asset.jpg", URL: "https://skyli.xyz/current/asset.jpg", WebP600Filename: "asset_w600.webp", WebP600URL: "https://skyli.xyz/current/asset_w600.webp"})
	for _, tc := range []struct {
		name, summary string
		used          bool
	}{
		{"original cover", "Cover %https://skyli.xyz/current/asset.jpg% description", true},
		{"relative variant cover", "%/images/asset_w600.webp?version=1#cover%", true},
		{"external filename", "%https://outside.test/images/asset.jpg%", false},
		{"later token is not cover", "%https://outside.test/other.jpg% %/images/asset.jpg%", false},
		{"plain URL is not cover", "https://skyli.xyz/current/asset.jpg", false},
		{"different filename", "%/images/prefix-asset.jpg%", false},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.used, match.summaryCoverReferencedBy(tc.summary)) })
	}
}
