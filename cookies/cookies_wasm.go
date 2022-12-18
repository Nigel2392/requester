//go:build js && wasm
// +build js,wasm

package cookies

import (
	"errors"
	"strings"
	"syscall/js"
)

func GetDOMCookie(name string) (*Cookie, error) {
	var cookies = js.Global().Get("document").Get("cookie").String()
	var cookieArray = strings.Split(cookies, ";")
	for _, cookie := range cookieArray {
		var cookieName = strings.Split(cookie, "=")[0]
		if cookieName == name {
			var cookieValue = strings.Split(cookie, "=")[1]
			return &Cookie{Name: cookieName, Value: cookieValue}, nil
		}
	}
	return nil, errors.New("cookie not found")
}

func SetDOMCookie(name, value string) {
	var cookie = name + "=" + value + ";"
	js.Global().Get("document").Set("cookie", cookie)
}

func (cj *CookieJar) DelCookieIndex(index int) error {
	cj.mut.Lock()
	defer cj.mut.Unlock()
	if index < 0 || index >= len(cj.cookies) {
		return errors.New("index out of range")
	}
	cookiename := cj.cookies[index].Name
	SetDOMCookie(cookiename, "")
	cj.cookies = append(cj.cookies[:index], cj.cookies[index+1:]...)
	return nil
}

func (cj *CookieJar) DelCookie(name string) error {
	cj.mut.Lock()
	defer cj.mut.Unlock()
	for i, cookie := range cj.cookies {
		if cookie.Name == name {
			SetDOMCookie(name, "")
			cj.cookies = append(cj.cookies[:i], cj.cookies[i+1:]...)
			return nil
		}
	}
	return errors.New("cookie not found")
}

func (cj *CookieJar) SetCookie(cookie *Cookie) error {
	if !cookie.IsValid() {
		return errors.New("cookie is invalid")
	}
	cj.mut.Lock()
	defer cj.mut.Unlock()
	cj.cookies = append(cj.cookies, cookie)
	SetDOMCookie(cookie.Name, cookie.Value)
	return nil
}
