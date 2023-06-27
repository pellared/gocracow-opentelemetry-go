package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("pgx", "postgres://postgres:pswd@localhost:5432/postgres")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connection to database: %v\n", err)
		os.Exit(1)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		tasks, err := listTasks()
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, t := range tasks {
			_, _ = fmt.Fprintf(rw, "%d. %s\n", t.id, t.description)
		}
	}).Methods("GET")

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		desc, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		err = addTask(string(desc))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}).Methods("POST")

	r.HandleFunc("/{id:[0-9]+}", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		desc, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		err = updateTask(int32(id), string(desc))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}).Methods("PUT")

	r.HandleFunc("/{id:[0-9]+}", func(rw http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		err = removeTask(int32(id))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8000", r))
}

type task struct {
	id          int32
	description string
}

func listTasks() ([]task, error) {
	var tasks []task
	rows, err := db.Query("SELECT id, description FROM tasks")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var t task
		if err := rows.Scan(&t.id, &t.description); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

func addTask(description string) error {
	_, err := db.Exec("INSERT INTO tasks(description) VALUES($1)", description)
	return err
}

func updateTask(itemNum int32, description string) error {
	_, err := db.Exec("UPDATE tasks SET description=$1 WHERE id=$2", description, itemNum)
	return err
}

func removeTask(itemNum int32) error {
	_, err := db.Exec("DELETE FROM tasks WHERE id=$1", itemNum)
	return err
}
