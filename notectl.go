package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

func (n *note) Save(path string) error {
	database, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, time INTEGER, notetext BLOB, tags TEXT)")
	statement.Exec()
	statement, _ = database.Prepare("INSERT INTO notes (time, notetext, tags) VALUES (?, ?, ?)")
	statement.Exec(n.Time.Unix(), n.Text, n.Tags.String())
	database.Close()
	return nil
}

func showAllNotes(path string) error {
	database, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	rows, _ := database.Query("SELECT * FROM notes")
	var id int
	var _time int
	var notetext string
	var tags string
	for rows.Next() {
		rows.Scan(&id, &_time, &notetext, &tags)
		fmt.Println(strconv.Itoa(id) + " - " + time.Unix(int64(_time), 0).Format(time.RFC822) + ": " + notetext + ", Tags: " + tags)
	}
	return nil
}

func main() {
	dbpath := fmt.Sprintf("%s/notectl.db", os.Getenv("HOME"))

	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	showCommand := flag.NewFlagSet("show", flag.ExitOnError)

	var newTagList tagList
	notePtr := newCommand.String("n", "", "Note text.")
	newCommand.Var(&newTagList, "t", "A comma-delimited list of tags.")

	showAllPtr := showCommand.Bool("all", false, "Show all notes.")

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
		if *notePtr == "" && newCommand.NFlag() > 0 {
			newCommand.PrintDefaults()
			os.Exit(1)
		}
		if len(newTagList) == 0 {
			newTagList.Set("generic")
		}
		if newCommand.NFlag() == 0 {
			noteVal := strings.Join(newCommand.Args(), " ")
			*notePtr = noteVal
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
	}
}
