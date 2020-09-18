package main

import (
	"bou.ke/monkey"
	"fmt"
	"golang.org/x/net/http/httpguts"
	"net/http"
)

func main() {
	// disable validation
	monkey.Patch(httpguts.ValidHeaderFieldValue, func(s string) bool {
		return true
	})

	req, err := http.NewRequest(http.MethodGet, "https://sven.wiltink.dev", nil)
	if err != nil {
		panic(err)
	}

	req.Header["SomeHeader"] = []string{"SomeValue\nOtherHeader: OtherValue"}

	_, err = http.DefaultClient.Do(req)
	fmt.Println(err)
}
