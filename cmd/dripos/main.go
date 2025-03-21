package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kieracarman/dripos-demo/internal/inventory"
	"github.com/kieracarman/dripos-demo/internal/loadtest"
	"github.com/kieracarman/dripos-demo/internal/menu"
	"github.com/kieracarman/dripos-demo/internal/payment"
)

func main() {
	fmt.Println("=== DripOS Go Backend Demo ===")
	fmt.Println("Choose a demo to run:")
	fmt.Println("1. Payment Processor")
	fmt.Println("2. Menu Cache System")
	fmt.Println("3. Load Testing Tool")
	fmt.Println("4. Inventory Syncer")
	fmt.Println("5. Run All Demos")
	fmt.Println("q. Quit")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nEnter your choice: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			fmt.Println("\nRunning Payment Processor Demo...")
			payment.RunDemo()
		case "2":
			fmt.Println("\nRunning Menu Cache Demo...")
			menu.RunDemo()
		case "3":
			fmt.Println("\nRunning Load Testing Demo...")
			loadtest.RunDemo()
		case "4":
			fmt.Println("\nRunning Inventory Syncer Demo...")
			inventory.RunDemo()
		case "5":
			fmt.Println("\nRunning All Demos...\n")
			fmt.Println("\n=== Payment Processor Demo ===")
			payment.RunDemo()
			fmt.Println("\n=== Menu Cache Demo ===")
			menu.RunDemo()
			fmt.Println("\n=== Load Testing Demo ===")
			loadtest.RunDemo()
			fmt.Println("\n=== Inventory Syncer Demo ===")
			inventory.RunDemo()
		case "q", "Q", "quit", "exit":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}
