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

func TestResolveEndpoints_Single(t *testing.T) {
	got, err := ResolveEndpoints("bing")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != EndpointBing {
		t.Fatalf("got %v", got)
	}
}

func TestResolveEndpoints_MultiPreserveOrder(t *testing.T) {
	got, err := ResolveEndpoints("yandex, bing , naver")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{EndpointYandex, EndpointBing, EndpointNaver}
	if !equalStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveEndpoints_Dedup(t *testing.T) {
	got, err := ResolveEndpoints("bing,Bing,bing")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != EndpointBing {
		t.Fatalf("got %v", got)
	}
}

func TestResolveEndpoints_MixedAliasAndURL(t *testing.T) {
	got, err := ResolveEndpoints("bing,https://custom.example.com/indexnow")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{EndpointBing, "https://custom.example.com/indexnow"}
	if !equalStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveEndpoints_UnknownItemFails(t *testing.T) {
	if _, err := ResolveEndpoints("bing,google"); !errors.Is(err, ErrUnknownEndpoint) {
		t.Fatalf("got %v, want ErrUnknownEndpoint", err)
	}
}

func TestResolveEndpoints_BlankSpec(t *testing.T) {
	for _, spec := range []string{"", " , , "} {
		if _, err := ResolveEndpoints(spec); !errors.Is(err, ErrUnknownEndpoint) {
			t.Fatalf("spec %q: got %v, want ErrUnknownEndpoint", spec, err)
		}
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
