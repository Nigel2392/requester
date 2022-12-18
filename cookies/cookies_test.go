package cookies_test

import (
	"testing"
	"time"

	"github.com/Nigel2392/requester/cookies"
)

var CookieJAR = cookies.NewCookieJar()

func TestCollect(t *testing.T) {
	CookieJAR.SetCookie(&cookies.Cookie{
		Name:     "testing_jar",
		Value:    "testing_jar",
		Path:     "/",
		Domain:   "localhost",
		Expires:  time.Now().Add(1 * time.Second),
		MaxAge:   2,
		Secure:   false,
		HttpOnly: false},
	)

	cookie, err := CookieJAR.Get("testing_jar")
	if err != nil {
		t.Error(err)
	}
	if cookie.Name != "testing_jar" {
		t.Error("Cookie name does not match")
	}
	if cookie.Value != "testing_jar" {
		t.Error("Cookie value does not match")
	}
	if cookie.Path != "/" {
		t.Error("Cookie path does not match")
	}
	if cookie.Domain != "localhost" {
		t.Error("Cookie domain does not match")
	}
	if cookie.Expires.Before(time.Now()) {
		t.Error("Cookie expires before now")
	}
	if cookie.MaxAge != 2 {
		t.Error("Cookie max age does not match")
	}
	if cookie.Secure != false {
		t.Error("Cookie secure does not match")
	}
	if cookie.HttpOnly != false {
		t.Error("Cookie http only does not match")
	}

	t.Log(cookie)
	t.Log(CookieJAR.All())

	// Test cookie expiration
	time.Sleep(2 * time.Second)
	_, err = CookieJAR.Get("testing_jar")
	if err == nil {
		t.Error("Cookie should have expired")
	}

	t.Log(CookieJAR.All())
}
