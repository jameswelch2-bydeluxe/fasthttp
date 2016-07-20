// FASTHTTP.GO
// Performs HTTP get operations using parallel threads working on
// ranged portions of the resouce, to increase network saturation,
// and increase speed of large file tranfers.
//
// Copyright 2016 Deluxe Media
// Author: James Welch

package fasthttp

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

// Perform a threaded HTTP GET on the input UEL, using the specified
// number of threads, and return the reponse data as a slice of bytes.
// Useful for embedding in an application workflow, but limited by the
// valid size of a byte slice, and avilable system memory resources.
func Get(u *url.URL, threads byte) ([]byte, error) {
	// Calculate the content length
	length, err := getContentLength(u)
	if err != nil {
		return nil, err
	}
	
	// Create the "file" we're going to catch the response into
	var f bufferWriterAt
	if(length == 0) {
		// Okay, no size. So we create the highest reasonable capcity.
		f.buffer = make([]byte, 0, math.MaxInt32)
	} else {
		// Better. Let's create a real-sized buffer.
		f.buffer = make([]byte, length, length)
	}
	
	// Actually perform the download.
	err = download(&f, u, threads, length)
	
	return f.buffer, err
}

// Perform a threaded HTTP GET on the input UEL, using the specified
// number of threads, and save the results to a file at the specified
// path. Allows large file downloads, without being limited by the
// golang slice limitations, but necessitates use of a temp file if
// the intended target is not a file.
func Save(u *url.URL, path string, threads byte) error {
	// Calculate the content length
	length, err := getContentLength(u)
	if err != nil {
		return err
	}
	
	// Let's prepare a file to be written to. We'll need to create any
	// parent directories first.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	
	// Now we can actually open the file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	// Actually perform the download.
	err = download(f, u, threads, length)
	return err
}

// Perform a head request to get the size, in bytes, of the resource.
func getContentLength(u *url.URL) (int64, error) {
	// Make the head request.
	resp, err := http.Head(u.String())
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	
	// Make sure the response is good.
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("bad response code: %d", resp.StatusCode)
	}
	
	// If no range request is accepted, no point in going further
	// Warning: According to the HTTP spec, a server doesn't *have* to
	// report this. But we need to know for sure for the rest of the
	// library to work properly.
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return 0, nil
	}
	
	// If cntent length wasn't returned, no point in going further.
	if resp.ContentLength < 0 {
		return 0, nil
	}

	// Okay, let's just report what we got then.
	return resp.ContentLength, nil
}

// This is a stream that targets a buffer (so that we can return a
// buffer for the Get() method using the same logic as the Save()
// method).
type bufferWriterAt struct{
	buffer []byte
}

// Our stream needs to export the WriteAt if it is going to satisfy
// the io.WriterAt inetrface.
func (w *bufferWriterAt) WriteAt(p []byte, off int64) (int, error) {
	// First, make sure the requested fits within the slice.
	if (off < 0) || (len(p) + int(off) > len(w.buffer)) {
		w.buffer = w.buffer[0:len(p) + int(off)]
	}

	// Do a simple copy loop, incrementing the offset as we go.
	for _, b := range p {
		w.buffer[off] = b
		off++
	}
	return len(p), nil
}

func download(f io.WriterAt, u *url.URL, threads byte, length int64) error {
	// They did specify to actually use threads, yes?
	if threads == 0 {
		return fmt.Errorf("cannot download \"%s\" download using %d threads", u.String(), threads)
	}
	
	// If the file size is smaller than the number of threads
	// specified, then just use one. This will also handle cases where
	// length is zero or unknown, or where range requests aren't
	// supported.
	if length < int64(threads) {
		threads = 1
	}
	
	// If we're only using one thread, then a range request isn't
	// necessary.
	if threads == 1 {
		return getRange(f, u, 0, (-1))
	}
	
	// Okay, this is our job. Now we need to find our transfer size per
	// connection. This probably won't divide evenly, so we have to
	// track the modulo also, so we can have the first connection grab
	// the leftover bytes (which should be a very small number anyway.)
	blocksize := length / int64(threads)
	remainder := length % int64(threads)
	
	// It's spawing time. We'll need a channel and wait group to know
	// if/when the transfers are done, and catch any errors that might
	// have occuured.
	errors := make(chan error)
    var wg sync.WaitGroup
	
	// We'll need to track our range offset as we spawn the
	// downloaders.
	offset := int64(0)
	
	// Spawn each of the download threads, each having a specific
	// range (that was caculated above.)
	for offset < length {	
			start := offset
			end := start + blocksize + remainder - 1
			wg.Add(1)
		    go func() {
		        defer wg.Done()
		        errors <- getRange(f, u, start, end)
		    }()
			offset = end + 1
			remainder = 0
	}
	
	// We need to listen for errors.
	var err error
    go func() {
        for e := range errors {
            err = e
        }
    }()
	
	// Now we wait for the downloaders to exit and we know we're done
	wg.Wait()
	return err
}

// Perform a range request, of zero-indexed bytes start through end. 
// Write the response bytes into the corresponding offset of the 
// WriteAt stream. *Special Case: If end is before start, don't
// perform a range request. Just perform a standard get instead.
func getRange(f io.WriterAt, u *url.URL, start int64, end int64) error {
	// Turn the request arguments into a proper HTTP request
	req, _ := http.NewRequest("GET", u.String(), nil)
	
	// If the end is less than start, this is not a range request
	if end >= start {
    	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	}

	// Make the request.
    var client http.Client
    resp, err := client.Do(req)
	if err != nil {
		return err
	}
	
	defer resp.Body.Close()
	
	// Is the response good?
	if end < start {
		// This was NOT a range request.
		if resp.StatusCode != 200 {
			return fmt.Errorf("bad response code: %d", resp.StatusCode)
		}
	} else {
		// This was a range request
		if resp.StatusCode != 206 {
			return fmt.Errorf("bad response code: %d while reading bytes %d through %d", resp.StatusCode, start, end)
		}
	}
  
	// Setup buffers for data transfer. We're using a small buffer
	// because we can't make assumptions about block size on the host,
	// and we want the write loop to iterate as quickly as reasonable.
	payload := make([]byte, 256)
	var eof error
	for eof == nil {
		// Get the payload...
		var bytes int
		bytes, eof = resp.Body.Read(payload)
		
		// ... and write it to the stream.
		f.WriteAt(payload[0:bytes], start)
		
		// Update our position.
		start += int64(bytes)
	}

	return nil
}
