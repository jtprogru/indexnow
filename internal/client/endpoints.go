package client

import "strings"

// Known IndexNow endpoints. A submission to any one of them is shared
// with every other participating engine per the protocol spec.
const (
	EndpointAPI    = "https://api.indexnow.org/indexnow"
	EndpointBing   = "https://www.bing.com/indexnow"
	EndpointYandex = "https://yandex.com/indexnow"
	EndpointNaver  = "https://searchadvisor.naver.com/indexnow"
	EndpointSeznam = "https://search.seznam.cz/indexnow"
	EndpointYep    = "https://indexnow.yep.com/indexnow"
)

var endpointAliases = map[string]string{
	"api":    EndpointAPI,
	"bing":   EndpointBing,
	"yandex": EndpointYandex,
	"naver":  EndpointNaver,
	"seznam": EndpointSeznam,
	"yep":    EndpointYep,
}

// ResolveEndpoint maps a short alias (api, bing, yandex, naver, seznam, yep)
// to its full URL. If the input already looks like an absolute http(s) URL
// it is returned as-is. Unknown aliases return ErrUnknownEndpoint.
func ResolveEndpoint(nameOrURL string) (string, error) {
	if nameOrURL == "" {
		return "", ErrUnknownEndpoint
	}
	if strings.HasPrefix(nameOrURL, "http://") || strings.HasPrefix(nameOrURL, "https://") {
		return nameOrURL, nil
	}
	if ep, ok := endpointAliases[strings.ToLower(nameOrURL)]; ok {
		return ep, nil
	}
	return "", ErrUnknownEndpoint
}
