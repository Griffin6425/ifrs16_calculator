package main

import (
	"fmt"
	"log"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	// Create a new Excel file
	f := excelize.NewFile()

	// Create a sheet
	sheetName := "Leases"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		log.Fatalf("Error creating sheet: %v", err)
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1") // Delete default sheet

	// Set headers
	headers := []string{"ID", "StartDate", "EndDate", "PaymentAmount", "PaymentFrequency", "DiscountRate"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Set some example data
	exampleData := [][]interface{}{
		{"L001", "2023-01-01", "2027-12-31", 5000, "Monthly", 0.05},
		{"L002", "2023-02-01", "2026-01-31", 10000, "Quarterly", 0.045},
	}

	for i, row := range exampleData {
		for j, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+j, i+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// Create directory if it doesn't exist
	os.MkdirAll("web/static/templates", 0755)

	// Save the file
	if err := f.SaveAs("web/static/templates/lease_template.xlsx"); err != nil {
		log.Fatalf("Error saving Excel template: %v", err)
	}

	fmt.Println("Excel template created successfully at web/static/templates/lease_template.xlsx")
}
