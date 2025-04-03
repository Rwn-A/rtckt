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

var active_action_element tview.Primitive
var active_directory string
var active_file string

func main() {
	root_path, err := core.Setup()
	if err != nil {
		fmt.Println(err)
		return
	}

	app := tview.NewApplication()

	//config
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	//file viewer
	root_node := tview.NewTreeNode("Root Directory").SetReference(root_path)
	root_node.SetTextStyle(tcell.Style{}.Foreground(tcell.ColorYellow))
	root_node.SetSelectedTextStyle(tcell.Style{}.Foreground(tcell.ColorYellow))
	populateFileTree(root_node, root_path)
	file_tree := tview.NewTreeView().SetRoot(root_node).SetCurrentNode(root_node)
	_ = file_tree.SetBorder(true) //chaining this method turns the tree into a box so cant chain it

	//default action box
	default_action_box := tview.NewTextView()
	default_action_box.SetBorder(true)
	default_action_box.SetTextAlign(tview.AlignCenter)
	default_action_box.SetText("Select a ticket to view/edit")
	active_action_element = default_action_box

	layout := tview.NewFlex().AddItem(file_tree, 0, 1, true).AddItem(default_action_box, 0, 2, false) // 3/4 width for the other box

	//view/edit ticket box
	ticket_action_box := tview.NewFlex().SetDirection(tview.FlexRow)
	ticket_action_box.SetBorder(true)

	//new ticket form
	ticketForm := tview.NewForm()
	populateTicketForm(ticketForm)

	//new project form
	newProjForm := tview.NewForm()
	populateProjForm(newProjForm)

	button_style := tcell.Style{}.Background(tcell.ColorGray)
	button_style_active := tcell.Style{}.Background(tcell.ColorRed)

	newProjForm.AddButton("Create", func() {
		name := newProjForm.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
		if active_directory == "" {
			core.NewProject(filepath.Join(root_path, name))
		} else {
			core.NewProject(filepath.Join(active_directory, name))
		}
		swapToDefaultPane(app, default_action_box, layout)
		refreshTree(root_node, root_path)
		newProjForm.Clear(false)
		populateProjForm(newProjForm)
		app.SetFocus(file_tree)
	}).SetButtonStyle(button_style).SetButtonActivatedStyle(button_style_active)
	newProjForm.AddButton("Cancel", func() {
		swapToDefaultPane(app, default_action_box, layout)
		newProjForm.Clear(false)
		populateProjForm(newProjForm)
		app.SetFocus(file_tree)
	}).SetButtonStyle(button_style).SetButtonActivatedStyle(button_style_active)

	ticketForm.AddButton("Save", func() {
		submitNewTicket(ticketForm)
		swapToDefaultPane(app, default_action_box, layout)
		refreshTree(root_node, root_path)
		app.SetFocus(file_tree)
		resetTicketForm(ticketForm)
	}).SetButtonStyle(button_style).SetButtonActivatedStyle(button_style_active)
	ticketForm.AddButton("Cancel", func() {
		swapToDefaultPane(app, default_action_box, layout)
		resetTicketForm(ticketForm)
		app.SetFocus(file_tree)
	}).SetButtonStyle(button_style).SetButtonActivatedStyle(button_style_active)

	//handle selections
	file_tree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		path := reference.(string)
		if reference == nil {
			active_directory = ""
			return // Selecting the root node does nothing.
		}
		info, _ := os.Stat(path)

		if !info.IsDir() { //this indicates a file
			ticket, err := core.GetTicket(path)
			if err != nil {
				fmt.Println(err)
				return
			}
			active_file = path
			layout.RemoveItem(active_action_element)
			ticket_action_box.SetTitle(ticket.Name)
			layout.AddItem(ticket_action_box, 0, 2, true)
			active_action_element = ticket_action_box
			populateTicketBox(ticket_action_box, ticket)
		} else {
			active_file = ""
			active_directory = path
			node.SetExpanded(!node.IsExpanded())
		}
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlO:
			if active_directory == "" {
				break
			}

			layout.RemoveItem(active_action_element)
			layout.AddItem(ticketForm, 0, 2, true)
			for _, file := range readWholeDirectory(active_directory) {
				lbl := strings.TrimSuffix(file, filepath.Ext(file))
				ticketForm.AddCheckbox(lbl, false, nil)
				focused_style := tcell.Style{}.Blink(true)
				ticketForm.GetFormItemByLabel(lbl).(*tview.Checkbox).SetUncheckedString("☐").SetActivatedStyle(focused_style)
			}
			ticketForm.GetFormItemByLabel("Project:").(*tview.TextView).SetText(active_directory)
			active_action_element = ticketForm
			app.SetFocus(active_action_element)
		case tcell.KeyCtrlD:
			if active_file == "" {
				break
			}
			core.DeleteTicket(active_file)
			swapToDefaultPane(app, default_action_box, layout)
			refreshTree(root_node, root_path)
			app.SetFocus(file_tree)
		case tcell.KeyCtrlB:
			if active_directory == "" {
				break
			}
			core.DeleteProject(active_directory)
			swapToDefaultPane(app, default_action_box, layout)
			active_directory = ""
			refreshTree(root_node, root_path)
			app.SetFocus(file_tree)
		case tcell.KeyCtrlP:
			layout.RemoveItem(active_action_element)
			layout.AddItem(newProjForm, 0, 2, true)
			active_action_element = newProjForm
			app.SetFocus(active_action_element)
		case tcell.KeyCtrlR:
			if active_file == "" {
				break
			}
			core.CloseTicket(active_file)
			swapToDefaultPane(app, default_action_box, layout)
		}
		return event
	})
	app.SetRoot(layout, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func submitNewTicket(ticketForm *tview.Form) {
	name := ticketForm.GetFormItemByLabel("Name:").(*tview.InputField).GetText()
	details := ticketForm.GetFormItemByLabel("Details:").(*tview.TextArea).GetText()

	deps := make([]string, 0)
	for i := 0; i < ticketForm.GetFormItemCount(); i += 1 {
		form_item := ticketForm.GetFormItem(i)
		if checkbox, ok := form_item.(*tview.Checkbox); ok {
			file := checkbox.GetLabel()
			path := filepath.Join(active_directory, file+".json")
			is_dep := checkbox.IsChecked() && !core.IsClosed(path)
			if is_dep {
				deps = append(deps, file)
			}
		}
	}

	status := core.STATUS_OPEN
	if len(deps) > 0 {
		status = core.STATUS_BLOCKED
	}

	t := core.Ticket{
		Name:         name,
		Detail:       details,
		Dependencies: deps,
		Status:       status,
	}

	core.SaveTicket(active_directory, t)
}

func populateTicketForm(ticketForm *tview.Form) {
	field_style := tcell.Style{}.Underline(true)
	ticketForm.AddTextView("Project:", active_directory, 30, 1, false, false)
	ticketForm.AddInputField("Name:", "", 30, nil, nil).SetFieldStyle(field_style)
	ticketForm.AddTextArea("Details:", "", 40, 5, 0, nil).SetFieldStyle(field_style)
	ticketForm.SetBorder(true)
	ticketForm.SetTitle("New Ticket Form")
}

func resetTicketForm(ticketForm *tview.Form) {
	ticketForm.Clear(false)
	populateTicketForm(ticketForm)
}

func swapToDefaultPane(app *tview.Application, defaultBox *tview.TextView, layout *tview.Flex) {
	layout.RemoveItem(active_action_element)
	layout.AddItem(defaultBox, 0, 2, true)
	active_action_element = defaultBox
}

func refreshTree(root_node *tview.TreeNode, root_path string) {
	root_node.ClearChildren()
	populateFileTree(root_node, root_path)
}

func readWholeDirectory(path string) []string {
	entries, _ := os.ReadDir(active_directory)

	files := []string{}
	for _, entry := range entries {
		path := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			readWholeDirectory(path)
		} else {
			files = append(files, entry.Name())
		}
	}
	return files
}

