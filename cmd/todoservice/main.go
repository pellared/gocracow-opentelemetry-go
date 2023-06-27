package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/XSAM/otelsql"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	todootel "todo/otel"
)

var db *sql.DB

func main() {
	// handle CTRL+C gracefully
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shutdown, err := todootel.Run(ctx, "todo-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run OpenTelemetry: %v\n", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shutdown OpenTelemetry: %v\n", err)
		}
	}()

	// Instrument database/sql.
	db, err = otelsql.Open("pgx", "postgres://postgres:pswd@localhost:5432/postgres", otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if err != nil {
		log.Fatalf("Unable to connection to database: %v\n", err)
	}
	defer db.Close()

	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if err != nil {
		log.Fatalf("Unable to RegisterDBStatsMetrics: %v\n", err)
	}

	// Add a custom metric.
	meter := otel.Meter("todoservice")
	taskCnt, err := meter.Int64UpDownCounter("task.count")
	if err != nil {
		log.Fatalf("Unable to create list_tasks metrics counter: %v\n", err)
	}

	r := mux.NewRouter()

	// Instrument gorilla/mux with OpenTelemetry tracing.
	r.Use(otelmux.Middleware("mux-server"))

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		tasks, err := listTasks(r.Context())
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
		err = addTask(r.Context(), string(desc))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		taskCnt.Add(context.Background(), -1)
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
		err = updateTask(r.Context(), int32(id), string(desc))
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
		err = removeTask(r.Context(), int32(id))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		taskCnt.Add(context.Background(), -1)
	}).Methods("DELETE")

	srv := &http.Server{Addr: ":8000", Handler: r}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

type task struct {
	id          int32
	description string
}

func listTasks(ctx context.Context) ([]task, error) {
	var tasks []task
	rows, err := db.QueryContext(ctx, "SELECT id, description FROM tasks")
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

func addTask(ctx context.Context, description string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO tasks(description) VALUES($1)", description)
	return err
}

func updateTask(ctx context.Context, itemNum int32, description string) error {
	_, err := db.ExecContext(ctx, "UPDATE tasks SET description=$1 WHERE id=$2", description, itemNum)
	return err
}

func removeTask(ctx context.Context, itemNum int32) error {
	_, err := db.ExecContext(ctx, "DELETE FROM tasks WHERE id=$1", itemNum)
	return err
}
