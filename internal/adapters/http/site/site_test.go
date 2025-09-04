package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSiteHandler(t *testing.T) {
	Convey("Given a site handler", t, func() {
		ctx := context.Background()
		mux := http.NewServeMux()

		Convey("When registering the site handler", func() {
			Register(ctx, mux)

			Convey("Then it should handle /docs route", func() {
				req := httptest.NewRequest("GET", "/docs", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				// The handler might redirect /docs to /docs/ or serve directly
				So(w.Code, ShouldBeIn, []int{http.StatusOK, http.StatusMovedPermanently})
				if w.Code == http.StatusOK {
					So(w.Header().Get("Content-Type"), ShouldContainSubstring, "text/html")
				}
			})

			Convey("And it should handle /docs/ route", func() {
				req := httptest.NewRequest("GET", "/docs/", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				So(w.Code, ShouldEqual, http.StatusOK)
				So(w.Header().Get("Content-Type"), ShouldContainSubstring, "text/html")
			})

			Convey("And it should handle /docs/subpath route", func() {
				req := httptest.NewRequest("GET", "/docs/pages/conventions.html", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				// Subpaths are handled by the /docs/ route, not the /docs route
				So(w.Code, ShouldEqual, http.StatusOK)
			})

			Convey("And it should not handle root / route", func() {
				req := httptest.NewRequest("GET", "/", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})

			Convey("And it should not handle root subpath route", func() {
				req := httptest.NewRequest("GET", "/some-asset", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func TestSiteErrors(t *testing.T) {
	Convey("Given site error constants", t, func() {
		Convey("Then ErrGenerate should be defined", func() {
			So(ErrGenerate, ShouldNotBeNil)
			So(ErrGenerate.Error(), ShouldEqual, "docs site generation failed")
		})

		Convey("And ErrServe should be defined", func() {
			So(ErrServe, ShouldNotBeNil)
			So(ErrServe.Error(), ShouldEqual, "docs site serve failed")
		})

		Convey("And errors should be different", func() {
			So(ErrGenerate, ShouldNotEqual, ErrServe)
		})
	})
}

func TestSiteHandlerWithNilMux(t *testing.T) {
	Convey("Given a nil mux", t, func() {
		ctx := context.Background()

		Convey("When registering the site handler", func() {
			Convey("Then it should panic", func() {
				So(func() {
					Register(ctx, nil)
				}, ShouldPanic)
			})
		})
	})
}

func TestSiteHandlerWithNilContext(t *testing.T) {
	Convey("Given a nil context", t, func() {
		mux := http.NewServeMux()

		Convey("When registering the site handler", func() {
			Convey("Then it should not panic", func() {
				So(func() {
					Register(context.TODO(), mux)
				}, ShouldNotPanic)
			})
		})
	})
}
