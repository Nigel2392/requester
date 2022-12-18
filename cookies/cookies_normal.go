//go:build !js && !wasm
// +build !js,!wasm

package cookies

import "errors"

func (cj *CookieJar) DelCookieIndex(index int) error {
	cj.mut.Lock()
	defer cj.mut.Unlock()
	if index < 0 || index >= len(cj.cookies) {
		return errors.New("index out of range")
	}
	cj.cookies = append(cj.cookies[:index], cj.cookies[index+1:]...)
	return nil
}

func (cj *CookieJar) DelCookie(name string) error {
	cj.mut.Lock()
	defer cj.mut.Unlock()
	for i, cookie := range cj.cookies {
		if cookie.Name == name {
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
	return nil
}
