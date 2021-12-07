/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package tests

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/chromium"
	"github.com/oxtoacart/bpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	k6common "go.k6.io/k6/js/common"
	k6http "go.k6.io/k6/js/modules/k6/http"
	k6lib "go.k6.io/k6/lib"
	k6metrics "go.k6.io/k6/lib/metrics"
	k6test "go.k6.io/k6/lib/testutils/httpmultibin"
	k6stats "go.k6.io/k6/stats"
	"gopkg.in/guregu/null.v3"
)

// Browser is a test browser for integration testing.
type Browser struct {
	Ctx          context.Context
	Runtime      *goja.Runtime
	State        *k6lib.State
	HTTPMultiBin *k6test.HTTPMultiBin
	Samples      chan k6stats.SampleContainer
	api.Browser
}

// TestBrowser configures and launches a chrome browser.
// It automatically closes the browser when `t` returns.
//
// opts provides a way to customize the TestBrowser.
// see: withLaunchOptions for an example.
func TestBrowser(t testing.TB, opts ...interface{}) *Browser {
	launchOpts := defaultLaunchOpts()
	if len(opts) == 1 {
		switch opt := opts[0].(type) {
		case withLaunchOptions:
			launchOpts = opt
		}
	}

	var (
		tb      = k6test.NewHTTPMultiBin(t)
		samples = make(chan k6stats.SampleContainer, 1000)
	)

	root, err := k6lib.NewGroup("", nil)
	require.NoError(t, err)

	state := &k6lib.State{
		Options: k6lib.Options{
			MaxRedirects: null.IntFrom(10),
			UserAgent:    null.StringFrom("TestUserAgent"),
			Throw:        null.BoolFrom(true),
			SystemTags:   &k6stats.DefaultSystemTagSet,
			Batch:        null.IntFrom(20),
			BatchPerHost: null.IntFrom(20),
			// HTTPDebug:    null.StringFrom("full"),
		},
		Logger:         logrus.StandardLogger(),
		Group:          root,
		TLSConfig:      tb.TLSClientConfig,
		Transport:      tb.HTTPTransport,
		BPool:          bpool.NewBufferPool(1),
		Samples:        samples,
		Tags:           k6lib.NewTagMap(map[string]string{"group": root.Path}),
		BuiltinMetrics: k6metrics.RegisterBuiltinMetrics(k6metrics.NewRegistry()),
	}
	ctx := k6lib.WithState(tb.Context, state)

	rt := goja.New()
	rt.SetFieldNameMapper(k6common.FieldNameMapper{})
	ctx = k6common.WithRuntime(ctx, rt)

	err = rt.Set("http", k6common.Bind(rt, new(k6http.GlobalHTTP).NewModuleInstancePerVU(), &ctx))
	require.NoError(t, err)

	b := chromium.NewBrowserType(ctx).(*chromium.BrowserType)
	browser := b.Launch(rt.ToValue(launchOpts))
	t.Cleanup(browser.Close)

	return &Browser{
		Ctx:          b.Ctx, // This context has the additional wrapping of common.WithLaunchOptions
		Runtime:      rt,
		State:        state,
		Browser:      browser,
		HTTPMultiBin: tb,
		Samples:      samples,
	}
}

// WithHandle adds the given handler to the HTTP test server and makes it
// accessible with the given pattern.
func (b *Browser) WithHandle(pattern string, handler http.HandlerFunc) *Browser {
	b.HTTPMultiBin.Mux.Handle(pattern, handler)
	return b
}

const testBrowserStaticDir = "static"

// WithStaticFiles adds a file server to the HTTP test server that is accessible
// via `testBrowserStaticDir` prefix.
func (b *Browser) WithStaticFiles() *Browser {
	const (
		slash = string(os.PathSeparator)
		path  = slash + testBrowserStaticDir + slash
	)

	fs := http.FileServer(http.Dir(testBrowserStaticDir))

	return b.WithHandle(path, http.StripPrefix(path, fs).ServeHTTP)
}

// URL returns the listening HTTP test server's URL combined with the given path.
func (b *Browser) URL(path string) string {
	return b.HTTPMultiBin.ServerHTTP.URL + path
}

// StaticURL is a helper for URL("/`testBrowserStaticDir`/"+ path).
func (b *Browser) StaticURL(path string) string {
	return b.URL("/" + testBrowserStaticDir + "/" + path)
}

// AttachFrame attaches the frame to the page and returns it.
func (b *Browser) AttachFrame(page api.Page, frameID string, url string) api.Frame {
	pageFn := `
	async (frameId, url) => {
		const frame = document.createElement('iframe');
		frame.src = url;
		frame.id = frameId;
		document.body.appendChild(frame);
		await new Promise(x => frame.onload = x);
		return frame;
	}
	`
	return page.EvaluateHandle(
		b.Runtime.ToValue(pageFn),
		b.Runtime.ToValue(frameID),
		b.Runtime.ToValue(url)).
		AsElement().
		ContentFrame()
}

// DetachFrame detaches the frame from the page.
func (b *Browser) DetachFrame(page api.Page, frameID string) {
	pageFn := `
	frameId => {
        	document.getElementById(frameId).remove();
    	}
	`
	page.Evaluate(
		b.Runtime.ToValue(pageFn),
		b.Runtime.ToValue(frameID))
}

// launchOptions provides a way to customize browser type
// launch options in tests.
type launchOptions struct {
	Debug    bool   `js:"debug"`
	Headless bool   `js:"headless"`
	SlowMo   string `js:"slowMo"`
	Timeout  string `js:"timeout"`
}

// withLaunchOptions is a helper for increasing readability
// in tests while customizing the browser type launch options.
//
// example:
//
//    b := TestBrowser(t, withLaunchOptions{
//        SlowMo:  "100s",
//        Timeout: "30s",
//    })
//
type withLaunchOptions = launchOptions

// defaultLaunchOptions returns defaults for browser type launch options.
// TestBrowser uses this for launching a browser type by default.
func defaultLaunchOpts() launchOptions {
	var (
		debug    = false
		headless = true
	)
	if v, found := os.LookupEnv("XK6_BROWSER_TEST_DEBUG"); found {
		debug, _ = strconv.ParseBool(v)
	}
	if v, found := os.LookupEnv("XK6_BROWSER_TEST_HEADLESS"); found {
		headless, _ = strconv.ParseBool(v)
	}

	return launchOptions{
		Debug:    debug,
		Headless: headless,
		SlowMo:   "0s",
		Timeout:  "30s",
	}
}
