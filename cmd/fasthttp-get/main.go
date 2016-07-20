// MAIN.GO
// Provides a command-line interface for the fasthttp library.
//
// Copyright 2016 Deluxe Media
// Author: James Welch

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"net/url"
	
	"github.com/jameswelch2-bydeluxe/fasthttp"
)

func main() {
	// Let go parse the command line
	const (
		threadDefault	=	1
		threadUsage		=	"number of threads to use (positive integer < 256)"
	)
	var threads uint64
	flag.Uint64Var(&threads, "threads", threadDefault, threadUsage)
	flag.Uint64Var(&threads, "t", threadDefault, threadUsage)
	flag.Parse()

	// Do we have the correct number of arguments?
	args := flag.Args()
	if (len(args) < 1) || (len(args) > 2) {
		log.Fatalln("Expected two args: URL and file path.")
	}
	
	// Is the thread count valid?
	if threads > 255 {
		log.Fatalf("Invalid thread count: %d is not a byte value.\n", threads)
	}
	
	// The first argument should be a valid URL
	u, err := url.Parse(args[0])
	if err != nil {
		log.Fatalf("\"%s\" is not a valid URL.\n", args[0])
	}
	
	// Okay, now actually make the call to the library
	if len(args) == 1 {
		// No output path specified, so let's use binary mode, and dump
		// to standard out.
		var bytes []byte
		bytes, err = fasthttp.Get(u, byte(threads))
		fmt.Printf("%s\n", bytes)
	} else {
		// Output pth is prsent, so let's use file mode.
		err = fasthttp.Save(u, args[1], byte(threads))
	}

	if err != nil {
		log.Fatalln(err)
	}
	
	// We're done.
	log.Println("Success!")
	os.Exit(0)
}