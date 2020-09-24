package main

import (
	"bou.ke/monkey"
	"fmt"
	"net/http"
	"reflect"
	_ "unsafe"
)

func main() {

	// disable validation
	monkey.Patch(linknameMagic, func(s string) bool {
		fmt.Println("the patched function was called!")
		return true
	})

	fmt.Printf("pointer in main: %d\n", reflect.ValueOf(linknameMagic).Pointer())

	req, err := http.NewRequest(http.MethodGet, "https://sven.wiltink.dev", nil)
	if err != nil {
		panic(err)
	}

	req.Header["SomeHeader"] = []string{"SomeValue\nOtherHeader: OtherValue"}

	_, err = http.DefaultClient.Do(req)
	fmt.Println(err)
}

//go:linkname linknameMagic vendor/golang.org/x/net/http/httpguts.ValidHeaderFieldValue
func linknameMagic(a string) bool