// GET_FASTHTTP.GO
// Provides a go-getter driver wrapper for the fasthttp library.
//
// Copyright 2016 Deluxe Media
// Author: James Welch

package getter

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	
	"github.com/jameswelch2-bydeluxe/fasthttp"
)

type FastHttpGetter struct{}

func (g *FastHttpGetter) Get(dst string, u *url.URL) error {
	// Check the fragment of the URL. Was the number of connections
	// specified? (That should be a byte value > 1.)
	threads, err := strconv.ParseUint(u.Fragment, 10, 8);
	if err != nil {
		threads = 1
	}
	
	// Is threads a byte value? If not, we need to ignoreit.
	if threads > 255 {
		threads = 1
	}
	
	// Does the dst end with a "/"? If not, we need to add it.
	if !strings.HasSuffix(dst, "/") {
		dst += "/"
	}
	
	// Is the URL a directory path? If not, just pass this along to
	// GetFile.
	if !strings.HasSuffix(u.Path, "/") {
		return g.GetFile(dst + u.Path[strings.LastIndex(u.Path, "/") + 1:len(u.Path)], u)
	}
	
	// Okay, let's get the directory listing, then
	index, _ := fasthttp.Get(u, byte(threads))
	if err != nil {
		return err
	}
	
	// Now let's fiter out the hrefs...
	re := regexp.MustCompile("\\shref=\\\".+\\\"")
	hrefs := re.FindAllString(string(index), -1)
	
	// ...then interate them to dispatch the downloads.
	for _, href := range hrefs {
		// Trim off the leftovers from the regular expression search.
		href = href[7:len(href) - 1]
		
		// Only if its a relative URL do we work with it.
		if (!strings.Contains(href, ":")) && (!strings.HasPrefix(href, "/")) {
			// Copy our base URL so we can append the href to it.
			v := *u
			v.Path += href
			
			// Is this a directory? If so, we need to add it to the
			// destinaton path.
			if strings.HasSuffix(href, "/") {
				dst += href
			}
			
			// Recurse into this method
			err = g.Get(dst, &v)
			if err != nil {
				return err	
			}
		}
	}

	// If we reached here, we're done. Let's exit successfully.
	return nil
}

func (g *FastHttpGetter) GetFile(dst string, u *url.URL) error {
	// Check the fragment of the URL. Was the number of connections
	// specified? (That should be a byte value > 1.)
	threads, err := strconv.ParseUint(u.Fragment, 10, 8);
	if err != nil {
		threads = 1
	}
	
	// Is threads a byte value? If not, we need to ignoreit.
	if threads > 255 {
		threads = 1
	}
	
	// Okay, now actually make the call to the library.
	return fasthttp.Save(u, dst, byte(threads))
}