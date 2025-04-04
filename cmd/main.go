package main

import (
	"fmt"
	"os"
	"path/filepath"
	"rtckt/core"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppState struct {
	rootPath        string
	activeDirectory string
	activeFile      string
}

type UIComponents struct {
	app              *tview.Application
	fileTree         *tview.TreeView
	layout           *tview.Flex
	defaultActionBox *tview.TextView
	ticketActionBox  *tview.Flex
	ticketForm       *tview.Form
	newProjForm      *tview.Form
	activeElement    tview.Primitive
	projectOverview  *tview.Flex
}

var (
	defaultStyle      = tcell.Style{}
	headingStyle      = tcell.Style{}.Underline(true)
	fieldStyle        = tcell.Style{}.Underline(true)
	buttonStyle       = tcell.Style{}.Background(tcell.ColorGray)
	buttonActiveStyle = tcell.Style{}.Background(tcell.ColorRed)
	dirTextStyle      = tcell.Style{}.Foreground(tcell.ColorWhiteSmoke)
	dirSelectedStyle  = tcell.Style{}.Underline(true).Foreground(tcell.ColorRed)
	fileTextStyle     = tcell.Style{}.Foreground(tcell.ColorAquaMarine)
	fileSelectedStyle = tcell.Style{}.Underline(true).Foreground(tcell.ColorGreen)
	rootTextStyle     = tcell.Style{}.Foreground(tcell.ColorYellow)
	rootSelectedStyle = tcell.Style{}.Foreground(tcell.ColorYellow)
	focusedCheckStyle = tcell.Style{}.Blink(true)
)

var statusText = map[core.Status]string{
	core.STATUS_OPEN:    "Open",
	core.STATUS_BLOCKED: "Blocked",
	core.STATUS_CLOSED:  "Closed",
}

func main() {
	state := &AppState{}
	ui := &UIComponents{}

	var err error
	state.rootPath, err = core.Setup()
	if err != nil {
		fmt.Println(err)
		return
	}

	initializeUI(state, ui)

	if err := ui.app.Run(); err != nil {
		panic(err)
	}
}

func initializeUI(state *AppState, ui *UIComponents) {
	ui.app = tview.NewApplication()

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	initializeFileTree(state, ui)
	initializeDefaultActionBox(ui)
	initializeTicketActionBox(ui)
	initializeTicketForm(state, ui)
	initializeProjectForm(state, ui)
	initializeProjectOverview(ui)

	ui.layout = tview.NewFlex().AddItem(ui.fileTree, 0, 1, true).AddItem(ui.defaultActionBox, 0, 2, false)

	ui.activeElement = ui.defaultActionBox

	setupKeyBindings(state, ui)

	ui.app.SetRoot(ui.layout, true)
}

func initializeProjectOverview(ui *UIComponents) {
	ui.projectOverview = tview.NewFlex()
	ui.projectOverview.SetBorder(true)
}

func initializeFileTree(state *AppState, ui *UIComponents) {
	rootNode := tview.NewTreeNode("Root Directory").SetReference(state.rootPath)
	rootNode.SetTextStyle(rootTextStyle)
	rootNode.SetSelectedTextStyle(rootSelectedStyle)

	populateFileTree(rootNode, state.rootPath, state.activeDirectory)

	ui.fileTree = tview.NewTreeView().
		SetRoot(rootNode).
		SetCurrentNode(rootNode)

	ui.fileTree.SetBorder(true)

	//on select, used for collapsing
	ui.fileTree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			state.activeDirectory = ""
			return
		}

		path := reference.(string)
		info, err := os.Stat(path)
		if err != nil {
			fmt.Println("Error accessing path:", err)
			return
		}

		if info.IsDir() {
			state.activeFile = ""
			node.SetExpanded(!node.IsExpanded())
		}
	})

	//hovers
	ui.fileTree.SetChangedFunc(func(node *tview.TreeNode) {
		handleNodeHover(node, state, ui)
	})
}

