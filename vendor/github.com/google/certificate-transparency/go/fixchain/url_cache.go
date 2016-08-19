package fixchain

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type urlCache struct {
	client *http.Client
	cache  map[string][]byte
	// counters may not be totally accurate due to non-atomicity
	hit       uint
	miss      uint
	errors    uint
	badStatus uint
	readFail  uint
}

func (u *urlCache) getURL(url string) ([]byte, error) {
	r, ok := u.cache[url]
	if ok {
		u.hit++
		return r, nil
	}
	c, err := u.client.Get(url)
	if err != nil {
		u.errors++
		return nil, err
	}
	defer c.Body.Close()
	// TODO(katjoyce): Add caching of permanent errors.
	if c.StatusCode != 200 {
		u.badStatus++
		return nil, fmt.Errorf("can't deal with status %d", c.StatusCode)
	}
	r, err = ioutil.ReadAll(c.Body)
	if err != nil {
		u.readFail++
		return nil, err
	}
	u.miss++
	u.cache[url] = r
	return r, nil
}

func newURLCache(c *http.Client, logStats bool) *urlCache {
	u := &urlCache{cache: make(map[string][]byte), client: c}

	if logStats {
		t := time.NewTicker(time.Second)
		go func() {
			for _ = range t.C {
				log.Printf("cache: %d hits, %d misses, %d errors, "+
					"%d bad status, %d read fail, %d cached", u.hit,
					u.miss, u.errors, u.badStatus, u.readFail,
					len(u.cache))
			}
		}()
	}

	return u
}
