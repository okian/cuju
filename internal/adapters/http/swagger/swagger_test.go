package swagger

import (
	"context"
	"errors"
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

func TestSwaggerErrorStruct(t *testing.T) {
	Convey("Given a swagger Error struct", t, func() {
		Convey("When creating an error with all fields", func() {
			baseErr := errors.New("base error")
			swaggerErr := &Error{
				Op:   "test operation",
				Kind: ErrServe,
				Err:  baseErr,
			}

			Convey("Then it should format correctly", func() {
				So(swaggerErr.Error(), ShouldEqual, "test operation: swagger serve failed: base error")
			})

			Convey("And it should unwrap correctly", func() {
				So(swaggerErr.Unwrap(), ShouldEqual, baseErr)
			})

			Convey("And it should match the kind", func() {
				So(swaggerErr.Is(ErrServe), ShouldBeTrue)
			})

			Convey("And it should match the base error", func() {
				So(swaggerErr.Is(baseErr), ShouldBeTrue)
			})
		})

		Convey("When creating an error with only operation and kind", func() {
			swaggerErr := &Error{
				Op:   "test operation",
				Kind: ErrServe,
			}

			Convey("Then it should format correctly", func() {
				So(swaggerErr.Error(), ShouldEqual, "test operation: swagger serve failed")
			})

			Convey("And it should match the kind", func() {
				So(swaggerErr.Is(ErrServe), ShouldBeTrue)
			})
		})

		Convey("When creating an error with only operation and base error", func() {
			baseErr := errors.New("base error")
			swaggerErr := &Error{
				Op:  "test operation",
				Err: baseErr,
			}

			Convey("Then it should format correctly", func() {
				So(swaggerErr.Error(), ShouldEqual, "test operation: base error")
			})

			Convey("And it should match the base error", func() {
				So(swaggerErr.Is(baseErr), ShouldBeTrue)
			})
		})

		Convey("When creating an error with only kind", func() {
			swaggerErr := &Error{
				Kind: ErrServe,
			}

			Convey("Then it should format correctly", func() {
				So(swaggerErr.Error(), ShouldEqual, "swagger serve failed")
			})

			Convey("And it should match the kind", func() {
				So(swaggerErr.Is(ErrServe), ShouldBeTrue)
			})
		})

		Convey("When creating an error with only base error", func() {
			baseErr := errors.New("base error")
			swaggerErr := &Error{
				Err: baseErr,
			}

			Convey("Then it should format correctly", func() {
				So(swaggerErr.Error(), ShouldEqual, "base error")
			})

			Convey("And it should match the base error", func() {
				So(swaggerErr.Is(baseErr), ShouldBeTrue)
			})
		})

		Convey("When creating an error with no fields", func() {
			swaggerErr := &Error{}

			Convey("Then it should format as unknown error", func() {
				So(swaggerErr.Error(), ShouldEqual, "unknown error")
			})

			Convey("And it should not match any error", func() {
				So(swaggerErr.Is(ErrServe), ShouldBeFalse)
			})
		})

		Convey("When creating a nil error", func() {
			var swaggerErr *Error

			Convey("Then it should format as nil", func() {
				So(swaggerErr.Error(), ShouldEqual, "<nil>")
			})

			Convey("And it should not match any error", func() {
				So(swaggerErr.Is(ErrServe), ShouldBeFalse)
			})
		})
	})
}

func TestSwaggerErrorWrapping(t *testing.T) {
	Convey("Given error wrapping functions", t, func() {
		baseErr := errors.New("base error")

		Convey("When wrapping an error", func() {
			wrappedErr := Wrap("test operation", baseErr)

			Convey("Then it should be a swagger Error", func() {
				So(wrappedErr, ShouldHaveSameTypeAs, &Error{})
			})

			Convey("And it should contain the operation and base error", func() {
				swaggerErr := wrappedErr.(*Error)
				So(swaggerErr.Op, ShouldEqual, "test operation")
				So(swaggerErr.Err, ShouldEqual, baseErr)
				So(swaggerErr.Kind, ShouldBeNil)
			})
		})

		Convey("When wrapping with kind", func() {
			wrappedErr := WrapKind("test operation", ErrServe, baseErr)

			Convey("Then it should be a swagger Error", func() {
				So(wrappedErr, ShouldHaveSameTypeAs, &Error{})
			})

			Convey("And it should contain the operation, kind, and base error", func() {
				swaggerErr := wrappedErr.(*Error)
				So(swaggerErr.Op, ShouldEqual, "test operation")
				So(swaggerErr.Kind, ShouldEqual, ErrServe)
				So(swaggerErr.Err, ShouldEqual, baseErr)
			})
		})

		Convey("When wrapping with kind but no base error", func() {
			wrappedErr := WrapKind("test operation", ErrServe, nil)

			Convey("Then it should be a swagger Error", func() {
				So(wrappedErr, ShouldHaveSameTypeAs, &Error{})
			})

			Convey("And it should contain the operation and kind", func() {
				swaggerErr := wrappedErr.(*Error)
				So(swaggerErr.Op, ShouldEqual, "test operation")
				So(swaggerErr.Kind, ShouldEqual, ErrServe)
				So(swaggerErr.Err, ShouldBeNil)
			})
		})

		Convey("When creating a new kind error", func() {
			kindErr := NewKind("test operation", ErrServe)

			Convey("Then it should be a swagger Error", func() {
				So(kindErr, ShouldHaveSameTypeAs, &Error{})
			})

			Convey("And it should contain the operation and kind", func() {
				swaggerErr := kindErr.(*Error)
				So(swaggerErr.Op, ShouldEqual, "test operation")
				So(swaggerErr.Kind, ShouldEqual, ErrServe)
				So(swaggerErr.Err, ShouldBeNil)
			})
		})

		Convey("When wrapping a nil error", func() {
			wrappedErr := Wrap("test operation", nil)

			Convey("Then it should return nil", func() {
				So(wrappedErr, ShouldBeNil)
			})
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
					Register(nil, mux)
				}, ShouldNotPanic)
			})
		})
	})
}
