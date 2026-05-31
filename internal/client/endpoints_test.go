package client

import (
	"errors"
	"testing"
)

func TestResolveEndpoint_Aliases(t *testing.T) {
	cases := map[string]string{
		"api":    EndpointAPI,
		"bing":   EndpointBing,
		"yandex": EndpointYandex,
		"naver":  EndpointNaver,
		"seznam": EndpointSeznam,
		"yep":    EndpointYep,
		"API":    EndpointAPI, // case-insensitive
		"Bing":   EndpointBing,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := ResolveEndpoint(in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestResolveEndpoint_PassthroughURL(t *testing.T) {
	cases := []string{
		"https://custom.example.com/indexnow",
		"http://localhost:8080/indexnow",
		"https://api.indexnow.org/indexnow",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := ResolveEndpoint(in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != in {
				t.Fatalf("got %q, want %q (passthrough)", got, in)
			}
		})
	}
}

func TestResolveEndpoint_Unknown(t *testing.T) {
	for _, in := range []string{"", "google", "ya.ru", "ftp://foo/bar"} {
		t.Run(in, func(t *testing.T) {
			_, err := ResolveEndpoint(in)
			if !errors.Is(err, ErrUnknownEndpoint) {
				t.Fatalf("got err=%v, want ErrUnknownEndpoint", err)
			}
		})
	}
}
