package swagger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestSwaggerHandler(t *testing.T) {
	convey.Convey("Given a swagger handler", t, func() {
		ctx := context.Background()
		mux := http.NewServeMux()

		convey.Convey("When registering the swagger handler", func() {
			Register(ctx, mux)

			convey.Convey("Then it should handle /openapi.yaml route", func() {
				req := httptest.NewRequest("GET", "/openapi.yaml", http.NoBody)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				convey.So(w.Code, convey.ShouldEqual, http.StatusOK)
				convey.So(w.Header().Get("Content-Type"), convey.ShouldEqual, "application/yaml; charset=utf-8")
				convey.So(w.Body.Len(), convey.ShouldBeGreaterThan, 0)
			})

			convey.Convey("And it should handle /api-docs route", func() {
				req := httptest.NewRequest("GET", "/api-docs", http.NoBody)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				convey.So(w.Code, convey.ShouldEqual, http.StatusOK)
				convey.So(w.Header().Get("Content-Type"), convey.ShouldEqual, "text/html; charset=utf-8")
				convey.So(w.Body.String(), convey.ShouldContainSubstring, "API Docs – ReDoc")
				convey.So(w.Body.String(), convey.ShouldContainSubstring, "redoc-container")
			})

			convey.Convey("And it should handle /api-docs/ subpath route", func() {
				req := httptest.NewRequest("GET", "/api-docs/redoc.standalone.js", http.NoBody)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				convey.So(w.Code, convey.ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestSwaggerErrors(t *testing.T) {
	convey.Convey("Given swagger error constants", t, func() {
		convey.Convey("Then ErrServe should be defined", func() {
			convey.So(ErrServe, convey.ShouldNotBeNil)
			convey.So(ErrServe.Error(), convey.ShouldEqual, "swagger serve failed")
		})
	})
}

func TestSwaggerHandlerWithNilMux(t *testing.T) {
	convey.Convey("Given a nil mux", t, func() {
		ctx := context.Background()

		convey.Convey("When registering the swagger handler", func() {
			convey.Convey("Then it should panic", func() {
				convey.So(func() {
					Register(ctx, nil)
				}, convey.ShouldPanic)
			})
		})
	})
}

func TestSwaggerHandlerWithNilContext(t *testing.T) {
	convey.Convey("Given a nil context", t, func() {
		mux := http.NewServeMux()

		convey.Convey("When registering the swagger handler", func() {
			convey.Convey("Then it should not panic", func() {
				convey.So(func() {
					Register(context.TODO(), mux)
				}, convey.ShouldNotPanic)
			})
		})
	})
}
