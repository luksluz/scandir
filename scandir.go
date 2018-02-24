package scanDir

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// DBNAME ...
	DBNAME string = "data.b"
	// EVFILEADDED ...
	EVFILEADDED = iota
	// EVFILEREMOVED ...
	EVFILEREMOVED
)

var (
	// Wts ...
	Wts Watchers
	wg  sync.WaitGroup
)

// Watchers ...
type Watchers struct {
	Watchers map[string]Watcher
	handler  handler
}

// Watcher ...
type Watcher struct {
	Path  string
	Files []File
}

// File ...
type File struct {
	Name string
	Tags []string
	Seen bool
}

// Events ...
type Events struct {
	EvType int
	Path   string
	FileEv *File
}

type handler func(Events)
type files []string

// New ...
func New(handlerFunc handler, dirs ...string) {
	Wts.handler = handlerFunc
	Wts.Watchers = make(map[string]Watcher)
	for _, dir := range dirs {
		if (strings.Compare(dir, ".")) == 0 {
			dir, _ = os.Getwd()
		}
		Wts.Watchers[dir] = Watcher{Path: dir}
	}
}

// Scan ...
func Scan() {
	wg.Add(1)
	go func() {
		_, err := os.Stat(DBNAME)

		if !os.IsNotExist(err) {
			bts, _ := ioutil.ReadFile(DBNAME)
			dec := gob.NewDecoder(bytes.NewBuffer(bts))
			dec.Decode(&Wts)
		}

		for _, w := range Wts.Watchers {
			wg.Add(1)
			go start(w)
		}
		wg.Wait()
		defer wg.Done()
	}()
}

// start ...
func start(w Watcher) {
	for {
		var cFiles []string
		changed := false

		fs, err := ioutil.ReadDir(w.Path)
		if err != nil {
			log.Println(err)
		}

		for _, f := range fs {
			if !f.IsDir() {
				cFiles = append(cFiles, f.Name())
			}
		}

		for _, fileName := range cFiles {
			if !w.fileScanned(fileName) {
				changed = true
				fileAdded := File{Name: fileName, Seen: false}
				w.Files = append(w.Files, fileAdded)
				ev := Wts.handler
				ev(Events{EvType: EVFILEADDED, Path: w.Path, FileEv: &fileAdded})
				// log.Println("[+]", fileName)

			}
		}

		for index, file := range w.Files {
			if w.fileToRemove(file.Name, cFiles) {
				changed = true
				w.Files = append(w.Files[:index], w.Files[index+1:]...)
				ev := Wts.handler
				ev(Events{EvType: EVFILEREMOVED, Path: w.Path, FileEv: &file})
				// log.Println("[-]", fileName)
			}
		}

		if changed {
			var bf bytes.Buffer
			Wts.Watchers[w.Path] = w
			gob.NewEncoder(&bf).Encode(Wts)
			err := ioutil.WriteFile(DBNAME, bf.Bytes(), 0666)
			if err != nil {
				panic(err)
			}
		}
		time.Sleep(time.Second * 3)
	}
}

func (w Watcher) fileScanned(fileName string) bool {
	for _, key := range w.Files {
		if key.Name == fileName {
			return true
		}
	}
	return false
}

func (w Watcher) fileToRemove(fileName string, cFiles []string) bool {
	for _, key := range cFiles {
		if key == fileName {
			return false
		}
	}
	return true
}

// ShowAllFiles ...
func ShowAllFiles() Watchers {
	return Wts
}

// Wait ...
func Wait() {
	wg.Wait()
}
