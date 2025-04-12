# IFRS 16 Lease Calculator

A modern web application for calculating lease values and schedules according to IFRS 16 standards.

## Features

- Upload lease data in CSV or Excel format
- Calculate initial lease liability and right-of-use asset values
- Generate amortization schedules for both lease liability and RoU asset
- Export results to Excel for reporting and analysis
- Clean, minimalist Notion-inspired user interface

## Getting Started

### Prerequisites

- Go 1.16 or higher
- Web browser

### Installation

1. Clone the repository:
   ```
   git clone [repository-url]
   cd ifrs_16_golang
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Run the application:
   ```
   go run cmd/server/main.go
   ```

4. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

## Usage

1. Prepare your lease data in CSV or Excel format with the following columns:
   - ID - Unique lease identifier
   - StartDate - Lease start date (YYYY-MM-DD)
   - EndDate - Lease end date (YYYY-MM-DD)
   - PaymentAmount - Regular payment amount
   - PaymentFrequency - Payment frequency (Monthly, Quarterly, or Annually)
   - DiscountRate - Incremental borrowing rate as decimal (e.g., 0.05 for 5%)

2. Navigate to the Calculate page and upload your file

3. Review the calculation results displayed on screen

4. Export the results to Excel for reporting and further analysis

## Project Structure

```
ifrs_16_golang/
├── cmd/
│   └── server/
│       └── main.go           # Main application server
├── internal/
│   ├── calculation/          # IFRS 16 calculation logic
│   ├── lease/                # Lease data structures
│   └── platform/
│       ├── export/           # Excel export functionality
│       └── parsing/          # File parsing logic
└── web/
    ├── static/               # Static assets (CSS, JS, templates)
    └── templates/            # HTML templates
```

## API Endpoints

- `GET /` - Home page
- `GET /calculate` - Lease calculation page
- `POST /calculate` - API endpoint for calculation
- `POST /export` - API endpoint for Excel export
- `GET /documentation` - Documentation page

## Built With

- [Go](https://golang.org/) - Backend language
- [Excelize](https://github.com/xuri/excelize) - Excel file handling
- HTML/CSS/JavaScript - Frontend technologies

## License

This project is licensed under the MIT License - see the LICENSE file for details. 