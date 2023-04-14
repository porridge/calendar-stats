package auth

// This file contains functions originally copied from
// https://github.com/googleapis/google-api-go-client/blob/be028cf5a1764737b56c22aeb484b780c06322a4/examples/main.go
// with further modifications to integrate them into this program.

// Copyright 2011 Google LLC. All rights reserved.
// Copyright 2023 Marcin Owsiany <marcin@owsiany.pl>
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-google-api-go-client.txt file.
import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"

	"github.com/xyproto/randomstring"
	"golang.org/x/oauth2"
)

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := randomstring.CookieFriendlyString(32)
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		log.Printf("no code")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return token
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser, please open it manually.")
}