func populateFileTree(node *tview.TreeNode, path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("could not read directory:", path)
		return
	}
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		childNode := tview.NewTreeNode(entry.Name()).
			SetReference(entryPath).
			SetSelectable(true)

		if entry.IsDir() {
			if entryPath == active_directory {
				childNode.SetExpanded(true)
			} else {
				childNode.SetExpanded(false)
			}

			childNode.SetTextStyle(tcell.Style{}.Foreground(tcell.ColorWhiteSmoke))
			childNode.SetSelectedTextStyle(tcell.Style{}.Underline(true).Foreground(tcell.ColorRed))
		} else {
			childNode.SetText(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			childNode.SetTextStyle(tcell.Style{}.Foreground(tcell.ColorAquaMarine))
			childNode.SetSelectedTextStyle(tcell.Style{}.Underline(true).Foreground(tcell.ColorGreen))
		}

		node.AddChild(childNode)

		if entry.IsDir() {
			populateFileTree(childNode, entryPath)
		}
	}
}

func populateTicketBox(node *tview.Flex, ticket core.Ticket) {
	statusText := map[core.Status]string{
		0: "Open",
		1: "Blocked",
		2: "Closed",
	}

	default_style := tcell.Style{}
	heading_style := tcell.Style{}.Underline(true)
	node.Clear()
	node.AddItem(tview.NewTextView().SetText("Name:").SetTextStyle(heading_style), 1, 1, false)
	node.AddItem(tview.NewTextView().SetText(ticket.Name).SetTextStyle(default_style), 2, 1, false)
	node.AddItem(tview.NewTextView().SetText("Status:").SetTextStyle(heading_style), 1, 1, false)
	node.AddItem(tview.NewTextView().SetText(statusText[ticket.Status]).SetTextStyle(default_style), 2, 1, false)
	node.AddItem(tview.NewTextView().SetText("Details:").SetTextStyle(heading_style), 1, 1, false)
	node.AddItem(tview.NewTextView().SetText(ticket.Detail).SetTextStyle(default_style), 2, 1, false)
	node.AddItem(tview.NewTextView().SetText("Dependencies:").SetTextStyle(heading_style), 1, 1, false)
	for _, v := range ticket.Dependencies {
		node.AddItem(tview.NewTextView().SetText(v).SetTextStyle(default_style), 1, 1, false)
	}
}

func populateProjForm(form *tview.Form) {
	field_style := tcell.Style{}.Underline(true)
	form.SetTitle("New Project")
	form.AddInputField("Name:", "", 30, nil, nil).SetFieldStyle(field_style)
}
