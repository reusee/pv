package main

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/reusee/jsonfile"
	"github.com/reusee/lgtk"
)

var (
	p = fmt.Printf
)

func main() {
	root := "."
	args := os.Args
	if len(os.Args) > 1 {
		root = os.Args[1]
		args = args[1:]
	}
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		log.Fatalf("invalid path %v", err)
	}

	var newOnly bool
	for _, arg := range args {
		switch arg {
		case "new":
			newOnly = true
		}
	}

	var images []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		what := mime.TypeByExtension(filepath.Ext(path))
		if !strings.HasPrefix(what, "image/") {
			return nil
		}
		images = append(images, path)
		return nil
	})
	if len(images) == 0 {
		p("no image.\n")
		return
	}

	sort.Sort(sort.Reverse(sort.StringSlice(images)))

	data := &struct {
		Count map[string]int
	}{
		Count: make(map[string]int),
	}

	dbPath := filepath.Join(root, ".picture_viewer.json")
	file, err := jsonfile.New(&data, dbPath, 51294)
	if err != nil {
		log.Fatalf("open data file %v", err)
	}
	defer file.Save()

	var filtered []string
	for _, img := range images {
		if newOnly {
			if data.Count[img] > 0 {
				continue
			}
		}
		filtered = append(filtered, img)
	}
	images = filtered

	keys := make(chan rune)
	g, err := lgtk.New(`
GdkPixbuf = lgi.GdkPixbuf

win = Gtk.Window{
	Gtk.Grid{
		expand = true,
		orientation = 'VERTICAL',
		Gtk.Label{
			id = 'filename',
		},
		Gtk.ScrolledWindow{
			id = 'scroll',
			Gtk.Image{
				id = 'img',
				expand = true,
			},
			expand = true,
		},
	},
}
win.on_destroy:connect(Gtk.main_quit)
win.on_key_press_event:connect(function(_, ev)
	key_press(ev.keyval)
	return true
end)

win:show_all()
	`,
		"key_press", func(k rune) {
			select {
			case keys <- k:
			default:
			}
		})
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()

	index := 0
	showImage := func() {
		g.ExecEval(`
print(F)
buf, err = GdkPixbuf.Pixbuf.new_from_file(F)
win.child.img:set_from_pixbuf(buf)
win.child.scroll.vadjustment:set_value(0)
win.child.filename:set_label(F)
`,
			"F", images[index])
	}
	showImage()

loop:
	for key := range keys {
		switch key {
		case 'q':
			break loop
		case ' ':
			if index == len(images)-1 {
				continue loop
			}
			data.Count[images[index]]++
			index++
			showImage()
			time.Sleep(time.Millisecond * 500)
		case 'z':
			if index == 0 {
				continue loop
			}
			index--
			showImage()
		}
	}
}
