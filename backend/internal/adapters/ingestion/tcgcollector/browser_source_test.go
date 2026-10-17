package tcgcollector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// normalPage is a trimmed copy of the saved sets page in
// references/tcg-collector: Cloudflare's window.__CF$cv$params block and the
// precursor script it appends ride on every normal page, so neither may mark
// it as a challenge.
const normalPage = `<html><head><title>Pokémon TCG sets (Indonesian TCG) – TCG Collector</title>
<script>window.__CF$cv$params={r:'a4bcd58e7e60ce63',t:'MTc5MjIxNDIxNw==',u:'01a0756244307b428e74e70222d81ac1',ut:'w3TG1JRh6KN1.Gv6KLjxAFRoijq5uTbV5U0rmp4SCto-1792214218-1.2.1.1-kUR3OgCWn3mm9ABQFFrXXf3k60dsJ8GosKbnwGlsytpYZTHa8Gs7NwtIQasFXsuKZLYK0Q7JhtBUpmFPFdq1.mpLE0WRFc1jLZox7P205x0',i:60};(function(){if(!document.body)return;var s=document.createElement('script');s.src='/cdn-cgi/challenge-platform/scripts/precursor/main.js';document.head.appendChild(s);})();</script>
</head><body><main><h1>Indonesian TCG Sets</h1></main></body></html>`

// interstitialPage is a trimmed copy of Cloudflare's challenge page: its
// challenge config key and its title are the markers that are on screen.
const interstitialPage = `<html lang="en-US"><head><title>Just a moment...</title>
<meta http-equiv="refresh" content="390">
</head><body class="no-js"><div class="main-wrapper" role="main"><div class="main-content"><noscript>Enable JavaScript and cookies to continue</noscript></div></div>
<script>(function(){window._cf_chl_opt={cvId:'3',cZone:'tcgcollector.com',cType:'managed',cRay:'8b1f2c3d4e5f6a7b'};var cpo=document.createElement('script');cpo.src='/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1?ray=8b1f2c3d4e5f6a7b';document.getElementsByTagName('head')[0].appendChild(cpo);})();</script>
</body></html>`

// configKeyPage is the interstitial under a normal page title, so only its
// challenge config key can mark it.
const configKeyPage = `<html><head><title>Indonesian Sets | TCG Collector</title>
<script>window._cf_chl_opt={cvId:'3',cZone:'tcgcollector.com',cType:'managed',cRay:'8b1f2c3d4e5f6a7b'};</script>
</head><body></body></html>`

func TestCheckPageURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"sets page", "https://www.tcgcollector.com/sets/id", false},
		{"card list", "https://www.tcgcollector.com/sets/id/SM-P", false},
		{"page parameter", "https://www.tcgcollector.com/sets/id?page=2", true},
		{"empty page parameter", "https://www.tcgcollector.com/sets/id?page=", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPageURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsChallenge(t *testing.T) {
	tests := []struct {
		name string
		page string
		want bool
	}{
		{"normal page carrying Cloudflare's precursor script and cv params", normalPage, false},
		{"plain page without Cloudflare's scripts", `<html><head><title>Sets</title></head><body><main>Sets</main></body></html>`, false},
		{"interstitial with its challenge config key and title", interstitialPage, true},
		{"interstitial under a normal title, config key only", configKeyPage, true},
		{"interstitial with its title only", `<html><head><title>Just a moment...</title></head><body></body></html>`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isChallenge(tt.page))
		})
	}
}

func TestPaceWaitsForGapSinceLastRequest(t *testing.T) {
	s := &browserSource{gap: 50 * time.Millisecond, last: time.Now()}

	start := time.Now()
	assert.NoError(t, s.pace(context.Background()))
	assert.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond)
}

func TestPaceReturnsAtOnceWhenNoRecentRequest(t *testing.T) {
	s := &browserSource{gap: time.Hour}

	start := time.Now()
	assert.NoError(t, s.pace(context.Background()))
	assert.Less(t, time.Since(start), time.Second)
}

func TestFetchRefusesPageParameterBeforeLoading(t *testing.T) {
	source, cleanup := NewBrowserSource(t.TempDir())
	defer cleanup()

	_, err := source.Fetch(context.Background(), "https://www.tcgcollector.com/sets/id?page=2")

	assert.Error(t, err)
}
