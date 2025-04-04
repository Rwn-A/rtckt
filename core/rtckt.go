package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Status int

const STATUS_OPEN = Status(0)
const STATUS_BLOCKED = Status(1)
const STATUS_CLOSED = Status(2)

type Ticket struct {
	Name         string   `json:"name"`
	Dependencies []string `json:"dependencies"`
	Status       Status   `json:"status"`
	Detail       string   `json:"detail"`
}

func Setup() (string, error) {
	user_home_directory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	root_path := path.Join(user_home_directory, "/rtckt")

	if err := os.Chdir(user_home_directory); err != nil {
		return "", err
	}

	if err := os.MkdirAll("rtckt", 0755); err != nil {
		return "", err
	}

	return root_path, nil
}

func NewProject(path string) {
	_ = os.Mkdir(path, 0755)
}

func GetTicket(path string) (Ticket, error) {
	fp, err := os.Open(path)
	if err != nil {
		return Ticket{}, fmt.Errorf("could not open file: %w", err)
	}
	defer fp.Close()

	var ticket Ticket
	decoder := json.NewDecoder(fp)
	if err := decoder.Decode(&ticket); err != nil {
		return Ticket{}, fmt.Errorf("could not decode JSON: %w", err)
	}

	return ticket, nil
}

func DeleteTicket(path string) {
	_ = os.Remove(path)
}

func DeleteProject(path string) {
	_ = os.RemoveAll(path)
}

func IsClosed(path string) bool {
	t, _ := GetTicket(path)
	return t.Status == STATUS_CLOSED
}

func CloseTicket(path string) {
	this_name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	t, err := GetTicket(path)
	if t.Status == STATUS_BLOCKED {
		return
	}
	if err != nil {
		panic(err)
	}

	fp, _ := os.Create("Debug.txt")
	defer fp.Close()

	t.Status = STATUS_CLOSED

	fmt.Fprintf(fp, "%s, %s\n", filepath.Dir(path), this_name)
	SaveTicket(filepath.Dir(path), t)

	possibly_deps := ReadWholeDirectory(filepath.Dir(path))

	//yes, this is not ideal. but it honestly doesnt seem to slow the app down.
	for _, pdep := range possibly_deps {
		t2, _ := GetTicket(pdep)
		if t2.Name == this_name {
			continue
		}
		for i, dep := range t2.Dependencies {
			fmt.Fprintf(fp, "%s, %s", this_name, dep)
			if this_name == dep {
				t2.Dependencies = removeUnordered(t2.Dependencies, i)
			}
		}
		if len(t2.Dependencies) <= 0 {
			if t2.Status == STATUS_BLOCKED {
				t2.Status = STATUS_OPEN
			}
		}
		SaveTicket(filepath.Dir(pdep), t2)
	}

}

func removeUnordered(slice []string, index int) []string {
	if index < 0 || index >= len(slice) {
		return slice
	}
	slice[index] = slice[len(slice)-1]
	return slice[:len(slice)-1]
}

func ReadWholeDirectory(path string) []string {
	entries, _ := os.ReadDir(path)
	filepaths := []string{}
	for _, entry := range entries {
		path := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			ReadWholeDirectory(path)
		} else {
			filepaths = append(filepaths, path)
		}
	}
	return filepaths
}

func SaveTicket(path string, t Ticket) error {
	save_path := filepath.Join(path, t.Name)
	file, err := os.Create(save_path + ".json")
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(t); err != nil {
		return err
	}

	return nil
}
