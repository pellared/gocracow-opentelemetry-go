package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

const url = "http://localhost:8000"

var client *http.Client = http.DefaultClient

func main() {
	if len(os.Args) == 1 {
		printHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "list":
		if len(os.Args) != 2 {
			fmt.Fprintln(os.Stderr, "Invalid usage")
			printHelp()
			os.Exit(1)
		}
		err := listTasks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to list tasks: %v\n", err)
			os.Exit(1)
		}

	case "add":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Invalid usage")
			printHelp()
			os.Exit(1)
		}
		err := addTask(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to add task: %v\n", err)
			os.Exit(1)
		}

	case "update":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "Invalid usage")
			printHelp()
			os.Exit(1)
		}
		n, err := strconv.ParseInt(os.Args[2], 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable convert task_num into int32: %v\n", err)
			os.Exit(1)
		}
		err = updateTask(int32(n), os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to update task: %v\n", err)
			os.Exit(1)
		}

	case "remove":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Invalid usage")
			printHelp()
			os.Exit(1)
		}
		n, err := strconv.ParseInt(os.Args[2], 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable convert task_num into int32: %v\n", err)
			os.Exit(1)
		}
		err = removeTask(int32(n))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to remove task: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintln(os.Stderr, "Invalid command")
		printHelp()
		os.Exit(1)
	}
}

func listTasks() error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New("HTTP: " + resp.Status)
	}

	defer resp.Body.Close()
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func addTask(description string) error {
	var buf bytes.Buffer
	buf.WriteString(description)
	resp, err := client.Post(url, "text/plain", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New("HTTP: " + resp.Status)
	}
	return nil
}

func updateTask(itemNum int32, description string) error {
	var buf bytes.Buffer
	buf.WriteString(description)
	req, _ := http.NewRequest("PUT", url+"/"+strconv.Itoa(int(itemNum)), &buf)
	req.Header.Set("Content-Type", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New("HTTP: " + resp.Status)
	}
	return nil
}

func removeTask(itemNum int32) error {
	req, _ := http.NewRequest("DELETE", url+"/"+strconv.Itoa(int(itemNum)), nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New("HTTP: " + resp.Status)
	}
	return nil
}

func printHelp() {
	fmt.Print(`TODO CLI application
Usage:
  todo list
  todo add task
  todo update task_num item
  todo remove task_num
Example:
  todo add 'Learn Go'
  todo list
`)
}
