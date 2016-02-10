# [Base62.go](http://libraries.io/go/github.com%2Fandrew%2Fbase62.go)

* *Original author*: Andrew Nesbitt
* *Forked from*: https://github.com/andrew/base62.go

An attempt at a go library to provide Base62 encoding, perfect for URL safe values, and url shorteners.


```go

	// Encode a value
	urlVal := "http://www.biglongurl.io/?utm_content=content&utm_campaign=campaign"
	encodedUrl := base62.StdEncoding.EncodeToString([]byte(urlVal))
	

	// Unencoded it
	byteUrl, err := base62.StdEncoding.DecodeString(encodedUrl)


```