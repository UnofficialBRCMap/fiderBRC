package identity_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WeCanHearYou/wechy/app"
	"github.com/WeCanHearYou/wechy/app/identity"
	"github.com/WeCanHearYou/wechy/app/mock"
	"github.com/labstack/echo"
	. "github.com/onsi/gomega"
)

var testCases = []struct {
	domain string
	tenant *app.Tenant
	hosts  []string
}{
	{
		"orange.test.canhearyou.com",
		&app.Tenant{Name: "The Orange Inc."},
		[]string{
			"orange.test.canhearyou.com",
			"orange.test.canhearyou.com:3000",
		},
	},
	{
		"trishop.test.canhearyou.com",
		&app.Tenant{Name: "The Triathlon Shop"},
		[]string{
			"trishop.test.canhearyou.com",
			"trishop.test.canhearyou.com:1231",
			"trishop.test.canhearyou.com:80",
		},
	},
}

type mockTenantService struct{}

func (svc mockTenantService) GetByDomain(domain string) (*app.Tenant, error) {
	for _, testCase := range testCases {
		if testCase.domain == domain {
			return testCase.tenant, nil
		}
	}
	return nil, app.ErrNotFound
}

func TestMultiTenant(t *testing.T) {
	RegisterTestingT(t)

	for _, testCase := range testCases {
		for _, host := range testCase.hosts {

			server := mock.NewServer()
			req, _ := http.NewRequest(echo.GET, "/", nil)
			rec := httptest.NewRecorder()
			c := server.NewContext(req, rec)
			c.Request().Host = host

			mw := identity.MultiTenant(&mockTenantService{})
			mw(func(c app.Context) error {
				return c.String(http.StatusOK, c.Tenant().Name)
			})(c)

			Expect(rec.Code).To(Equal(200))
			Expect(rec.Body.String()).To(Equal(testCase.tenant.Name))
		}
	}
}

func TestMultiTenant_UnknownDomain(t *testing.T) {
	RegisterTestingT(t)

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().Host = "somedomain.com"

	mw := identity.MultiTenant(&mockTenantService{})
	mw(func(c app.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})(c)

	Expect(rec.Code).To(Equal(404))
}

func TestJwtGetter_NoCookie(t *testing.T) {
	RegisterTestingT(t)

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)

	mw := identity.JwtGetter()
	mw(func(c app.Context) error {
		if c.IsAuthenticated() {
			return c.NoContent(http.StatusOK)
		} else {
			return c.NoContent(http.StatusNoContent)
		}
	})(c)

	Expect(rec.Code).To(Equal(http.StatusNoContent))
}

func TestJwtGetter_WithCookie(t *testing.T) {
	RegisterTestingT(t)

	token, _ := identity.Encode(&app.WechyClaims{
		UserName: "Jon Snow",
	})

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	mw := identity.JwtGetter()
	mw(func(c app.Context) error {
		return c.String(http.StatusOK, c.Claims().UserName)
	})(c)

	Expect(rec.Code).To(Equal(http.StatusOK))
	Expect(rec.Body.String()).To(Equal("Jon Snow"))
}

func TestJwtSetter_WithoutJwt(t *testing.T) {
	RegisterTestingT(t)

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/abc", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().Host = "orange.test.canhearyou.com"

	mw := identity.JwtSetter()
	mw(func(c app.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	Expect(rec.Code).To(Equal(http.StatusOK))
}

func TestJwtSetter_WithJwt_WithoutParameter(t *testing.T) {
	RegisterTestingT(t)

	token, _ := identity.Encode(&app.WechyClaims{
		UserName: "Jon Snow",
	})

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/abc?jwt="+token, nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().Host = "orange.test.canhearyou.com"

	mw := identity.JwtSetter()
	mw(func(c app.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	Expect(rec.Code).To(Equal(http.StatusTemporaryRedirect))
	Expect(rec.Header().Get("Location")).To(Equal("http://orange.test.canhearyou.com/abc"))
}

func TestJwtSetter_WithJwt_WithParameter(t *testing.T) {
	RegisterTestingT(t)

	token, _ := identity.Encode(&app.WechyClaims{
		UserName: "Jon Snow",
	})

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/abc?jwt="+token+"&foo=bar", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().Host = "orange.test.canhearyou.com"

	mw := identity.JwtSetter()
	mw(func(c app.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	Expect(rec.Code).To(Equal(http.StatusTemporaryRedirect))
	Expect(rec.Header().Get("Location")).To(Equal("http://orange.test.canhearyou.com/abc?foo=bar"))
}

func TestHostChecker(t *testing.T) {
	RegisterTestingT(t)

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().Host = "login.test.canhearyou.com"

	mw := identity.HostChecker("http://login.test.canhearyou.com")
	mw(func(c app.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)

	Expect(rec.Code).To(Equal(http.StatusOK))
}

func TestHostChecker_DifferentHost(t *testing.T) {
	RegisterTestingT(t)

	server := mock.NewServer()
	req, _ := http.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := server.NewContext(req, rec)
	c.Request().Host = "orange.test.canhearyou.com"

	mw := identity.HostChecker("login.test.canhearyou.com")
	mw(app.HandlerFunc(func(c app.Context) error {
		return c.NoContent(http.StatusOK)
	}))(c)

	Expect(rec.Code).To(Equal(http.StatusBadRequest))
}