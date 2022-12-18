package cookies

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type SameSite int

const (
	SameSiteDefaultMode SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

const (
	MaxAgeNotSet = 0
)

var GlobalJar = NewCookieJar()

type Cookie struct {
	Name  string
	Value string

	Path    string
	Domain  string
	Expires time.Time

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	MaxAge              int
	Secure              bool
	HttpOnly            bool
	SameSite            SameSite
	Raw                 string
	CalledCloseToExpire bool
}

func (c *Cookie) String() string {
	var len = len(c.Name) + len(c.Value) + 1
	cookieBuf := make([]byte, 0, len)
	cookieBuf = append(cookieBuf, c.Name...)
	cookieBuf = append(cookieBuf, '=')
	cookieBuf = append(cookieBuf, c.Value...)
	return string(cookieBuf)
}

func (c *Cookie) IsValid() bool {
	if c.Expires.Before(time.Now()) {
		return false
	} else if c.MaxAge < MaxAgeNotSet {
		return false
	}
	return true
}

type CookieJar struct {
	cookies                  []*Cookie
	mut                      *sync.Mutex
	terminate                chan bool
	OnExpire                 func(cookie *Cookie)
	CloseToExpire            func(cookie *Cookie)
	CloseToExpireTimeSeconds int
}

func NewCookieJar() *CookieJar {
	var cj = &CookieJar{
		cookies:                  make([]*Cookie, 0),
		mut:                      &sync.Mutex{},
		terminate:                make(chan bool),
		CloseToExpireTimeSeconds: 60,
	}
	cj.collect()
	return cj
}

func (cj *CookieJar) collect() {
	var ticker = time.NewTicker(1 * time.Second)
	go func(c *CookieJar) {
		for looping := true; looping; {
			select {
			case <-c.terminate:
				looping = false
			case <-ticker.C:
				for i, cookie := range cj.cookies {
					if !cookie.IsValid() || cookie.MaxAge-1 == MaxAgeNotSet {
						if cj.OnExpire != nil {
							cj.OnExpire(cookie)
						}
						cj.DelCookieIndex(i)
					} else if cookie.MaxAge-1 <= cj.CloseToExpireTimeSeconds || cookie.Expires.Before(time.Now().Add(time.Duration(cj.CloseToExpireTimeSeconds)*time.Second)) {
						if cj.CloseToExpire != nil && !cookie.CalledCloseToExpire {
							cj.mut.Lock()
							cj.CloseToExpire(cookie)
							cookie.CalledCloseToExpire = true
							// cj.cookies[i] = cookie
							cj.mut.Unlock()
						}
					} else {
						cj.mut.Lock()
						cookie.MaxAge = cookie.MaxAge - 1
						cj.mut.Unlock()
						// cj.cookies[i] = cookie
					}
				}
			}
		}
		close(cj.terminate)
	}(cj)
}

func (cj *CookieJar) Stop() {
	cj.terminate <- true
}

func (cj *CookieJar) SetHTTPCookie(cookie *http.Cookie) error {
	return cj.SetCookie(&Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Expires:  cookie.Expires,
		MaxAge:   cookie.MaxAge,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HttpOnly,
		SameSite: SameSite(cookie.SameSite),
		Raw:      cookie.Raw,
	})
}

func (cj *CookieJar) SetHTTPCookies(cookies []*http.Cookie) error {
	for _, cookie := range cookies {
		err := cj.SetHTTPCookie(cookie)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cj *CookieJar) Dump(r *http.Request, cookies ...string) error {
	for _, cookie := range cj.cookies {
		if !cookie.IsValid() {
			continue
		}
		var found = false
		for _, name := range cookies {
			if cookie.Name == name {
				found = true
				break
			}
		}
		if len(cookies) > 0 && !found {
			continue
		} else if len(cookies) == 0 {
			found = true
		}
		r.AddCookie(&http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  cookie.Expires,
			MaxAge:   cookie.MaxAge,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
			SameSite: http.SameSite(cookie.SameSite),
			Raw:      cookie.Raw,
		})
	}
	return nil
}

func (cj *CookieJar) DumpForm(rq *http.Request, cookies ...string) {
	var cookieMap = cj.AsMap(cookies...)
	var form = url.Values{}
	for key, value := range cookieMap {
		form.Add(key, value)
	}
	rq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rq.Body = io.NopCloser(strings.NewReader(form.Encode()))
}

func (cj *CookieJar) DumpJson(rq *http.Request, cookies ...string) error {
	var json, err = cj.AsJson(cookies...)
	if err != nil {
		return err
	}
	rq.Header.Set("Content-Type", "application/json")
	rq.Body = io.NopCloser(bytes.NewReader(json))
	return nil
}

func (cj *CookieJar) AsMap(cookies ...string) map[string]string {
	var kv = make(map[string]string)
	for _, cookie := range cj.cookies {
		if !cookie.IsValid() {
			continue
		}
		var found = false
		for _, name := range cookies {
			if cookie.Name == name {
				found = true
				break
			}
		}
		if len(cookies) > 0 && !found {
			continue
		} else if len(cookies) == 0 {
			found = true
		}
		kv[cookie.Name] = cookie.Value
	}
	return kv
}

func (cj *CookieJar) AsJson(cookies ...string) ([]byte, error) {
	return json.Marshal(cj.AsMap(cookies...))
}

func (cj *CookieJar) Get(name string) (*Cookie, error) {
	for _, cookie := range cj.cookies {
		if cookie.Name == name {
			if !cookie.IsValid() {
				return nil, errors.New("cookie is invalid")
			}
			return cookie, nil
		}
	}
	return nil, errors.New("cookie not found")
}

func (cj *CookieJar) All() []*Cookie {
	return cj.cookies
}
