package tui

type screen uint8

const (
	home screen = iota
	createProject
	openProject
	pluginMarketplace
	settings
)

type menuItem struct {
	label  string
	screen screen
}

var homeMenu = []menuItem{
	{label: "Create Project", screen: createProject},
	{label: "Open Existing Project", screen: openProject},
	{label: "Plugin Marketplace", screen: pluginMarketplace},
	{label: "Settings", screen: settings},
}

type navigation struct {
	current  screen
	selected int
	history  []screen
}

func (n *navigation) move(delta int) {
	if n.current != home || len(homeMenu) == 0 {
		return
	}
	n.selected = (n.selected + delta + len(homeMenu)) % len(homeMenu)
}

func (n *navigation) selectItem() {
	if n.current != home {
		return
	}
	n.history = append(n.history, n.current)
	n.current = homeMenu[n.selected].screen
}

func (n *navigation) back() {
	if len(n.history) == 0 {
		return
	}
	last := len(n.history) - 1
	n.current = n.history[last]
	n.history = n.history[:last]
}
