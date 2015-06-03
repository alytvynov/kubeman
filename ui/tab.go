package ui

import (
	"reflect"
	"sort"
	"sync"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
	"github.com/alytvynov/kubeman/client"
	"github.com/gizak/termui"
)

type tab interface {
	dataUpdate(Event)
	uiUpdate(*client.Client, termui.Event)
	toRows() []*termui.Row
}

type listTab struct {
	mu       *sync.Mutex
	headers  []header
	items    []listItem
	itemType reflect.Type
	selected int
}

type header struct {
	name string
	span int
}

func (h header) build() *termui.Row {
	l := label(h.name)
	l.TextFgColor = termui.ColorWhite | termui.AttrBold
	l.PaddingLeft = 1
	return termui.NewCol(h.span, 0, l)
}

type listItem interface {
	toRows() []*termui.Row
	setData(interface{})
	sameData(interface{}) bool
	less(listItem) bool
	handleEvent(*client.Client, termui.Event)
}

func (pl *listTab) Len() int           { return len(pl.items) }
func (pl *listTab) Less(i, j int) bool { return pl.items[i].less(pl.items[j]) }
func (pl *listTab) Swap(i, j int)      { pl.items[i], pl.items[j] = pl.items[j], pl.items[i] }

func (pl *listTab) uiUpdate(c *client.Client, e termui.Event) {
	switch e.Type {
	case termui.EventKey:
		switch e.Key {
		case termui.KeyArrowDown:
			pl.mu.Lock()
			pl.selected++
			if pl.selected >= len(pl.items) {
				pl.selected = len(pl.items) - 1
			}
			pl.mu.Unlock()
			return
		case termui.KeyArrowUp:
			pl.mu.Lock()
			pl.selected--
			if pl.selected < 0 {
				pl.selected = 0
			}
			pl.mu.Unlock()
			return
		}
	}
	if len(pl.items) > 0 {
		pl.items[pl.selected].handleEvent(c, e)
	}
}

func (pl *listTab) dataUpdate(e Event) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	var existing listItem
	var existingi int
	for i, li := range pl.items {
		if li.sameData(e.Data) {
			existing = li
			existingi = i
			break
		}
	}
	switch e.Type {
	case watch.Added, watch.Modified:
		if existing != nil {
			existing.setData(e.Data)
		} else {
			item := reflect.New(pl.itemType).Interface().(listItem)
			item.setData(e.Data)
			pl.items = append(pl.items, item)
		}
	case watch.Deleted:
		if existing != nil {
			pl.Swap(existingi, pl.Len()-1)
			pl.items = pl.items[:pl.Len()-1]
		}
	}
	sort.Sort(pl)
}

func (pl *listTab) toRows() []*termui.Row {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	rows := make([]*termui.Row, 0)

	headerTmps := make([]*termui.Row, 0, len(pl.headers))
	for _, h := range pl.headers {
		headerTmps = append(headerTmps, h.build())
	}

	rows = append(rows, termui.NewRow(headerTmps...))
	for i, p := range pl.items {
		row := p.toRows()
		for _, r := range row {
			for _, c := range r.Cols {
				if p, ok := c.Widget.(*termui.Par); ok {
					// simulate padding without loosing bg color
					p.Text = " " + p.Text
					if i == pl.selected {
						p.BgColor = termui.ColorCyan
						p.TextFgColor = termui.ColorBlack
						p.TextBgColor = termui.ColorCyan
					}
				}
			}
		}
		rows = append(rows, row...)
	}
	return rows
}