func handleNodeHover(node *tview.TreeNode, state *AppState, ui *UIComponents) {
	reference := node.GetReference()
	if reference == nil {
		return
	}

	path := reference.(string)
	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error accessing path:", err)
		return
	}

	if !info.IsDir() {
		ticket, err := core.GetTicket(path)
		if err != nil {
			fmt.Println(err)
			return
		}

		state.activeFile = path
		SwapToTicketBox(ui, ticket)
	} else {
		state.activeDirectory = path
		if state.activeDirectory == state.rootPath {
			swapToDefaultPane(ui)
			return
		}
		SwapToProjectBox(ui, state)
	}
}

func SwapToTicketBox(ui *UIComponents, ticket core.Ticket) {
	ui.layout.RemoveItem(ui.activeElement)
	ui.ticketActionBox.SetTitle(ticket.Name)
	ui.layout.AddItem(ui.ticketActionBox, 0, 2, false)
	ui.activeElement = ui.ticketActionBox
	populateTicketBox(ui.ticketActionBox, ticket)
}

func SwapToProjectBox(ui *UIComponents, state *AppState) {
	ui.layout.RemoveItem(ui.activeElement)
	ui.projectOverview.SetTitle(state.activeDirectory)
	ui.layout.AddItem(ui.projectOverview, 0, 2, false)
	ui.activeElement = ui.projectOverview
	populateProjectOverview(ui.projectOverview, state)
}

func populateProjectOverview(overview *tview.Flex, state *AppState) {
	overview.Clear().SetDirection(tview.FlexRow)
	all_tickets := core.ReadWholeDirectory(state.activeDirectory)

	var open int
	var blocked int
	var closed int
	var open_names []string
	for _, ticket_path := range all_tickets {
		t, _ := core.GetTicket(ticket_path)
		switch t.Status {
		case core.STATUS_BLOCKED:
			blocked += 1
		case core.STATUS_OPEN:
			open += 1
			open_names = append(open_names, strings.TrimSuffix(filepath.Base(ticket_path), filepath.Ext(ticket_path)))
		case core.STATUS_CLOSED:
			closed += 1
		}
	}
	var completion float32
	if blocked+open+closed > 0 {
		completion = float32(closed) / float32((open + blocked + closed))
	} else {
		completion = 1
	}
	addLabeledField(overview, "Open Tickets:", fmt.Sprintf("%d", open))
	addLabeledField(overview, "Blocked Tickets:", fmt.Sprintf("%d", blocked))
	addLabeledField(overview, "Closed Tickets:", fmt.Sprintf("%d", closed))
	addLabeledField(overview, "Completion %", fmt.Sprintf("%d %%", int32(completion*100)))
	if len(open_names) > 0 {
		overview.AddItem(tview.NewTextView().SetText("Tickets Ready to Work on:").SetTextStyle(headingStyle), 1, 1, false)
	}
	for _, name := range open_names {
		overview.AddItem(tview.NewTextView().SetText(name).SetTextStyle(defaultStyle), 1, 1, false)
	}
}

func initializeDefaultActionBox(ui *UIComponents) {
	ui.defaultActionBox = tview.NewTextView()
	ui.defaultActionBox.SetBorder(true)
	ui.defaultActionBox.SetTextAlign(tview.AlignCenter)
	ui.defaultActionBox.SetText("Select a ticket to view/edit")
}

func initializeTicketActionBox(ui *UIComponents) {
	ui.ticketActionBox = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.ticketActionBox.SetBorder(true)
}

func initializeTicketForm(state *AppState, ui *UIComponents) {
	ui.ticketForm = tview.NewForm()
	populateTicketForm(ui.ticketForm, state.activeDirectory)

	ui.ticketForm.AddButton("Save", func() {
		submitNewTicket(ui.ticketForm, state.activeDirectory)
		swapToDefaultPane(ui)
		refreshTree(ui.fileTree.GetRoot(), state.rootPath, state.activeDirectory)
		ui.app.SetFocus(ui.fileTree)
		resetTicketForm(ui.ticketForm, state.activeDirectory)
	}).SetButtonStyle(buttonStyle).SetButtonActivatedStyle(buttonActiveStyle)

	ui.ticketForm.AddButton("Cancel", func() {
		swapToDefaultPane(ui)
		resetTicketForm(ui.ticketForm, state.activeDirectory)
		ui.app.SetFocus(ui.fileTree)
	}).SetButtonStyle(buttonStyle).SetButtonActivatedStyle(buttonActiveStyle)
}

