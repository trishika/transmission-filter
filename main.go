package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

import "github.com/jessevdk/go-flags"
import "github.com/trishika/transmission-go"

type Options struct {
	Out string `short:"o" long:"out" description:"Output directory" default:"."`
	Url string `short:"u" long:"url" description:"Transmission url" default:"127.0.0.1:9091"`
}

var options Options

var parser = flags.NewParser(&options, flags.Default)

var IGNORE = [...]string{"the", "and", "an", "a"}

var EXTENSION = [...]string{".mp4", ".mkv", ".avi"}

func contains(slice []string, search string) bool {
	for _, value := range slice {
		if value == search {
			return true
		}
	}
	return false
}

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
				if !contains(IGNORE[:], ss) {
					if strings.Contains(torrentName, ss) {
						count = count + 1
					} else {
						count = -1
						break
					}
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

	max_count := 0
	max_name := ""

	for name, count := range match {
		if count > max_count {
			max_count = count
			max_name = name
		}
	}

	fmt.Printf("Found match %s", max_name)
	fmt.Println("")

	return max_name, nil
}

func moveFile(file string, to string) error {
	if contains(EXTENSION[:], path.Ext(file)) {
		fmt.Printf("Moving %s to %s\n", file, to)
		return os.Rename(file, path.Join(options.Out, to, path.Base(file)))
	}
	fmt.Printf("Skipping %s\n", file)
	return nil
}

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

	fmt.Println("Filtering...")

	conf := transmission.Config{
		Address: fmt.Sprintf("http://%s/transmission/rpc", options.Url),
	}
	t, err := transmission.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	torrents, err := t.GetTorrents()
	if err != err {
		log.Fatal(err)
	}

	fmt.Printf("%d torrents found\n", len(torrents))
	for _, torrent := range torrents {
		if torrent.Status == 0 || torrent.Status == 6 { // Finished or seeding
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
				fmt.Printf("Error moving file from torrent %s", torrent.Name)
				continue
			}
			t.RemoveTorrents([]*transmission.Torrent{torrent}, true)
		}
	}
}
