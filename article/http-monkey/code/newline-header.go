package main

import (
	"fmt"
	"net/http"
)

func main() {

	req, err := http.NewRequest(http.MethodGet, "https://sven.wiltink.dev", nil)
	if err != nil {
		panic(err)
	}

	req.Header["SomeHeader"] = []string{"SomeValue\nOtherHeader: OtherValue"}

	_, err = http.DefaultClient.Do(req)
	fmt.Println(err)
}