func initializeProjectForm(state *AppState, ui *UIComponents) {
	ui.newProjForm = tview.NewForm()
	populateProjForm(ui.newProjForm)

	ui.newProjForm.AddButton("Create", func() {
		name := ui.newProjForm.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
		projectPath := state.rootPath
		if state.activeDirectory != "" {
			projectPath = state.activeDirectory
		}

		core.NewProject(filepath.Join(projectPath, name))
		swapToDefaultPane(ui)
		refreshTree(ui.fileTree.GetRoot(), state.rootPath, state.activeDirectory)
		ui.newProjForm.Clear(false)
		populateProjForm(ui.newProjForm)
		ui.app.SetFocus(ui.fileTree)
	}).SetButtonStyle(buttonStyle).SetButtonActivatedStyle(buttonActiveStyle)

	ui.newProjForm.AddButton("Cancel", func() {
		swapToDefaultPane(ui)
		ui.newProjForm.Clear(false)
		populateProjForm(ui.newProjForm)
		ui.app.SetFocus(ui.fileTree)
	}).SetButtonStyle(buttonStyle).SetButtonActivatedStyle(buttonActiveStyle)
}

func setupKeyBindings(state *AppState, ui *UIComponents) {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlO: // New ticket
			handleNewTicket(state, ui)
		case tcell.KeyCtrlD: // Delete ticket
			handleDeleteTicket(state, ui)
		case tcell.KeyCtrlB: // Delete project
			handleDeleteProject(state, ui)
		case tcell.KeyCtrlP: // New project
			handleNewProject(ui)
		case tcell.KeyCtrlR: // Close ticket
			handleCloseTicket(state, ui)
		}
		return event
	})
}

func handleNewTicket(state *AppState, ui *UIComponents) {
	if state.activeDirectory == "" {
		return
	}

	ui.layout.RemoveItem(ui.activeElement)
	ui.layout.AddItem(ui.ticketForm, 0, 2, true)

	for _, file := range getDirectoryFiles(state.activeDirectory) {
		lbl := strings.TrimSuffix(file, filepath.Ext(file))
		ui.ticketForm.AddCheckbox(lbl, false, nil)
		ui.ticketForm.GetFormItemByLabel(lbl).(*tview.Checkbox).
			SetUncheckedString("☐").
			SetActivatedStyle(focusedCheckStyle)
	}
	ui.ticketForm.GetFormItemByLabel("Project:").(*tview.TextView).SetText(state.activeDirectory)
	ui.activeElement = ui.ticketForm
	ui.app.SetFocus(ui.activeElement)
}

func handleDeleteTicket(state *AppState, ui *UIComponents) {
	if state.activeFile == "" {
		return
	}
	core.DeleteTicket(state.activeFile)
	swapToDefaultPane(ui)
	refreshTree(ui.fileTree.GetRoot(), state.rootPath, state.activeDirectory)
	ui.app.SetFocus(ui.fileTree)
}

func handleDeleteProject(state *AppState, ui *UIComponents) {
	if state.activeDirectory == "" {
		return
	}
	core.DeleteProject(state.activeDirectory)
	swapToDefaultPane(ui)
	state.activeDirectory = ""
	refreshTree(ui.fileTree.GetRoot(), state.rootPath, state.activeDirectory)
	ui.app.SetFocus(ui.fileTree)
}

func handleNewProject(ui *UIComponents) {
	ui.layout.RemoveItem(ui.activeElement)
	ui.layout.AddItem(ui.newProjForm, 0, 2, true)
	ui.activeElement = ui.newProjForm
	ui.app.SetFocus(ui.activeElement)
}

func handleCloseTicket(state *AppState, ui *UIComponents) {
	if state.activeFile == "" {
		return
	}

	core.CloseTicket(state.activeFile)
	t, _ := core.GetTicket(state.activeFile)
	SwapToTicketBox(ui, t)
}

