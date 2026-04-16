package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/tfior/doc-tracker/internal/users"
	"github.com/tfior/doc-tracker/platform"
	"golang.org/x/term"
)

func main() {
	cfg := platform.LoadConfig()

	db, err := platform.OpenDatabase(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	scanner := bufio.NewScanner(os.Stdin)

	email := prompt(scanner, "Email: ")
	firstName := prompt(scanner, "First name: ")
	lastName := prompt(scanner, "Last name: ")

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		log.Fatalf("read password: %v", err)
	}
	password := strings.TrimSpace(string(passwordBytes))

	if email == "" || firstName == "" || lastName == "" || password == "" {
		log.Fatal("all fields are required")
	}

	svc := users.NewService(users.NewStore(db))
	user, err := svc.Create(context.Background(), email, firstName, lastName, password)
	if err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Printf("Created user %s (%s %s)\n", user.Email, user.FirstName, user.LastName)
}

func prompt(scanner *bufio.Scanner, label string) string {
	fmt.Print(label)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
