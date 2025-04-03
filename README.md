# rtckt
A TUI for managing a ticket based todo app. Developed for my own use.

> [!WARNING]
> This software is currently buggy, unpolished and the code quality is sub-par. Use at your own risk.

## Current Functionality
1. **Create projects:** A project is a collection of tickets, and/or sub-projects
2. **Create tickets:** A ticket is either open, blocked or closed. Blocked inidcates it depends on other tickets. A ticket contains a name, status (open, blocked, closed), its dependencies and some optional extra text information.
3. **Close tickets:** When a ticket is closed the other tickets will no longer be blocked by it.
4. **Delete Projects & Tickets:** This will delete the files, it will break any dependency links. Use for closed tickets no longer needed, finished projects, or to clean up mistakes.

## Keybinds
**These are not great, and will change.**
-  Ctrl-O: Open ticket
-  Ctrl-D: Delete ticket
-  Ctrl-R: Close ticket (Will not close if blocked)
-  Ctrl-P: New Project
-  Ctrl-B: Delete Project

## Installation
Install like any other go module. To build the tui...
```sh
$ go build -o rtckt cmd/main.go #build
$ ./rtckt  #run the TUI
```
By default the files are stored in your `$HOME/rtckt` directory, or the windows equivalent. 