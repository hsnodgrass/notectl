package main

import (
	"bufio"
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

// DefaultEditor Default text editor for notes
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

func connectToDatabase(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	return database, nil
}

func createTableIfNotExist(database *sql.DB) error {
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, day INTEGER, month INTEGER, year INTEGER, timestamp INTEGER, notetext BLOB, tags TEXT)")
	statement.Exec()
	return nil
}

func (n *note) Save(database *sql.DB) error {
	statement, _ := database.Prepare("INSERT INTO notes (day, month, year, timestamp, notetext, tags) VALUES (?, ?, ?, ?, ?, ?)")
	statement.Exec(n.Time.Day(), n.Time.Month(), n.Time.Year(), n.Time.Unix(), n.Text, n.Tags.String())
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

func showAllNotes(database *sql.DB) error {
	rows, _ := database.Query("SELECT * FROM notes")
	printRows(rows)
	return nil
}

func showNoteByID(id int, database *sql.DB) error {
	rows, _ := database.Query("SELECT * FROM notes WHERE id = (?)", id)
	printRows(rows)
	return nil
}

// Defaults to this month and this year
func showNoteByDay(day int, database *sql.DB) error {
	rows, _ := database.Query("SELECT * FROM notes WHERE day = (?) AND month = (?) AND year = (?)", day, time.Now().Month(), time.Now().Year())
	printRows(rows)
	return nil
}

// Defaults to this year
func showNoteByMonth(month int, database *sql.DB) error {
	rows, _ := database.Query("SELECT * FROM notes WHERE month = (?) AND year = (?)", month, time.Now().Year())
	printRows(rows)
	return nil
}

func showNoteByYear(year int, database *sql.DB) error {
	rows, _ := database.Query("SELECT * FROM notes WHERE year = (?)", year)
	printRows(rows)
	return nil
}

func showNoteByDate(date string, usa bool, database *sql.DB) error {
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
	rows, _ := database.Query("SELECT * FROM notes WHERE day = (?) AND month = (?) AND year = (?)", day, month, year)
	printRows(rows)
	return nil
}

func deleteAll(database *sql.DB) error {
	fmt.Println("Are you sure you want to delete all notes? (y/n)")
	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		panic(err)
	}
	if char == 'y' || char == 'Y' {
		fmt.Println("Deleting all notes...")
		statement, _ := database.Prepare("DROP TABLE notes")
		statement.Exec()
		createTableIfNotExist(database)
	} else {
		fmt.Println("Not deleting notes, everything is still there.")
	}
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
	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)

	var newTagList tagList
	newNotePtr := newCommand.String("n", "", "Note text.")
	newEditorNotePtr := newCommand.Bool("e", false, "Create a new file with a text editor.")
	newCommand.Var(&newTagList, "t", "A comma-delimited list of tags.")

	showAllPtr := showCommand.Bool("all", false, "Show all notes.")
	showByIDPtr := showCommand.Int("i", -1, "Show a note based of the ID it has assigned to it.")
	showByDayPtr := showCommand.Int("day", -1, "Show notes from the specified day of the current month and year.")
	showByMonthPtr := showCommand.Int("month", -1, "Show notes from the specified month of the current year.")
	showByYearPtr := showCommand.Int("year", -1, "Show notes from the specified year.")
	showByDatePtr := showCommand.String("date", "", "Show notes by date in the format <d>/<m>/<y>.")
	showUSADatePtr := showCommand.Bool("usa", false, "Allows for searching by date in US format <m>/<d>/<y>.")

	deleteAllPtr := deleteCommand.Bool("all", false, "Delete all stored notes.")

	if len(os.Args) < 2 {
		fmt.Println("subcommand required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		newCommand.Parse(os.Args[2:])
	case "show":
		showCommand.Parse(os.Args[2:])
	case "delete":
		deleteCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if newCommand.Parsed() {
		database, err := connectToDatabase(dbpath)
		if err != nil {
			panic(err)
		}
		createTableIfNotExist(database)
		if *newNotePtr == "" && newCommand.NFlag() > 0 && !*newEditorNotePtr {
			newCommand.PrintDefaults()
			os.Exit(1)
		}
		if len(newTagList) == 0 {
			newTagList.Set("generic")
		}
		// We default to opening a text editor if there are no flags and no extra args
		if newCommand.NFlag() == 0 || *newEditorNotePtr {
			if len(os.Args[2:]) == 0 || *newEditorNotePtr {
				noteValBytes, err := captureFromEditor()
				if err != nil {
					panic(err)
				}
				noteValString := bytes.NewBuffer(noteValBytes).String()
				*newNotePtr = noteValString
			} else {
				noteVal := strings.Join(newCommand.Args(), " ")
				*newNotePtr = noteVal
			}
		}
		timeStamp := time.Now()
		note := note{Time: timeStamp, Text: *newNotePtr, Tags: newTagList}
		note.PrintConsole()
		note.Save(database)
		database.Close()
	}

	if showCommand.Parsed() {
		database, err := connectToDatabase(dbpath)
		if err != nil {
			panic(err)
		}
		createTableIfNotExist(database)
		if *showAllPtr {
			showAllNotes(database)
		} else if *showByIDPtr != -1 {
			showNoteByID(*showByIDPtr, database)
		} else if *showByDayPtr != -1 {
			showNoteByDay(*showByDayPtr, database)
		} else if *showByMonthPtr != -1 {
			showNoteByMonth(*showByMonthPtr, database)
		} else if *showByYearPtr != -1 {
			showNoteByYear(*showByYearPtr, database)
		} else if *showByDatePtr != "" {
			showNoteByDate(*showByDatePtr, *showUSADatePtr, database)
		} else {
			showCommand.PrintDefaults()
			os.Exit(1)
		}
		database.Close()
	}

	if deleteCommand.Parsed() {
		database, err := connectToDatabase(dbpath)
		if err != nil {
			panic(err)
		}
		createTableIfNotExist(database)
		if *deleteAllPtr {
			deleteAll(database)
		} else {
			deleteCommand.PrintDefaults()
			os.Exit(1)
		}
		database.Close()
	}
}
