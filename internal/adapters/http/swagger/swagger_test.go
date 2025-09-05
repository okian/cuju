package swagger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSwaggerHandler(t *testing.T) {
	Convey("Given a swagger handler", t, func() {
		ctx := context.Background()
		mux := http.NewServeMux()

		Convey("When registering the swagger handler", func() {
			Register(ctx, mux)

			Convey("Then it should handle /openapi.yaml route", func() {
				req := httptest.NewRequest("GET", "/openapi.yaml", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				So(w.Code, ShouldEqual, http.StatusOK)
				So(w.Header().Get("Content-Type"), ShouldEqual, "application/yaml; charset=utf-8")
				So(w.Body.Len(), ShouldBeGreaterThan, 0)
			})

			Convey("And it should handle /api-docs route", func() {
				req := httptest.NewRequest("GET", "/api-docs", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				So(w.Code, ShouldEqual, http.StatusOK)
				So(w.Header().Get("Content-Type"), ShouldEqual, "text/html; charset=utf-8")
				So(w.Body.String(), ShouldContainSubstring, "API Docs – ReDoc")
				So(w.Body.String(), ShouldContainSubstring, "redoc-container")
			})

			Convey("And it should handle /api-docs/ subpath route", func() {
				req := httptest.NewRequest("GET", "/api-docs/redoc.standalone.js", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				So(w.Code, ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestSwaggerErrors(t *testing.T) {
	Convey("Given swagger error constants", t, func() {
		Convey("Then ErrServe should be defined", func() {
			So(ErrServe, ShouldNotBeNil)
			So(ErrServe.Error(), ShouldEqual, "swagger serve failed")
		})
	})
}

func TestSwaggerHandlerWithNilMux(t *testing.T) {
	Convey("Given a nil mux", t, func() {
		ctx := context.Background()

		Convey("When registering the swagger handler", func() {
			Convey("Then it should panic", func() {
				So(func() {
					Register(ctx, nil)
				}, ShouldPanic)
			})
		})
	})
}

func TestSwaggerHandlerWithNilContext(t *testing.T) {
	Convey("Given a nil context", t, func() {
		mux := http.NewServeMux()

		Convey("When registering the swagger handler", func() {
			Convey("Then it should not panic", func() {
				So(func() {
					Register(context.TODO(), mux)
				}, ShouldNotPanic)
			})
		})
	})
}
