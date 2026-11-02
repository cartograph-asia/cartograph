package web

import (
	"testing"

	"github.com/cartograph-asia/cartograph/internal/views"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
)

// The photo page prints the camera line when there is camera data, escaped, and
// nothing when there is none (docs/CAMERA_BRIEF.md).
func TestPhotoPageCamera(t *testing.T) {
	p := viewstest.Photo()
	body := render(t, views.PagePhoto, p)
	must(t, body, `<p class="camera">Apple iPhone 17 Pro · 6.8 mm (48 mm equiv.) · f/1.8 · 1/250 s · ISO 64</p>`)
	mustNot(t, body, "back triple camera") // the lens is not on the line

	p.Camera = &views.Camera{Label: `<script>alert(1)</script> "Cam"`}
	body = render(t, views.PagePhoto, p)
	must(t, body, `<p class="camera">&lt;script&gt;alert(1)&lt;/script&gt; &#34;Cam&#34;</p>`)
	mustNot(t, body, "<script>alert")

	p.Camera = nil
	body = render(t, views.PagePhoto, p)
	mustNot(t, body, `class="camera"`, "Unknown camera")
}
