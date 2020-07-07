package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const DefaultEditor = "vi"

type tagList []string

func (s *tagList) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *tagList) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

type note struct {
	Time time.Time
	Text string
	Tags tagList
}

func (n *note) PrintConsole() error {
	fmt.Printf("%s : Saving note \"%s\", tags: %s\n", n.Time.Format(time.RFC822), n.Text, n.Tags.String())
	return nil
}

func connectToDatabase(path string) (sql.DB, error) {
	database, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	return *database, nil
}

func (n *note) Save(path string) error {
	database, _ := connectToDatabase(path)
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, day INTEGER, month INTEGER, year INTEGER, timestamp INTEGER, notetext BLOB, tags TEXT)")
	statement.Exec()
	statement, _ = database.Prepare("INSERT INTO notes (day, month, year, timestamp, notetext, tags) VALUES (?, ?, ?, ?, ?, ?)")
	statement.Exec(n.Time.Day(), n.Time.Month(), n.Time.Year(), n.Time.Unix(), n.Text, n.Tags.String())
	database.Close()
	return nil
}

func printRows(rows *sql.Rows) error {
	var id int
	var day int
	var month string
	var year int
	var timestamp int
	var notetext string
	var tags string
	for rows.Next() {
		rows.Scan(&id, &day, &month, &year, &timestamp, &notetext, &tags)
		fmt.Printf("%d - %s: %s, tags: %s\n", id, time.Unix(int64(timestamp), 0).Format(time.RFC822), notetext, tags)
	}
	return nil
}

func showAllNotes(path string) error {
	database, _ := connectToDatabase(path)
	rows, _ := database.Query("SELECT * FROM notes")
	database.Close()
	printRows(rows)
	return nil
}

func showNoteByID(id int, path string) error {
	database, _ := connectToDatabase(path)
	rows, _ := database.Query("SELECT * FROM notes WHERE id = (?)", id)
	database.Close()
	printRows(rows)
	return nil
}

// Defaults to this month and this year
func showNoteByDay(day int, path string) error {
	database, _ := connectToDatabase(path)
	rows, _ := database.Query("SELECT * FROM notes WHERE day = (?) AND month = (?) AND year = (?)", day, time.Now().Month(), time.Now().Year())
	database.Close()
	printRows(rows)
	return nil
}

// Defaults to this year
func showNoteByMonth(month int, path string) error {
	database, _ := connectToDatabase(path)
	rows, _ := database.Query("SELECT * FROM notes WHERE month = (?) AND year = (?)", month, time.Now().Year())
	database.Close()
	printRows(rows)
	return nil
}

func showNoteByYear(year int, path string) error {
	database, _ := connectToDatabase(path)
	rows, _ := database.Query("SELECT * FROM notes WHERE year = (?)", year)
	database.Close()
	printRows(rows)
	return nil
}

func showNoteByDate(date string, path string, usa bool) error {
	d := strings.Split(date, "/")
	var day int
	var month int
	var year int
	if usa {
		day, _ = strconv.Atoi(d[1])
		month, _ = strconv.Atoi(d[0])
		year, _ = strconv.Atoi(d[2])
	} else {
		day, _ = strconv.Atoi(d[0])
		month, _ = strconv.Atoi(d[1])
		year, _ = strconv.Atoi(d[2])
	}
	database, _ := connectToDatabase(path)
	rows, _ := database.Query("SELECT * FROM notes WHERE day = (?) AND month = (?) AND year = (?)", day, month, year)
	database.Close()
	printRows(rows)
	return nil
}

func openFileInEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = DefaultEditor
	}

	executable, err := exec.LookPath(editor)
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func captureFromEditor() ([]byte, error) {
	file, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return []byte{}, err
	}

	filename := file.Name()

	defer os.Remove(filename)

	if err = file.Close(); err != nil {
		return []byte{}, err
	}

	if err = openFileInEditor(filename); err != nil {
		return []byte{}, err
	}

	bytes, err := ioutil.ReadFile(filename)

	return bytes, nil
}

func main() {
	dbpath := fmt.Sprintf("%s/notectl.db", os.Getenv("HOME"))

	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	showCommand := flag.NewFlagSet("show", flag.ExitOnError)

	var newTagList tagList
	notePtr := newCommand.String("n", "", "Note text.")
	editorPtr := newCommand.Bool("e", false, "Create a new file with a text editor.")
	newCommand.Var(&newTagList, "t", "A comma-delimited list of tags.")

	showAllPtr := showCommand.Bool("all", false, "Show all notes.")
	showByIDPtr := showCommand.Int("i", -1, "Show a note based of the ID it has assigned to it.")
	showByDayPtr := showCommand.Int("day", -1, "Show notes from the specified day of the current month and year.")
	showByMonthPtr := showCommand.Int("month", -1, "Show notes from the specified month of the current year.")
	showByYearPtr := showCommand.Int("year", -1, "Show notes from the specified year.")
	showByDatePtr := showCommand.String("date", "", "Show notes by date in the format <d>/<m>/<y>.")
	showUSADatePtr := showCommand.Bool("usa", false, "Allows for searching by date in US format <m>/<d>/<y>.")

	if len(os.Args) < 2 {
		fmt.Println("subcommand required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		newCommand.Parse(os.Args[2:])
	case "show":
		showCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if newCommand.Parsed() {
		if *notePtr == "" && newCommand.NFlag() > 0 && !*editorPtr {
			newCommand.PrintDefaults()
			os.Exit(1)
		}
		if len(newTagList) == 0 {
			newTagList.Set("generic")
		}
		// We default to opening a text editor if there are no flags and no extra args
		if newCommand.NFlag() == 0 || *editorPtr {
			if len(os.Args[2:]) == 0 || *editorPtr {
				noteValBytes, err := captureFromEditor()
				if err != nil {
					panic(err)
				}
				noteValString := bytes.NewBuffer(noteValBytes).String()
				*notePtr = noteValString
			} else {
				noteVal := strings.Join(newCommand.Args(), " ")
				*notePtr = noteVal
			}
		}
		timeStamp := time.Now()
		note := note{Time: timeStamp, Text: *notePtr, Tags: newTagList}
		note.PrintConsole()
		note.Save(dbpath)
	}

	if showCommand.Parsed() {
		if *showAllPtr {
			showAllNotes(dbpath)
		}
		if *showByIDPtr != -1 {
			showNoteByID(*showByIDPtr, dbpath)
		}
		if *showByDayPtr != -1 {
			showNoteByDay(*showByDayPtr, dbpath)
		}
		if *showByMonthPtr != -1 {
			showNoteByMonth(*showByMonthPtr, dbpath)
		}
		if *showByYearPtr != -1 {
			showNoteByYear(*showByYearPtr, dbpath)
		}
		if *showByDatePtr != "" {
			showNoteByDate(*showByDatePtr, dbpath, *showUSADatePtr)
		}
	}
}