func submitNewTicket(ticketForm *tview.Form, activeDirectory string) {
	name := ticketForm.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
	details := ticketForm.GetFormItemByLabel("Details:").(*tview.TextArea).GetText()

	deps := collectDependencies(ticketForm, activeDirectory)

	status := core.STATUS_OPEN
	if len(deps) > 0 {
		status = core.STATUS_BLOCKED
	}

	ticket := core.Ticket{
		Name:         name,
		Detail:       details,
		Dependencies: deps,
		Status:       status,
	}

	core.SaveTicket(activeDirectory, ticket)
}

func collectDependencies(form *tview.Form, activeDirectory string) []string {
	deps := make([]string, 0)

	for i := range form.GetFormItemCount() {
		formItem := form.GetFormItem(i)
		checkbox, ok := formItem.(*tview.Checkbox)
		if !ok {
			continue
		}

		file := checkbox.GetLabel()
		path := filepath.Join(activeDirectory, file+".json")
		isDep := checkbox.IsChecked() && !core.IsClosed(path)

		if isDep {
			deps = append(deps, file)
		}
	}

	return deps
}

func populateTicketForm(ticketForm *tview.Form, activeDirectory string) {
	ticketForm.AddTextView("Project:", activeDirectory, 30, 1, false, false)
	ticketForm.AddInputField("Name:", "", 30, nil, nil).SetFieldStyle(fieldStyle)
	ticketForm.AddTextArea("Details:", "", 40, 5, 0, nil).SetFieldStyle(fieldStyle)
	ticketForm.SetBorder(true)
	ticketForm.SetTitle("New Ticket Form")
}

func resetTicketForm(ticketForm *tview.Form, activeDirectory string) {
	ticketForm.Clear(false)
	populateTicketForm(ticketForm, activeDirectory)
}

func swapToDefaultPane(ui *UIComponents) {
	ui.layout.RemoveItem(ui.activeElement)
	ui.layout.AddItem(ui.defaultActionBox, 0, 2, true)
	ui.activeElement = ui.defaultActionBox
}

func refreshTree(rootNode *tview.TreeNode, rootPath string, activeDirectory string) {
	rootNode.ClearChildren()
	populateFileTree(rootNode, rootPath, activeDirectory)
}

func getDirectoryFiles(dirPath string) []string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return []string{}
	}

	files := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files
}

func populateFileTree(node *tview.TreeNode, path string, activeDirectory string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Could not read directory:", path)
		return
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		childNode := tview.NewTreeNode(entry.Name()).
			SetReference(entryPath).
			SetSelectable(true)

		if entry.IsDir() {
			childNode.SetExpanded(entryPath == activeDirectory)
			childNode.SetTextStyle(dirTextStyle)
			childNode.SetSelectedTextStyle(dirSelectedStyle)
			node.AddChild(childNode)
			populateFileTree(childNode, entryPath, activeDirectory)
		} else {
			displayName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			childNode.SetText(displayName)
			childNode.SetTextStyle(fileTextStyle)
			childNode.SetSelectedTextStyle(fileSelectedStyle)
			node.AddChild(childNode)
		}
	}
}

func populateTicketBox(ticketBox *tview.Flex, ticket core.Ticket) {
	ticketBox.Clear()

	addLabeledField(ticketBox, "Name:", ticket.Name)
	addLabeledField(ticketBox, "Status:", statusText[ticket.Status])
	addLabeledField(ticketBox, "Details:", ticket.Detail)

	ticketBox.AddItem(tview.NewTextView().SetText("Dependencies:").SetTextStyle(headingStyle), 1, 1, false)
	for _, dep := range ticket.Dependencies {
		ticketBox.AddItem(tview.NewTextView().SetText(dep).SetTextStyle(defaultStyle), 1, 1, false)
	}
}

func addLabeledField(flex *tview.Flex, label string, value string) {
	flex.AddItem(tview.NewTextView().SetText(label).SetTextStyle(headingStyle), 1, 1, false)
	flex.AddItem(tview.NewTextView().SetText(value).SetTextStyle(defaultStyle), 2, 1, false)
}

func populateProjForm(form *tview.Form) {
	form.SetTitle("New Project")
	form.AddInputField("Name:", "", 30, nil, nil).SetFieldStyle(fieldStyle)
}
