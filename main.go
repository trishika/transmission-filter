/*
 * Copyright (C) 2018 Aurélien Chabot <aurelien@chabot.fr>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"unicode/utf8"
)

import "github.com/jessevdk/go-flags"
import "github.com/trishika/transmission-go"

type Options struct {
	Out string `short:"o" long:"out" description:"Output directory" default:"."`
	URL string `short:"u" long:"url" description:"Transmission url" default:"127.0.0.1:9091"`
	Ext string `short:"e" long:"extension" description:"File extension to filter" default:"mp4,mkv,avi,srt,mp3,ogg"`
}

var options Options

var parser = flags.NewParser(&options, flags.Default)

var extensions []string

// contains test if an array of string contains a string
func contains(slice []string, search string) bool {
	for _, value := range slice {
		if value == search {
			return true
		}
	}
	return false
}

// splitter split a string using multiple delimiter
func splitter(s string, splits string) []string {
	m := make(map[rune]int)
	for _, r := range splits {
		m[r] = 1
	}

	splitter := func(r rune) bool {
		return m[r] == 1
	}

	return strings.FieldsFunc(s, splitter)
}

// findMatch find the match for a torrent string name inside the directory
// folder name.
// The folder name substrings must all be present in the torrent name, except
// substring inside brackets. If there is multiple match the one with more
// substring will be selected.
func findMatch(torrent string) (string, error) {
	torrentName := strings.ToLower(torrent)

	files, err := ioutil.ReadDir(options.Out)
	if err != nil {
		log.Fatal(err)
	}

	match := make(map[string]int)

	for _, f := range files {
		if f.IsDir() {
			s := splitter(strings.ToLower(f.Name()), ". ")
			count := 0
			for _, ss := range s {
				// Skip what is inside brackets
				if r, _ := utf8.DecodeRuneInString(ss); r == '(' {
					continue
				}
				// All other substring need to be in the torrent name
				if strings.Contains(torrentName, ss) {
					count = count + 1
				} else {
					count = -1
					break
				}
			}
			if count > 0 {
				match[f.Name()] = count
			}
		}
	}

	if len(match) == 0 {
		return "", fmt.Errorf("no match found for %s", torrent)
	}

	maxCount := 0
	maxName := ""

	// If there is multiple match select the longest name
	for name, count := range match {
		if count > maxCount {
			maxCount = count
			maxName = name
		}
	}

	fmt.Printf("Found match %s\n", maxName)

	return maxName, nil
}

// moveFile move a file to the given directory
func moveFile(file string, to string) error {
	ext := strings.ToLower(path.Ext(file))
	if len(ext) > 0 && contains(extensions[:], ext[1:]) {
		fmt.Printf("Moving %s to %s\n", file, to)
		return os.Rename(file, path.Join(options.Out, to, path.Base(file)))
	}
	fmt.Printf("Skipping %s\n", file)
	return nil
}

// move move all file from a torrent matching the filter to the given directory
func move(torrentDir string, to string) error {
	f, err := os.Stat(torrentDir)
	if err != nil {
		log.Fatal(err)
	}
	if f.Mode().IsDir() {
		files, err := ioutil.ReadDir(torrentDir)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range files {
			err := moveFile(path.Join(torrentDir, f.Name()), to)
			if err != nil {
				return err
			}
		}
	} else {
		err := moveFile(torrentDir, to)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	extensions = strings.Split(options.Ext, ",")

	fmt.Println("Filtering...")

	// Create the transmission connection object
	conf := transmission.Config{
		Address: fmt.Sprintf("http://%s/transmission/rpc", options.URL),
	}

	// Connect to transmission
	t, err := transmission.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Get all torrent
	torrents, err := t.GetTorrents()
	if err != err {
		log.Fatal(err)
	}

	fmt.Printf("%d torrents found\n", len(torrents))
	for _, torrent := range torrents {
		// Filter only finished or seeding torrent
		if torrent.Status == 0 || torrent.Status == 6 {
			fmt.Println()
			fmt.Printf("Torrent : %s\n", torrent.Name)
			match, err := findMatch(torrent.Name)
			if err != nil {
				fmt.Println(err)
				continue
			}
			torrentDir := path.Join(torrent.DownloadDir, torrent.Name)
			err = move(torrentDir, match)
			if err != nil {
				fmt.Printf("Error moving file from torrent %s\n", torrent.Name)
				continue
			}
			t.RemoveTorrents([]*transmission.Torrent{torrent}, true)
		}
	}
}
